package manifest

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// SchemaVersion es la única versión de esquema que este motor entiende. Si el
// manifiesto declara otra, se rechaza (evita aplicar un contenido pensado para
// otro motor).
const SchemaVersion = 1

// profilePlaceholder es el comodín que, dentro de una capa, se sustituye por el
// nombre del perfil activo. Ej.: "machines/{profile}" -> "machines/minipc".
const profilePlaceholder = "{profile}"

// Manifest es el modelo tipado de dots.yaml.
type Manifest struct {
	Version   int                `yaml:"version"`
	Profiles  map[string]Profile `yaml:"profiles"`
	Stow      Stow               `yaml:"stow"`
	Packages  Packages           `yaml:"packages"`
	System    []SystemFile       `yaml:"system"`
	Hooks     Hooks              `yaml:"hooks"`
	Externals []External         `yaml:"externals"`
}

// Profile describe un perfil de máquina (se elige explícitamente en `dots init`).
type Profile struct {
	Description string `yaml:"description"`
}

// Stow describe qué se despliega por symlinks y cómo se apilan las capas.
type Stow struct {
	// Target es el destino del stow (normalmente "~", que es $HOME).
	Target string `yaml:"target"`
	// Subdir es el subdirectorio dentro de cada capa que se stowea (ej. "home").
	Subdir string `yaml:"subdir"`
	// Layers es la lista explícita de capas, apiladas en orden. Puede contener
	// el comodín {profile}. El motor no trata ninguna capa de forma especial.
	Layers []string `yaml:"layers"`
}

// Packages agrupa la configuración de paquetes por gestor.
type Packages struct {
	// Managers son los gestores activos (ej. dnf). El motor une
	// <capa>/packages/<gestor>.txt de cada capa activa.
	Managers []string `yaml:"managers"`
	// DNF lleva opciones específicas de dnf (repos COPR a habilitar). Opcional.
	DNF *DNFOptions `yaml:"dnf"`
}

// DNFOptions son las opciones específicas del gestor dnf.
type DNFOptions struct {
	// COPR son los repos COPR a habilitar antes de instalar (ej. "atim/lazygit").
	COPR []string `yaml:"copr"`
}

// SystemFile describe un fichero de sistema (fuera de $HOME, requiere root).
type SystemFile struct {
	Src      string `yaml:"src"`      // ruta dentro del contenido (system/...)
	Dest     string `yaml:"dest"`     // ruta absoluta de destino (/etc/...)
	Type     string `yaml:"type"`     // "symlink" | "copy"
	Mode     string `yaml:"mode"`     // opcional, octal (ej. "0440")
	Owner    string `yaml:"owner"`    // opcional, "user:group" (default root:root)
	Validate string `yaml:"validate"` // opcional, comando de validación sobre copia temporal
}

// Hooks describe los hooks de ciclo de vida.
type Hooks struct {
	// Dir es el directorio raíz de los hooks dentro del contenido (default "hooks").
	Dir string `yaml:"dir"`
	// Points son los puntos de ciclo de vida activos (subconjunto de KnownHookPoints).
	Points []string `yaml:"points"`
}

// External describe un recurso externo que el motor mantiene (git o fichero).
type External struct {
	Dest string `yaml:"dest"` // destino directo (NO se stowea)
	Type string `yaml:"type"` // "git" | "file"
	Repo string `yaml:"repo"` // requerido si type == git
	URL  string `yaml:"url"`  // requerido si type == file
	Ref  string `yaml:"ref"`  // rama/tag/commit para type == git
	Post string `yaml:"post"` // script post a ejecutar (relativo al contenido)
}

// Conjuntos de valores conocidos, usados por la validación.
var (
	// KnownManagers son los gestores de paquetes que el esquema admite. Solo dnf
	// está implementado en v1, pero el esquema reconoce los demás para no romper
	// manifiestos que los declaren de cara a futuras fases.
	KnownManagers = []string{"dnf", "pip", "npm", "cargo"}
	// KnownHookPoints son los puntos de ciclo de vida válidos.
	KnownHookPoints = []string{"post-link", "post-packages", "post-init"}
	// knownSystemTypes son los tipos válidos de fichero de system.
	knownSystemTypes = []string{"symlink", "copy"}
	// knownExternalTypes son los tipos válidos de external.
	knownExternalTypes = []string{"git", "file"}
)

// Load lee, parsea, aplica defaults y valida un dots.yaml desde disco. Es el
// punto de entrada normal del motor. Devuelve el manifiesto ya listo para usar,
// o un error explicando qué falla (agregado, todas las pegas de una vez).
func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no he podido leer el manifiesto %q: %w", path, err)
	}
	m, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("en %q: %w", path, err)
	}
	return m, nil
}

// Parse hace lo mismo que Load pero sobre bytes en memoria (sin tocar disco).
// Separado para poder testear el parseo y la validación sin ficheros.
func Parse(data []byte) (*Manifest, error) {
	var m Manifest
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	// KnownFields hace que un campo desconocido (ej. una errata) sea un error,
	// en vez de ignorarse en silencio.
	dec.KnownFields(true)
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("el YAML no es válido: %w", err)
	}
	m.applyDefaults()
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// applyDefaults rellena los valores por defecto de los campos opcionales, para
// que el resto del motor no tenga que repetir esa lógica.
func (m *Manifest) applyDefaults() {
	if m.Stow.Target == "" {
		m.Stow.Target = "~"
	}
	if m.Stow.Subdir == "" {
		m.Stow.Subdir = "home"
	}
	if m.Hooks.Dir == "" {
		m.Hooks.Dir = "hooks"
	}
	// Si no se declaran puntos de hook, se asumen todos los conocidos.
	if len(m.Hooks.Points) == 0 {
		m.Hooks.Points = append([]string(nil), KnownHookPoints...)
	}
}

// ResolvedLayers devuelve las capas del manifiesto con el comodín {profile}
// sustituido por el perfil dado, en orden. Es una operación pura derivada del
// manifiesto (no comprueba que el perfil exista: eso es responsabilidad de quien
// llama, que ya conoce el perfil activo).
func (m *Manifest) ResolvedLayers(profile string) []string {
	out := make([]string, len(m.Stow.Layers))
	for i, layer := range m.Stow.Layers {
		out[i] = strings.ReplaceAll(layer, profilePlaceholder, profile)
	}
	return out
}

// HasManager indica si un gestor de paquetes está activo en el manifiesto.
func (m *Manifest) HasManager(name string) bool {
	for _, mgr := range m.Packages.Managers {
		if mgr == name {
			return true
		}
	}
	return false
}
