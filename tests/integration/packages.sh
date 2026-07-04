#!/usr/bin/env bash
#
# Integración de paquetes: se ejecuta DENTRO del contenedor Fedora (como root).
# Comprueba el "hecho cuando" de F1 en la parte de paquetes: que init deja los
# paquetes declarados instalados (idempotente) y que add registra + instala.
set -euo pipefail

fail() {
	printf 'FALLO: %s\n' "$*" >&2
	exit 1
}

C=/tmp/content
mkdir -p "$C/shared/packages" "$C/hooks/post-link" "$C/hooks/post-init" "$C/system"
cat >"$C/dots.yaml" <<'EOF'
version: 1
profiles: { minipc: { description: "integración" } }
stow: { layers: [shared, "machines/{profile}"] }
packages: { managers: [dnf] }
hooks: { points: [post-link, post-packages, post-init] }
system:
  - { src: system/dotsmithy.conf, dest: /etc/dotsmithy.conf, type: copy, mode: "0644", validate: "grep -q hola {file}" }
  - { src: system/dots.link, dest: /etc/dotsmithy.link, type: symlink }
EOF
# Un paquete pequeño y con pocas dependencias.
echo tree >"$C/shared/packages/dnf.txt"
# Ficheros de sistema de prueba (copy con validación + symlink).
echo "hola=mundo" >"$C/system/dotsmithy.conf"
echo "soy un target de symlink" >"$C/system/dots.link"
# Hooks que dejan un rastro para comprobar que se dispararon (el heredoc con
# 'EOF' entrecomillado escribe las variables literales, sin expandirlas ahora).
cat >"$C/hooks/post-link/10-marker.sh" <<'EOF'
echo "$DOTS_PROFILE" > "$DOTS_TARGET/.dots-postlink"
EOF
cat >"$C/hooks/post-init/10-once.sh" <<'EOF'
touch "$DOTS_TARGET/.dots-postinit"
EOF

echo ">> init --dry-run no toca nada (ni instala, ni estado, ni hooks)"
dots init --profile minipc --content "$C" --dry-run
if rpm -q tree >/dev/null 2>&1; then fail "dry-run NO debería instalar tree"; fi
if [ -e "$HOME/.config/dots/state.yaml" ]; then fail "dry-run NO debería escribir el estado"; fi
if [ -e "$HOME/.dots-postlink" ]; then fail "dry-run NO debería ejecutar hooks"; fi
if [ -e /etc/dotsmithy.conf ]; then fail "dry-run NO debería tocar /etc"; fi

echo ">> init instala los paquetes declarados y dispara los hooks"
dots init --profile minipc --content "$C"
rpm -q tree >/dev/null 2>&1 || fail "tree no quedó instalado tras init"
[ -f "$HOME/.dots-postlink" ] || fail "el hook post-link no se ejecutó"
[ -f "$HOME/.dots-postinit" ] || fail "el hook post-init no se ejecutó"

echo ">> system: copia validada a /etc y symlink"
[ -f /etc/dotsmithy.conf ] || fail "system copy no se aplicó"
grep -q hola /etc/dotsmithy.conf || fail "system copy: contenido incorrecto"
[ "$(stat -c %a /etc/dotsmithy.conf)" = "644" ] || fail "system copy: mode incorrecto"
[ -L /etc/dotsmithy.link ] || fail "system symlink no se creó"

echo ">> init es idempotente (segunda pasada: al día)"
out="$(dots init --profile minipc --content "$C")"
grep -q "al día" <<<"$out" || fail "la segunda pasada de init debería decir «al día»"

echo ">> add registra e instala un paquete nuevo"
dots add dnf jq --content "$C"
rpm -q jq >/dev/null 2>&1 || fail "jq no quedó instalado tras add"
grep -qx jq "$C/shared/packages/dnf.txt" || fail "jq no quedó registrado en la lista"

echo "OK: init deja lo declarado instalado (idempotente) y add registra+instala."
