// Paquete system: ficheros de sistema fuera de $HOME (/etc, systemd, sudoers).
// Es la única parte del motor que necesita root (ADR 0012).
//
// Responsabilidad: colocar ficheros en el sistema por symlink o copia. Para las
// copias con validación, aplica el patrón anti-lockout: copia a un temporal,
// valida ahí (ej. `visudo -cf {file}`) y solo si pasa hace el reemplazo. Nunca
// deja un /etc roto.
//
// El plan (qué hacer con cada entrada) es puro y testeable; la escritura
// privilegiada se hace con sudo (o directa si ya somos root, ej. el contenedor
// de tests).
package system
