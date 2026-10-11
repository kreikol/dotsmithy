<!-- Generado por tools/docgen: no lo edites a mano. -->

# dots status

Enseña el perfil, las capas y el contenido de esta máquina (solo lectura).

status te cuenta cómo está montada esta máquina: qué perfil tiene, qué
máquina es, qué valor toma en cada dimensión, qué capas forman su pila y dónde
está el contenido. Si el contenido es un repo git, también la rama, si tienes
cambios sin commitear y si hay commits sin subir (según lo que sabe tu repo
local: no hace fetch). No toca nada.

Con --format eliges la salida: text (por defecto) para leerla tú; env, con
líneas VAR=valor listas para hacer «source» en bash (DOTS_PROFILE, DOTS_MACHINE,
DOTS_PARAM_&lt;DIMENSIÓN&gt;, DOTS_LAYERS, DOTS_CONTENT_DIR y DOTS_VERSION); o json,
un objeto JSON para scripts y otras herramientas.

En una máquina sin «dots init» (y sin --profile), avisa por la salida de error
y termina con código 1, sin escribir nada en la salida normal. Así un script
puede comprobarlo sin tener que leer el texto.

Perfil y contenido se resuelven como en link: flags o el estado local.

## Uso

```
dots status [opciones]
```

## Opciones

| Opción | Qué hace |
| --- | --- |
| `--content <texto>` | ruta al repo de contenido (por defecto: la del estado local, o el directorio actual) |
| `--format <texto>` | formato de salida: text, env o json (por defecto: `text`) |
| `--profile <texto>` | perfil del que enseñar el estado (por defecto: el del estado local) |

## Opciones globales

| Opción | Qué hace |
| --- | --- |
| `-n`, `--dry-run` | enseña qué haría sin tocar nada |
| `-h`, `--help` | enseña esta ayuda |
| `-v`, `--verbose` | saca más detalle de lo que va pasando |
| `-y`, `--yes` | no pregunta nada (modo no interactivo) |

## Ver también

- [`dots`](dots.md): dotsmithy: la herrería de tus dotfiles
- [Todos los comandos](README.md)
