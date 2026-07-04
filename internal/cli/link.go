package cli

import "github.com/spf13/cobra"

// newLinkCmd: despliega los symlinks a $HOME (modelo Stow), resolviendo las
// capas (base + perfil). Idempotente: re-ejecutar no rompe nada.
func newLinkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "link",
		Short: "despliega los dotfiles a $HOME (symlinks, modelo Stow)",
		Long: `link resuelve las capas (base + perfil de máquina) y crea los symlinks
en tu $HOME apuntando a los ficheros reales del repo. Es idempotente.`,
		RunE: notImplemented("link"),
	}
}
