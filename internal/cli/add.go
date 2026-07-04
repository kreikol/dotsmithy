package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"go.kreikol.dev/dotsmithy/internal/packages"
)

// newAddCmd: da de alta uno o varios paquetes: los registra en la lista de una
// capa del contenido (por defecto shared) y los instala (D18). Así el alta queda
// declarada en el repo y aplicada en la máquina de una vez.
func newAddCmd() *cobra.Command {
	var layer string
	var contentDir string

	cmd := &cobra.Command{
		Use:   "add <gestor> <paquete>...",
		Short: "registra e instala uno o varios paquetes",
		Long: `add coge uno o varios paquetes de un gestor (ej. dnf), los registra en la
lista de una capa del contenido (por defecto «shared», usa --layer para otra) y
los instala. Deja el alta declarada en el repo y aplicada en la máquina.

Ejemplo: dots add dnf ripgrep fzf`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			mgrName, pkgs := args[0], args[1:]

			mgr, ok := packages.Get(mgrName)
			if !ok {
				return fmt.Errorf("el gestor %q no está soportado todavía", mgrName)
			}

			content, err := resolveContent(contentDir)
			if err != nil {
				return err
			}

			if dryRun {
				fmt.Printf("dry-run: registraría en %s/packages/%s.txt e instalaría: %v\n", layer, mgrName, pkgs)
				return nil
			}

			// 1) Registrar en la lista de la capa (declarativo).
			added, err := packages.AddToList(content, layer, mgrName, pkgs)
			if err != nil {
				return err
			}
			if len(added) > 0 {
				fmt.Printf("registrados en %s/packages/%s.txt: %v\n", layer, mgrName, added)
			} else {
				fmt.Println("ya estaban todos registrados; no toco la lista.")
			}

			// 2) Instalar (dnf es idempotente: los que ya estén no se tocan).
			fmt.Printf("instalando: %v\n", pkgs)
			return mgr.Install(pkgs)
		},
	}

	cmd.Flags().StringVar(&layer, "layer", "shared", "capa del contenido donde registrar (ej. shared, machines/minipc)")
	cmd.Flags().StringVar(&contentDir, "content", "", "ruta al repo de contenido (por defecto: la del estado local, o el directorio actual)")
	return cmd
}
