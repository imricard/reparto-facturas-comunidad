package informe

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"reparto/internal/modelo"
	"reparto/internal/periodo"
	"reparto/internal/reparto"
)

func d(y int, m time.Month, dia int) time.Time { return time.Date(y, m, dia, 0, 0, 0, 0, time.UTC) }

func escenarioDePrueba(t *testing.T) modelo.Escenario {
	t.Helper()
	vent := func(ini, fin time.Time) periodo.Periodo {
		p, err := periodo.Nuevo(ini, fin)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	return modelo.Escenario{
		FinObra: d(2025, 6, 12),
		Pagadores: []modelo.Pagador{
			{Nombre: "Promotora", Tipo: modelo.TipoConstructora},
			{Nombre: "Ana", Tipo: modelo.TipoVecino, Coeficiente: 60, FechaCompra: d(2025, 6, 1), FechaAltaLuz: d(2025, 7, 1), FechaAltaAgua: d(2025, 7, 1)},
			{Nombre: "Bruno", Tipo: modelo.TipoVecino, Coeficiente: 40, FechaCompra: d(2025, 6, 15), FechaAltaLuz: d(2025, 8, 1), FechaAltaAgua: d(2025, 8, 1)},
		},
		Facturas: []modelo.Factura{
			{Nombre: "Comunidad-junio", Tipo: "comunidad", Importe: modelo.Importe{Otros: 100}, Ventana: vent(d(2025, 6, 1), d(2025, 6, 30))},
			{Nombre: "Luz-julio", Tipo: modelo.TipoLuz, Importe: modelo.Importe{Consumo: 93, Otros: 31}, Ventana: vent(d(2025, 7, 1), d(2025, 7, 31))},
			{Nombre: "Luz-septiembre", Tipo: modelo.TipoLuz, Importe: modelo.Importe{Consumo: 60, Otros: 30}, Ventana: vent(d(2025, 9, 1), d(2025, 9, 30))},
		},
	}
}

// textoDelPDF usa pdftotext (poppler-utils) para poder comprobar el
// contenido real del PDF generado, no solo que el fichero exista.
func textoDelPDF(t *testing.T, ruta string) string {
	t.Helper()
	salida, err := exec.Command("pdftotext", "-layout", ruta, "-").Output()
	if err != nil {
		t.Fatalf("pdftotext falló: %v", err)
	}
	return string(salida)
}

func TestRenderizarGeneraUnPDFConElContenidoEsperado(t *testing.T) {
	e := escenarioDePrueba(t)
	res, err := reparto.Calcular(e)
	if err != nil {
		t.Fatal(err)
	}
	datos := NuevosDatos(e, res)

	ruta := filepath.Join(t.TempDir(), "informe.pdf")
	if err := Renderizar(datos, ruta); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(ruta)
	if err != nil || info.Size() == 0 {
		t.Fatalf("el PDF no se generó correctamente: %v", err)
	}

	texto := textoDelPDF(t, ruta)
	requeridos := []string{
		"Reparto de facturas", "Supuestos", "12/06/2025",
		"Ana", "Bruno", "Resumen", "Desglose por factura",
		"Comunidad-junio", "Luz-julio", "Luz-septiembre",
	}
	for _, r := range requeridos {
		if !strings.Contains(texto, r) {
			t.Errorf("el PDF debería contener %q; texto extraído:\n%s", r, texto)
		}
	}
}

func TestRenderizarFallaConRutaInvalida(t *testing.T) {
	e := escenarioDePrueba(t)
	res, err := reparto.Calcular(e)
	if err != nil {
		t.Fatal(err)
	}
	err = Renderizar(NuevosDatos(e, res), filepath.Join("directorio", "que", "no", "existe", "informe.pdf"))
	if err == nil {
		t.Fatal("quería un error al escribir en una ruta inexistente")
	}
}
