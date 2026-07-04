package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"go.kreikol.dev/dotsmithy/internal/state"
)

// newInitCmd: arranque en frío de una máquina. Elige perfil, consigue el
// contenido (clonándolo o usando uno local), guarda el estado local y aplica
// todo por primera vez (link). Es el comando al que cede el control el shim.
//
// En esta versión init asume que la clave SSH ya está lista y registrada (caso
// del piloto minipc): no genera ni gestiona claves. Si el clone falla por auth,
// lo dice claro.
func newInitCmd() *cobra.Command {
	var profile string
	var remote string
	var contentDir string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "arranca una máquina desde cero (perfil, contenido, estado, aplica)",
		Long: `init deja una máquina lista: elige el perfil, consigue tu repo de contenido
(clonándolo desde --remote por SSH, o usando el de --content si ya lo tienes en
local), guarda el estado local y aplica todo (link).

Asume que tu clave SSH ya existe y está registrada en GitHub. Si el clone falla
por permisos, revisa que la clave tenga acceso al repo.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if profile == "" {
				return fmt.Errorf("necesito el perfil de máquina: pásalo con --profile <nombre>")
			}
			if remote == "" && contentDir == "" {
				return fmt.Errorf("necesito saber de dónde sale el contenido: usa --remote <url> (para clonarlo) o --content <dir> (uno local)")
			}
			if remote != "" && contentDir != "" {
				return fmt.Errorf("usa --remote o --content, no los dos a la vez")
			}

			// 1) Conseguir el contenido.
			var savedRemote string
			if contentDir == "" {
				dest, err := defaultContentDir()
				if err != nil {
					return err
				}
				if err := cloneOrReuse(remote, dest); err != nil {
					return err
				}
				contentDir = dest
				savedRemote = remote
			}
			absContent, err := filepath.Abs(contentDir)
			if err != nil {
				return err
			}

			// 2) Cargar el manifiesto y validar el perfil (antes de guardar nada).
			m, err := loadManifestForProfile(absContent, profile)
			if err != nil {
				return err
			}

			// 3) Guardar el estado local (perfil + ubicación del contenido).
			if err := state.SaveDefault(&state.State{
				Profile: profile,
				Content: absContent,
				Remote:  savedRemote,
			}); err != nil {
				return err
			}
			fmt.Printf("estado guardado: perfil %q, contenido en %s\n", profile, absContent)

			// 4) Aplicar (link).
			return applyStow(m, absContent, profile)
		},
	}

	cmd.Flags().StringVar(&profile, "profile", "", "perfil de máquina a instalar (ej. minipc)")
	cmd.Flags().StringVar(&remote, "remote", "", "URL (SSH) del repo de contenido a clonar")
	cmd.Flags().StringVar(&contentDir, "content", "", "ruta a un repo de contenido ya presente en local (alternativa a --remote)")
	return cmd
}

// defaultContentDir es dónde init clona el contenido por defecto:
// $XDG_DATA_HOME/dots/content (o ~/.local/share/dots/content).
func defaultContentDir() (string, error) {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("no he podido averiguar tu HOME: %w", err)
		}
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, "dots", "content"), nil
}

// cloneOrReuse clona el contenido en dest, o lo reutiliza si ya hay un repo ahí
// (idempotente). Si en dest hay algo que no es un repo git, aborta.
func cloneOrReuse(remote, dest string) error {
	if _, err := os.Stat(dest); err == nil {
		// dest existe: ¿es ya un repo git?
		if _, err := os.Stat(filepath.Join(dest, ".git")); err == nil {
			fmt.Printf("ya tienes el contenido en %s, lo reutilizo (no clono otra vez)\n", dest)
			return nil
		}
		return fmt.Errorf("en %s hay algo que no es un repo de contenido; quítalo o usa otra ruta", dest)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("no he podido mirar %s: %w", dest, err)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("no he podido crear el directorio para el contenido: %w", err)
	}
	fmt.Printf("clonando el contenido desde %s …\n", remote)
	cmd := exec.Command("git", "clone", remote, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("no he podido clonar %q. ¿Tu clave SSH está registrada en GitHub y tiene acceso al repo? Detalle: %w", remote, err)
	}
	return nil
}
