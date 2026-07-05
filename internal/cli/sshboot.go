package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// El bootstrap SSH prepara lo mínimo para poder clonar el contenido (y los
// externals) por SSH en una máquina limpia: confía en el host y se asegura de
// que existe una clave. Si la clave no está registrada en GitHub, init lo detecta
// al fallar el clone y guía para añadirla (guideSSHKey). No registra por ti la
// clave: eso es manual e inevitable.

// isSSHRemote indica si un remoto se clona por SSH (git@host:… o ssh://…).
func isSSHRemote(remote string) bool {
	return strings.HasPrefix(remote, "git@") || strings.HasPrefix(remote, "ssh://")
}

// sshHost extrae el host de un remoto SSH (para confiar en él en known_hosts).
// Devuelve "" si no lo reconoce.
func sshHost(remote string) string {
	switch {
	case strings.HasPrefix(remote, "ssh://"):
		rest := strings.TrimPrefix(remote, "ssh://")
		if at := strings.IndexByte(rest, '@'); at >= 0 {
			rest = rest[at+1:]
		}
		rest = strings.SplitN(rest, "/", 2)[0] // quita la ruta
		rest = strings.SplitN(rest, ":", 2)[0] // quita el puerto
		return rest
	case strings.HasPrefix(remote, "git@"):
		rest := strings.TrimPrefix(remote, "git@")
		return strings.SplitN(rest, ":", 2)[0]
	default:
		return ""
	}
}

// prepareSSH deja el terreno listo para clonar por SSH: confía en el host y se
// asegura de que hay una clave (la genera si no hay ninguna). Devuelve la ruta
// de la clave pública elegida (para poder guiar si luego el clone falla).
func prepareSSH(remote string) (pubKey string, err error) {
	ensureKnownHost(sshHost(remote))
	return ensureSSHKey()
}

// ensureSSHKey devuelve la clave pública a usar: si ya hay una, la reutiliza; si
// no hay ninguna, genera una ed25519 sin passphrase. Nunca toca una clave que ya
// exista.
func ensureSSHKey() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	sshDir := filepath.Join(home, ".ssh")

	if pub := existingPubKey(sshDir); pub != "" {
		return pub, nil
	}

	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		return "", fmt.Errorf("no tienes clave SSH y no encuentro ssh-keygen para crearte una: %w", err)
	}
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return "", fmt.Errorf("no he podido crear %s: %w", sshDir, err)
	}
	keyPath := filepath.Join(sshDir, "id_ed25519")
	hostname, _ := os.Hostname()
	comment := "dotsmithy@" + hostname
	fmt.Println("no tienes clave SSH; te genero una (ed25519)…")
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-C", comment, "-f", keyPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("no he podido generar la clave SSH: %w", err)
	}
	return keyPath + ".pub", nil
}

// existingPubKey devuelve la primera clave pública conocida que exista en sshDir,
// o "" si no hay ninguna.
func existingPubKey(sshDir string) string {
	for _, name := range []string{"id_ed25519.pub", "id_ecdsa.pub", "id_rsa.pub"} {
		p := filepath.Join(sshDir, name)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// ensureKnownHost añade la clave de host a ~/.ssh/known_hosts si falta, para que
// el clone no se cuelgue en la verificación del host en una máquina limpia. Es
// best-effort: si algo falla, avisa y sigue (no aborta).
func ensureKnownHost(host string) {
	if host == "" {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	sshDir := filepath.Join(home, ".ssh")
	known := filepath.Join(sshDir, "known_hosts")

	// ¿ya confiamos en él?
	if exec.Command("ssh-keygen", "-F", host, "-f", known).Run() == nil {
		return
	}
	out, err := exec.Command("ssh-keyscan", host).Output()
	if err != nil || len(out) == 0 {
		fmt.Printf("aviso: no he podido obtener la clave de host de %s; si el clone se cuelga verificando el host, añádela a mano.\n", host)
		return
	}
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(known, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	if _, err := f.Write(out); err == nil {
		fmt.Printf("añadida la clave de host de %s a known_hosts.\n", host)
	}
}

// guideSSHKey se llama cuando el clone por SSH falla: lo más probable es que la
// clave no esté registrada en GitHub. Imprime la clave pública y cómo añadirla.
func guideSSHKey(remote, pubKey string) {
	fmt.Println()
	fmt.Println("No he podido clonar por SSH. Lo más habitual: tu clave aún no está en GitHub (o no tiene acceso al repo).")
	if pubKey != "" {
		if data, err := os.ReadFile(pubKey); err == nil {
			fmt.Println("Esta es tu clave pública:")
			fmt.Println()
			fmt.Print(string(data))
			fmt.Println()
		}
	}
	fmt.Println("Añádela en GitHub: https://github.com/settings/ssh/new (pega la clave y guarda).")
	fmt.Printf("Luego reejecuta:  dots init --profile <perfil> --remote %s\n", remote)
}
