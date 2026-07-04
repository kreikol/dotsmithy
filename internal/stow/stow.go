package stow

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ignoredBasenames son ficheros que NO se despliegan aunque estén en una capa:
// centinelas que solo sirven para que git conserve directorios vacíos. Nunca son
// dotfiles de verdad.
var ignoredBasenames = map[string]bool{
	".gitkeep": true,
	".keep":    true,
}

// ActionKind clasifica qué hay que hacer con un destino concreto.
type ActionKind int

const (
	// ActionCreate: el destino no existe, se creará el symlink.
	ActionCreate ActionKind = iota
	// ActionUpdate: ya hay un symlink pero apunta a otro sitio, se repuntará.
	ActionUpdate
	// ActionSkip: ya existe el symlink correcto, no hay nada que hacer.
	ActionSkip
	// ActionConflict: hay un fichero o directorio REAL en medio, no se toca.
	ActionConflict
)

func (k ActionKind) String() string {
	switch k {
	case ActionCreate:
		return "crear"
	case ActionUpdate:
		return "actualizar"
	case ActionSkip:
		return "ya ok"
	case ActionConflict:
		return "conflicto"
	default:
		return "?"
	}
}

// Action es una entrada del plan: un symlink destino -> fuente y qué hacer con él.
type Action struct {
	Dest   string // ruta absoluta del symlink en el destino ($HOME/...)
	Source string // ruta absoluta del fichero real en el contenido
	Kind   ActionKind
}

// Plan es el conjunto de acciones calculado, en orden determinista por destino.
type Plan struct {
	Actions []Action
}

// Conflicts devuelve solo las acciones en conflicto.
func (p Plan) Conflicts() []Action {
	var out []Action
	for _, a := range p.Actions {
		if a.Kind == ActionConflict {
			out = append(out, a)
		}
	}
	return out
}

// Changes cuenta las acciones que cambiarían algo (crear o actualizar).
func (p Plan) Changes() int {
	n := 0
	for _, a := range p.Actions {
		if a.Kind == ActionCreate || a.Kind == ActionUpdate {
			n++
		}
	}
	return n
}

// BuildPlan calcula el plan de symlinks: recorre cada capa (en orden), junta los
// ficheros a enlazar (la última capa gana ante un mismo destino) y clasifica qué
// hacer con cada destino mirando el estado actual del disco. No modifica nada.
//
//   - contentDir: raíz del repo de contenido.
//   - subdir: subdirectorio de cada capa que se stowea (ej. "home").
//   - target: destino ("~" se expande a $HOME).
//   - layers: capas ya resueltas (sin comodines), en orden de aplicación.
func BuildPlan(contentDir, subdir, target string, layers []string) (Plan, error) {
	targetDir, err := ExpandTarget(target)
	if err != nil {
		return Plan{}, err
	}

	// desired: destino absoluto -> fuente absoluta. order preserva el primer
	// encuentro para no depender del mapa; luego se ordena para ser determinista.
	desired := make(map[string]string)
	for _, layer := range layers {
		root := filepath.Join(contentDir, layer, subdir)
		info, err := os.Stat(root)
		if err != nil {
			if os.IsNotExist(err) {
				// Una capa puede no tener subdirectorio de stow: no pasa nada.
				continue
			}
			return Plan{}, fmt.Errorf("no he podido mirar la capa %q: %w", root, err)
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if ignoredBasenames[d.Name()] {
				// Centinela de git (.gitkeep/.keep): no se despliega.
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			src, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			dest := filepath.Join(targetDir, rel)
			desired[dest] = src // la última capa gana
			return nil
		})
		if err != nil {
			return Plan{}, fmt.Errorf("recorriendo la capa %q: %w", root, err)
		}
	}

	dests := make([]string, 0, len(desired))
	for dest := range desired {
		dests = append(dests, dest)
	}
	sort.Strings(dests)

	plan := Plan{Actions: make([]Action, 0, len(dests))}
	for _, dest := range dests {
		src := desired[dest]
		kind, err := classify(dest, src)
		if err != nil {
			return Plan{}, err
		}
		plan.Actions = append(plan.Actions, Action{Dest: dest, Source: src, Kind: kind})
	}
	return plan, nil
}

// classify decide qué hacer con un destino comparando su estado actual con el
// symlink deseado.
func classify(dest, src string) (ActionKind, error) {
	fi, err := os.Lstat(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return ActionCreate, nil
		}
		return ActionConflict, fmt.Errorf("no he podido mirar %q: %w", dest, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		cur, err := os.Readlink(dest)
		if err != nil {
			return ActionConflict, fmt.Errorf("no he podido leer el symlink %q: %w", dest, err)
		}
		if cur == src {
			return ActionSkip, nil
		}
		return ActionUpdate, nil
	}
	// Fichero o directorio real: no lo tocamos.
	return ActionConflict, nil
}

// Apply ejecuta el plan. Si hay conflictos, aborta ANTES de tocar nada y los
// devuelve como error (nunca pisa ficheros reales). Con dryRun no modifica el
// disco: solo informa por log. log puede ser nil.
func (p Plan) Apply(dryRun bool, log func(string)) error {
	if log == nil {
		log = func(string) {}
	}

	if conflicts := p.Conflicts(); len(conflicts) > 0 {
		var b strings.Builder
		fmt.Fprintf(&b, "hay %d conflicto(s): tienes ficheros reales donde irían symlinks. Quítalos o muévelos y reintenta:", len(conflicts))
		for _, c := range conflicts {
			fmt.Fprintf(&b, "\n  - %s", c.Dest)
		}
		if dryRun {
			// En dry-run informamos igualmente pero no es un fallo de ejecución.
			log(b.String())
		} else {
			return fmt.Errorf("%s", b.String())
		}
	}

	for _, a := range p.Actions {
		switch a.Kind {
		case ActionSkip:
			log(fmt.Sprintf("= %s (ya ok)", a.Dest))
		case ActionConflict:
			// Ya informado arriba; en dry-run se muestra el detalle.
			if dryRun {
				log(fmt.Sprintf("! %s (conflicto: hay algo real ahí)", a.Dest))
			}
		case ActionCreate, ActionUpdate:
			verb := "+"
			if a.Kind == ActionUpdate {
				verb = "~"
			}
			if dryRun {
				log(fmt.Sprintf("%s %s -> %s", verb, a.Dest, a.Source))
				continue
			}
			if err := linkFile(a); err != nil {
				return err
			}
			log(fmt.Sprintf("%s %s -> %s", verb, a.Dest, a.Source))
		}
	}
	return nil
}

// linkFile crea (o repunta) un symlink, creando los directorios intermedios.
func linkFile(a Action) error {
	if err := os.MkdirAll(filepath.Dir(a.Dest), 0o755); err != nil {
		return fmt.Errorf("no he podido crear el directorio de %q: %w", a.Dest, err)
	}
	if a.Kind == ActionUpdate {
		// Solo llegamos aquí si el destino es un symlink (classify lo garantiza),
		// así que borrarlo es seguro: no es un fichero real.
		if err := os.Remove(a.Dest); err != nil {
			return fmt.Errorf("no he podido quitar el symlink viejo %q: %w", a.Dest, err)
		}
	}
	if err := os.Symlink(a.Source, a.Dest); err != nil {
		return fmt.Errorf("no he podido crear el symlink %q: %w", a.Dest, err)
	}
	return nil
}

// ExpandTarget expande un destino que empiece por "~" a la home del usuario.
// Exportada para que otros paquetes (ej. hooks) resuelvan el mismo destino.
func ExpandTarget(target string) (string, error) {
	if target == "~" || strings.HasPrefix(target, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("no he podido averiguar tu HOME: %w", err)
		}
		if target == "~" {
			return home, nil
		}
		return filepath.Join(home, target[2:]), nil
	}
	return target, nil
}
