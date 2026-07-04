package cli

import "github.com/spf13/cobra"

// newAddCmd: adopta un fichero de $HOME al repo (lo mueve al sitio que toca
// según la capa y crea el symlink de vuelta). Alta cómoda de dotfiles.
func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <ruta>",
		Short: "adopta un fichero de $HOME al repo de contenido",
		Long: `add coge un fichero que ya tienes en $HOME, lo lleva al repo de contenido
(a la capa que le corresponda) y deja el symlink apuntando a él.`,
		RunE: notImplemented("add"),
	}
}
