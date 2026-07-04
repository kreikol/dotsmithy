package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"go.kreikol.dev/dotsmithy/internal/externals"
	"go.kreikol.dev/dotsmithy/internal/manifest"
	"go.kreikol.dev/dotsmithy/internal/stow"
)

// newExternalsCmd: gestiona los recursos externos declarados (fetch + post por
// external). Best-effort: si uno falla, avisa y sigue con el resto. Invocable
// suelto, además de correr como fase final de init y update (ADR 0010).
func newExternalsCmd() *cobra.Command {
	var profile string
	var contentDir string

	cmd := &cobra.Command{
		Use:   "externals",
		Short: "trae y prepara los recursos externos declarados",
		Long: `externals baja los recursos externos que declaras (repos git o ficheros)
a su destino y ejecuta su post cuando los trae o cuando cambian. Va en modo
best-effort: si alguno falla, te avisa y sigue con el resto.

Perfil y contenido se resuelven como en link: flags o el estado local.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			profile, contentDir, err := resolveProfileAndContent(profile, contentDir)
			if err != nil {
				return err
			}
			m, err := loadManifestForProfile(contentDir, profile)
			if err != nil {
				return err
			}
			// Suelto sí señala fallo (exit != 0) para que se note.
			if failed := runExternals(m, contentDir, profile); failed > 0 {
				return fmt.Errorf("%d external(s) fallaron", failed)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "perfil de máquina (por defecto: el del estado local)")
	cmd.Flags().StringVar(&contentDir, "content", "", "ruta al repo de contenido (por defecto: la del estado local, o el directorio actual)")
	return cmd
}

// runExternals procesa la fase de externals y devuelve cuántos fallaron. Imprime
// un resumen. Lo comparten init, update y el comando suelto.
func runExternals(m *manifest.Manifest, contentDir, profile string) int {
	if len(m.Externals) == 0 {
		return 0
	}
	target, err := stow.ExpandTarget(m.Stow.Target)
	if err != nil {
		// No debería pasar; si pasa, lo tratamos como fallo de la fase.
		fmt.Printf("· externals: no he podido resolver el destino base: %v\n", err)
		return len(m.Externals)
	}
	fmt.Printf("· externals: %d recurso(s)…\n", len(m.Externals))
	ctx := externals.Context{
		Profile:    profile,
		ContentDir: contentDir,
		Target:     target,
		Layers:     m.ResolvedLayers(profile),
	}
	failed := externals.Apply(m.Externals, ctx, dryRun, func(s string) {
		fmt.Println("  " + s)
	})
	if failed > 0 {
		fmt.Printf("· externals: %d fallaron (best-effort: el resto se aplicó).\n", failed)
	}
	return failed
}
