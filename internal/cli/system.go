package cli

import (
	"fmt"

	"go.kreikol.dev/dotsmithy/internal/manifest"
	"go.kreikol.dev/dotsmithy/internal/system"
)

// applySystem coloca los ficheros de sistema declarados (/etc, systemd, sudoers).
// Es la única parte que necesita root (sudo). Respeta --dry-run. Se aplica como
// parte de la fase link, antes del hook post-link (ADR 0012).
func applySystem(m *manifest.Manifest, contentDir string) error {
	if len(m.System) == 0 {
		return nil
	}
	fmt.Printf("· system: %d fichero(s) de sistema…\n", len(m.System))
	return system.Apply(m.System, contentDir, dryRun, func(s string) {
		fmt.Println("  " + s)
	})
}
