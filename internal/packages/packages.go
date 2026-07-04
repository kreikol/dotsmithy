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

// AddToList registra paquetes en la lista de un gestor dentro de una capa
// (<contentDir>/<layer>/packages/<manager>.txt), sin duplicar los que ya estén.
// Devuelve los que ha añadido de verdad. No instala nada (de eso se encarga el
// gestor); solo toca el fichero declarativo.
func AddToList(contentDir, layer, manager string, pkgs []string) ([]string, error) {
	path := filepath.Join(contentDir, layer, "packages", manager+".txt")

	existing := make(map[string]bool)
	if data, err := os.ReadFile(path); err == nil {
		for _, p := range parseList(data) {
			existing[p] = true
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("no he podido leer %q: %w", path, err)
	}

	var toAdd []string
	seen := make(map[string]bool)
	for _, p := range pkgs {
		p = strings.TrimSpace(p)
		if p == "" || existing[p] || seen[p] {
			continue
		}
		seen[p] = true
		toAdd = append(toAdd, p)
	}
	if len(toAdd) == 0 {
		return nil, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("no he podido crear el directorio de la lista: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("no he podido abrir %q para escribir: %w", path, err)
	}
	defer f.Close()
	for _, p := range toAdd {
		if _, err := fmt.Fprintln(f, p); err != nil {
			return nil, fmt.Errorf("no he podido escribir en %q: %w", path, err)
		}
	}
	return toAdd, nil
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
