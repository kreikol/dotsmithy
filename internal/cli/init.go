package cli

import "github.com/spf13/cobra"

// newInitCmd: arranque en frío de una máquina. Gestiona la clave SSH, elige
// perfil, clona el contenido privado y aplica todo por primera vez. Es el
// comando al que cede el control el shim de bootstrap.
func newInitCmd() *cobra.Command {
	var profile string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "arranca una máquina desde cero (clave SSH, perfil, clon, aplica)",
		Long: `init deja una máquina limpia lista: se asegura de la clave SSH, te deja
elegir el perfil de máquina, clona tu repo de contenido y aplica todo.`,
		RunE: notImplemented("init"),
	}
	// El perfil se indica explícitamente (nada de autodetección).
	cmd.Flags().StringVar(&profile, "profile", "", "perfil de máquina a instalar (ej. minipc)")
	return cmd
}
