#!/usr/bin/env bash
#
# Orquesta el test de integración en contenedor:
#   1) compila el motor para linux/amd64, estático (como en una release);
#   2) construye la imagen Fedora con el binario dentro;
#   3) corre los escenarios (smoke + init/link/paquetes/system/hooks/externals).
#
# Necesita un motor de contenedores en el HOST (podman o docker).
#
# ¿Por qué no va en el devbox del proyecto? podman/docker son herramientas de
# sistema con integración de host (rootless, subuid/subgid, cgroups, storage) que
# vía Nix/devbox dan guerra y rara vez quedan limpias. El toolchain reproducible
# (go, shellcheck, goreleaser) sí vive en el devbox; el motor de contenedores se
# toma del sistema. Puedes forzar cuál con DOTS_CONTAINER_ENGINE.
set -euo pipefail

# Motor de contenedores: el indicado en DOTS_CONTAINER_ENGINE, o podman, o docker.
ENGINE="${DOTS_CONTAINER_ENGINE:-}"
if [ -z "$ENGINE" ]; then
	if command -v podman >/dev/null 2>&1; then
		ENGINE=podman
	elif command -v docker >/dev/null 2>&1; then
		ENGINE=docker
	fi
fi
if [ -z "$ENGINE" ] || ! command -v "$ENGINE" >/dev/null 2>&1; then
	cat >&2 <<'MSG'
Necesito un motor de contenedores (podman o docker) en el sistema para la
integración local, y no lo encuentro.
  - Instala podman (recomendado en Fedora) o docker, o
  - define DOTS_CONTAINER_ENGINE con el que quieras usar.
No va en el devbox a propósito: es una herramienta de sistema (rootless, cgroups,
storage) que en Nix da guerra. El resto del toolchain sí está en el devbox.
MSG
	exit 1
fi

# Raíz del repo (dos niveles por encima de este script).
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
readonly IMAGE="dotsmithy-integration:latest"

echo ">> usando motor de contenedores: $ENGINE"

echo ">> compilando el motor (linux/amd64, estático)…"
(
	cd "$ROOT"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -trimpath -ldflags "-s -w" -o "$HERE/dots" ./cmd/dots
)

echo ">> construyendo la imagen de integración…"
"$ENGINE" build -t "$IMAGE" "$HERE"

echo ">> corriendo el smoke test dentro del contenedor…"
"$ENGINE" run --rm "$IMAGE"

echo ">> corriendo la integración de paquetes (necesita red para dnf)…"
"$ENGINE" run --rm "$IMAGE" /usr/local/bin/packages.sh

echo ">> limpiando el binario temporal…"
rm -f "$HERE/dots"

echo ">> integración OK."
