package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.kreikol.dev/dotsmithy/internal/state"
)

// writeStatusContent crea un contenido con dos dimensiones y un perfil, en una
// carpeta cuyo nombre lleva espacios y una comilla simple (para probar el
// quoting de --format env). Devuelve su ruta.
func writeStatusContent(t *testing.T) string {
	t.Helper()
	content := filepath.Join(t.TempDir(), "mis dots 'raros'")
	manifest := `version: 1
dimensions: [ux, identity]
profiles:
  laptop:
    machine: lap
    params: { ux: desktop, identity: work }
`
	for _, dir := range []string{"ux/desktop", "identity/work", "machines/lap"} {
		if err := os.MkdirAll(filepath.Join(content, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(content, "dots.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return content
}

// isolateStatus aísla HOME, XDG y la config de git en temporales, para que ni
// el estado ni la configuración de quien corre los tests se cuelen.
func isolateStatus(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

// saveStatusState deja el estado local como si se hubiera hecho «dots init».
func saveStatusState(t *testing.T, content, remote string) {
	t.Helper()
	if err := state.SaveDefault(&state.State{Profile: "laptop", Content: content, Remote: remote}); err != nil {
		t.Fatal(err)
	}
}

// runStatus ejecuta «dots status» con los args dados y devuelve su salida.
func runStatus(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := newStatusCmd()
	// Como en la raíz real: los errores los saca Execute, no cobra.
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	err := cmd.Execute()
	return out.String(), err
}

// git ejecuta un comando git en dir y falla el test si no sale bien.
func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	base := []string{"-C", dir, "-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false"}
	if out, err := exec.Command("git", append(base, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestStatusWithoutStateFails(t *testing.T) {
	isolateStatus(t)
	for _, format := range []string{"text", "env", "json"} {
		out, err := runStatus(t, "--format", format)
		if !errors.Is(err, errNoState) {
			t.Errorf("--format %s sin estado: quiero errNoState, tengo %v", format, err)
		}
		if out != "" {
			t.Errorf("--format %s sin estado no debería escribir nada, y escribe:\n%s", format, out)
		}
	}
}

func TestStatusRejectsUnknownFormat(t *testing.T) {
	isolateStatus(t)
	_, err := runStatus(t, "--format", "yaml")
	if err == nil || !strings.Contains(err.Error(), `no conozco el formato "yaml": usa text, env o json`) {
		t.Errorf("formato desconocido: error inesperado %v", err)
	}
}

func TestStatusText(t *testing.T) {
	isolateStatus(t)
	content := writeStatusContent(t)
	saveStatusState(t, content, "git@example.com:yo/dots.git")

	out, err := runStatus(t)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"dotsmithy dev\n",
		"  perfil      laptop\n",
		"  máquina     lap\n",
		"  dimensiones ux=desktop, identity=work\n",
		"  capas       shared → ux/desktop → identity/work → machines/lap\n",
		"  contenido   " + content + "\n",
		"  remoto      git@example.com:yo/dots.git\n",
		"  git         el contenido no es un repo git\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en la salida:\n%s", want, out)
		}
	}
}

func TestStatusFlagsWithoutState(t *testing.T) {
	// Con --profile y --content no hace falta haber hecho init.
	isolateStatus(t)
	content := writeStatusContent(t)
	out, err := runStatus(t, "--profile", "laptop", "--content", content)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "  perfil      laptop\n") || strings.Contains(out, "remoto") {
		t.Errorf("salida inesperada:\n%s", out)
	}
}

func TestStatusEnvIsSourceable(t *testing.T) {
	isolateStatus(t)
	content := writeStatusContent(t)
	saveStatusState(t, content, "")

	out, err := runStatus(t, "--format", "env")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "status.env")
	if err := os.WriteFile(file, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	// Lo carga bash de verdad y enseña cada variable en una línea.
	script := `source "$1" && printf '%s\n' "$DOTS_PROFILE" "$DOTS_MACHINE" "$DOTS_PARAM_UX" "$DOTS_PARAM_IDENTITY" "$DOTS_LAYERS" "$DOTS_CONTENT_DIR" "$DOTS_VERSION"`
	got, err := exec.Command("bash", "-c", script, "bash", file).Output()
	if err != nil {
		t.Fatalf("bash no ha podido cargar la salida: %v\n%s", err, out)
	}
	want := strings.Join([]string{"laptop", "lap", "desktop", "work",
		"shared ux/desktop identity/work machines/lap", content, "dev"}, "\n") + "\n"
	if string(got) != want {
		t.Errorf("variables cargadas:\n%s\nquiero:\n%s", got, want)
	}
}

func TestStatusJSON(t *testing.T) {
	isolateStatus(t)
	content := writeStatusContent(t)
	saveStatusState(t, content, "")

	out, err := runStatus(t, "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	var info statusInfo
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		t.Fatalf("no es JSON válido: %v\n%s", err, out)
	}
	if info.Profile != "laptop" || info.Machine != "lap" || info.Content != content || info.Git != nil {
		t.Errorf("JSON inesperado:\n%s", out)
	}
	// Las dimensiones, en el orden del manifiesto.
	if len(info.Params) != 2 || info.Params[0] != (statusParam{"ux", "desktop"}) || info.Params[1] != (statusParam{"identity", "work"}) {
		t.Errorf("params: %+v", info.Params)
	}
	// Las claves son contrato: si cambian, este test lo dice.
	for _, key := range []string{`"version"`, `"profile"`, `"machine"`, `"params"`, `"layers"`, `"content"`, `"remote"`, `"git": null`} {
		if !strings.Contains(out, key) {
			t.Errorf("falta la clave %s en el JSON:\n%s", key, out)
		}
	}
}

func TestStatusGit(t *testing.T) {
	isolateStatus(t)
	content := writeStatusContent(t)
	saveStatusState(t, content, "")
	git(t, content, "init", "-q", "-b", "main")
	git(t, content, "add", ".")
	git(t, content, "commit", "-q", "-m", "inicio")

	gitLine := func() string {
		t.Helper()
		out, err := runStatus(t)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range strings.Split(out, "\n") {
			if strings.HasPrefix(l, "  git ") {
				return strings.TrimSpace(strings.TrimPrefix(l, "  git"))
			}
		}
		t.Fatalf("no hay línea de git:\n%s", out)
		return ""
	}

	steps := []struct {
		name  string
		setup func()
		want  string
	}{
		{"limpio y sin upstream", func() {},
			"rama main · sin cambios · sin rama remota configurada"},
		{"con un fichero nuevo", func() {
			if err := os.WriteFile(filepath.Join(content, "nuevo"), []byte("x"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "rama main · 1 fichero(s) con cambios sin commitear · sin rama remota configurada"},
		{"subido al remoto", func() {
			remote := filepath.Join(t.TempDir(), "remote.git")
			git(t, filepath.Dir(remote), "init", "-q", "--bare", remote)
			git(t, content, "add", ".")
			git(t, content, "commit", "-q", "-m", "nuevo")
			git(t, content, "remote", "add", "origin", remote)
			git(t, content, "push", "-q", "-u", "origin", "main")
		}, "rama main · sin cambios · todo subido a origin/main"},
		{"con un commit sin subir", func() {
			if err := os.WriteFile(filepath.Join(content, "otro"), []byte("y"), 0o644); err != nil {
				t.Fatal(err)
			}
			git(t, content, "add", ".")
			git(t, content, "commit", "-q", "-m", "otro")
		}, "rama main · sin cambios · 1 commit(s) sin subir a origin/main"},
	}
	for _, s := range steps {
		s.setup()
		if got := gitLine(); got != s.want {
			t.Errorf("%s:\n tengo %q\nquiero %q", s.name, got, s.want)
		}
	}
}
