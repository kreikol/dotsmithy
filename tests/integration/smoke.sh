#!/usr/bin/env bash
#
# Smoke test de integración: se ejecuta DENTRO del contenedor Fedora limpio.
# Comprueba lo mínimo de F0: que el binario del motor arranca en una Fedora
# pelada (sin runtime de Go) y que la CLI responde por sus dos nombres.
set -euo pipefail

fail() {
	printf 'FALLO: %s\n' "$*" >&2
	exit 1
}

# 1) El binario existe y es ejecutable por su nombre canónico.
command -v dots >/dev/null 2>&1 || fail "«dots» no está en el PATH"
# 2) Y por su atajo.
command -v ds >/dev/null 2>&1 || fail "«ds» (atajo) no está en el PATH"

# 3) --help arranca y lista los comandos de la v1.
help_out="$(dots --help)"
for c in init link update add sync externals; do
	grep -q -- "$c" <<<"$help_out" || fail "«dots --help» no menciona el comando «$c»"
done

# 4) El atajo «ds» se comporta igual que «dots».
ds --help >/dev/null 2>&1 || fail "«ds --help» no arranca"

echo "OK: el motor arranca en Fedora limpia y la CLI responde (dots + ds)."
