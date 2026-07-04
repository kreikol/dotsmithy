# dotsmithy

La **herrería de tus dotfiles**. `dotsmithy` es un gestor de dotfiles pensado como
producto: un **motor** genérico (este repo, en Go) y tu **contenido** personal
(dotfiles reales, paquetes, perfiles de máquina) en un repo aparte. El único
contrato entre ambos es un manifiesto declarativo, `dots.yaml`.

> Estado: **F0 (andamiaje)**. El esqueleto compila y la CLI lista sus comandos,
> pero todavía no hacen nada. La lógica llega en las siguientes fases.

## Idea en dos líneas

- **Motor** (este repo): bootstrap, despliegue de symlinks (modelo Stow), paquetes,
  perfiles de máquina, hooks y externals. Genérico, versionado y testeado. No guarda
  nada personal.
- **Contenido** (repo privado tuyo): los ficheros de verdad. Se engancha al motor por
  `dots.yaml`.

Es el patrón librería/SDK + aplicación: el motor es la pieza reutilizable; el
contenido, tu instancia.

## Instalación (bootstrap)

En una máquina limpia, una sola línea baja el motor y cede el control a `dots init`:

```sh
curl -fsSL https://raw.githubusercontent.com/kreikol/dotsmithy/main/script/install.sh | bash
```

El instalador deja el mando `dots` (y su atajo `ds`) a mano.

## Comandos

| Comando          | Qué hace                                                        |
| ---------------- | -------------------------------------------------------------- |
| `dots init`      | Arranca una máquina desde cero (clave SSH, perfil, clon, aplica) |
| `dots link`      | Despliega los dotfiles a `$HOME` (symlinks, modelo Stow)        |
| `dots update`    | Trae los cambios del contenido y los aplica                     |
| `dots add`       | Adopta un fichero de `$HOME` al repo de contenido               |
| `dots sync`      | Concilia los paquetes con lo declarado (o informa del drift)    |
| `dots externals` | Trae y prepara los recursos externos declarados                 |

Flags globales: `--dry-run` (`-n`), `--yes` (`-y`), `--verbose` (`-v`).

## Desarrollo

El toolchain vive en un [devbox](https://www.jetify.com/devbox) local (nada global):

```sh
devbox shell           # entra al entorno (go, shellcheck, goreleaser)
devbox run build       # compila el binario en dist/dots
devbox run test        # go test ./...
devbox run lint-shim   # shellcheck del shim de bootstrap
```

## Licencia

[MIT](LICENSE).
