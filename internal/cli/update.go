package cli

import "github.com/spf13/cobra"

// newUpdateCmd: trae los últimos cambios del contenido y los aplica (pull del
// repo + re-link + re-sync de paquetes + hooks). El día a día.
func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "trae los cambios del contenido y los aplica",
		Long: `update hace pull de tu repo de contenido y vuelve a aplicar: enlaza lo
nuevo, sincroniza paquetes y lanza los hooks que toquen.`,
		RunE: notImplemented("update"),
	}
}
