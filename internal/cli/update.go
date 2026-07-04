package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

// newUpdateCmd: trae los últimos cambios del contenido y los reaplica. Reutiliza
// link y paquetes (con sus hooks post-link y post-packages). NO ejecuta post-init:
// ese es one-off del primer init (ADR 0007).
func newUpdateCmd() *cobra.Command {
	var profile string
	var contentDir string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "trae los cambios del contenido y los aplica",
		Long: `update hace pull de tu repo de contenido y vuelve a aplicar: enlaza lo
nuevo (link + post-link), sincroniza paquetes (+ post-packages) y deja todo al
día. No ejecuta post-init (eso es del primer init).

Perfil y contenido se resuelven como en link: flags o el estado local.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			profile, contentDir, err := resolveProfileAndContent(profile, contentDir)
			if err != nil {
				return err
			}

			// 1) Traer los cambios del contenido (si es un repo git).
			if err := pullContent(contentDir); err != nil {
				return err
			}

			// 2) Cargar el manifiesto ya actualizado y validar el perfil.
			m, err := loadManifestForProfile(contentDir, profile)
			if err != nil {
				return err
			}

			// 3) Reaplicar: link + system → post-link → paquetes → post-packages.
			if err := applyStow(m, contentDir, profile); err != nil {
				return err
			}
			if err := applySystem(m, contentDir); err != nil {
				return err
			}
			if err := applyHook(m, contentDir, profile, "post-link"); err != nil {
				return err
			}
			if err := applyPackages(m, contentDir, m.ResolvedLayers(profile)); err != nil {
				return err
			}
			if err := applyHook(m, contentDir, profile, "post-packages"); err != nil {
				return err
			}
			// Fase final: externals (best-effort, no aborta el update).
			runExternals(m, contentDir, profile)
			return nil
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "perfil de máquina (por defecto: el del estado local)")
	cmd.Flags().StringVar(&contentDir, "content", "", "ruta al repo de contenido (por defecto: la del estado local)")
	return cmd
}

// pullContent hace git pull --ff-only en el contenido si es un repo. Si no lo es
// (ej. un --content local suelto), avisa y sigue sin pull. En dry-run no toca.
func pullContent(contentDir string) error {
	if _, err := os.Stat(filepath.Join(contentDir, ".git")); err != nil {
		fmt.Printf("el contenido en %s no es un repo git; me salto el pull.\n", contentDir)
		return nil
	}
	if dryRun {
		fmt.Println("dry-run: haría git pull --ff-only del contenido.")
		return nil
	}
	fmt.Println("trayendo cambios del contenido (git pull)…")
	cmd := exec.Command("git", "-C", contentDir, "pull", "--ff-only")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("no he podido hacer pull del contenido: %w", err)
	}
	return nil
}
