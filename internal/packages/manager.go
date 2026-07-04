package packages

import (
	"fmt"
	"os/exec"
	"strings"
)

// Manager abstrae el gestor de paquetes del sistema. En este PR solo se usa la
// consulta (qué hay instalado), que es de solo lectura; la instalación llegará
// con el comando add.
type Manager interface {
	// Installed devuelve el conjunto de paquetes instalados (por nombre).
	Installed() (map[string]bool, error)
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
