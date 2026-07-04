package stow

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile crea un fichero con su árbol de directorios y contenido dado.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

// setup crea un contenido de prueba con una capa shared y (opcional) minipc, y
// devuelve contentDir y targetDir (un tmp aislado, sin tocar el HOME real).
func setup(t *testing.T) (contentDir, targetDir string) {
	t.Helper()
	root := t.TempDir()
	contentDir = filepath.Join(root, "content")
	targetDir = filepath.Join(root, "home")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return contentDir, targetDir
}

func TestApplyCreatesSymlink(t *testing.T) {
	content, target := setup(t)
	writeFile(t, filepath.Join(content, "shared", "home", ".bashrc"), "export A=1\n")

	plan, err := BuildPlan(content, "home", target, []string{"shared"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Changes() != 1 {
		t.Fatalf("quiero 1 cambio, tengo %d (%+v)", plan.Changes(), plan.Actions)
	}
	if err := plan.Apply(false, nil); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(target, ".bashrc")
	got, err := os.Readlink(dest)
	if err != nil {
		t.Fatalf("el destino no es un symlink: %v", err)
	}
	want := filepath.Join(content, "shared", "home", ".bashrc")
	if got != want {
		t.Errorf("symlink apunta a %q, quiero %q", got, want)
	}
	// El contenido se lee a través del symlink.
	data, err := os.ReadFile(dest)
	if err != nil || string(data) != "export A=1\n" {
		t.Errorf("no leo bien a través del symlink: %q, %v", data, err)
	}
}

func TestIdempotent(t *testing.T) {
	content, target := setup(t)
	writeFile(t, filepath.Join(content, "shared", "home", ".bashrc"), "x\n")

	plan, _ := BuildPlan(content, "home", target, []string{"shared"})
	if err := plan.Apply(false, nil); err != nil {
		t.Fatal(err)
	}
	// Segunda pasada: nada que cambiar.
	plan2, _ := BuildPlan(content, "home", target, []string{"shared"})
	if plan2.Changes() != 0 {
		t.Errorf("la segunda pasada quiere cambiar %d cosa(s), debería ser 0", plan2.Changes())
	}
	if plan2.Actions[0].Kind != ActionSkip {
		t.Errorf("quiero ActionSkip, tengo %v", plan2.Actions[0].Kind)
	}
}

func TestOverlayLastLayerWins(t *testing.T) {
	content, target := setup(t)
	writeFile(t, filepath.Join(content, "shared", "home", ".bashrc"), "shared\n")
	writeFile(t, filepath.Join(content, "machines", "minipc", "home", ".bashrc"), "minipc\n")

	layers := []string{"shared", "machines/minipc"}
	plan, err := BuildPlan(content, "home", target, layers)
	if err != nil {
		t.Fatal(err)
	}
	// Un solo destino (.bashrc), aunque aparece en dos capas.
	if len(plan.Actions) != 1 {
		t.Fatalf("quiero 1 acción, tengo %d", len(plan.Actions))
	}
	if err := plan.Apply(false, nil); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(target, ".bashrc"))
	if string(data) != "minipc\n" {
		t.Errorf("gana la última capa: quiero %q, tengo %q", "minipc\n", string(data))
	}
}

func TestNestedDirsCreated(t *testing.T) {
	content, target := setup(t)
	writeFile(t, filepath.Join(content, "shared", "home", ".config", "nvim", "init.lua"), "-- vim\n")

	plan, _ := BuildPlan(content, "home", target, []string{"shared"})
	if err := plan.Apply(false, nil); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(target, ".config", "nvim", "init.lua")
	if _, err := os.Readlink(dest); err != nil {
		t.Errorf("no se creó el symlink anidado: %v", err)
	}
}

func TestConflictWithRealFileAborts(t *testing.T) {
	content, target := setup(t)
	writeFile(t, filepath.Join(content, "shared", "home", ".bashrc"), "nuevo\n")
	// Ya hay un .bashrc REAL en el destino.
	writeFile(t, filepath.Join(target, ".bashrc"), "el mío de siempre\n")

	plan, _ := BuildPlan(content, "home", target, []string{"shared"})
	if len(plan.Conflicts()) != 1 {
		t.Fatalf("quiero 1 conflicto, tengo %d", len(plan.Conflicts()))
	}
	// Apply (no dry-run) debe fallar y NO tocar el fichero real.
	if err := plan.Apply(false, nil); err == nil {
		t.Fatal("esperaba error por conflicto")
	}
	data, _ := os.ReadFile(filepath.Join(target, ".bashrc"))
	if string(data) != "el mío de siempre\n" {
		t.Errorf("el fichero real se tocó: %q", string(data))
	}
}

func TestUpdateRepointsSymlink(t *testing.T) {
	content, target := setup(t)
	src := filepath.Join(content, "shared", "home", ".bashrc")
	writeFile(t, src, "x\n")
	// El destino ya es un symlink, pero a otro sitio.
	dest := filepath.Join(target, ".bashrc")
	if err := os.Symlink(filepath.Join(content, "otra-cosa"), dest); err != nil {
		t.Fatal(err)
	}

	plan, _ := BuildPlan(content, "home", target, []string{"shared"})
	if plan.Actions[0].Kind != ActionUpdate {
		t.Fatalf("quiero ActionUpdate, tengo %v", plan.Actions[0].Kind)
	}
	if err := plan.Apply(false, nil); err != nil {
		t.Fatal(err)
	}
	got, _ := os.Readlink(dest)
	want, _ := filepath.Abs(src)
	if got != want {
		t.Errorf("no repuntó: apunta a %q, quiero %q", got, want)
	}
}

func TestDryRunDoesNotTouchDisk(t *testing.T) {
	content, target := setup(t)
	writeFile(t, filepath.Join(content, "shared", "home", ".bashrc"), "x\n")

	plan, _ := BuildPlan(content, "home", target, []string{"shared"})
	if err := plan.Apply(true, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(target, ".bashrc")); !os.IsNotExist(err) {
		t.Errorf("dry-run creó algo en disco (o error raro): %v", err)
	}
}

func TestGitkeepIgnored(t *testing.T) {
	content, target := setup(t)
	// La capa tiene un .gitkeep (centinela) y un fichero real.
	writeFile(t, filepath.Join(content, "shared", "home", ".gitkeep"), "")
	writeFile(t, filepath.Join(content, "shared", "home", ".keep"), "")
	writeFile(t, filepath.Join(content, "shared", "home", ".bashrc"), "x\n")

	plan, err := BuildPlan(content, "home", target, []string{"shared"})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 {
		t.Fatalf("quiero 1 acción (solo .bashrc), tengo %d: %+v", len(plan.Actions), plan.Actions)
	}
	if err := plan.Apply(false, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(target, ".gitkeep")); !os.IsNotExist(err) {
		t.Errorf(".gitkeep NO debería desplegarse")
	}
	if _, err := os.Lstat(filepath.Join(target, ".bashrc")); err != nil {
		t.Errorf(".bashrc sí debería desplegarse: %v", err)
	}
}

func TestMissingLayerSubdirSkipped(t *testing.T) {
	content, target := setup(t)
	// shared tiene home/, machines/minipc NO tiene home/ (capa sin stow).
	writeFile(t, filepath.Join(content, "shared", "home", ".bashrc"), "x\n")
	writeFile(t, filepath.Join(content, "machines", "minipc", "packages", "dnf.txt"), "git\n")

	plan, err := BuildPlan(content, "home", target, []string{"shared", "machines/minipc"})
	if err != nil {
		t.Fatalf("no esperaba error por capa sin home/: %v", err)
	}
	if len(plan.Actions) != 1 {
		t.Errorf("quiero 1 acción (solo shared), tengo %d", len(plan.Actions))
	}
}
