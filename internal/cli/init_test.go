package cli

import (
	"os"
	"path/filepath"
	"testing"
)

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
