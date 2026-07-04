// Paquete stow: despliegue de dotfiles a $HOME por symlinks (modelo Stow).
//
// Responsabilidad (F1): resolver el overlay de capas (base + perfil) y crear /
// reconciliar los symlinks en $HOME de forma idempotente, con soporte de
// dry-run. En F0 es solo el hueco.
package stow
