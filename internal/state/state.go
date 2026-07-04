// Paquete state: estado LOCAL del motor en cada máquina (ADR 0003).
//
// Lo que cambia por máquina no es el manifiesto, sino el "contexto de
// instalación": qué perfil está activo aquí y dónde está el repo de contenido.
// Eso se resuelve una vez en `dots init` y se guarda fuera del repo, en
// ~/.config/dots/state.yaml, para que el resto de comandos no tengan que
// repetirlo.
package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// fileHeader se antepone al YAML guardado, para dejar claro que lo gestiona el
// motor y no conviene editarlo a mano.
const fileHeader = "# Estado local de dotsmithy (lo gestiona el motor). No lo edites a mano.\n"

// State es el estado local de una máquina.
type State struct {
	// Profile es el perfil de máquina activo aquí (ej. "minipc").
	Profile string `yaml:"profile"`
	// Content es la ruta local al repo de contenido (donde vive dots.yaml).
	Content string `yaml:"content"`
	// Remote es la URL de origen del contenido (opcional, informativa: de dónde
	// se clonó, útil para update).
	Remote string `yaml:"remote,omitempty"`
}

// DefaultPath devuelve la ruta canónica del estado: $XDG_CONFIG_HOME/dots/state.yaml
// (o ~/.config/dots/state.yaml si XDG no está definido).
func DefaultPath() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("no he podido averiguar tu directorio de config: %w", err)
	}
	return filepath.Join(cfg, "dots", "state.yaml"), nil
}

// Load lee el estado desde una ruta. Si el fichero no existe, el error cumple
// errors.Is(err, fs.ErrNotExist), para que quien llama distinga "aún no hay init".
func Load(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err // se conserva fs.ErrNotExist para que el llamante lo detecte
	}
	var s State
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("el estado local %q está corrupto: %w", path, err)
	}
	return &s, nil
}

// LoadDefault carga el estado de la ruta canónica. found indica si existía; si no
// existe, devuelve (nil, false, nil) sin error (es un caso normal: falta init).
func LoadDefault() (s *State, found bool, err error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, false, err
	}
	s, err = Load(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return s, true, nil
}

// Save escribe el estado en una ruta, creando los directorios que hagan falta.
func Save(path string, s *State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("no he podido crear el directorio del estado: %w", err)
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("no he podido serializar el estado: %w", err)
	}
	if err := os.WriteFile(path, append([]byte(fileHeader), data...), 0o644); err != nil {
		return fmt.Errorf("no he podido guardar el estado en %q: %w", path, err)
	}
	return nil
}

// SaveDefault guarda el estado en la ruta canónica.
func SaveDefault(s *State) error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	return Save(path, s)
}
