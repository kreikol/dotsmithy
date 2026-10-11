# El día a día: actualizar y sincronizar

Has cambiado algo en una máquina y lo quieres en las demás. O al revés: llegas a
una máquina que lleva días sin tocar y quieres ponerla al día.

## En la máquina donde cambias algo

Tu contenido es un repo git normal. Cuando cambias un dotfile (o `dots add`
apunta un paquete), el cambio queda en el repo; súbelo como siempre:

```sh
cd ~/.local/share/dots/content
git status
git add .
git commit -m "Cambia los alias de git"
git push
```

> [!TIP]
> Como tus dotfiles son symlinks al repo, editar `~/.bashrc.d/aliases.sh` ya es
> editar el fichero del repo. Solo falta el commit.

## En las demás máquinas

```sh
dots update
```

Trae los cambios del repo y vuelve a aplicarlo todo, en este orden:

1. Enlaza los dotfiles nuevos y coloca los ficheros de sistema.
2. Ejecuta los hooks `post-link`.
3. Instala los paquetes que falten (y habilita sus repos COPR).
4. Ejecuta los hooks `post-packages`.
5. Trae los externals.

Todo es repetible: si algo ya está bien, no lo toca. Si quieres ver qué haría sin
hacer nada, añade `--dry-run`.

> [!WARNING]
> `dots update` hace un `git pull` que solo avanza (`--ff-only`). Si tienes
> commits locales sin subir y en el remoto hay otros, se para. Resuélvelo con git
> (un `git pull --rebase` en el contenido suele bastar) y repite.

## Ver cómo está una máquina

¿Con qué perfil la montaste? ¿Tienes cambios en el contenido sin subir? Te lo
cuenta `dots status`:

```console
$ dots status
dotsmithy 1.0.0

  perfil      laptop
  máquina     laptop
  dimensiones ux=desktop
  capas       shared → ux/desktop → machines/laptop
  contenido   /home/tu/.local/share/dots/content
  remoto      git@github.com:tu-usuario/mis-dots.git
  git         rama main · 1 fichero(s) con cambios sin commitear · todo subido a origin/main
```

> [!TIP]
> Si quieres usar esos datos en tus dotfiles (por ejemplo, para enseñar el perfil
> en el prompt), `dots status --format env` los da como variables listas para
> `source`, y `--format json` como JSON.

Para saber si falta algún paquete, sin instalar nada:

```sh
dots sync
```

Y para ver qué symlinks cambiarían:

```sh
dots link --dry-run
```

## Ver también

- [Añadir un dotfile](add-a-dotfile.md)
- [Instalar paquetes](manage-packages.md)
- [Comandos](../reference/commands/README.md)
