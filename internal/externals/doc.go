// Paquete externals: recursos externos declarados en el manifiesto (ADR 0009).
//
// Un external es contenido que no escribes tú (plugins de terceros, o un repo
// tuyo con vida propia) que el motor trae directamente a un destino y mantiene.
// Sustituye a los submódulos de git: solo se versiona su declaración.
//
// Responsabilidad: por cada external, hacer el fetch (git clone/pull, o descarga
// de un fichero) y, si ha cambiado (o es nuevo), ejecutar su script post. Es una
// fase best-effort (ADR 0010): si un external falla, se avisa y se sigue con el
// resto, con un resumen final. El post recibe el contrato de env de los hooks
// más DOTS_EXTERNAL_DEST.
package externals
