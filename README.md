# dotsmithy 🔨

<p align="center">
  <img src="docs/assets/banner.svg" alt="dotsmithy: la herrería de tus dotfiles"/>
</p>

<p align="center">
  <strong>Tus dotfiles, tus paquetes y los ajustes de cada máquina, guardados en un repo tuyo y aplicados con un solo comando.</strong>
</p>

<p align="center">
  <a href="https://github.com/kreikol/dotsmithy/releases">
    <img alt="Release" src="https://img.shields.io/github/v/release/kreikol/dotsmithy?include_prereleases&style=for-the-badge&color=ff8c42&label=release"/>
  </a>
  <a href="https://github.com/kreikol/dotsmithy/actions/workflows/ci.yml">
    <img alt="CI" src="https://img.shields.io/github/actions/workflow/status/kreikol/dotsmithy/ci.yml?branch=main&style=for-the-badge&label=CI"/>
  </a>
</p>

<p align="center">
  <a href="https://fedoraproject.org">
    <img alt="SO: Fedora" src="https://img.shields.io/badge/SO-Fedora-51A2DA?style=flat-square&logo=fedora&logoColor=white"/>
  </a>
  <a href="https://go.dev">
    <img alt="Go" src="https://img.shields.io/github/go-mod/go-version/kreikol/dotsmithy?style=flat-square&logo=go&logoColor=white&color=00ADD8"/>
  </a>
  <a href="LICENSE">
    <img alt="Licencia MIT" src="https://img.shields.io/badge/licencia-MIT-blue?style=flat-square"/>
  </a>
</p>

---

dotsmithy tiene dos piezas: un **motor** (este repo, el comando `dots`) que sabe
hacer el trabajo, y tu **contenido** (un repo privado con tus dotfiles de verdad)
que dice qué hay que hacer. Se entienden a través de un fichero, `dots.yaml`. Tú
editas tus ficheros como siempre y `dots` se encarga de que cada máquina tenga lo
suyo.

> [!NOTE]
> dotsmithy está pensado para **Fedora**. Funciona y está cubierto por tests, pero
> todavía no ha llegado a la v1.0, así que puede cambiar algún detalle.

## ✨ Qué hace

- **Tus ficheros de verdad, sin plantillas**: cada dotfile vive en tu repo y se
  enlaza en tu `$HOME` con un symlink. Editarlo es editar el repo.
- **Cada máquina recibe lo suyo**: los perfiles apilan capas (lo común, lo de su
  tipo y lo de la propia máquina) sin copiar ni un fichero.
- **Paquetes declarados**: listas por capa, repos COPR incluidos. `dots add`
  apunta e instala en un paso.
- **Hooks en el momento justo**: tus scripts, después de enlazar, de instalar
  paquetes o al estrenar la máquina.
- **Recursos externos**: plugins y repos de terceros que se descargan y se
  mantienen al día solos.
- **Ficheros de sistema sin sustos**: lo de `/etc` se puede validar antes de
  reemplazarlo, para no dejar nunca un fichero roto.
- **Repetible y sin miedo**: puedes lanzarlo mil veces; lo que ya está bien no se
  toca, y `--dry-run` te enseña el plan antes.
- **Una máquina limpia en una línea**: el instalador trae `dots`, y `dots init`
  clona tu contenido (y te ayuda con la clave SSH si hace falta).

## 🚀 Pruébalo en un minuto

En una Fedora limpia, esta línea instala `dots` (y su atajo `ds`):

```sh
curl -fsSL https://raw.githubusercontent.com/kreikol/dotsmithy/main/script/install.sh | bash
```

Luego le dices de dónde sale tu contenido y qué perfil quieres para esta máquina:

```sh
dots init --remote git@github.com:<tu-usuario>/<tu-contenido>.git --profile <perfil>
```

¿Aún no tienes repo de contenido? Créalo desde la
[plantilla](https://github.com/kreikol/dotsmithy-content-template): el
[tutorial](docs/tutorials/first-machine.md) te lleva de la mano.

## 🧭 ¿Qué quieres hacer?

| | |
| --- | --- |
| 🧭 **Aprender desde cero** | Monta tu primera máquina paso a paso: [tutoriales](docs/tutorials/README.md) |
| 🛠️ **Resolver algo concreto** | Añadir un dotfile, un paquete, un hook...: [guías](docs/how-to/README.md) |
| 📖 **Consultar un detalle** | Comandos, `dots.yaml`, cómo se organiza el contenido: [referencia](docs/reference/README.md) |
| 💡 **Entender cómo funciona** | Por qué está hecho así: [explicaciones](docs/explanation/README.md) |

Todo junto, en el [índice de la documentación](docs/README.md).

## 🤝 ¿Quieres meterle mano al código?

Las ideas y los fallos, en las [issues](https://github.com/kreikol/dotsmithy/issues).
Para trabajar en el motor, mira [CONTRIBUTING.md](CONTRIBUTING.md).

## 📜 Licencia

[MIT](LICENSE).
