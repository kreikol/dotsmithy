// Comando dots: punto de entrada del motor dotsmithy.
//
// El binario es único y estáticamente enlazado. Toda la lógica de comandos
// vive en internal/cli; aquí solo se cede el control al árbol de cobra.
package main

import "go.kreikol.dev/dotsmithy/internal/cli"

func main() {
	cli.Execute()
}
