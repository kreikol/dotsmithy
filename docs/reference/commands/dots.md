<!-- Generado por tools/docgen: no lo edites a mano. -->

# dots

dotsmithy: la herrería de tus dotfiles.

dots es el motor de dotsmithy: despliega tus dotfiles (modelo Stow),
gestiona paquetes, perfiles de máquina, hooks y externals.

El motor es genérico y no guarda nada personal: tu contenido (dotfiles, listas
de paquetes, perfiles) vive en un repo aparte y se engancha por el manifiesto
dots.yaml.

## Uso

```
dots [opciones]
dots [comando]
```

## Comandos

| Comando | Qué hace |
| --- | --- |
| [`add`](add.md) | registra e instala uno o varios paquetes |
| [`completion`](completion.md) | genera el script de autocompletado para tu shell |
| [`externals`](externals.md) | trae y prepara los recursos externos declarados |
| [`init`](init.md) | arranca una máquina desde cero (perfil, contenido, estado, aplica) |
| [`link`](link.md) | despliega los dotfiles a $HOME (symlinks, modelo Stow) |
| [`status`](status.md) | enseña el perfil, las capas y el contenido de esta máquina (solo lectura) |
| [`sync`](sync.md) | informa de qué paquetes declarados faltan por instalar (solo lectura) |
| [`update`](update.md) | trae los cambios del contenido y los aplica |

## Opciones

| Opción | Qué hace |
| --- | --- |
| `-n`, `--dry-run` | enseña qué haría sin tocar nada |
| `-h`, `--help` | enseña esta ayuda |
| `-v`, `--verbose` | saca más detalle de lo que va pasando |
| `--version` | enseña la versión de dots |
| `-y`, `--yes` | no pregunta nada (modo no interactivo) |

## Ver también

- [Todos los comandos](README.md)
