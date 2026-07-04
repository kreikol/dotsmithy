#!/usr/bin/env bash
#
# Orquesta el test de integración en contenedor:
#   1) compila el motor para linux/amd64, estático (como en una release);
#   2) construye la imagen Fedora con el binario dentro;
#   3) corre el smoke test dentro del contenedor.
#
# Requiere podman. Pensado para correr en local y en CI.
set -euo pipefail

# Raíz del repo (dos niveles por encima de este script).
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
readonly IMAGE="dotsmithy-integration:latest"

echo ">> compilando el motor (linux/amd64, estático)…"
(
	cd "$ROOT"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -trimpath -ldflags "-s -w" -o "$HERE/dots" ./cmd/dots
)

echo ">> construyendo la imagen de integración…"
podman build -t "$IMAGE" "$HERE"

echo ">> corriendo el smoke test dentro del contenedor…"
podman run --rm "$IMAGE"

echo ">> corriendo la integración de paquetes (necesita red para dnf)…"
podman run --rm "$IMAGE" /usr/local/bin/packages.sh

echo ">> limpiando el binario temporal…"
rm -f "$HERE/dots"

echo ">> integración OK."
