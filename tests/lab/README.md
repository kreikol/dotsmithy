# Fixture del lab

Contenido de muestra (fake) que usa el workflow manual **Lab**
(`.github/workflows/lab.yml`) para levantar una Fedora con dotsmithy aplicado y
poder trastear a mano por SSH.

- `content/` es un repo de contenido de mentira, mínimo pero representativo:
  un dotfile (bajo `.config/` para no chocar con el `.bashrc` de la imagen base),
  una lista de paquetes dnf y un hook `post-link`.
- El binario del motor se compila a `tests/lab/dots` (ignorado por git) y se
  monta junto a este `content/` en el contenedor.

Cuando exista contenido real (`dotsmithy-content`), el lab puede pasar a
clonarlo (deploy key) en vez de usar este fixture.
