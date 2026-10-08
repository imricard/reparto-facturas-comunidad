// Package informe genera el PDF de resultado a partir de una plantilla
// llamada report-template: una estructura fija de secciones (cabecera,
// supuestos, resumen por vecino, desglose por factura) que se rellena con los
// datos de un cálculo ya hecho. Es el único paquete que sabe cómo se ve el
// informe; no vuelve a calcular nada del reparto.
package informe

import (
	"sort"

	"reparto/internal/modelo"
	"reparto/internal/reparto"
)

// Datos es la información ya calculada que la plantilla necesita para
// rellenarse. Se obtiene una única vez con NuevosDatos y no depende de fpdf
// ni de ningún detalle de maquetación.
type Datos struct {
	Escenario      modelo.Escenario
	Resultado      reparto.Resultado
	TotalPorVecino map[string]float64
	TotalGeneral   float64
}

// NuevosDatos organiza el resultado de un cálculo para la plantilla.
func NuevosDatos(e modelo.Escenario, res reparto.Resultado) Datos {
	d := Datos{Escenario: e, Resultado: res, TotalPorVecino: map[string]float64{}}
	for _, pago := range res.Pagos {
		for _, r := range pago.Reparto {
			d.TotalPorVecino[r.Pagador] += r.Importe
			d.TotalGeneral += r.Importe
		}
	}
	return d
}

// VecinosOrdenados devuelve los vecinos en el mismo orden que el escenario
// (el orden de entrada, que es el que espera ver el usuario).
func (d Datos) VecinosOrdenados() []modelo.Pagador { return d.Escenario.Vecinos() }

// PagoDe devuelve el pago de una factura por su nombre.
func (d Datos) PagoDe(nombreFactura string) modelo.Pago {
	for _, p := range d.Resultado.Pagos {
		if p.Factura == nombreFactura {
			return p
		}
	}
	return modelo.Pago{}
}

// ordenNombres devuelve una copia ordenada alfabéticamente de una lista de
// nombres; se usa para que las tablas sean deterministas.
func ordenNombres(nombres []string) []string {
	out := append([]string(nil), nombres...)
	sort.Strings(out)
	return out
}
