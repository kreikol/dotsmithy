#!/usr/bin/env bash
#
# Shim de bootstrap de dotsmithy.
#
# Es lo ÚNICO que va en bash, y su único trabajo es: en una máquina limpia,
# bajar el binario del motor desde las Releases de GitHub, dejarlo instalado
# (con los mandos «dots» y su atajo «ds») y cederle el control con «dots init».
#
# Uso típico (una línea, en una Fedora recién puesta):
#   curl -fsSL https://raw.githubusercontent.com/kreikol/dotsmithy/main/script/install.sh | bash
#
# Variables que puedes tunear:
#   DOTS_VERSION  versión a instalar (por defecto: la última release)
#   DOTS_BIN      dónde instalar los binarios (por defecto: ~/.local/bin)
#
# Nota F0: es un esqueleto funcional. La resolución de "última release" y el
# formato exacto de los assets se afinan cuando exista la primera release.

set -euo pipefail

readonly REPO="kreikol/dotsmithy"
readonly BIN_NAME="dots"
readonly ALIAS_NAME="ds"

# Directorio de instalación de los binarios.
DOTS_BIN="${DOTS_BIN:-$HOME/.local/bin}"
# Versión a instalar (vacío = última).
DOTS_VERSION="${DOTS_VERSION:-}"

# say imprime un mensaje al usuario, en tono cercano.
say() {
	printf '  %s\n' "$*"
}

# die aborta con un mensaje de error por stderr.
die() {
	printf 'ups: %s\n' "$*" >&2
	exit 1
}

# need_cmd comprueba que una herramienta imprescindible está disponible.
need_cmd() {
	command -v "$1" >/dev/null 2>&1 || die "me falta «$1» para poder seguir."
}

# detect_arch mapea la arquitectura de la máquina al nombre que usan los assets.
detect_arch() {
	local arch
	arch="$(uname -m)"
	case "$arch" in
	x86_64 | amd64) printf 'amd64' ;;
	aarch64 | arm64) printf 'arm64' ;;
	*) die "arquitectura no soportada todavía: $arch" ;;
	esac
}

# detect_os mapea el sistema operativo al nombre que usan los assets.
detect_os() {
	local os
	os="$(uname -s)"
	case "$os" in
	Linux) printf 'linux' ;;
	*) die "sistema no soportado todavía: $os" ;;
	esac
}

main() {
	need_cmd uname
	need_cmd install
	# Necesitamos uno de los dos para descargar.
	if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
		die "necesito «curl» o «wget» para bajar el motor."
	fi

	local os arch
	os="$(detect_os)"
	arch="$(detect_arch)"

	say "instalando dotsmithy para ${os}/${arch}…"
	say "(F0) aquí bajaría el binario del motor desde las Releases de $REPO"
	say "     versión: ${DOTS_VERSION:-última}"
	say "     destino: ${DOTS_BIN}/${BIN_NAME} (+ atajo «${ALIAS_NAME}»)"

	# TODO(F1): descargar el asset, extraer «dots», instalarlo en DOTS_BIN,
	# crear el symlink del atajo y ceder el control con «dots init "$@"».
	#
	#   mkdir -p "$DOTS_BIN"
	#   install -m 0755 "$tmp/$BIN_NAME" "$DOTS_BIN/$BIN_NAME"
	#   ln -sf "$BIN_NAME" "$DOTS_BIN/$ALIAS_NAME"
	#   exec "$DOTS_BIN/$BIN_NAME" init "$@"

	say "listo (esqueleto): cuando haya release, esto dejará «dots» y «ds» a mano."
}

main "$@"
