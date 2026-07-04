package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"go.kreikol.dev/dotsmithy/internal/manifest"
	"go.kreikol.dev/dotsmithy/internal/stow"
)

// newLinkCmd: despliega los symlinks a $HOME (modelo Stow), resolviendo las
// capas (shared + perfil). Idempotente: re-ejecutar no rompe nada.
func newLinkCmd() *cobra.Command {
	var contentDir string
	var profile string

	cmd := &cobra.Command{
		Use:   "link",
		Short: "despliega los dotfiles a $HOME (symlinks, modelo Stow)",
		Long: `link resuelve las capas (shared + perfil de máquina) y crea los symlinks
en tu $HOME apuntando a los ficheros reales del repo. Es idempotente: si ya
está enlazado, no hace nada; si falta, lo crea. Con --dry-run enseña el plan
sin tocar nada.

Nota: de momento el perfil y la ruta del contenido se pasan por flags. Cuando
exista el estado local (init), link los cogerá de ahí si no los indicas.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if profile == "" {
				return fmt.Errorf("necesito saber el perfil: pásalo con --profile <nombre>")
			}

			manifestPath := filepath.Join(contentDir, "dots.yaml")
			m, err := manifest.Load(manifestPath)
			if err != nil {
				return err
			}
			if _, ok := m.Profiles[profile]; !ok {
				return fmt.Errorf("el perfil %q no está en el manifiesto (tienes: %s)",
					profile, strings.Join(profileNames(m), ", "))
			}

			layers := m.ResolvedLayers(profile)
			plan, err := stow.BuildPlan(contentDir, m.Stow.Subdir, m.Stow.Target, layers)
			if err != nil {
				return err
			}

			if dryRun {
				fmt.Printf("dry-run del perfil %q (capas: %s):\n", profile, strings.Join(layers, ", "))
			}

			logfn := func(s string) {
				// Las líneas "= ..." (ya ok) solo se enseñan en verbose.
				if !verbose && strings.HasPrefix(s, "= ") {
					return
				}
				fmt.Println("  " + s)
			}
			if err := plan.Apply(dryRun, logfn); err != nil {
				return err
			}

			changes := plan.Changes()
			conflicts := len(plan.Conflicts())
			okAlready := len(plan.Actions) - changes - conflicts
			if dryRun {
				fmt.Printf("dry-run: %d cambio(s), %d ya ok, %d conflicto(s).\n", changes, okAlready, conflicts)
			} else {
				fmt.Printf("listo: %d aplicado(s), %d ya estaban, %d conflicto(s).\n", changes, okAlready, conflicts)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&contentDir, "content", ".", "ruta al repo de contenido (donde está dots.yaml)")
	cmd.Flags().StringVar(&profile, "profile", "", "perfil de máquina a aplicar")
	return cmd
}

// profileNames devuelve los nombres de perfil ordenados, para mensajes de ayuda.
func profileNames(m *manifest.Manifest) []string {
	names := make([]string, 0, len(m.Profiles))
	for name := range m.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
