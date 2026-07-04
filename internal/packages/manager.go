package packages

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Manager abstrae el gestor de paquetes del sistema.
type Manager interface {
	// Installed devuelve el conjunto de paquetes instalados (por nombre).
	Installed() (map[string]bool, error)
	// Install instala los paquetes dados (idempotente: si ya están, no pasa nada).
	Install(pkgs []string) error
}

// COPRManager lo implementan los gestores que saben habilitar repos COPR (dnf).
// Es opcional: se comprueba con un type assertion.
type COPRManager interface {
	EnableCOPR(repos []string) error
}

// supported son los gestores implementados de verdad. El manifiesto admite otros
// gestores en el esquema (pip, npm, cargo) de cara a futuras fases, pero aquí
// solo dnf está soportado.
var supported = map[string]Manager{
	"dnf": dnfManager{},
}

// Get devuelve el gestor implementado para name, y si está soportado.
func Get(name string) (Manager, bool) {
	m, ok := supported[name]
	return m, ok
}

// dnfManager habla con dnf/rpm en sistemas Fedora.
type dnfManager struct{}

// Installed lista los paquetes instalados vía rpm (solo lectura, rápido).
func (dnfManager) Installed() (map[string]bool, error) {
	out, err := exec.Command("rpm", "-qa", "--queryformat", "%{NAME}\\n").Output()
	if err != nil {
		return nil, fmt.Errorf("no he podido consultar los paquetes instalados con rpm: %w", err)
	}
	set := make(map[string]bool)
	for _, name := range strings.Split(string(out), "\n") {
		name = strings.TrimSpace(name)
		if name != "" {
			set[name] = true
		}
	}
	return set, nil
}

// Install instala paquetes con dnf (no interactivo). dnf ya es idempotente: si un
// paquete está, lo deja como está.
func (dnfManager) Install(pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}
	return runPrivileged("dnf", append([]string{"install", "-y"}, pkgs...)...)
}

// EnableCOPR habilita los repos COPR indicados (ej. "atim/lazygit") antes de
// instalar (D10: los COPR van en el manifiesto, no como hook).
func (dnfManager) EnableCOPR(repos []string) error {
	for _, repo := range repos {
		if err := runPrivileged("dnf", "copr", "enable", "-y", repo); err != nil {
			return fmt.Errorf("no he podido habilitar el COPR %q: %w", repo, err)
		}
	}
	return nil
}

// runPrivileged ejecuta un comando que necesita root. Si ya somos root lo lanza
// tal cual; si no, lo antepone con sudo. La salida va a la terminal.
func runPrivileged(name string, args ...string) error {
	if os.Geteuid() != 0 {
		args = append([]string{name}, args...)
		name = "sudo"
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("falló «%s %s»: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
