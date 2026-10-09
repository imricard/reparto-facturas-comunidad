package reparto

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"reparto/internal/modelo"
)

// ErrSinReferencia indica que no se puede estimar el consumo comunitario de
// un servicio porque no hay facturas de un periodo en el que todos los
// vecinos lo tuvieran dado de alta.
var ErrSinReferencia = errors.New("no hay consumo comunitario de referencia")

// Referencia describe el consumo comunitario estimado de un servicio: la
// media diaria de las facturas de los periodos en los que todos los vecinos
// tenían el servicio dado de alta y, por tanto, la factura de la constructora
// solo recogía consumo de zonas comunes.
type Referencia struct {
	Servicio string
	// EurosPorDia es la media ponderada por días del consumo (sin la parte
	// estructural) de las facturas de referencia.
	EurosPorDia float64
	// Desde es el primer día en que todos los vecinos tienen el servicio.
	Desde time.Time
	// Facturas son los nombres de las facturas usadas para la media.
	Facturas []string
}

// esDeReferencia indica si la factura empieza cuando ya todos los vecinos
// tenían el servicio dado de alta.
func (r Referencia) esDeReferencia(f modelo.Factura) bool {
	return !f.Ventana.Inicio().Before(r.Desde)
}

// consumoComunitario devuelve la parte del consumo de la factura atribuible
// a la comunidad:
//   - En una factura de referencia, todo el consumo es comunitario.
//   - En las demás, la media diaria de referencia por los días de la factura,
//     sin superar el consumo real de la factura.
func (r Referencia) consumoComunitario(f modelo.Factura) float64 {
	if r.esDeReferencia(f) {
		return f.Importe.Consumo
	}
	return math.Min(f.Importe.Consumo, r.EurosPorDia*float64(f.Ventana.NumDias()))
}

// referencias indexa las referencias por servicio.
type referencias map[string]Referencia

// ordenadas devuelve las referencias ordenadas por servicio.
func (r referencias) ordenadas() []Referencia {
	out := make([]Referencia, 0, len(r))
	for _, ref := range r {
		out = append(out, ref)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Servicio < out[j].Servicio })
	return out
}

// calcularReferencias calcula la referencia de cada servicio (luz, agua) del
// que haya alguna factura.
func calcularReferencias(e modelo.Escenario) (referencias, error) {
	refs := referencias{}
	for _, f := range e.Facturas {
		if !f.EsSuministro() {
			continue
		}
		if _, hecho := refs[f.Tipo]; hecho {
			continue
		}
		ref, err := nuevaReferencia(e, f.Tipo)
		if err != nil {
			return nil, err
		}
		refs[f.Tipo] = ref
	}
	return refs, nil
}

func nuevaReferencia(e modelo.Escenario, servicio string) (Referencia, error) {
	desde, ok := e.FechaAltaCompleta(servicio)
	if !ok {
		return Referencia{}, fmt.Errorf("%w de %s: algún vecino aún no ha dado de alta el servicio",
			ErrSinReferencia, servicio)
	}

	ref := Referencia{Servicio: servicio, Desde: desde}
	var consumo float64
	var dias int
	for _, f := range e.Facturas {
		// Una factura asignada íntegramente a un vecino no representa
		// consumo comunitario: se excluye de la media de referencia.
		if f.Tipo == servicio && !f.EsAsignada() && ref.esDeReferencia(f) {
			consumo += f.Importe.Consumo
			dias += f.Ventana.NumDias()
			ref.Facturas = append(ref.Facturas, f.Nombre)
		}
	}
	if dias == 0 {
		return Referencia{}, fmt.Errorf("%w de %s: ninguna factura empieza a partir de %s, cuando todos los vecinos ya tienen el servicio",
			ErrSinReferencia, servicio, desde.Format("02/01/2006"))
	}
	ref.EurosPorDia = consumo / float64(dias)
	return ref, nil
}
