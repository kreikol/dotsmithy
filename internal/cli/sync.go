package cli

import "github.com/spf13/cobra"

// newSyncCmd: concilia los paquetes instalados con los declarados en el
// manifiesto (unión por capa). En su forma read-only, informa del drift.
func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "concilia los paquetes con lo declarado (o informa del drift)",
		Long: `sync mira los paquetes que declaras (por capa) y los pone al día. En modo
solo-lectura te cuenta las diferencias sin tocar nada.`,
		RunE: notImplemented("sync"),
	}
}
