package packages

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadDeclaredUnionYComentarios(t *testing.T) {
	content := t.TempDir()
	// shared: git, ripgrep (con comentarios y líneas vacías).
	write(t, filepath.Join(content, "shared", "packages", "dnf.txt"), `
# paquetes comunes
git
ripgrep   # buscador

`)
	// minipc: fzf y git (repetido, debe deduplicarse).
	write(t, filepath.Join(content, "machines", "minipc", "packages", "dnf.txt"), "fzf\ngit\n")

	got, err := ReadDeclared(content, []string{"shared", "machines/minipc"}, "dnf")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"fzf", "git", "ripgrep"} // unión, deduplicada, ordenada
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ReadDeclared: quiero %v, tengo %v", want, got)
	}
}

func TestReadDeclaredSinFicheros(t *testing.T) {
	// Ninguna capa declara el gestor: unión vacía, sin error.
	content := t.TempDir()
	got, err := ReadDeclared(content, []string{"shared"}, "dnf")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("quiero lista vacía, tengo %v", got)
	}
}

func TestDrift(t *testing.T) {
	declared := []string{"fzf", "git", "ripgrep"}
	installed := map[string]bool{"git": true, "bash": true}
	got := Drift(declared, installed)
	want := []string{"fzf", "ripgrep"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Drift: quiero %v, tengo %v", want, got)
	}
}

func TestGetSupported(t *testing.T) {
	if _, ok := Get("dnf"); !ok {
		t.Error("dnf debería estar soportado")
	}
	if _, ok := Get("pip"); ok {
		t.Error("pip no debería estar soportado todavía")
	}
}

func TestDNFInstalled(t *testing.T) {
	// Ojo: Ubuntu trae un binario rpm (para manejar .rpm) pero SIN base de datos,
	// así que "rpm" en el PATH no basta. dnf sí es propio de Fedora: lo usamos
	// como señal de que estamos en un sistema donde la consulta tiene sentido.
	if _, err := exec.LookPath("dnf"); err != nil {
		t.Skip("no hay dnf en esta máquina (no es Fedora); salto la consulta real")
	}
	set, err := dnfManager{}.Installed()
	if err != nil {
		t.Fatal(err)
	}
	if len(set) == 0 {
		t.Error("esperaba al menos algún paquete instalado")
	}
	// rpm siempre está si rpm existe.
	if !set["rpm"] {
		t.Error("esperaba que 'rpm' figurara como instalado")
	}
}
