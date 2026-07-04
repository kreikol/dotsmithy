package system

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"go.kreikol.dev/dotsmithy/internal/manifest"
)

// Op es qué hay que hacer con una entrada de system.
type Op int

const (
	OpSkip    Op = iota // ya está como debe
	OpSymlink           // crear/repuntar el symlink
	OpCopy              // copiar (con validación anti-lockout si procede)
)

// Action es la decisión para una entrada, ya con rutas y dueño resueltos.
type Action struct {
	Src      string // ruta absoluta del origen (en el contenido)
	Dest     string // ruta absoluta del destino (en el sistema)
	Op       Op
	Mode     string // perms para copy (ej. "0440"), opcional
	User     string // dueño resuelto (default root)
	Group    string // grupo resuelto (default root)
	Validate string // comando de validación, opcional ({file} = fichero a validar)
}

// Plan calcula qué hacer con cada entrada de system, SIN modificar nada (solo
// lee para decidir idempotencia). Falla si un origen no existe.
func Plan(entries []manifest.SystemFile, contentDir string) ([]Action, error) {
	var actions []Action
	for i, e := range entries {
		absSrc, err := filepath.Abs(filepath.Join(contentDir, e.Src))
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(absSrc); err != nil {
			return nil, fmt.Errorf("system[%d]: no encuentro el origen %q: %w", i, absSrc, err)
		}
		user, group := splitOwner(e.Owner)
		a := Action{Src: absSrc, Dest: e.Dest, Mode: e.Mode, User: user, Group: group, Validate: e.Validate}

		switch e.Type {
		case "symlink":
			if cur, err := os.Readlink(e.Dest); err == nil && cur == absSrc {
				a.Op = OpSkip
			} else {
				a.Op = OpSymlink
			}
		case "copy":
			if sameCopy(absSrc, e.Dest, e.Mode) {
				a.Op = OpSkip
			} else {
				a.Op = OpCopy
			}
		default:
			return nil, fmt.Errorf("system[%d]: type %q no soportado", i, e.Type)
		}
		actions = append(actions, a)
	}
	return actions, nil
}

// Apply ejecuta el plan. Valida ANTES de tocar nada (anti-lockout) y hace las
// escrituras con privilegios. En dry-run solo informa. log puede ser nil.
func Apply(entries []manifest.SystemFile, contentDir string, dryRun bool, log func(string)) error {
	if log == nil {
		log = func(string) {}
	}
	actions, err := Plan(entries, contentDir)
	if err != nil {
		return err
	}

	for _, a := range actions {
		if a.Op == OpSkip {
			log(fmt.Sprintf("= %s (ya ok)", a.Dest))
			continue
		}
		// Validación anti-lockout: sobre una copia temporal, antes de escribir.
		if a.Validate != "" {
			if err := runValidate(a.Src, a.Validate); err != nil {
				return err
			}
		}
		if dryRun {
			verb := "enlazaría"
			if a.Op == OpCopy {
				verb = "copiaría"
			}
			log(fmt.Sprintf("+ %s %s -> %s", verb, a.Src, a.Dest))
			continue
		}
		if err := writeAction(a); err != nil {
			return err
		}
		log(fmt.Sprintf("+ %s -> %s", a.Src, a.Dest))
	}
	return nil
}

// writeAction hace la escritura privilegiada de una acción.
func writeAction(a Action) error {
	switch a.Op {
	case OpSymlink:
		return runPrivileged("ln", "-sfn", a.Src, a.Dest)
	case OpCopy:
		// Copiamos a un temporal EN EL MISMO directorio del destino (para que el
		// move sea atómico dentro del mismo sistema de ficheros) y luego movemos.
		tmp := a.Dest + ".dots-tmp"
		args := []string{"install"}
		if a.Mode != "" {
			args = append(args, "-m", a.Mode)
		}
		args = append(args, "-o", a.User, "-g", a.Group, a.Src, tmp)
		if err := runPrivileged(args[0], args[1:]...); err != nil {
			return err
		}
		return runPrivileged("mv", "-f", tmp, a.Dest)
	}
	return nil
}

// runValidate copia el origen a un temporal y corre ahí el comando de validación
// ({file} se sustituye por la ruta temporal). Si falla, devuelve error nombrando
// el fichero: así nunca se instala un /etc roto.
func runValidate(src, tmpl string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("no he podido leer %q para validar: %w", src, err)
	}
	tmp, err := os.CreateTemp("", "dots-validate-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	cmdline := strings.ReplaceAll(tmpl, "{file}", tmp.Name())
	cmd := exec.Command("bash", "-c", cmdline)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("la validación de %q falló (%s): %w; no lo instalo", src, cmdline, err)
	}
	return nil
}

// sameCopy indica si el destino ya es una copia idéntica del origen (mismo
// contenido y, si se pide mode, mismos permisos). Si no se puede leer el destino
// (no existe o hace falta root), devuelve false: mejor aplicar.
func sameCopy(src, dest, mode string) bool {
	srcData, err := os.ReadFile(src)
	if err != nil {
		return false
	}
	destData, err := os.ReadFile(dest)
	if err != nil {
		return false
	}
	if !bytes.Equal(srcData, destData) {
		return false
	}
	if mode != "" {
		fi, err := os.Stat(dest)
		if err != nil {
			return false
		}
		want, err := strconv.ParseUint(mode, 8, 32)
		if err != nil {
			return false
		}
		if fi.Mode().Perm() != os.FileMode(want).Perm() {
			return false
		}
	}
	return true
}

// splitOwner parte "user:group" en sus dos partes. Vacío = root:root. Si solo se
// da el usuario, el grupo toma ese mismo nombre.
func splitOwner(owner string) (user, group string) {
	if strings.TrimSpace(owner) == "" {
		return "root", "root"
	}
	parts := strings.SplitN(owner, ":", 2)
	if len(parts) == 2 && parts[1] != "" {
		return parts[0], parts[1]
	}
	return parts[0], parts[0]
}

// runPrivileged ejecuta un comando que necesita root. Si ya somos root lo lanza
// tal cual; si no, lo antepone con sudo.
func runPrivileged(name string, args ...string) error {
	if os.Geteuid() != 0 {
		args = append([]string{name}, args...)
		name = "sudo"
	}
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("falló «%s %s»: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
