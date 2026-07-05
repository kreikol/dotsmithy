package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsSSHRemote(t *testing.T) {
	yes := []string{"git@github.com:kreikol/x.git", "ssh://git@github.com/kreikol/x"}
	no := []string{"https://github.com/kreikol/x.git", "/ruta/local", ""}
	for _, r := range yes {
		if !isSSHRemote(r) {
			t.Errorf("%q debería ser SSH", r)
		}
	}
	for _, r := range no {
		if isSSHRemote(r) {
			t.Errorf("%q NO debería ser SSH", r)
		}
	}
}

func TestSSHHost(t *testing.T) {
	cases := map[string]string{
		"git@github.com:kreikol/x.git":         "github.com",
		"ssh://git@github.com/kreikol/x":       "github.com",
		"ssh://git@example.org:2222/kreikol/x": "example.org",
		"https://github.com/kreikol/x":         "",
	}
	for in, want := range cases {
		if got := sshHost(in); got != want {
			t.Errorf("sshHost(%q): quiero %q, tengo %q", in, want, got)
		}
	}
}

func TestEnsureSSHKeyGenerarYReutilizar(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("no hay ssh-keygen; salto")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Sin clave: la genera.
	pub, err := ensureSSHKey()
	if err != nil {
		t.Fatal(err)
	}
	if pub != filepath.Join(home, ".ssh", "id_ed25519.pub") {
		t.Errorf("clave pública en sitio inesperado: %s", pub)
	}
	if _, err := os.Stat(pub); err != nil {
		t.Errorf("no se creó la clave pública: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".ssh", "id_ed25519")); err != nil {
		t.Errorf("no se creó la clave privada: %v", err)
	}

	// Segunda llamada: reutiliza (no regenera). Lo comprobamos por el contenido.
	before, _ := os.ReadFile(pub)
	pub2, err := ensureSSHKey()
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(pub2)
	if string(before) != string(after) {
		t.Error("no debería regenerar una clave existente")
	}
}

func TestExistingPubKeyPrefiereEd25519(t *testing.T) {
	sshDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sshDir, "id_rsa.pub"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := existingPubKey(sshDir); filepath.Base(got) != "id_rsa.pub" {
		t.Errorf("con solo id_rsa, quiero id_rsa.pub, tengo %q", got)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "id_ed25519.pub"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := existingPubKey(sshDir); filepath.Base(got) != "id_ed25519.pub" {
		t.Errorf("con id_ed25519, debería preferirla, tengo %q", got)
	}
}
