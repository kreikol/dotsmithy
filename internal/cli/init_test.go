package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeMinimalContent crea un contenido local mínimo (dots.yaml + un dotfile) y
// devuelve su ruta. Sirve para probar init con --content (sin red).
func writeMinimalContent(t *testing.T) string {
	t.Helper()
	content := t.TempDir()
	manifest := "version: 1\nprofiles: { minipc: { description: t } }\nstow: { layers: [shared] }\n"
	if err := os.WriteFile(filepath.Join(content, "dots.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(content, "shared", "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".bashrc"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return content
}

// runInit ejecuta el comando init con los args dados, aislando HOME y XDG en
// directorios temporales. Devuelve (homeDir, xdgConfigDir) para las asserts.
func runInit(t *testing.T, dry bool, args ...string) (string, string) {
	t.Helper()
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)

	// dryRun es un flag global (lo pone la raíz); en el test lo fijamos a mano.
	old := dryRun
	dryRun = dry
	t.Cleanup(func() { dryRun = old })

	cmd := newInitCmd()
	cmd.SetArgs(args)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init falló: %v", err)
	}
	return home, xdg
}

func TestInitDryRunNoTocaDisco(t *testing.T) {
	content := writeMinimalContent(t)
	home, xdg := runInit(t, true, "--profile", "minipc", "--content", content)

	if _, err := os.Stat(filepath.Join(xdg, "dots", "state.yaml")); !os.IsNotExist(err) {
		t.Error("en dry-run NO debería escribirse el estado")
	}
	if _, err := os.Lstat(filepath.Join(home, ".bashrc")); !os.IsNotExist(err) {
		t.Error("en dry-run NO debería crearse el symlink")
	}
}

func TestInitAplicaYGuardaEstado(t *testing.T) {
	content := writeMinimalContent(t)
	home, xdg := runInit(t, false, "--profile", "minipc", "--content", content)

	// Estado escrito con el perfil correcto.
	data, err := os.ReadFile(filepath.Join(xdg, "dots", "state.yaml"))
	if err != nil {
		t.Fatalf("debería haberse escrito el estado: %v", err)
	}
	if !strings.Contains(string(data), "profile: minipc") {
		t.Errorf("el estado no tiene el perfil esperado:\n%s", data)
	}
	// Symlink creado.
	if _, err := os.Lstat(filepath.Join(home, ".bashrc")); err != nil {
		t.Errorf("debería haberse creado el symlink: %v", err)
	}
}

func TestDefaultContentDirRespetaXDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	got, err := defaultContentDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, "dots", "content")
	if got != want {
		t.Errorf("defaultContentDir: quiero %q, tengo %q", want, got)
	}
}

func TestCloneOrReuseReutilizaRepoExistente(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "content")
	// Simula un repo ya clonado: dest con un .git.
	if err := os.MkdirAll(filepath.Join(dest, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// No debe intentar clonar (no hay red aquí); reutiliza y devuelve nil.
	if err := cloneOrReuse("git@github.com:x/inexistente.git", dest); err != nil {
		t.Errorf("esperaba reutilizar sin error, tengo: %v", err)
	}
}

func TestCloneOrReuseChocaConDirNoRepo(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "content")
	// dest existe pero NO es un repo git (hay un fichero suelto).
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "algo.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := cloneOrReuse("git@github.com:x/y.git", dest); err == nil {
		t.Error("esperaba error por dir no-repo en medio")
	}
}
