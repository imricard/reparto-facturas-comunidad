package informe

import (
	"fmt"
	"os"

	"github.com/go-pdf/fpdf"

	"reparto/internal/modelo"
)

// reportTemplate es la plantilla del informe: una secuencia fija de
// secciones que se rellenan con los Datos de un cálculo. Cambiar el aspecto
// del informe significa cambiar los métodos de este fichero, no el resto del
// paquete.
type reportTemplate struct {
	pdf        *fpdf.Fpdf
	dirFuentes string
}

const (
	familiaFuente = "DejaVu"
	colorTitulo   = 31 // gris oscuro, en escala de grises de fpdf
)

// nuevaPlantilla crea el documento PDF (A4, fuente DejaVu para tildes y €) y
// prepara sus estilos base. El llamador debe invocar cerrar() al terminar.
func nuevaPlantilla() (*reportTemplate, error) {
	dir, err := extraerFuentes()
	if err != nil {
		return nil, err
	}
	pdf := fpdf.New("P", "mm", "A4", dir)
	pdf.SetMargins(18, 20, 18)
	pdf.AddUTF8Font(familiaFuente, "", "DejaVuSansCondensed.ttf")
	pdf.AddUTF8Font(familiaFuente, "B", "DejaVuSansCondensed-Bold.ttf")
	pdf.AddUTF8Font(familiaFuente, "I", "DejaVuSansCondensed-Oblique.ttf")
	pdf.SetAutoPageBreak(true, 20)
	pdf.SetFont(familiaFuente, "", 10)
	pdf.AddPage()
	if err := pdf.Error(); err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("preparando el PDF: %w", err)
	}
	return &reportTemplate{pdf: pdf, dirFuentes: dir}, nil
}

func (t *reportTemplate) cerrar() { os.RemoveAll(t.dirFuentes) }

// Renderizar rellena la plantilla report-template con los datos indicados y
// escribe el PDF resultante en ruta.
func Renderizar(d Datos, ruta string) error {
	t, err := nuevaPlantilla()
	if err != nil {
		return err
	}
	defer t.cerrar()

	t.cabecera(d)
	t.supuestos(d)
	t.resumenPorVecino(d)
	t.desglosePorFactura(d)

	if err := t.pdf.Error(); err != nil {
		return fmt.Errorf("componiendo el informe: %w", err)
	}
	if err := t.pdf.OutputFileAndClose(ruta); err != nil {
		return fmt.Errorf("generando %s: %w", ruta, err)
	}
	return nil
}

// --- utilidades de maquetación compartidas por las secciones ---

func (t *reportTemplate) tituloSeccion(texto string) {
	t.pdf.Ln(4)
	t.pdf.SetFont(familiaFuente, "B", 13)
	t.pdf.CellFormat(0, 8, texto, "", 1, "L", false, 0, "")
	t.pdf.SetDrawColor(colorTitulo, colorTitulo, colorTitulo)
	anchoPagina, _ := t.pdf.GetPageSize()
	x, y := t.pdf.GetXY()
	t.pdf.Line(t.pdf.GetX(), y, anchoPagina-18, y)
	t.pdf.SetXY(x, y+3)
	t.pdf.SetFont(familiaFuente, "", 10)
}

func euros(v float64) string { return fmt.Sprintf("%.2f €", v) }

func fechaOTexto(p modelo.Pagador, servicio string) string {
	f := p.FechaAlta(servicio)
	if f.IsZero() {
		return "no dado de alta"
	}
	return f.Format("02/01/2006")
}
