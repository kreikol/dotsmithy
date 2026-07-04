#!/usr/bin/env bash
#
# Shim de bootstrap de dotsmithy.
#
# Es lo ÚNICO que va en bash, y su único trabajo es: en una máquina limpia,
# bajar el binario del motor desde las Releases de GitHub, dejarlo instalado
# (con los mandos «dots» y su atajo «ds») y, si le pasas argumentos, ceder el
# control a «dots init».
#
# Uso típico (una línea, en una Fedora recién puesta):
#   curl -fsSL https://raw.githubusercontent.com/kreikol/dotsmithy/main/script/install.sh | bash
# Y luego:
#   dots init --profile <perfil> --remote <url-ssh-del-contenido>
#
# Variables que puedes tunear:
#   DOTS_VERSION      versión a instalar (ej. v0.1.0-alpha; por defecto: la última)
#   DOTS_BIN          dónde instalar los binarios (por defecto: ~/.local/bin)
#   DOTS_RELEASE_URL  URL de metadatos de la release (uso interno/tests; por
#                     defecto se resuelve contra la API de GitHub)

set -euo pipefail

readonly REPO="kreikol/dotsmithy"
readonly BIN_NAME="dots"
readonly ALIAS_NAME="ds"

# Directorio de instalación de los binarios.
DOTS_BIN="${DOTS_BIN:-$HOME/.local/bin}"
# Versión a instalar (vacío = última).
DOTS_VERSION="${DOTS_VERSION:-}"
# URL de metadatos de la release (override para tests). Vacío = API de GitHub.
DOTS_RELEASE_URL="${DOTS_RELEASE_URL:-}"

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

# fetch vuelca una URL a stdout (con curl o wget, lo que haya). Soporta file://
# para poder testear sin red.
fetch() {
	local url="$1"
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$url"
	else
		wget -qO- "$url"
	fi
}

# fetch_to guarda una URL en un fichero.
fetch_to() {
	local url="$1" dest="$2"
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL -o "$dest" "$url"
	else
		wget -qO "$dest" "$url"
	fi
}

# release_metadata_url devuelve la URL de la que sacar los metadatos de la
# release (la última, o la de DOTS_VERSION), o el override de DOTS_RELEASE_URL.
release_metadata_url() {
	if [ -n "$DOTS_RELEASE_URL" ]; then
		printf '%s' "$DOTS_RELEASE_URL"
	elif [ -n "$DOTS_VERSION" ]; then
		printf 'https://api.github.com/repos/%s/releases/tags/%s' "$REPO" "$DOTS_VERSION"
	else
		printf 'https://api.github.com/repos/%s/releases/latest' "$REPO"
	fi
}

# asset_url resuelve, de los metadatos de la release, la URL de descarga del
# asset que corresponde a este os/arch.
asset_url() {
	local os="$1" arch="$2" meta url
	meta="$(fetch "$(release_metadata_url)")" || die "no he podido consultar la release."
	# De los pares "browser_download_url": "…", quedarnos con el del os/arch.
	url="$(printf '%s\n' "$meta" |
		grep -oE '"browser_download_url"[ ]*:[ ]*"[^"]+"' |
		grep "_${os}_${arch}\.tar\.gz" |
		head -1 |
		cut -d'"' -f4)"
	[ -n "$url" ] || die "no encuentro un binario para ${os}/${arch} en la release."
	printf '%s' "$url"
}

main() {
	need_cmd uname
	need_cmd install
	need_cmd tar
	if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
		die "necesito «curl» o «wget» para bajar el motor."
	fi

	local os arch url
	os="$(detect_os)"
	arch="$(detect_arch)"

	say "instalando dotsmithy para ${os}/${arch} (${DOTS_VERSION:-última versión})…"
	url="$(asset_url "$os" "$arch")"

	# tmp es global a propósito: la trap EXIT se ejecuta en scope global, así que
	# no puede ser una variable local de main. El ${tmp:-} evita fallos si la trap
	# saltara antes de asignarlo (con set -u).
	tmp="$(mktemp -d)"
	trap 'rm -rf "${tmp:-}"' EXIT

	say "bajando $url"
	fetch_to "$url" "$tmp/dots.tar.gz" || die "falló la descarga del binario."
	tar -xzf "$tmp/dots.tar.gz" -C "$tmp" || die "no he podido descomprimir el binario."
	[ -f "$tmp/$BIN_NAME" ] || die "el paquete no trae el binario «$BIN_NAME»."

	mkdir -p "$DOTS_BIN"
	install -m 0755 "$tmp/$BIN_NAME" "$DOTS_BIN/$BIN_NAME"
	ln -sf "$BIN_NAME" "$DOTS_BIN/$ALIAS_NAME"
	say "instalado: $DOTS_BIN/$BIN_NAME (+ atajo «$ALIAS_NAME»)"

	# Aviso si el destino no está en el PATH.
	case ":$PATH:" in
	*":$DOTS_BIN:"*) : ;;
	*) say "ojo: $DOTS_BIN no está en tu PATH; añádelo para usar «$BIN_NAME»." ;;
	esac

	# Si te han pasado argumentos, cedemos el control a «dots init». Si no, te
	# decimos cómo seguir (no forzamos un init a ciegas).
	if [ "$#" -gt 0 ]; then
		exec "$DOTS_BIN/$BIN_NAME" init "$@"
	fi
	say "ahora arranca tu entorno con:"
	say "  $BIN_NAME init --profile <perfil> --remote <url-ssh-del-contenido>"
}

main "$@"
