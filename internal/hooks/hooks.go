package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Points son los puntos de ciclo de vida válidos, en el orden en que se disparan
// durante un init (ADR 0007). update reutiliza post-link y post-packages.
var Points = []string{"post-link", "post-packages", "post-init"}

// Context es lo que se pasa a cada hook por variables de entorno.
type Context struct {
	Profile    string   // perfil activo (DOTS_PROFILE)
	ContentDir string   // raíz del contenido (DOTS_CONTENT_DIR y cwd de los hooks)
	Target     string   // destino del stow ya expandido, ej. $HOME (DOTS_TARGET)
	Layers     []string // capas resueltas (DOTS_LAYERS, separadas por espacio)
	HooksDir   string   // subdirectorio de hooks dentro del contenido (ej. "hooks")
}

// Run ejecuta los hooks de un punto: hooks/<punto>/*.sh en orden léxico, con
// bash y cwd = raíz del contenido. Fail-fast: si uno sale con código != 0, se
// aborta nombrándolo. Si no hay directorio o no hay scripts, no hace nada.
//
// En dry-run NO se ejecuta nada: solo se informa de qué correría. Es la opción
// segura (el motor no puede garantizar que un hook respete el modo). log puede
// ser nil.
func Run(point string, ctx Context, dryRun bool, log func(string)) error {
	if log == nil {
		log = func(string) {}
	}

	dir := filepath.Join(ctx.ContentDir, ctx.HooksDir, point)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no hay hooks para este punto: normal
		}
		return fmt.Errorf("no he podido leer los hooks de %s: %w", point, err)
	}

	var scripts []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sh") {
			scripts = append(scripts, e.Name())
		}
	}
	sort.Strings(scripts) // orden léxico (convención NN-nombre.sh)
	if len(scripts) == 0 {
		return nil
	}

	for _, name := range scripts {
		rel := filepath.Join(ctx.HooksDir, point, name)
		if dryRun {
			log(fmt.Sprintf("· (dry-run) ejecutaría %s", rel))
			continue
		}
		log(fmt.Sprintf("· ejecutando %s", rel))
		cmd := exec.Command("bash", filepath.Join(dir, name))
		cmd.Dir = ctx.ContentDir
		cmd.Env = append(os.Environ(), envVars(point, ctx)...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("el hook %s falló: %w", rel, err) // fail-fast
		}
	}
	return nil
}

// envVars arma el contrato de variables de entorno de los hooks (ADR 0007).
func envVars(point string, ctx Context) []string {
	return []string{
		"DOTS_HOOK=" + point,
		"DOTS_PROFILE=" + ctx.Profile,
		"DOTS_CONTENT_DIR=" + ctx.ContentDir,
		"DOTS_TARGET=" + ctx.Target,
		"DOTS_LAYERS=" + strings.Join(ctx.Layers, " "),
	}
}
