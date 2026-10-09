package modelo

import (
	"strings"
	"testing"
	"time"

	"reparto/internal/periodo"
)

func d(y int, m time.Month, dia int) time.Time {
	return time.Date(y, m, dia, 0, 0, 0, 0, time.UTC)
}

func ventana(t *testing.T, ini, fin time.Time) periodo.Periodo {
	t.Helper()
	p, err := periodo.Nuevo(ini, fin)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestImporteTotal(t *testing.T) {
	if got := (Importe{Consumo: 10.5, Otros: 4.25}).Total(); got != 14.75 {
		t.Fatalf("Total() = %v", got)
	}
}

func TestEsSuministro(t *testing.T) {
	for tipo, want := range map[string]bool{"luz": true, "agua": true, "ascensor": false, "": false} {
		if got := (Factura{Tipo: tipo}).EsSuministro(); got != want {
			t.Errorf("EsSuministro(%q) = %v, quería %v", tipo, got, want)
		}
	}
}

func TestEsPropietarioIncluyeElDiaDeCompra(t *testing.T) {
	p := Pagador{FechaCompra: d(2025, 6, 20)}
	for dia, want := range map[time.Time]bool{
		d(2025, 6, 19): false,
		d(2025, 6, 20): true,
		d(2025, 6, 21): true,
	} {
		if got := p.EsPropietario(dia); got != want {
			t.Errorf("EsPropietario(%v) = %v, quería %v", dia, got, want)
		}
	}
	if (Pagador{}).EsPropietario(d(2030, 1, 1)) {
		t.Error("sin fecha de compra nunca es propietario")
	}
}

func TestTieneAlta(t *testing.T) {
	p := Pagador{FechaAltaLuz: d(2025, 7, 1)}
	if p.TieneAlta(TipoLuz, d(2025, 6, 30)) || !p.TieneAlta(TipoLuz, d(2025, 7, 1)) {
		t.Error("el alta de luz debe contar desde el día indicado")
	}
	if p.TieneAlta(TipoAgua, d(2030, 1, 1)) {
		t.Error("sin fecha de alta de agua nunca tiene el servicio")
	}
	if p.TieneAlta("gas", d(2030, 1, 1)) {
		t.Error("servicio desconocido")
	}
}

func TestPago(t *testing.T) {
	p := Pago{Factura: "F", Reparto: []Reparto{{"A", 10}, {"B", 5.5}}}
	if p.Total() != 15.5 || p.ImporteDe("B") != 5.5 || p.ImporteDe("Z") != 0 {
		t.Fatalf("Pago mal calculado: %+v", p)
	}
}

func escenarioValido(t *testing.T) Escenario {
	return Escenario{
		FinObra: d(2025, 6, 12),
		Facturas: []Factura{
			{Nombre: "F1", Tipo: TipoLuz, Ventana: ventana(t, d(2025, 6, 1), d(2025, 6, 30)), Importe: Importe{1, 1}},
		},
		Pagadores: []Pagador{
			{Nombre: "Promo", Tipo: TipoConstructora},
			{Nombre: "A", Tipo: TipoVecino, Coeficiente: 60, FechaAltaLuz: d(2025, 7, 1)},
			{Nombre: "B", Tipo: TipoVecino, Coeficiente: 40, FechaAltaLuz: d(2025, 8, 1)},
		},
	}
}

func TestEscenarioAccesores(t *testing.T) {
	e := escenarioValido(t)
	if len(e.Vecinos()) != 2 || e.Constructora().Nombre != "Promo" {
		t.Fatalf("Vecinos/Constructora incorrectos")
	}
	if !e.PrimerDiaReparto().Equal(d(2025, 6, 13)) {
		t.Fatalf("PrimerDiaReparto = %v", e.PrimerDiaReparto())
	}
	if e.SumaCoeficientes() != 100 {
		t.Fatalf("SumaCoeficientes = %v", e.SumaCoeficientes())
	}
}

func TestFechaAltaCompleta(t *testing.T) {
	e := escenarioValido(t)
	if got, ok := e.FechaAltaCompleta(TipoLuz); !ok || !got.Equal(d(2025, 8, 1)) {
		t.Fatalf("FechaAltaCompleta(luz) = %v, %v", got, ok)
	}
	if _, ok := e.FechaAltaCompleta(TipoAgua); ok {
		t.Fatal("nadie ha dado de alta el agua: no debe haber fecha completa")
	}
}

func TestValidar(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*Escenario)
		quiere string // "" = válido
	}{
		{"válido", func(*Escenario) {}, ""},
		{"sin fin de obra", func(e *Escenario) { e.FinObra = time.Time{} }, "fin de obra"},
		{"sin constructora", func(e *Escenario) { e.Pagadores = e.Pagadores[1:] }, "exactamente una constructora"},
		{"coeficientes de más", func(e *Escenario) { e.Pagadores[1].Coeficiente = 90 }, "más del 100"},
		{"coeficiente cero", func(e *Escenario) { e.Pagadores[1].Coeficiente = 0 }, "coeficiente"},
		{"nombre duplicado", func(e *Escenario) { e.Pagadores[2].Nombre = "A" }, "duplicado"},
		{"tipo de pagador raro", func(e *Escenario) { e.Pagadores[2].Tipo = "gnomo" }, "desconocido"},
		{"factura sin ventana", func(e *Escenario) { e.Facturas[0].Ventana = periodo.Periodo{} }, "ventana"},
		{"importe negativo", func(e *Escenario) { e.Facturas[0].Importe.Otros = -1 }, "importe"},
		{"factura duplicada", func(e *Escenario) { e.Facturas = append(e.Facturas, e.Facturas[0]) }, "duplicado"},
		{"sin vecinos", func(e *Escenario) { e.Pagadores = e.Pagadores[:1] }, "ningún vecino"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			e := escenarioValido(t)
			c.mutar(&e)
			err := e.Validar()
			switch {
			case c.quiere == "" && err != nil:
				t.Fatalf("error inesperado: %v", err)
			case c.quiere != "" && (err == nil || !strings.Contains(err.Error(), c.quiere)):
				t.Fatalf("quería un error con %q, obtuve %v", c.quiere, err)
			}
		})
	}
}

func TestValidarAcumulaVariosErrores(t *testing.T) {
	e := escenarioValido(t)
	e.FinObra = time.Time{}
	e.Pagadores[1].Coeficiente = 0
	err := e.Validar()
	if err == nil || !strings.Contains(err.Error(), "fin de obra") || !strings.Contains(err.Error(), "coeficiente") {
		t.Fatalf("Validar debe informar de todos los problemas: %v", err)
	}
}

func TestEsAsignada(t *testing.T) {
	if (Factura{}).EsAsignada() {
		t.Error("sin AsignadoA no debería estar asignada")
	}
	if !(Factura{AsignadoA: "Cuarto"}).EsAsignada() {
		t.Error("con AsignadoA debería estar asignada")
	}
}

func TestValidarAsignadoA(t *testing.T) {
	e := escenarioValido(t)
	e.Facturas[0].AsignadoA = "A" // "A" es un vecino real del escenario
	if err := e.Validar(); err != nil {
		t.Fatalf("asignar a un vecino existente debería ser válido: %v", err)
	}

	e2 := escenarioValido(t)
	e2.Facturas[0].AsignadoA = "Quien-no-existe"
	err := e2.Validar()
	if err == nil || !strings.Contains(err.Error(), "no es ningún vecino") {
		t.Fatalf("asignar a alguien que no es vecino debería fallar: %v", err)
	}

	e3 := escenarioValido(t)
	e3.Facturas[0].AsignadoA = "Promo" // existe pero es la constructora, no un vecino
	err = e3.Validar()
	if err == nil || !strings.Contains(err.Error(), "no es ningún vecino") {
		t.Fatalf("asignar a la constructora debería fallar: %v", err)
	}
}
