package externals

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.kreikol.dev/dotsmithy/internal/manifest"
	"go.kreikol.dev/dotsmithy/internal/stow"
)

// httpClient con timeout para las descargas de externals de tipo file.
var httpClient = &http.Client{Timeout: 60 * time.Second}

// Context es el contexto que se pasa al post de cada external (mismo contrato
// que los hooks, más DOTS_EXTERNAL_DEST por external).
type Context struct {
	Profile    string
	ContentDir string
	Target     string
	Layers     []string
}

// Apply procesa todos los externals en modo best-effort: por cada uno hace el
// fetch y, si es nuevo o ha cambiado, ejecuta su post. Si uno falla (fetch o
// post), lo avisa y sigue con el resto. Devuelve cuántos fallaron. En dry-run
// solo informa. log puede ser nil.
func Apply(exts []manifest.External, ctx Context, dryRun bool, log func(string)) int {
	if log == nil {
		log = func(string) {}
	}
	failed := 0
	for _, ext := range exts {
		if err := applyOne(ext, ctx, dryRun, log); err != nil {
			log(fmt.Sprintf("! %s: %v (sigo con el resto)", ext.Dest, err))
			failed++
		}
	}
	return failed
}

// applyOne procesa un external: fetch + post si cambió.
func applyOne(ext manifest.External, ctx Context, dryRun bool, log func(string)) error {
	dest, err := stow.ExpandTarget(ext.Dest)
	if err != nil {
		return err
	}

	var changed bool
	switch ext.Type {
	case "git":
		changed, err = fetchGit(ext, dest, dryRun, log)
	case "file":
		changed, err = fetchFile(ext, dest, dryRun, log)
	default:
		return fmt.Errorf("type %q no soportado", ext.Type)
	}
	if err != nil {
		return err
	}

	if ext.Post == "" {
		return nil
	}
	// El post corre al traerlo y cuando cambia (ADR 0009).
	if !changed {
		log(fmt.Sprintf("= %s (sin cambios, no re-ejecuto post)", ext.Dest))
		return nil
	}
	if dryRun {
		log(fmt.Sprintf("· (dry-run) ejecutaría post %s", ext.Post))
		return nil
	}
	log(fmt.Sprintf("· post %s", ext.Post))
	return runPost(ext, ctx, dest)
}

// fetchGit clona el repo si no está, o lo actualiza al ref si ya está. Devuelve
// si el HEAD ha cambiado respecto a antes (para decidir si re-ejecutar el post).
func fetchGit(ext manifest.External, dest string, dryRun bool, log func(string)) (bool, error) {
	if log == nil {
		log = func(string) {}
	}
	if dryRun {
		log(fmt.Sprintf("+ (dry-run) traería %s (%s) -> %s", ext.Repo, ext.Ref, dest))
		return false, nil
	}

	if !isGitRepo(dest) {
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return false, err
		}
		if out, err := git("clone", ext.Repo, dest); err != nil {
			return false, fmt.Errorf("clone falló: %v (%s)", err, out)
		}
		if ext.Ref != "" {
			if out, err := git("-C", dest, "checkout", "--quiet", ext.Ref); err != nil {
				return false, fmt.Errorf("checkout %q falló: %v (%s)", ext.Ref, err, out)
			}
		}
		log(fmt.Sprintf("+ clonado %s -> %s", ext.Repo, dest))
		return true, nil
	}

	old, _ := git("-C", dest, "rev-parse", "HEAD")
	if out, err := git("-C", dest, "fetch", "--prune", "origin"); err != nil {
		return false, fmt.Errorf("fetch falló: %v (%s)", err, out)
	}
	if ext.Ref != "" {
		if out, err := git("-C", dest, "checkout", "--force", "--quiet", ext.Ref); err != nil {
			return false, fmt.Errorf("checkout %q falló: %v (%s)", ext.Ref, err, out)
		}
	}
	// Si el ref es una rama, avanzamos al remoto (fast-forward). Si es tag/commit
	// (HEAD separado), el merge no aplica y se ignora el error.
	_, _ = git("-C", dest, "merge", "--ff-only", "--quiet")
	newHead, _ := git("-C", dest, "rev-parse", "HEAD")

	changed := old != newHead
	if changed {
		log(fmt.Sprintf("~ actualizado %s -> %s", ext.Repo, dest))
	} else {
		log(fmt.Sprintf("= %s (ya al día)", dest))
	}
	return changed, nil
}

// fetchFile descarga un fichero suelto al destino. Devuelve si el contenido ha
// cambiado respecto a lo que ya hubiera.
func fetchFile(ext manifest.External, dest string, dryRun bool, log func(string)) (bool, error) {
	if log == nil {
		log = func(string) {}
	}
	if dryRun {
		log(fmt.Sprintf("+ (dry-run) descargaría %s -> %s", ext.URL, dest))
		return false, nil
	}

	resp, err := httpClient.Get(ext.URL)
	if err != nil {
		return false, fmt.Errorf("descarga falló: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("descarga de %s: HTTP %d", ext.URL, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	if existing, err := os.ReadFile(dest); err == nil && bytes.Equal(existing, data) {
		log(fmt.Sprintf("= %s (ya al día)", dest))
		return false, nil
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return false, err
	}
	// Escritura atómica: temporal en el mismo dir + rename.
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".dots-ext-*")
	if err != nil {
		return false, err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return false, err
	}
	tmp.Close()
	if err := os.Rename(tmpName, dest); err != nil {
		os.Remove(tmpName)
		return false, err
	}
	log(fmt.Sprintf("+ descargado %s -> %s", ext.URL, dest))
	return true, nil
}

// runPost ejecuta el script post del external, con bash, cwd = raíz del
// contenido, y el contrato de env de los hooks más DOTS_EXTERNAL_DEST.
func runPost(ext manifest.External, ctx Context, dest string) error {
	script := filepath.Join(ctx.ContentDir, ext.Post)
	cmd := exec.Command("bash", script)
	cmd.Dir = ctx.ContentDir
	cmd.Env = append(os.Environ(),
		"DOTS_HOOK=externals",
		"DOTS_PROFILE="+ctx.Profile,
		"DOTS_CONTENT_DIR="+ctx.ContentDir,
		"DOTS_TARGET="+ctx.Target,
		"DOTS_LAYERS="+strings.Join(ctx.Layers, " "),
		"DOTS_EXTERNAL_DEST="+dest,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("el post %s falló: %w", ext.Post, err)
	}
	return nil
}

// git ejecuta un comando git y devuelve su salida combinada (recortada).
func git(args ...string) (string, error) {
	out, err := exec.Command("git", args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// isGitRepo indica si dir es un repo git (tiene .git).
func isGitRepo(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}
