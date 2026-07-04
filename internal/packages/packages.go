package packages

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ReadDeclared lee las listas de paquetes declaradas de un gestor a lo largo de
// las capas activas y devuelve su unión, deduplicada y ordenada.
//
// Cada capa puede tener un fichero <capa>/packages/<gestor>.txt (opcional). Se
// leen, no se stowean: son la fuente de la verdad de qué paquetes quieres.
func ReadDeclared(contentDir string, layers []string, manager string) ([]string, error) {
	set := make(map[string]struct{})
	for _, layer := range layers {
		path := filepath.Join(contentDir, layer, "packages", manager+".txt")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue // la capa no declara este gestor: normal
			}
			return nil, fmt.Errorf("no he podido leer %q: %w", path, err)
		}
		for _, pkg := range parseList(data) {
			set[pkg] = struct{}{}
		}
	}

	out := make([]string, 0, len(set))
	for pkg := range set {
		out = append(out, pkg)
	}
	sort.Strings(out)
	return out, nil
}

// parseList extrae los nombres de paquete de una lista: un paquete por línea,
// ignorando líneas vacías y comentarios (las que empiezan por #). Se admiten
// comentarios al final de línea.
func parseList(data []byte) []string {
	var out []string
	for _, raw := range strings.Split(string(data), "\n") {
		line := raw
		// Comentario al final de línea.
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

// Drift devuelve los paquetes declarados que aún NO están instalados, en el
// mismo orden en que llegan (ReadDeclared los da ordenados).
//
// Solo miramos "lo declarado que falta": no reportamos como sobrantes los
// paquetes del sistema no declarados, porque dotsmithy no gestiona su borrado
// (el modelo es aditivo por capa).
func Drift(declared []string, installed map[string]bool) []string {
	var missing []string
	for _, pkg := range declared {
		if !installed[pkg] {
			missing = append(missing, pkg)
		}
	}
	return missing
}
