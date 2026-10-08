package informe

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed fuentes/*.ttf
var ficherosFuentes embed.FS

// extraerFuentes copia las fuentes embebidas (DejaVu, con soporte de tildes y
// el símbolo €) a un directorio temporal, porque fpdf.AddUTF8Font necesita
// una ruta en disco. Devuelve ese directorio; el llamador debe borrarlo.
func extraerFuentes() (string, error) {
	dir, err := os.MkdirTemp("", "reparto-fuentes-*")
	if err != nil {
		return "", fmt.Errorf("creando directorio temporal de fuentes: %w", err)
	}
	entradas, err := ficherosFuentes.ReadDir("fuentes")
	if err != nil {
		return "", err
	}
	for _, e := range entradas {
		datos, err := ficherosFuentes.ReadFile(filepath.Join("fuentes", e.Name()))
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(dir, e.Name()), datos, 0o644); err != nil {
			return "", fmt.Errorf("escribiendo fuente %s: %w", e.Name(), err)
		}
	}
	return dir, nil
}
