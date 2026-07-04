// Paquete cli define el árbol de comandos del motor (raíz dots + subcomandos).
//
// En F0 los subcomandos son stubs: solo declaran su interfaz (nombre, ayuda,
// flags) para fijar el contrato de la CLI. La lógica real llega en F1+.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Flags globales, compartidos por todos los subcomandos. Se resuelven una vez
// y la lógica de cada comando los consulta.
var (
	// dryRun: enseña qué haría sin tocar nada. Clave para probar sin miedo.
	dryRun bool
	// assumeYes: no pregunta nada (modo no interactivo, para CI y scripts).
	assumeYes bool
	// verbose: saca más detalle de lo que va pasando.
	verbose bool
)

// version la inyecta el linker en los builds de release (goreleaser, vía -X).
// En builds de desarrollo se queda como "dev".
var version = "dev"

// rootCmd es la raíz del árbol. El binario canónico es "dots" (con alias "ds"
// creado por el instalador como symlink).
var rootCmd = &cobra.Command{
	Use:   "dots",
	Short: "dotsmithy: la herrería de tus dotfiles",
	Long: `dots es el motor de dotsmithy: despliega tus dotfiles (modelo Stow),
gestiona paquetes, perfiles de máquina, hooks y externals.

El motor es genérico y no guarda nada personal: tu contenido (dotfiles, listas
de paquetes, perfiles) vive en un repo aparte y se engancha por el manifiesto
dots.yaml.`,
	// Silenciamos el uso/errores automáticos: los gestionamos nosotros para
	// dar mensajes coloquiales y en castellano.
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version,
}

// Execute arranca la CLI. Es el único símbolo que main necesita.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "ups, algo ha fallado:", err)
		os.Exit(1)
	}
}

func init() {
	// Flags globales (persistentes: valen para la raíz y todos los hijos).
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n", false,
		"enseña qué haría sin tocar nada")
	rootCmd.PersistentFlags().BoolVarP(&assumeYes, "yes", "y", false,
		"no pregunta nada (modo no interactivo)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"saca más detalle de lo que va pasando")

	// Registro de los subcomandos de la v1 (D18).
	rootCmd.AddCommand(
		newInitCmd(),
		newLinkCmd(),
		newUpdateCmd(),
		newAddCmd(),
		newSyncCmd(),
		newExternalsCmd(),
	)
}

// notImplemented es el cuerpo compartido de los stubs de F0: avisa, en tono
// cercano, de que el comando todavía no hace nada.
func notImplemented(name string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		fmt.Printf("«dots %s» todavía no hace nada: es un esqueleto de F0.\n", name)
		return nil
	}
}
