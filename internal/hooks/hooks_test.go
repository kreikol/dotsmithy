package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeHook crea un script de hook en hooks/<punto>/<name> dentro de content.
func writeHook(t *testing.T, content, point, name, body string) {
	t.Helper()
	dir := filepath.Join(content, "hooks", point)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func ctxFor(content, target string) Context {
	return Context{
		Profile:    "minipc",
		ContentDir: content,
		Target:     target,
		Layers:     []string{"shared", "machines/minipc"},
		HooksDir:   "hooks",
	}
}

func TestRunOrderAndEnv(t *testing.T) {
	content, target := t.TempDir(), t.TempDir()
	// Dos hooks: deben correr en orden léxico (10 antes que 20).
	writeHook(t, content, "post-link", "20-b.sh", `echo b >> "$DOTS_TARGET/order.txt"`)
	writeHook(t, content, "post-link", "10-a.sh", `echo a >> "$DOTS_TARGET/order.txt"`)
	// Un hook que vuelca el contrato de env vars y el cwd.
	writeHook(t, content, "post-link", "30-env.sh", `{
  echo "profile=$DOTS_PROFILE"
  echo "content=$DOTS_CONTENT_DIR"
  echo "target=$DOTS_TARGET"
  echo "layers=$DOTS_LAYERS"
  echo "hook=$DOTS_HOOK"
  echo "pwd=$(pwd)"
} > "$DOTS_TARGET/env.txt"`)

	if err := Run("post-link", ctxFor(content, target), false, nil); err != nil {
		t.Fatal(err)
	}

	order, _ := os.ReadFile(filepath.Join(target, "order.txt"))
	if string(order) != "a\nb\n" {
		t.Errorf("orden léxico: quiero \"a\\nb\\n\", tengo %q", string(order))
	}

	env, _ := os.ReadFile(filepath.Join(target, "env.txt"))
	s := string(env)
	for _, want := range []string{
		"profile=minipc",
		"content=" + content,
		"target=" + target,
		"layers=shared machines/minipc",
		"hook=post-link",
		"pwd=" + content, // cwd = raíz del contenido
	} {
		if !strings.Contains(s, want) {
			t.Errorf("env: falta %q en:\n%s", want, s)
		}
	}
}

func TestRunFailFast(t *testing.T) {
	content, target := t.TempDir(), t.TempDir()
	writeHook(t, content, "post-packages", "10-ok.sh", `touch "$DOTS_TARGET/ran-10"`)
	writeHook(t, content, "post-packages", "20-fail.sh", `exit 3`)
	writeHook(t, content, "post-packages", "30-after.sh", `touch "$DOTS_TARGET/ran-30"`)

	err := Run("post-packages", ctxFor(content, target), false, nil)
	if err == nil {
		t.Fatal("esperaba error por hook que falla")
	}
	if !strings.Contains(err.Error(), "20-fail.sh") {
		t.Errorf("el error debería nombrar el hook que falló; error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "ran-10")); err != nil {
		t.Error("el hook 10 debería haber corrido antes del fallo")
	}
	if _, err := os.Stat(filepath.Join(target, "ran-30")); err == nil {
		t.Error("el hook 30 NO debería correr (fail-fast tras el 20)")
	}
}

func TestRunMissingDir(t *testing.T) {
	content, target := t.TempDir(), t.TempDir()
	// No hay hooks/post-init: no es error, simplemente no hace nada.
	if err := Run("post-init", ctxFor(content, target), false, nil); err != nil {
		t.Errorf("no esperaba error sin directorio de hooks: %v", err)
	}
}

func TestRunDryRunNoExec(t *testing.T) {
	content, target := t.TempDir(), t.TempDir()
	writeHook(t, content, "post-link", "10-a.sh", `touch "$DOTS_TARGET/ran"`)
	if err := Run("post-link", ctxFor(content, target), true, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "ran")); err == nil {
		t.Error("en dry-run el hook NO debería ejecutarse")
	}
}

func TestRunIgnoresNonSh(t *testing.T) {
	content, target := t.TempDir(), t.TempDir()
	writeHook(t, content, "post-link", "10-a.sh", `touch "$DOTS_TARGET/ran-sh"`)
	writeHook(t, content, "post-link", "README.md", `no soy un script`)
	if err := Run("post-link", ctxFor(content, target), false, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "ran-sh")); err != nil {
		t.Error("el .sh debería haber corrido")
	}
}
