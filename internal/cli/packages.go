package cli

import (
	"fmt"

	"go.kreikol.dev/dotsmithy/internal/manifest"
	"go.kreikol.dev/dotsmithy/internal/packages"
)

// applyPackages instala los paquetes declarados que falten, gestor por gestor.
// Habilita antes los repos COPR declarados (dnf). Respeta --dry-run (solo
// informa). Es idempotente: si todo está, no hace nada. Lo usa init.
func applyPackages(m *manifest.Manifest, contentDir string, layers []string) error {
	if len(m.Packages.Managers) == 0 {
		return nil
	}

	for _, mgrName := range m.Packages.Managers {
		mgr, ok := packages.Get(mgrName)
		if !ok {
			fmt.Printf("· paquetes %s: gestor no soportado todavía, lo salto.\n", mgrName)
			continue
		}
		declared, err := packages.ReadDeclared(contentDir, layers, mgrName)
		if err != nil {
			return err
		}
		installed, err := mgr.Installed()
		if err != nil {
			return err
		}
		missing := packages.Drift(declared, installed)

		if len(missing) == 0 {
			fmt.Printf("· paquetes %s: al día (%d declarado(s)).\n", mgrName, len(declared))
			continue
		}

		if dryRun {
			fmt.Printf("· paquetes %s: instalaría %d: %v\n", mgrName, len(missing), missing)
			continue
		}

		// COPR (dnf): habilitar los repos declarados antes de instalar.
		if mgrName == "dnf" && m.Packages.DNF != nil && len(m.Packages.DNF.COPR) > 0 {
			if cm, ok := mgr.(packages.COPRManager); ok {
				fmt.Printf("· habilitando %d repo(s) COPR…\n", len(m.Packages.DNF.COPR))
				if err := cm.EnableCOPR(m.Packages.DNF.COPR); err != nil {
					return err
				}
			}
		}

		fmt.Printf("· paquetes %s: instalando %d…\n", mgrName, len(missing))
		if err := mgr.Install(missing); err != nil {
			return err
		}
	}
	return nil
}
