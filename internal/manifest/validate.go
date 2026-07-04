package manifest

import (
	"fmt"
	"regexp"
	"strings"
)

// modeRe valida un modo octal de 3 o 4 dígitos (ej. "644", "0440").
var modeRe = regexp.MustCompile(`^[0-7]{3,4}$`)

// Validate comprueba que el manifiesto es coherente. Acumula TODOS los problemas
// y los devuelve juntos, para que la usuaria los vea de una sola pasada en vez de
// arreglarlos de uno en uno. Devuelve nil si todo está bien.
func (m *Manifest) Validate() error {
	var problems []string
	add := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf(format, args...))
	}

	// --- version ---
	if m.Version != SchemaVersion {
		add("version: es %d, pero este motor solo entiende la versión %d", m.Version, SchemaVersion)
	}

	// --- profiles ---
	if len(m.Profiles) == 0 {
		add("profiles: hace falta al menos un perfil")
	}
	for name := range m.Profiles {
		if strings.TrimSpace(name) == "" {
			add("profiles: hay un perfil con nombre vacío")
		}
	}

	// --- stow ---
	if len(m.Stow.Layers) == 0 {
		add("stow.layers: hace falta al menos una capa")
	}
	for i, layer := range m.Stow.Layers {
		if strings.TrimSpace(layer) == "" {
			add("stow.layers[%d]: capa con nombre vacío", i)
		}
	}

	// --- packages ---
	for i, mgr := range m.Packages.Managers {
		if !contains(KnownManagers, mgr) {
			add("packages.managers[%d]: gestor desconocido %q (conocidos: %s)", i, mgr, strings.Join(KnownManagers, ", "))
		}
	}
	// COPR solo tiene sentido si dnf está activo.
	if m.Packages.DNF != nil && len(m.Packages.DNF.COPR) > 0 && !m.HasManager("dnf") {
		add("packages.dnf.copr: hay repos COPR declarados pero dnf no está en packages.managers")
	}

	// --- system ---
	for i, s := range m.System {
		if strings.TrimSpace(s.Src) == "" {
			add("system[%d]: falta src", i)
		}
		if strings.TrimSpace(s.Dest) == "" {
			add("system[%d]: falta dest", i)
		}
		if !contains(knownSystemTypes, s.Type) {
			add("system[%d]: type %q inválido (usa: %s)", i, s.Type, strings.Join(knownSystemTypes, ", "))
		}
		if s.Mode != "" && !modeRe.MatchString(s.Mode) {
			add("system[%d]: mode %q no es un octal válido (ej. \"0440\")", i, s.Mode)
		}
	}

	// --- hooks ---
	for i, p := range m.Hooks.Points {
		if !contains(KnownHookPoints, p) {
			add("hooks.points[%d]: punto desconocido %q (conocidos: %s)", i, p, strings.Join(KnownHookPoints, ", "))
		}
	}

	// --- externals ---
	for i, e := range m.Externals {
		if strings.TrimSpace(e.Dest) == "" {
			add("externals[%d]: falta dest", i)
		}
		switch e.Type {
		case "git":
			if strings.TrimSpace(e.Repo) == "" {
				add("externals[%d]: type git requiere repo", i)
			}
		case "file":
			if strings.TrimSpace(e.URL) == "" {
				add("externals[%d]: type file requiere url", i)
			}
		default:
			add("externals[%d]: type %q inválido (usa: %s)", i, e.Type, strings.Join(knownExternalTypes, ", "))
		}
	}

	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("el manifiesto tiene %d problema(s):\n  - %s",
		len(problems), strings.Join(problems, "\n  - "))
}

// contains indica si el slice contiene el valor dado.
func contains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
