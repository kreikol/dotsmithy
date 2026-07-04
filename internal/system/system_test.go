package system

import (
	"os"
	"path/filepath"
	"testing"

	"go.kreikol.dev/dotsmithy/internal/manifest"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSplitOwner(t *testing.T) {
	cases := []struct{ in, u, g string }{
		{"", "root", "root"},
		{"mruiz", "mruiz", "mruiz"},
		{"root:wheel", "root", "wheel"},
	}
	for _, c := range cases {
		u, g := splitOwner(c.in)
		if u != c.u || g != c.g {
			t.Errorf("splitOwner(%q): quiero %s:%s, tengo %s:%s", c.in, c.u, c.g, u, g)
		}
	}
}

func TestPlanCopy(t *testing.T) {
	content := t.TempDir()
	sys := t.TempDir() // hace de "/etc" de mentira
	writeFile(t, filepath.Join(content, "system", "foo.conf"), "hola\n")
	dest := filepath.Join(sys, "foo.conf")

	entries := []manifest.SystemFile{{Src: "system/foo.conf", Dest: dest, Type: "copy"}}

	// Destino inexistente -> hay que copiar.
	plan, err := Plan(entries, content)
	if err != nil {
		t.Fatal(err)
	}
	if plan[0].Op != OpCopy {
		t.Errorf("destino ausente: quiero OpCopy, tengo %v", plan[0].Op)
	}

	// Destino idéntico -> skip.
	writeFile(t, dest, "hola\n")
	plan, _ = Plan(entries, content)
	if plan[0].Op != OpSkip {
		t.Errorf("destino idéntico: quiero OpSkip, tengo %v", plan[0].Op)
	}

	// Destino distinto -> copiar.
	writeFile(t, dest, "otra cosa\n")
	plan, _ = Plan(entries, content)
	if plan[0].Op != OpCopy {
		t.Errorf("destino distinto: quiero OpCopy, tengo %v", plan[0].Op)
	}
}

func TestPlanSymlink(t *testing.T) {
	content := t.TempDir()
	sys := t.TempDir()
	writeFile(t, filepath.Join(content, "system", "unit.service"), "x\n")
	absSrc := filepath.Join(content, "system", "unit.service")
	dest := filepath.Join(sys, "unit.service")

	entries := []manifest.SystemFile{{Src: "system/unit.service", Dest: dest, Type: "symlink"}}

	// Sin symlink -> crear.
	plan, err := Plan(entries, content)
	if err != nil {
		t.Fatal(err)
	}
	if plan[0].Op != OpSymlink {
		t.Errorf("sin symlink: quiero OpSymlink, tengo %v", plan[0].Op)
	}

	// Symlink correcto -> skip.
	if err := os.Symlink(absSrc, dest); err != nil {
		t.Fatal(err)
	}
	plan, _ = Plan(entries, content)
	if plan[0].Op != OpSkip {
		t.Errorf("symlink correcto: quiero OpSkip, tengo %v", plan[0].Op)
	}
}

func TestPlanMissingSrc(t *testing.T) {
	content := t.TempDir()
	entries := []manifest.SystemFile{{Src: "system/no-existe", Dest: "/tmp/x", Type: "copy"}}
	if _, err := Plan(entries, content); err == nil {
		t.Error("esperaba error por origen inexistente")
	}
}

func TestRunValidate(t *testing.T) {
	src := filepath.Join(t.TempDir(), "conf")
	writeFile(t, src, "clave = valor\n")

	// Validación que pasa.
	if err := runValidate(src, "grep -q clave {file}"); err != nil {
		t.Errorf("no esperaba error en validación que pasa: %v", err)
	}
	// Validación que falla: debe nombrar el fichero.
	err := runValidate(src, "grep -q ausente {file}")
	if err == nil {
		t.Fatal("esperaba error en validación que falla")
	}
}

func TestSameCopyMode(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dest := filepath.Join(dir, "dest")
	writeFile(t, src, "igual\n")
	writeFile(t, dest, "igual\n")
	if err := os.Chmod(dest, 0o600); err != nil {
		t.Fatal(err)
	}
	// Mismo contenido pero se pide mode 0644 y el destino es 0600 -> no es igual.
	if sameCopy(src, dest, "0644") {
		t.Error("con mode distinto NO debería considerarse igual")
	}
	// Ajustamos el mode: ahora sí.
	if err := os.Chmod(dest, 0o644); err != nil {
		t.Fatal(err)
	}
	if !sameCopy(src, dest, "0644") {
		t.Error("mismo contenido y mode debería ser igual")
	}
}
