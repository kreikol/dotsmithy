// Paquete stow: despliegue de dotfiles a $HOME por symlinks (modelo Stow).
//
// Responsabilidad: dado el contenido, las capas ya resueltas y el destino,
// calcular el PLAN de symlinks (qué crear, qué actualizar, qué ya está bien y
// qué choca) y aplicarlo de forma idempotente, con soporte de dry-run.
//
// Decisiones de diseño:
//   - Se enlazan FICHEROS individuales (no se "pliegan" directorios como en GNU
//     Stow). Los directorios intermedios se crean de verdad en el destino. Así
//     varias capas pueden aportar ficheros al mismo directorio (ej. ~/.config)
//     sin pisarse, que es justo lo que necesita el overlay por perfil.
//   - Las capas se aplican en orden: si un mismo destino aparece en dos capas,
//     gana la última (aunque la regla del contenido es que un fichero viva en
//     una sola capa).
//   - Nunca se borra un fichero o directorio real del destino: si hay uno en
//     medio, es un conflicto y se aborta sin tocar nada.
package stow
