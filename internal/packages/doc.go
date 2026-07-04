// Paquete packages: gestión de paquetes (dnf en v1; COPR y gestores de lenguaje
// más adelante).
//
// Responsabilidad: leer las listas de paquetes declaradas por capa y unirlas,
// consultar qué hay instalado en el sistema y reportar el drift (lo declarado
// que aún no está instalado). La lógica de listas y drift es pura y testeable;
// hablar con el gestor del sistema (dnf/rpm) queda detrás de la interfaz Manager.
package packages
