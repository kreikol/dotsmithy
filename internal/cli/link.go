package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"go.kreikol.dev/dotsmithy/internal/manifest"
	"go.kreikol.dev/dotsmithy/internal/state"
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

Los flags mandan; lo que no indiques, se coge del estado local (lo que dejó
init). Si no hay estado ni flags, no puede saber qué perfil aplicar.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			profile, contentDir, err := resolveProfileAndContent(profile, contentDir)
			if err != nil {
				return err
			}
			m, err := loadManifestForProfile(contentDir, profile)
			if err != nil {
				return err
			}
			return applyStow(m, contentDir, profile)
		},
	}

	cmd.Flags().StringVar(&contentDir, "content", "", "ruta al repo de contenido (por defecto: la del estado local, o el directorio actual)")
	cmd.Flags().StringVar(&profile, "profile", "", "perfil de máquina a aplicar (por defecto: el del estado local)")
	return cmd
}

// resolveProfileAndContent decide qué perfil y qué contenido usar: los flags
// mandan; lo que falte se coge del estado local (lo que dejó init); el contenido
// cae al directorio actual si no hay nada. Sin perfil, error que guía a init.
// Lo comparten link y sync.
func resolveProfileAndContent(profile, contentDir string) (string, string, error) {
	if profile == "" || contentDir == "" {
		if st, found, err := state.LoadDefault(); err != nil {
			return "", "", err
		} else if found {
			if profile == "" {
				profile = st.Profile
			}
			if contentDir == "" {
				contentDir = st.Content
			}
		}
	}
	if contentDir == "" {
		contentDir = "."
	}
	if profile == "" {
		return "", "", fmt.Errorf("no sé qué perfil usar: pásalo con --profile <nombre> o haz antes «dots init»")
	}
	return profile, contentDir, nil
}

// loadManifestForProfile carga el manifiesto del contenido y comprueba que el
// perfil pedido existe. Es el paso común de link e init.
func loadManifestForProfile(contentDir, profile string) (*manifest.Manifest, error) {
	m, err := manifest.Load(filepath.Join(contentDir, "dots.yaml"))
	if err != nil {
		return nil, err
	}
	if _, ok := m.Profiles[profile]; !ok {
		return nil, fmt.Errorf("el perfil %q no está en el manifiesto (tienes: %s)",
			profile, strings.Join(profileNames(m), ", "))
	}
	return m, nil
}

// applyStow construye el plan de symlinks del perfil y lo aplica (respetando los
// flags globales --dry-run y --verbose). Es el corazón de link, reutilizado por
// init tras clonar y guardar el estado.
func applyStow(m *manifest.Manifest, contentDir, profile string) error {
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
