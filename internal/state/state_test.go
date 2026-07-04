package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "state.yaml")
	in := &State{Profile: "minipc", Content: "/home/x/content", Remote: "git@github.com:x/c.git"}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if *out != *in {
		t.Errorf("round-trip: guardé %+v, leí %+v", in, out)
	}
	// El fichero lleva la cabecera de aviso.
	data, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(data), "# Estado local de dotsmithy") {
		t.Errorf("falta la cabecera de aviso; contenido:\n%s", data)
	}
}

func TestRemoteOmitidoSiVacio(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.yaml")
	if err := Save(path, &State{Profile: "minipc", Content: "/c"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "remote:") {
		t.Errorf("remote vacío no debería serializarse; contenido:\n%s", data)
	}
}

func TestDefaultPathRespetaXDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	got, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(tmp, "dots", "state.yaml")
	if got != want {
		t.Errorf("DefaultPath: quiero %q, tengo %q", want, got)
	}
}

func TestLoadDefaultNoExiste(t *testing.T) {
	// XDG apuntando a un tmp vacío: no hay estado todavía.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s, found, err := LoadDefault()
	if err != nil {
		t.Fatalf("no esperaba error cuando falta el estado: %v", err)
	}
	if found {
		t.Error("found debería ser false cuando no hay estado")
	}
	if s != nil {
		t.Error("s debería ser nil cuando no hay estado")
	}
}

func TestLoadDefaultExiste(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	if err := SaveDefault(&State{Profile: "minipc", Content: "/c"}); err != nil {
		t.Fatal(err)
	}
	s, found, err := LoadDefault()
	if err != nil || !found {
		t.Fatalf("esperaba encontrarlo: found=%v err=%v", found, err)
	}
	if s.Profile != "minipc" {
		t.Errorf("perfil: quiero minipc, tengo %q", s.Profile)
	}
}

func TestLoadCorrupto(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.yaml")
	// Campo desconocido: debe fallar (decodificación estricta).
	if err := os.WriteFile(path, []byte("profile: x\nbasura: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("esperaba error por estado corrupto")
	}
}
