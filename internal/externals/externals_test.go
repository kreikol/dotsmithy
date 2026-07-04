package externals

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.kreikol.dev/dotsmithy/internal/manifest"
)

// gitRun ejecuta un git y falla el test si el comando falla.
func gitRun(t *testing.T, args ...string) {
	t.Helper()
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// makeRemote crea un repo git (rama main) con un fichero y un commit, y devuelve
// su ruta. Sirve de "remoto" local para los externals de tipo git (sin red).
func makeRemote(t *testing.T, filename, content string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "remote")
	gitRun(t, "init", "-b", "main", dir)
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, "-C", dir, "add", ".")
	gitRun(t, "-C", dir, "-c", "user.email=t@e", "-c", "user.name=t", "commit", "-m", "init")
	return dir
}

func TestFetchGitCloneThenIdempotent(t *testing.T) {
	remote := makeRemote(t, "a.txt", "uno\n")
	dest := filepath.Join(t.TempDir(), "dest")
	ext := manifest.External{Dest: dest, Type: "git", Repo: remote, Ref: "main"}

	// Primer fetch: clona (cambió).
	changed, err := fetchGit(ext, dest, false, nil)
	if err != nil || !changed {
		t.Fatalf("clone: changed=%v err=%v", changed, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "a.txt")); err != nil {
		t.Errorf("el fichero clonado no está: %v", err)
	}

	// Segundo fetch sin cambios en el remoto: no cambió.
	changed, err = fetchGit(ext, dest, false, nil)
	if err != nil || changed {
		t.Errorf("sin cambios: changed=%v err=%v", changed, err)
	}

	// Nuevo commit en el remoto -> el siguiente fetch sí cambia.
	if err := os.WriteFile(filepath.Join(remote, "b.txt"), []byte("dos\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, "-C", remote, "add", ".")
	gitRun(t, "-C", remote, "-c", "user.email=t@e", "-c", "user.name=t", "commit", "-m", "second")

	changed, err = fetchGit(ext, dest, false, nil)
	if err != nil || !changed {
		t.Errorf("tras nuevo commit: changed=%v err=%v", changed, err)
	}
}

func TestFetchFile(t *testing.T) {
	body := "v1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "sub", "file.txt")
	ext := manifest.External{Dest: dest, Type: "file", URL: srv.URL}

	changed, err := fetchFile(ext, dest, false, nil)
	if err != nil || !changed {
		t.Fatalf("descarga: changed=%v err=%v", changed, err)
	}
	if data, _ := os.ReadFile(dest); string(data) != "v1" {
		t.Errorf("contenido: quiero v1, tengo %q", data)
	}

	// Mismo contenido -> no cambia.
	changed, _ = fetchFile(ext, dest, false, nil)
	if changed {
		t.Error("mismo contenido no debería cambiar")
	}

	// Cambia lo servido -> cambia.
	body = "v2"
	changed, _ = fetchFile(ext, dest, false, nil)
	if !changed {
		t.Error("contenido nuevo debería cambiar")
	}
	if data, _ := os.ReadFile(dest); string(data) != "v2" {
		t.Errorf("contenido actualizado: quiero v2, tengo %q", data)
	}
}

func TestApplyRunsPostOnChangeOnly(t *testing.T) {
	remote := makeRemote(t, "a.txt", "uno\n")
	content := t.TempDir()
	target := t.TempDir()

	// post que apunta una línea cada vez que corre, con el dest del external.
	postDir := filepath.Join(content, "hooks", "externals")
	if err := os.MkdirAll(postDir, 0o755); err != nil {
		t.Fatal(err)
	}
	post := "echo \"$DOTS_EXTERNAL_DEST\" >> \"$DOTS_TARGET/post.log\"\n"
	if err := os.WriteFile(filepath.Join(postDir, "sync.sh"), []byte(post), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "dest")
	ext := manifest.External{Dest: dest, Type: "git", Repo: remote, Ref: "main", Post: "hooks/externals/sync.sh"}
	ctx := Context{Profile: "minipc", ContentDir: content, Target: target, Layers: []string{"shared"}}

	// Primera vez: clona (cambia) -> post corre.
	if failed := Apply([]manifest.External{ext}, ctx, false, nil); failed != 0 {
		t.Fatalf("no esperaba fallos, hubo %d", failed)
	}
	if n := lines(t, filepath.Join(target, "post.log")); n != 1 {
		t.Errorf("post debería haber corrido 1 vez, corrió %d", n)
	}

	// Segunda vez sin cambios: post NO corre.
	if failed := Apply([]manifest.External{ext}, ctx, false, nil); failed != 0 {
		t.Fatalf("no esperaba fallos, hubo %d", failed)
	}
	if n := lines(t, filepath.Join(target, "post.log")); n != 1 {
		t.Errorf("post no debería re-ejecutarse sin cambios; corrió %d veces", n)
	}
}

func TestApplyBestEffort(t *testing.T) {
	// Un external roto (repo inexistente) y uno bueno (file). El bueno se aplica
	// y el roto solo cuenta como fallo.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	goodDest := filepath.Join(t.TempDir(), "good.txt")
	exts := []manifest.External{
		{Dest: filepath.Join(t.TempDir(), "bad"), Type: "git", Repo: "/no/existe/repo", Ref: "main"},
		{Dest: goodDest, Type: "file", URL: srv.URL},
	}
	ctx := Context{ContentDir: t.TempDir(), Target: t.TempDir()}

	failed := Apply(exts, ctx, false, nil)
	if failed != 1 {
		t.Errorf("quiero 1 fallo, tengo %d", failed)
	}
	if _, err := os.Stat(goodDest); err != nil {
		t.Errorf("el external bueno debería haberse aplicado pese al fallo del otro: %v", err)
	}
}

// lines cuenta las líneas no vacías de un fichero (0 si no existe).
func lines(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	n := 0
	for _, l := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}
