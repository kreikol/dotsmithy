package cli

import "github.com/spf13/cobra"

// newExternalsCmd: gestiona los recursos externos declarados (fetch + post por
// external). Best-effort: si uno falla, avisa y sigue con el resto.
func newExternalsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "externals",
		Short: "trae y prepara los recursos externos declarados",
		Long: `externals baja los recursos externos que declaras y ejecuta su post por
cada uno. Va en modo best-effort: si alguno falla, te avisa y sigue.`,
		RunE: notImplemented("externals"),
	}
}
