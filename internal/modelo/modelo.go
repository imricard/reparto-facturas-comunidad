// Package modelo define los datos del problema (facturas, pagadores y pagos)
// y las reglas que dependen únicamente de esos datos: quién es propietario o
// tiene un servicio dado de alta en un día concreto, y si un escenario es
// coherente.
//
// No sabe nada de cómo se reparte el dinero (paquete reparto), de cómo se
// leen los datos (config) ni de cómo se presentan (informe).
package modelo

import (
	"time"

	"reparto/internal/periodo"
)

// Tipos de factura con tratamiento especial. Cualquier otro tipo
// (comunidad, ascensor, ...) se reparte solo por propiedad.
const (
	TipoLuz  = "luz"
	TipoAgua = "agua"
)

// Tipos de pagador.
const (
	TipoVecino       = "vecino"
	TipoConstructora = "constructora"
)

// Importe es el importe de una factura, separado en la parte asociada al
// consumo y el resto (término fijo, potencia, cuotas, impuestos...).
type Importe struct {
	Consumo float64
	Otros   float64
}

// Total devuelve la suma de ambas partes.
func (i Importe) Total() float64 { return i.Consumo + i.Otros }

// Factura es una factura reclamada por la constructora.
type Factura struct {
	Nombre  string
	Importe Importe
	Tipo    string
	// Ventana es el periodo de facturación que cubre la factura.
	Ventana periodo.Periodo
	// AsignadoA, si no está vacío, es el nombre de un vecino que asume él
	// solo el importe íntegro de la factura (p. ej. una tasa que
	// corresponde a su vivienda en concreto). En ese caso la factura no se
	// reparte por coeficiente ni se tiene en cuenta para estimar el
	// consumo comunitario de referencia: ni el resto de vecinos ni la
	// constructora pagan nada de ella.
	AsignadoA string
}

// EsAsignada indica si la factura está asignada íntegramente a un único
// vecino en lugar de repartirse entre todos.
func (f Factura) EsAsignada() bool { return f.AsignadoA != "" }

// EsSuministro indica si la factura es de un servicio (luz o agua) que cada
// vecino debe dar de alta y cuyo consumo se separa en comunitario e
// individual.
func (f Factura) EsSuministro() bool { return f.Tipo == TipoLuz || f.Tipo == TipoAgua }

// Pagador es un vecino o la constructora.
type Pagador struct {
	Nombre string
	Tipo   string
	// FechaCompra es el primer día en que el vecino es propietario. La fecha
	// cero significa que todavía no ha comprado.
	FechaCompra time.Time
	// FechaAltaLuz y FechaAltaAgua son el primer día con el servicio a su
	// nombre. La fecha cero significa que aún no lo ha dado de alta.
	FechaAltaLuz  time.Time
	FechaAltaAgua time.Time
	// Coeficiente es el porcentaje (0-100) de participación en la comunidad.
	Coeficiente float64
}

// EsVecino indica si el pagador es un vecino (y no la constructora).
func (p Pagador) EsVecino() bool { return p.Tipo == TipoVecino }

// EsPropietario indica si el pagador es propietario el día indicado, es
// decir, si ese día es igual o posterior a su fecha de compra.
func (p Pagador) EsPropietario(dia time.Time) bool {
	compra := periodo.Dia(p.FechaCompra)
	return !compra.IsZero() && !periodo.Dia(dia).Before(compra)
}

// FechaAlta devuelve el día de alta del servicio ("luz" o "agua"); la fecha
// cero indica que no lo ha dado de alta o que el servicio es desconocido.
func (p Pagador) FechaAlta(servicio string) time.Time {
	switch servicio {
	case TipoLuz:
		return periodo.Dia(p.FechaAltaLuz)
	case TipoAgua:
		return periodo.Dia(p.FechaAltaAgua)
	default:
		return time.Time{}
	}
}

// TieneAlta indica si el servicio está dado de alta por el pagador el día
// indicado (el día de alta cuenta como dado de alta).
func (p Pagador) TieneAlta(servicio string, dia time.Time) bool {
	alta := p.FechaAlta(servicio)
	return !alta.IsZero() && !periodo.Dia(dia).Before(alta)
}

// Reparto es lo que paga un pagador de una factura.
type Reparto struct {
	Pagador string
	Importe float64
}

// Pago es el reparto completo de una factura entre todos los pagadores.
type Pago struct {
	Factura string
	Reparto []Reparto
}

// Total suma lo que pagan todos los pagadores.
func (p Pago) Total() float64 {
	var t float64
	for _, r := range p.Reparto {
		t += r.Importe
	}
	return t
}

// ImporteDe devuelve lo que paga el pagador indicado (0 si no aparece).
func (p Pago) ImporteDe(pagador string) float64 {
	for _, r := range p.Reparto {
		if r.Pagador == pagador {
			return r.Importe
		}
	}
	return 0
}
