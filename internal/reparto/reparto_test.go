package reparto

import (
	"errors"
	"strings"
	"testing"
	"time"

	"reparto/internal/modelo"
	"reparto/internal/periodo"
)

func d(y int, m time.Month, dia int) time.Time { return time.Date(y, m, dia, 0, 0, 0, 0, time.UTC) }

func vent(t *testing.T, ini, fin time.Time) periodo.Periodo {
	t.Helper()
	p, err := periodo.Nuevo(ini, fin)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func total(res Resultado) float64 {
	var s float64
	for _, p := range res.Pagos {
		s += p.Total()
	}
	return s
}

// escenarioBase: obra termina 12/06/2025; A (60%) compra el 1/06 y da de alta
// luz el 1/07; B (40%) compra el 15/06 y da de alta luz el 1/08.
func escenarioBase(t *testing.T) modelo.Escenario {
	return modelo.Escenario{
		FinObra: d(2025, 6, 12),
		Pagadores: []modelo.Pagador{
			{Nombre: "Promotora", Tipo: modelo.TipoConstructora},
			{Nombre: "A", Tipo: modelo.TipoVecino, Coeficiente: 60, FechaCompra: d(2025, 6, 1), FechaAltaLuz: d(2025, 7, 1)},
			{Nombre: "B", Tipo: modelo.TipoVecino, Coeficiente: 40, FechaCompra: d(2025, 6, 15), FechaAltaLuz: d(2025, 8, 1)},
		},
		Facturas: []modelo.Factura{
			{Nombre: "Comunidad-jun", Tipo: "comunidad", Importe: modelo.Importe{Otros: 100}, Ventana: vent(t, d(2025, 6, 1), d(2025, 6, 30))},
			{Nombre: "Luz-jul", Tipo: modelo.TipoLuz, Importe: modelo.Importe{Consumo: 93, Otros: 31}, Ventana: vent(t, d(2025, 7, 1), d(2025, 7, 31))},
			{Nombre: "Luz-sep", Tipo: modelo.TipoLuz, Importe: modelo.Importe{Consumo: 60, Otros: 30}, Ventana: vent(t, d(2025, 9, 1), d(2025, 9, 30))},
		},
	}
}

func TestCalcularRechazaEscenarioInvalido(t *testing.T) {
	e := escenarioBase(t)
	e.FinObra = time.Time{}
	if _, err := Calcular(e); err == nil {
		t.Fatal("quería un error de validación")
	}
}

func TestSumaTotalCoincide(t *testing.T) {
	e := escenarioBase(t)
	res, err := Calcular(e)
	if err != nil {
		t.Fatal(err)
	}
	var totalFacturas float64
	for _, f := range e.Facturas {
		totalFacturas += f.Importe.Total()
	}
	if got := total(res); absf(got-totalFacturas) > 1e-9 {
		t.Fatalf("suma de pagos = %v, suma de facturas = %v", got, totalFacturas)
	}
}

func TestObraSinTerminarLoPagaTodoLaConstructora(t *testing.T) {
	e := escenarioBase(t)
	e.Facturas = []modelo.Factura{
		{Nombre: "Antes", Tipo: "comunidad", Importe: modelo.Importe{Otros: 50}, Ventana: vent(t, d(2025, 6, 1), d(2025, 6, 12))},
	}
	res, err := Calcular(e)
	if err != nil {
		t.Fatal(err)
	}
	pago := res.Pagos[0]
	if pago.ImporteDe("A") != 0 || pago.ImporteDe("B") != 0 || pago.ImporteDe("Promotora") != 50 {
		t.Fatalf("antes del fin de obra todo debe ser de la constructora: %+v", pago)
	}
}

func TestPropietarioSoloPagaSuParteProporcional(t *testing.T) {
	e := escenarioBase(t)
	// Solo la factura de comunidad, para aislar el efecto de la fecha de compra.
	e.Facturas = e.Facturas[:1]
	res, err := Calcular(e)
	if err != nil {
		t.Fatal(err)
	}
	pago := res.Pagos[0]
	// B solo es propietario desde el 15/06 (16 de los 30 días de junio, y solo
	// a partir del 13/06 se reparte: 13,14 constructora-por-B-no-propietario,
	// 15..30 = 16 días de B como propietario sobre 18 días repartibles).
	if pago.ImporteDe("B") <= 0 || pago.ImporteDe("B") >= pago.ImporteDe("A") {
		t.Fatalf("B compra más tarde y tiene menor coeficiente: A=%.2f B=%.2f",
			pago.ImporteDe("A"), pago.ImporteDe("B"))
	}
	if pago.ImporteDe("Promotora") <= 0 {
		t.Fatalf("la promotora debe asumir los días de B antes de su compra")
	}
}

func TestConsumoIndividualSoloEntreQuienesNoTienenAlta(t *testing.T) {
	e := escenarioBase(t)
	res, err := Calcular(e)
	if err != nil {
		t.Fatal(err)
	}
	pagoJul := res.Pagos[1] // Luz-jul: A ya tiene alta (1/07), B no.
	// El consumo de referencia (media de Luz-sep, cuando ambos ya tienen
	// alta) es menor que el consumo real de julio, así que hay un exceso de
	// consumo individual que solo debe recaer sobre B (el único sin alta).
	// Por tanto B debe pagar más que su 40% proporcional del total.
	totalJul := e.Facturas[1].Importe.Total()
	if pagoJul.ImporteDe("B") <= totalJul*0.40+1e-9 {
		t.Fatalf("B, sin alta, debería cargar con consumo individual además de su parte: B=%.2f (40%% del total=%.2f)",
			pagoJul.ImporteDe("B"), totalJul*0.40)
	}
	if pagoJul.ImporteDe("A") >= totalJul*0.60-1e-9 {
		t.Fatalf("A, con alta, no debería cargar con consumo individual: A=%.2f (60%% del total=%.2f)",
			pagoJul.ImporteDe("A"), totalJul*0.60)
	}
}

func TestSinReferenciaDevuelveErrorClaro(t *testing.T) {
	e := escenarioBase(t)
	e.Pagadores[2].FechaAltaLuz = time.Time{} // B nunca da de alta la luz
	_, err := Calcular(e)
	if err == nil || !errors.Is(err, ErrSinReferencia) {
		t.Fatalf("quería ErrSinReferencia, obtuve %v", err)
	}
	if !strings.Contains(err.Error(), "luz") {
		t.Fatalf("el error debería mencionar el servicio: %v", err)
	}
}

// TestMayorCoeficienteYAltaMasTardiaPagaMas es el test end-to-end pedido:
// si X tiene coeficiente >= Y y contrata el servicio después que Y, X paga
// más que Y en esa factura de suministro.
func TestMayorCoeficienteYAltaMasTardiaPagaMas(t *testing.T) {
	e := escenarioBase(t) // A: 60% alta 1/07 (antes); B: 40% alta 1/08 (después)
	// Invertimos para que quien tiene mayor coeficiente sea también quien da
	// de alta más tarde, que es el caso exigido por el enunciado.
	e.Pagadores[1].Coeficiente, e.Pagadores[2].Coeficiente = 40, 60
	e.Pagadores[1].FechaAltaLuz, e.Pagadores[2].FechaAltaLuz = d(2025, 7, 1), d(2025, 8, 1)
	// Ahora B (60%, alta después) debe pagar más que A (40%, alta antes) en Luz-jul.
	res, err := Calcular(e)
	if err != nil {
		t.Fatal(err)
	}
	pagoJul := res.Pagos[1]
	if pagoJul.ImporteDe("B") <= pagoJul.ImporteDe("A") {
		t.Fatalf("B tiene mayor coeficiente y contrata después: debería pagar más. A=%.2f B=%.2f",
			pagoJul.ImporteDe("A"), pagoJul.ImporteDe("B"))
	}
}

// TestMenorCoeficienteYAltaMasTempranaPagaMenos es el caso inverso pedido: si
// X tiene coeficiente <= Y y contrata antes que Y, X paga menos que Y.
func TestMenorCoeficienteYAltaMasTempranaPagaMenos(t *testing.T) {
	e := escenarioBase(t)
	// X = A: coeficiente menor (30 <= 70) y alta más temprana (1/07 < 1/08).
	e.Pagadores[1].Coeficiente, e.Pagadores[2].Coeficiente = 30, 70
	res, err := Calcular(e)
	if err != nil {
		t.Fatal(err)
	}
	pagoJul := res.Pagos[1]
	if pagoJul.ImporteDe("A") >= pagoJul.ImporteDe("B") {
		t.Fatalf("A tiene menor coeficiente y contrata antes: debería pagar menos que B. A=%.2f B=%.2f",
			pagoJul.ImporteDe("A"), pagoJul.ImporteDe("B"))
	}
}

func absf(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// --- Tests del caso descrito por el usuario: un vecino se da de alta A
// MITAD de la ventana de la factura, y debe asumir el consumo individual
// íntegro de esa factura (no solo prorrateado por los días que estuvo sin
// alta), porque es el único rezagado en toda la ventana.

func TestConsumoIndividualIntegroCuandoElRezagadoSeDaDeAltaAMitadDeLaFactura(t *testing.T) {
	e := modelo.Escenario{
		FinObra: d(2025, 6, 12),
		Pagadores: []modelo.Pagador{
			{Nombre: "Promotora", Tipo: modelo.TipoConstructora},
			// Todos propietarios desde antes de agosto: para esta factura
			// nadie debería dejar nada sin asignar por no ser propietario.
			{Nombre: "A", Tipo: modelo.TipoVecino, Coeficiente: 30, FechaCompra: d(2025, 4, 15), FechaAltaLuz: d(2025, 7, 31)},
			{Nombre: "B", Tipo: modelo.TipoVecino, Coeficiente: 30, FechaCompra: d(2025, 7, 8), FechaAltaLuz: d(2025, 7, 20)},
			{Nombre: "C", Tipo: modelo.TipoVecino, Coeficiente: 20, FechaCompra: d(2025, 6, 27), FechaAltaLuz: d(2025, 7, 10)},
			// Rezagado: único sin alta durante la ventana de Luz-ago-set,
			// se da de alta a mitad de esa factura (20/09).
			{Nombre: "D", Tipo: modelo.TipoVecino, Coeficiente: 20, FechaCompra: d(2025, 6, 17), FechaAltaLuz: d(2025, 9, 20)},
		},
		Facturas: []modelo.Factura{
			// Factura de referencia: todos (incluido D) ya tienen alta.
			{Nombre: "Luz-referencia", Tipo: modelo.TipoLuz, Importe: modelo.Importe{Consumo: 60, Otros: 30}, Ventana: vent(t, d(2025, 10, 1), d(2025, 10, 30))},
			// La factura bajo prueba: D es el único rezagado, y deja de
			// serlo a mitad de la ventana (20/09, dentro de 31/08-30/09).
			{Nombre: "Luz-ago-set", Tipo: modelo.TipoLuz, Importe: modelo.Importe{Consumo: 67.09, Otros: 96.62}, Ventana: vent(t, d(2025, 8, 31), d(2025, 9, 30))},
		},
	}
	res, err := Calcular(e)
	if err != nil {
		t.Fatal(err)
	}
	pago := pagoDeTest(res, "Luz-ago-set")

	// La constructora no debe pagar nada: todos los vecinos son
	// propietarios durante toda la ventana de esta factura.
	if c := pago.ImporteDe("Promotora"); c > 1e-9 {
		t.Fatalf("la constructora no debería pagar nada en esta factura (todos son propietarios): pagó %.4f", c)
	}

	// D debe asumir el consumo individual íntegro de la factura, no solo la
	// parte proporcional a los días que estuvo sin alta (20 de 31).
	comunitario := 60.0 / 30 * 31 // EurosPorDia de la referencia * días de la factura
	individual := e.Facturas[1].Importe.Consumo - comunitario
	estructural := e.Facturas[1].Importe.Otros + comunitario
	// Parte estructural de D (por coeficiente, toda la ventana):
	estructuralD := estructural * 0.20
	esperadoD := estructuralD + individual
	if got := pago.ImporteDe("D"); abs2(got-esperadoD) > 0.02 {
		t.Fatalf("D debería asumir el consumo individual íntegro: got=%.4f, esperado=%.4f (individual=%.4f)",
			got, esperadoD, individual)
	}

	// La suma de la factura debe seguir cuadrando.
	if abs2(pago.Total()-e.Facturas[1].Importe.Total()) > 1e-6 {
		t.Fatalf("el pago no suma el total de la factura: %.4f vs %.4f", pago.Total(), e.Facturas[1].Importe.Total())
	}
}

func TestConsumoIndividualSeRepartePorCoeficienteYDiasSinAltaEntreVariosRezagados(t *testing.T) {
	// Replica el escenario de varios rezagados con fechas de alta
	// escalonadas: cada uno paga en proporción a coeficiente × días sin
	// alta, y la suma entre todos cubre el importe individual completo.
	e := modelo.Escenario{
		FinObra: d(2025, 6, 12),
		Pagadores: []modelo.Pagador{
			{Nombre: "Promotora", Tipo: modelo.TipoConstructora},
			{Nombre: "Tercero", Tipo: modelo.TipoVecino, Coeficiente: 14.61, FechaCompra: d(2025, 6, 27), FechaAltaLuz: d(2025, 8, 8)},
			{Nombre: "Bajos2", Tipo: modelo.TipoVecino, Coeficiente: 18.65, FechaCompra: d(2025, 7, 8), FechaAltaLuz: d(2025, 8, 9)},
			{Nombre: "Cuarto", Tipo: modelo.TipoVecino, Coeficiente: 19.4, FechaCompra: d(2025, 6, 17), FechaAltaLuz: d(2025, 8, 20)},
			{Nombre: "Bajos1", Tipo: modelo.TipoVecino, Coeficiente: 18.65, FechaCompra: d(2025, 4, 15), FechaAltaLuz: d(2025, 9, 20)},
			{Nombre: "Primero", Tipo: modelo.TipoVecino, Coeficiente: 14.16, FechaCompra: d(2025, 7, 7), FechaAltaLuz: d(2025, 7, 31)},
			{Nombre: "Segundo", Tipo: modelo.TipoVecino, Coeficiente: 14.53, FechaCompra: d(2025, 6, 12), FechaAltaLuz: d(2025, 7, 30)},
		},
		Facturas: []modelo.Factura{
			{Nombre: "Luz-referencia", Tipo: modelo.TipoLuz, Importe: modelo.Importe{Consumo: 60, Otros: 30}, Ventana: vent(t, d(2025, 10, 1), d(2025, 10, 30))},
			{Nombre: "Luz-jul-ago", Tipo: modelo.TipoLuz, Importe: modelo.Importe{Consumo: 105.08, Otros: 107.31}, Ventana: vent(t, d(2025, 7, 31), d(2025, 8, 31))},
		},
	}
	res, err := Calcular(e)
	if err != nil {
		t.Fatal(err)
	}
	pago := pagoDeTest(res, "Luz-jul-ago")

	comunitario := 60.0 / 30 * 32
	individual := e.Facturas[1].Importe.Consumo - comunitario

	// Días sin alta de cada rezagado dentro de la ventana 31/07-31/08:
	// Tercero: 31/07..07/08 = 8; Bajos2: 31/07..08/08 = 9;
	// Cuarto: 31/07..19/08 = 20; Bajos1: toda la ventana = 32.
	pesos := map[string]float64{
		"Tercero": 14.61 * 8,
		"Bajos2":  18.65 * 9,
		"Cuarto":  19.4 * 20,
		"Bajos1":  18.65 * 32,
	}
	var pesoTotal float64
	for _, p := range pesos {
		pesoTotal += p
	}

	var sumaIndividual float64
	for nombre, peso := range pesos {
		esperadoEstructural := (e.Facturas[1].Importe.Otros + comunitario) * (coefDe(e, nombre) / 100)
		esperadoIndividual := individual * peso / pesoTotal
		sumaIndividual += esperadoIndividual
		esperado := esperadoEstructural + esperadoIndividual
		if got := pago.ImporteDe(nombre); abs2(got-esperado) > 0.02 {
			t.Fatalf("%s: got=%.4f esperado=%.4f (estructural=%.4f individual=%.4f)",
				nombre, got, esperado, esperadoEstructural, esperadoIndividual)
		}
	}
	if abs2(sumaIndividual-individual) > 1e-6 {
		t.Fatalf("la suma del consumo individual entre rezagados (%.4f) no cubre el importe individual (%.4f)",
			sumaIndividual, individual)
	}

	// Primero y Segundo ya tenían alta antes de empezar la factura: no
	// asumen nada del consumo individual, solo su parte estructural.
	for _, nombre := range []string{"Primero", "Segundo"} {
		esperadoEstructural := (e.Facturas[1].Importe.Otros + comunitario) * (coefDe(e, nombre) / 100)
		if got := pago.ImporteDe(nombre); abs2(got-esperadoEstructural) > 0.02 {
			t.Fatalf("%s no debería asumir consumo individual: got=%.4f esperado=%.4f", nombre, got, esperadoEstructural)
		}
	}

	if c := pago.ImporteDe("Promotora"); c > 1e-9 {
		t.Fatalf("la constructora no debería pagar nada (todos propietarios): pagó %.4f", c)
	}
}

func pagoDeTest(res Resultado, nombreFactura string) modelo.Pago {
	for _, p := range res.Pagos {
		if p.Factura == nombreFactura {
			return p
		}
	}
	return modelo.Pago{}
}

func coefDe(e modelo.Escenario, nombre string) float64 {
	for _, v := range e.Vecinos() {
		if v.Nombre == nombre {
			return v.Coeficiente
		}
	}
	return 0
}

func abs2(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
