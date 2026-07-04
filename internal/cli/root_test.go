package cli

import (
	"sort"
	"testing"
)

// TestRootHasV1Commands verifica que la raíz registra exactamente los seis
// comandos de la v1 (D18). Es un smoke test del contrato de la CLI: si alguien
// añade o quita un comando sin querer, salta aquí.
func TestRootHasV1Commands(t *testing.T) {
	want := []string{"add", "externals", "init", "link", "sync", "update"}

	got := make([]string, 0, len(rootCmd.Commands()))
	for _, c := range rootCmd.Commands() {
		// cobra añade "help" y "completion" por su cuenta; los ignoramos.
		if c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		got = append(got, c.Name())
	}
	sort.Strings(got)

	if len(got) != len(want) {
		t.Fatalf("número de comandos: quiero %d %v, tengo %d %v", len(want), want, len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("comando en posición %d: quiero %q, tengo %q", i, want[i], got[i])
		}
	}
}

// TestGlobalFlags verifica que los flags globales (--dry-run, --yes, --verbose)
// están declarados como persistentes en la raíz.
func TestGlobalFlags(t *testing.T) {
	for _, name := range []string{"dry-run", "yes", "verbose"} {
		if rootCmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("falta el flag global persistente --%s", name)
		}
	}
}
