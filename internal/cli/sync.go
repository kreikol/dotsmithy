package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"go.kreikol.dev/dotsmithy/internal/packages"
)

// newSyncCmd: informa del drift de paquetes (qué declaras que aún no está
// instalado), por gestor y capa. En este PR es SOLO LECTURA: no instala nada.
// La instalación llegará con el comando add.
func newSyncCmd() *cobra.Command {
	var profile string
	var contentDir string

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "informa de qué paquetes declarados faltan por instalar (solo lectura)",
		Long: `sync mira los paquetes que declaras (unión de las capas activas) y los
compara con lo que hay instalado, contándote qué falta. De momento es solo
lectura: no instala nada (eso vendrá con add/instalación).

Perfil y contenido se resuelven igual que en link: flags, o el estado local.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			profile, contentDir, err := resolveProfileAndContent(profile, contentDir)
			if err != nil {
				return err
			}
			m, err := loadManifestForProfile(contentDir, profile)
			if err != nil {
				return err
			}

			layers := m.ResolvedLayers(profile)
			if len(m.Packages.Managers) == 0 {
				fmt.Println("no declaras ningún gestor de paquetes; nada que mirar.")
				return nil
			}

			totalMissing := 0
			for _, mgrName := range m.Packages.Managers {
				declared, err := packages.ReadDeclared(contentDir, layers, mgrName)
				if err != nil {
					return err
				}
				mgr, ok := packages.Get(mgrName)
				if !ok {
					fmt.Printf("· %s: %d declarado(s), pero el gestor aún no está soportado (lo salto).\n", mgrName, len(declared))
					continue
				}
				installed, err := mgr.Installed()
				if err != nil {
					return err
				}
				missing := packages.Drift(declared, installed)
				totalMissing += len(missing)

				fmt.Printf("· %s: %d declarado(s), %d por instalar.\n", mgrName, len(declared), len(missing))
				for _, pkg := range missing {
					fmt.Printf("    - %s\n", pkg)
				}
			}

			if totalMissing == 0 {
				fmt.Println("todo lo declarado está instalado. 👌")
			} else {
				fmt.Printf("faltan %d paquete(s) por instalar. (la instalación llegará con add.)\n", totalMissing)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "perfil de máquina (por defecto: el del estado local)")
	cmd.Flags().StringVar(&contentDir, "content", "", "ruta al repo de contenido (por defecto: la del estado local, o el directorio actual)")
	return cmd
}
