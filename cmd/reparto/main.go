// Comando reparto: lee un escenario (facturas, pagadores) de un fichero JSON
// o YAML, calcula cuánto debe pagar cada vecino y la constructora, y genera
// un informe en PDF con el resultado.
package main

import (
	"flag"
	"fmt"
	"os"

	"reparto/internal/config"
	"reparto/internal/informe"
	"reparto/internal/reparto"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("reparto", flag.ContinueOnError)
	entrada := fs.String("entrada", "", "fichero de entrada .json, .yaml o .yml (obligatorio)")
	salida := fs.String("salida", "reparto.pdf", "fichero PDF de salida")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *entrada == "" {
		fs.Usage()
		return fmt.Errorf("falta el fichero de entrada (usa -entrada)")
	}

	escenario, err := config.Leer(*entrada)
	if err != nil {
		return fmt.Errorf("leyendo la entrada: %w", err)
	}
	resultado, err := reparto.Calcular(escenario)
	if err != nil {
		return fmt.Errorf("calculando el reparto: %w", err)
	}
	if err := informe.Renderizar(informe.NuevosDatos(escenario, resultado), *salida); err != nil {
		return fmt.Errorf("generando el informe: %w", err)
	}

	fmt.Printf("Informe generado en %s\n", *salida)
	return nil
}
