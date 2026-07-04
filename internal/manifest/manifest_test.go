package manifest

import (
	"path/filepath"
	"strings"
	"testing"
)

// fullValid es un manifiesto válido que ejercita todas las secciones.
const fullValid = `
version: 1
profiles:
  minipc: { description: "Escritorio fijo" }
  portatil: { description: "Portátil" }
stow:
  target: ~
  subdir: home
  layers:
    - shared
    - "machines/{profile}"
packages:
  managers: [dnf, cargo]
  dnf:
    copr: [atim/lazygit]
system:
  - src: system/sudoers/mruiz
    dest: /etc/sudoers.d/mruiz
    type: copy
    mode: "0440"
    validate: "visudo -cf {file}"
hooks:
  dir: hooks
  points: [post-link, post-init]
externals:
  - dest: ~/.config/nvim
    type: git
    repo: git@github.com:miriam/nvim-config.git
    ref: main
    post: hooks/externals/nvim-sync.sh
  - dest: ~/.zim/zimfw.zsh
    type: file
    url: https://example.com/zimfw.zsh
`

func TestParseValidFull(t *testing.T) {
	m, err := Parse([]byte(fullValid))
	if err != nil {
		t.Fatalf("no esperaba error, tengo: %v", err)
	}
	if m.Version != 1 {
		t.Errorf("version: quiero 1, tengo %d", m.Version)
	}
	if len(m.Profiles) != 2 {
		t.Errorf("profiles: quiero 2, tengo %d", len(m.Profiles))
	}
	if got := m.Profiles["minipc"].Description; got != "Escritorio fijo" {
		t.Errorf("descripción de minipc: quiero %q, tengo %q", "Escritorio fijo", got)
	}
	if len(m.Stow.Layers) != 2 {
		t.Errorf("layers: quiero 2, tengo %d", len(m.Stow.Layers))
	}
	if m.Packages.DNF == nil || len(m.Packages.DNF.COPR) != 1 {
		t.Errorf("copr: quiero 1 repo, tengo %+v", m.Packages.DNF)
	}
	if len(m.System) != 1 || m.System[0].Type != "copy" {
		t.Errorf("system mal parseado: %+v", m.System)
	}
	if len(m.Externals) != 2 {
		t.Errorf("externals: quiero 2, tengo %d", len(m.Externals))
	}
}

func TestDefaultsApplied(t *testing.T) {
	// Manifiesto mínimo: los opcionales deben tomar sus defaults.
	const minimal = `
version: 1
profiles:
  minipc: {}
stow:
  layers: [shared]
`
	m, err := Parse([]byte(minimal))
	if err != nil {
		t.Fatalf("no esperaba error, tengo: %v", err)
	}
	if m.Stow.Target != "~" {
		t.Errorf("default target: quiero \"~\", tengo %q", m.Stow.Target)
	}
	if m.Stow.Subdir != "home" {
		t.Errorf("default subdir: quiero \"home\", tengo %q", m.Stow.Subdir)
	}
	if m.Hooks.Dir != "hooks" {
		t.Errorf("default hooks.dir: quiero \"hooks\", tengo %q", m.Hooks.Dir)
	}
	if len(m.Hooks.Points) != len(KnownHookPoints) {
		t.Errorf("default hooks.points: quiero %v, tengo %v", KnownHookPoints, m.Hooks.Points)
	}
}

func TestResolvedLayers(t *testing.T) {
	m := &Manifest{Stow: Stow{Layers: []string{"shared", "machines/{profile}", "extra/{profile}/x"}}}
	got := m.ResolvedLayers("minipc")
	want := []string{"shared", "machines/minipc", "extra/minipc/x"}
	if len(got) != len(want) {
		t.Fatalf("longitud: quiero %v, tengo %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("capa[%d]: quiero %q, tengo %q", i, want[i], got[i])
		}
	}
}

func TestHasManager(t *testing.T) {
	m := &Manifest{Packages: Packages{Managers: []string{"dnf", "npm"}}}
	if !m.HasManager("dnf") {
		t.Error("debería tener dnf")
	}
	if m.HasManager("cargo") {
		t.Error("no debería tener cargo")
	}
}

func TestParseUnknownFieldFails(t *testing.T) {
	// Una errata en un campo (typpo) debe fallar, no ignorarse.
	const withTypo = `
version: 1
profiles:
  minipc: {}
stow:
  layers: [shared]
  targett: ~
`
	if _, err := Parse([]byte(withTypo)); err == nil {
		t.Fatal("esperaba error por campo desconocido, no lo hubo")
	}
}

func TestValidationErrors(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want string // fragmento que debe aparecer en el error
	}{
		{
			name: "versión no soportada",
			yaml: "version: 2\nprofiles:\n  m: {}\nstow:\n  layers: [shared]\n",
			want: "version",
		},
		{
			name: "sin perfiles",
			yaml: "version: 1\nstow:\n  layers: [shared]\n",
			want: "al menos un perfil",
		},
		{
			name: "sin capas",
			yaml: "version: 1\nprofiles:\n  m: {}\nstow:\n  layers: []\n",
			want: "al menos una capa",
		},
		{
			name: "gestor desconocido",
			yaml: "version: 1\nprofiles:\n  m: {}\nstow:\n  layers: [shared]\npackages:\n  managers: [brew]\n",
			want: "gestor desconocido",
		},
		{
			name: "copr sin dnf",
			yaml: "version: 1\nprofiles:\n  m: {}\nstow:\n  layers: [shared]\npackages:\n  managers: [cargo]\n  dnf:\n    copr: [atim/lazygit]\n",
			want: "dnf no está",
		},
		{
			name: "system sin type válido",
			yaml: "version: 1\nprofiles:\n  m: {}\nstow:\n  layers: [shared]\nsystem:\n  - src: a\n    dest: /b\n    type: hardlink\n",
			want: "type \"hardlink\" inválido",
		},
		{
			name: "system con mode inválido",
			yaml: "version: 1\nprofiles:\n  m: {}\nstow:\n  layers: [shared]\nsystem:\n  - src: a\n    dest: /b\n    type: copy\n    mode: \"999x\"\n",
			want: "octal válido",
		},
		{
			name: "hook point desconocido",
			yaml: "version: 1\nprofiles:\n  m: {}\nstow:\n  layers: [shared]\nhooks:\n  points: [pre-link]\n",
			want: "punto desconocido",
		},
		{
			name: "external git sin repo",
			yaml: "version: 1\nprofiles:\n  m: {}\nstow:\n  layers: [shared]\nexternals:\n  - dest: ~/x\n    type: git\n",
			want: "type git requiere repo",
		},
		{
			name: "external file sin url",
			yaml: "version: 1\nprofiles:\n  m: {}\nstow:\n  layers: [shared]\nexternals:\n  - dest: ~/x\n    type: file\n",
			want: "type file requiere url",
		},
		{
			name: "external type inválido",
			yaml: "version: 1\nprofiles:\n  m: {}\nstow:\n  layers: [shared]\nexternals:\n  - dest: ~/x\n    type: svn\n",
			want: "inválido",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.yaml))
			if err == nil {
				t.Fatalf("esperaba error con %q, pero parseó bien", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("el error no menciona %q; error: %v", tc.want, err)
			}
		})
	}
}

func TestValidationAggregatesProblems(t *testing.T) {
	// Manifiesto con varias pegas a la vez: deben salir todas juntas.
	const many = `
version: 3
profiles: {}
stow:
  layers: []
`
	_, err := Parse([]byte(many))
	if err == nil {
		t.Fatal("esperaba error")
	}
	for _, frag := range []string{"version", "al menos un perfil", "al menos una capa"} {
		if !strings.Contains(err.Error(), frag) {
			t.Errorf("el error agregado no menciona %q; error: %v", frag, err)
		}
	}
}

func TestLoadFromDisk(t *testing.T) {
	m, err := Load(filepath.Join("testdata", "minipc.yaml"))
	if err != nil {
		t.Fatalf("no esperaba error cargando el fixture: %v", err)
	}
	if _, ok := m.Profiles["minipc"]; !ok {
		t.Error("el fixture debería tener el perfil minipc")
	}
	if got := m.ResolvedLayers("minipc"); got[1] != "machines/minipc" {
		t.Errorf("capa resuelta: quiero machines/minipc, tengo %q", got[1])
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join("testdata", "no-existe.yaml")); err == nil {
		t.Fatal("esperaba error al cargar un fichero inexistente")
	}
}
