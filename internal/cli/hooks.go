package cli

import (
	"fmt"

	"go.kreikol.dev/dotsmithy/internal/hooks"
	"go.kreikol.dev/dotsmithy/internal/manifest"
	"go.kreikol.dev/dotsmithy/internal/stow"
)

// applyHook ejecuta los hooks de un punto, si el manifiesto lo declara activo.
// Arma el contexto (perfil, contenido, destino expandido, capas) y respeta
// --dry-run. Lo usan init y update.
func applyHook(m *manifest.Manifest, contentDir, profile, point string) error {
	if !hookEnabled(m, point) {
		return nil
	}
	target, err := stow.ExpandTarget(m.Stow.Target)
	if err != nil {
		return err
	}
	ctx := hooks.Context{
		Profile:    profile,
		ContentDir: contentDir,
		Target:     target,
		Layers:     m.ResolvedLayers(profile),
		HooksDir:   m.Hooks.Dir,
	}
	return hooks.Run(point, ctx, dryRun, func(s string) {
		fmt.Println("  " + s)
	})
}

// hookEnabled indica si el manifiesto declara activo un punto de hook.
func hookEnabled(m *manifest.Manifest, point string) bool {
	for _, p := range m.Hooks.Points {
		if p == point {
			return true
		}
	}
	return false
}
