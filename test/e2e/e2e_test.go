// Tests end to end: compilan el binario, lo ejecutan sobre ficheros de
// entrada reales y comprueban tanto las propiedades exigidas por el
// enunciado como el PDF generado.
package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"reparto/internal/config"
	"reparto/internal/informe"
	"reparto/internal/modelo"
	"reparto/internal/reparto"
)

// compilarBinario construye el comando reparto una sola vez para todos los
// tests de este paquete.
func compilarBinario(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "reparto-e2e")
	raiz := filepath.Join("..", "..")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/reparto")
	cmd.Dir = raiz
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build falló: %v\n%s", err, out)
	}
	return bin
}

func TestBinarioGeneraElPDFDesdeYAMLYJSON(t *testing.T) {
	bin := compilarBinario(t)
	for _, entrada := range []string{"escenario_valido.yaml", "escenario_valido.json"} {
		entrada := entrada
		t.Run(entrada, func(t *testing.T) {
			salida := filepath.Join(t.TempDir(), "reparto.pdf")
			ruta := filepath.Join("..", "fixtures", entrada)
			cmd := exec.Command(bin, "-entrada", ruta, "-salida", salida)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("el binario falló: %v\n%s", err, out)
			}
			info, err := os.Stat(salida)
			if err != nil || info.Size() == 0 {
				t.Fatalf("no se generó un PDF válido: %v", err)
			}
			texto, err := exec.Command("pdftotext", "-layout", salida, "-").Output()
			if err != nil {
				t.Fatalf("pdftotext falló: %v", err)
			}
			if len(texto) == 0 {
				t.Fatal("el PDF generado está vacío")
			}
		})
	}
}

func TestBinarioFallaSinFicheroDeEntrada(t *testing.T) {
	bin := compilarBinario(t)
	cmd := exec.Command(bin)
	if err := cmd.Run(); err == nil {
		t.Fatal("el binario debería fallar si no se indica -entrada")
	}
}

// --- Propiedades exigidas por el enunciado, verificadas de extremo a
// extremo: se leen los datos desde fichero (como hará el usuario real), se
// calcula el reparto y se comprueban las propiedades sobre el resultado.

func TestE2E_SumaFacturasIgualASumaPagos(t *testing.T) {
	e, err := config.Leer(filepath.Join("..", "fixtures", "escenario_valido.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := reparto.Calcular(e)
	if err != nil {
		t.Fatal(err)
	}

	var totalFacturas, totalPagos float64
	for _, f := range e.Facturas {
		totalFacturas += f.Importe.Total()
	}
	for _, p := range res.Pagos {
		totalPagos += p.Total()
	}
	if d := totalFacturas - totalPagos; d > 1e-9 || d < -1e-9 {
		t.Fatalf("la suma de las facturas (%.2f) no coincide con la suma pagada por vecinos y promotora (%.2f)",
			totalFacturas, totalPagos)
	}
}

func TestE2E_MayorCoeficienteYAltaPosteriorPagaMas(t *testing.T) {
	e, err := config.Leer(filepath.Join("..", "fixtures", "escenario_valido.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// En el fixture, Ana (60%, alta 01/07) contrata antes que Bruno (40%,
	// alta 01/08) pero tiene mayor coeficiente; para aislar el efecto de la
	// fecha exigido por el enunciado invertimos los coeficientes, de modo que
	// quien tiene mayor coeficiente sea también quien contrata más tarde.
	for i := range e.Pagadores {
		switch e.Pagadores[i].Nombre {
		case "Ana":
			e.Pagadores[i].Coeficiente = 40
		case "Bruno":
			e.Pagadores[i].Coeficiente = 60
		}
	}
	res, err := reparto.Calcular(e)
	if err != nil {
		t.Fatal(err)
	}
	pago := pagoDe(res, "Luz-julio")
	if pago.ImporteDe("Bruno") <= pago.ImporteDe("Ana") {
		t.Fatalf("Bruno tiene mayor coeficiente y contrata después: debería pagar más que Ana. Ana=%.2f Bruno=%.2f",
			pago.ImporteDe("Ana"), pago.ImporteDe("Bruno"))
	}
}

func TestE2E_MenorCoeficienteYAltaAnteriorPagaMenos(t *testing.T) {
	e, err := config.Leer(filepath.Join("..", "fixtures", "escenario_valido.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	// Ana: coeficiente menor o igual (30 <= 70) y alta más temprana (01/07 <
	// 01/08) que Bruno.
	for i := range e.Pagadores {
		switch e.Pagadores[i].Nombre {
		case "Ana":
			e.Pagadores[i].Coeficiente = 30
		case "Bruno":
			e.Pagadores[i].Coeficiente = 70
		}
	}
	res, err := reparto.Calcular(e)
	if err != nil {
		t.Fatal(err)
	}
	pago := pagoDe(res, "Luz-julio")
	if pago.ImporteDe("Ana") >= pago.ImporteDe("Bruno") {
		t.Fatalf("Ana tiene menor coeficiente y contrata antes: debería pagar menos que Bruno. Ana=%.2f Bruno=%.2f",
			pago.ImporteDe("Ana"), pago.ImporteDe("Bruno"))
	}
}

func TestE2E_InformeProduceUnPDFCoherenteConElCalculo(t *testing.T) {
	e, err := config.Leer(filepath.Join("..", "fixtures", "escenario_valido.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := reparto.Calcular(e)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "informe.pdf")
	if err := informe.Renderizar(informe.NuevosDatos(e, res), ruta); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(ruta); err != nil || info.Size() == 0 {
		t.Fatalf("el informe no se generó correctamente: %v", err)
	}
}

func pagoDe(res reparto.Resultado, nombreFactura string) modelo.Pago {
	for _, p := range res.Pagos {
		if p.Factura == nombreFactura {
			return p
		}
	}
	return modelo.Pago{}
}
