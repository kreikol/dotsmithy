#!/usr/bin/env bash
#
# Test del shim (script/install.sh) con la descarga MOCKEADA vía file://, sin red
# ni release real. Comprueba que resuelve el asset del os/arch, descarga, extrae
# e instala dots + ds, y que cede a «dots init» cuando se le pasan argumentos.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"

fail() {
	printf 'FALLO: %s\n' "$*" >&2
	exit 1
}

os=linux
case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) fail "arquitectura no contemplada en el test: $(uname -m)" ;;
esac

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# Binario «dots» de mentira: un script que imprime sus argumentos, para poder
# comprobar que es el que instaló el shim y que init recibe lo que toca.
mkdir -p "$work/pkg"
cat >"$work/pkg/dots" <<'EOF'
#!/usr/bin/env bash
echo "FAKE-DOTS $*"
EOF
chmod +x "$work/pkg/dots"
tar -C "$work/pkg" -czf "$work/dotsmithy_0.1.0-alpha_${os}_${arch}.tar.gz" dots

# Metadatos de release simulados: incluyen un asset decoy de otra arch para
# comprobar que el shim filtra por el os/arch correcto.
cat >"$work/release.json" <<EOF
{
  "tag_name": "v0.1.0-alpha",
  "assets": [
    { "browser_download_url": "file://$work/dotsmithy_0.1.0-alpha_linux_ppc64.tar.gz" },
    { "browser_download_url": "file://$work/dotsmithy_0.1.0-alpha_${os}_${arch}.tar.gz" }
  ]
}
EOF

# 1) Sin argumentos: instala y NO ejecuta init.
bin="$work/bin"
DOTS_RELEASE_URL="file://$work/release.json" DOTS_BIN="$bin" bash "$ROOT/script/install.sh" >/dev/null
[ -x "$bin/dots" ] || fail "no se instaló dots"
[ -L "$bin/ds" ] || fail "no se creó el atajo ds (symlink)"
[ "$("$bin/dots" hola)" = "FAKE-DOTS hola" ] || fail "el dots instalado no funciona"

# 2) Con argumentos: cede el control a «dots init <args>».
bin2="$work/bin2"
out="$(DOTS_RELEASE_URL="file://$work/release.json" DOTS_BIN="$bin2" \
	bash "$ROOT/script/install.sh" --profile minipc --remote git@x:y.git 2>/dev/null)"
grep -q "FAKE-DOTS init --profile minipc --remote git@x:y.git" <<<"$out" ||
	fail "el shim no cede a «dots init» con los argumentos"

echo "OK: el shim descarga (mock), extrae, instala dots+ds y cede a init."
