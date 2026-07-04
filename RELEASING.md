# Publicar una release

Las releases se publican solas con **goreleaser** al empujar un **tag** semver:
el workflow `.github/workflows/release.yml` cross-compila (linux amd64 + arm64),
arma los `.tar.gz` (cada uno con el binario `dots`), genera checksums y crea la
**GitHub Release** con los binarios. El shim (`script/install.sh`) baja de ahí.

## Convención del tag

El tag **tiene que ser semver válido** (goreleaser lo exige): `vX.Y.Z`, con
sufijo de pre-release opcional (`-alpha`, `-beta`, `-rc.1`). Ejemplos válidos:
`v0.1.0-alpha`, `v0.1.0`, `v1.0.0`.

- **Primera release (piloto): `v0.1.0-alpha`.** Es "v0, alpha". Si se quiere, el
  **título** de la Release en GitHub puede ser algo más informal ("v0-alpha"),
  pero el **tag** debe ser el semver `v0.1.0-alpha`.

## Cómo publicar la release

Desde `main`, con todo verde:

```sh
git tag v0.1.0-alpha
git push origin v0.1.0-alpha
```

El push del tag dispara `release.yml`. Al terminar, la Release y sus binarios
quedan publicados y el shim ya puede instalarlos:

```sh
curl -fsSL https://raw.githubusercontent.com/kreikol/dotsmithy/main/script/install.sh | bash
```

(Para instalar una versión concreta en vez de la última: `DOTS_VERSION=v0.1.0-alpha`.)

## Antes de publicar

- `main` en verde (build + test + integración Fedora + shellcheck + shim).
- Sin dependencias ni versiones a medio en el código.
- Comprobar el empaquetado en local sin publicar: `goreleaser release --snapshot --clean`.
