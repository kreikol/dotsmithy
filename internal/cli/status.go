package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"go.kreikol.dev/dotsmithy/internal/hooks"
	"go.kreikol.dev/dotsmithy/internal/state"
)

// Formatos de salida de status.
const (
	statusFormatText = "text"
	statusFormatEnv  = "env"
	statusFormatJSON = "json"
)

// errNoState es el error de status en una máquina sin «dots init» (y sin flags
// que lo suplan). Va a stderr y sale con código 1, para que un script lo note.
var errNoState = errors.New("esta máquina aún no está montada con dotsmithy: haz «dots init» (o dime el perfil y el contenido con --profile y --content)")

// newStatusCmd construye «dots status»: cuenta qué tiene aplicado esta máquina
// (perfil, máquina, dimensiones, pila de capas y contenido). Es solo lectura.
func newStatusCmd() *cobra.Command {
	var contentDir, profile, format string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "enseña el perfil, las capas y el contenido de esta máquina (solo lectura)",
		Args:  noArgs,
		Long: `status te cuenta cómo está montada esta máquina: qué perfil tiene, qué
máquina es, qué valor toma en cada dimensión, qué capas forman su pila y dónde
está el contenido. Si el contenido es un repo git, también la rama, si tienes
cambios sin commitear y si hay commits sin subir (según lo que sabe tu repo
local: no hace fetch). No toca nada.

Con --format eliges la salida: text (por defecto) para leerla tú; env, con
líneas VAR=valor listas para hacer «source» en bash (DOTS_PROFILE, DOTS_MACHINE,
DOTS_PARAM_<DIMENSIÓN>, DOTS_LAYERS, DOTS_CONTENT_DIR y DOTS_VERSION); o json,
un objeto JSON para scripts y otras herramientas.

En una máquina sin «dots init» (y sin --profile), avisa por la salida de error
y termina con código 1, sin escribir nada en la salida normal. Así un script
puede comprobarlo sin tener que leer el texto.

Perfil y contenido se resuelven como en link: flags o el estado local.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if format != statusFormatText && format != statusFormatEnv && format != statusFormatJSON {
				return fmt.Errorf("no conozco el formato %q: usa text, env o json", format)
			}
			info, err := collectStatus(profile, contentDir)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			switch format {
			case statusFormatEnv:
				writeStatusEnv(out, info)
			case statusFormatJSON:
				return writeStatusJSON(out, info)
			default:
				writeStatusText(out, info)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&contentDir, "content", "", "ruta al repo de contenido (por defecto: la del estado local, o el directorio actual)")
	cmd.Flags().StringVar(&profile, "profile", "", "perfil del que enseñar el estado (por defecto: el del estado local)")
	cmd.Flags().StringVar(&format, "format", statusFormatText, "formato de salida: text, env o json")
	return cmd
}

// statusParam es el valor de un perfil en una dimensión. Va en lista, y no en
// mapa, para conservar el orden de las dimensiones del manifiesto.
type statusParam struct {
	Dimension string `json:"dimension"`
	Value     string `json:"value"`
}

// statusGit es el estado del contenido como repo git.
type statusGit struct {
	Branch string `json:"branch"`
	// Changes es el número de ficheros con cambios sin commitear (incluidos los
	// nuevos sin seguimiento).
	Changes int `json:"changes"`
	// Upstream es la rama remota que sigue la local; vacío si no tiene.
	Upstream string `json:"upstream"`
	// Ahead son los commits locales que aún no están en Upstream.
	Ahead int `json:"ahead"`
}

// statusInfo junta todo lo que enseña status. Los nombres JSON son parte del
// contrato de --format json: no se cambian a la ligera.
type statusInfo struct {
	Version string        `json:"version"`
	Profile string        `json:"profile"`
	Machine string        `json:"machine"`
	Params  []statusParam `json:"params"`
	Layers  []string      `json:"layers"`
	Content string        `json:"content"`
	Remote  string        `json:"remote"`
	// Git es nil si el contenido no es un repo git.
	Git *statusGit `json:"git"`
}

// collectStatus resuelve perfil y contenido (flags o estado local), carga el
// manifiesto y reúne la información de status. Sin perfil por ningún lado,
// devuelve errNoState.
func collectStatus(profile, contentDir string) (*statusInfo, error) {
	st, found, err := state.LoadDefault()
	if err != nil {
		return nil, err
	}
	if profile == "" && !found {
		return nil, errNoState
	}
	profile, contentDir, err = resolveProfileAndContent(profile, contentDir)
	if err != nil {
		return nil, err
	}
	m, err := loadManifestForProfile(contentDir, profile)
	if err != nil {
		return nil, err
	}
	if abs, err := filepath.Abs(contentDir); err == nil {
		contentDir = abs
	}

	p := m.Profiles[profile]
	info := &statusInfo{
		Version: version,
		Profile: profile,
		Machine: p.Machine,
		Params:  []statusParam{},
		Layers:  m.Stack(profile),
		Content: contentDir,
	}
	for _, dim := range m.Dimensions {
		info.Params = append(info.Params, statusParam{Dimension: dim, Value: p.Params[dim]})
	}
	if isGitRepo(contentDir) {
		info.Git = gitStatus(contentDir)
		info.Remote = gitOutput(contentDir, "remote", "get-url", "origin")
	}
	if info.Remote == "" && found {
		info.Remote = st.Remote
	}
	return info, nil
}

// gitStatus mira la rama, los cambios sin commitear y los commits sin subir del
// contenido. Lo que git no pueda decir (ej. no hay upstream) se queda vacío, sin
// fallar: status es informativo.
func gitStatus(dir string) *statusGit {
	g := &statusGit{Branch: gitOutput(dir, "rev-parse", "--abbrev-ref", "HEAD")}
	if porcelain := gitOutput(dir, "status", "--porcelain"); porcelain != "" {
		g.Changes = len(strings.Split(porcelain, "\n"))
	}
	g.Upstream = gitOutput(dir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if g.Upstream != "" {
		g.Ahead, _ = strconv.Atoi(gitOutput(dir, "rev-list", "--count", "@{u}..HEAD"))
	}
	return g
}

// gitOutput ejecuta un comando git en dir y devuelve su salida sin espacios al
// final, o "" si falla.
func gitOutput(dir string, args ...string) string {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// writeStatusText escribe la salida para personas.
func writeStatusText(w io.Writer, info *statusInfo) {
	fmt.Fprintf(w, "dotsmithy %s\n\n", info.Version)
	line := func(label, value string) { fmt.Fprintf(w, "  %-11s %s\n", label, value) }
	line("perfil", info.Profile)
	line("máquina", info.Machine)
	if len(info.Params) > 0 {
		parts := make([]string, 0, len(info.Params))
		for _, p := range info.Params {
			parts = append(parts, p.Dimension+"="+p.Value)
		}
		line("dimensiones", strings.Join(parts, ", "))
	}
	line("capas", strings.Join(info.Layers, " → "))
	line("contenido", info.Content)
	if info.Remote != "" {
		line("remoto", info.Remote)
	}
	if info.Git == nil {
		line("git", "el contenido no es un repo git")
		return
	}
	parts := []string{"rama " + info.Git.Branch}
	if info.Git.Changes == 0 {
		parts = append(parts, "sin cambios")
	} else {
		parts = append(parts, fmt.Sprintf("%d fichero(s) con cambios sin commitear", info.Git.Changes))
	}
	switch {
	case info.Git.Upstream == "":
		parts = append(parts, "sin rama remota configurada")
	case info.Git.Ahead > 0:
		parts = append(parts, fmt.Sprintf("%d commit(s) sin subir a %s", info.Git.Ahead, info.Git.Upstream))
	default:
		parts = append(parts, "todo subido a "+info.Git.Upstream)
	}
	line("git", strings.Join(parts, " · "))
}

// writeStatusEnv escribe líneas VAR=valor que se pueden cargar con «source» en
// bash. Las dimensiones usan el mismo nombre de variable que en los hooks.
func writeStatusEnv(w io.Writer, info *statusInfo) {
	kv := func(k, v string) { fmt.Fprintf(w, "%s=%s\n", k, shellQuote(v)) }
	kv("DOTS_PROFILE", info.Profile)
	kv("DOTS_MACHINE", info.Machine)
	for _, p := range info.Params {
		kv(hooks.ParamVar(p.Dimension), p.Value)
	}
	kv("DOTS_LAYERS", strings.Join(info.Layers, " "))
	kv("DOTS_CONTENT_DIR", info.Content)
	kv("DOTS_VERSION", info.Version)
}

// writeStatusJSON escribe la información como un objeto JSON indentado.
func writeStatusJSON(w io.Writer, info *statusInfo) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(info)
}

// shellQuote entrecomilla un valor con comillas simples para que bash lo lea
// tal cual (espacios, $, comillas...). Una comilla simple dentro se escribe
// como '\''.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
