// Package reparto calcula cuánto paga cada pagador de cada factura.
//
// Modelo de cálculo (todo el conocimiento del algoritmo vive en este
// paquete):
//
//  1. Hasta el fin de obra (incluido) todo lo paga la constructora; ningún
//     día anterior cuenta para ningún vecino (calculo.go: diasValidos).
//  2. Cada factura se descompone en componentes (descomponer, más abajo):
//     facturas que no son de luz/agua tienen un único componente (el
//     importe total); las de luz/agua se dividen en estructural+comunitario
//     y consumo individual.
//  3. Cada componente se reparte en forma cerrada (sin bucle día a día):
//     el estructural por coeficiente y fracción de días como propietario
//     (porPropiedad); el individual por coeficiente ponderado por los días
//     sin dar de alta, de modo que la suma entre vecinos sea exactamente el
//     importe del componente y nunca quede nada sin asignar que acabe
//     cayendo, por descuido, en la constructora (porConsumoIndividual).
//  4. Lo que ningún vecino cubre en el componente estructural (por no ser
//     aún propietario, o por estar la obra sin terminar) lo asume la
//     constructora; ver construirPago.
//  5. Los importes se redondean a céntimos sin perder ni crear ninguno
//     (paquete dinero).
package reparto

import (
	"time"

	"reparto/internal/dinero"
	"reparto/internal/modelo"
)

// Resultado es la salida del cálculo.
type Resultado struct {
	// Pagos contiene un pago por factura, en el mismo orden que la entrada.
	// Cada pago incluye una línea por pagador (también con importe 0).
	Pagos []modelo.Pago
	// Referencias documenta el consumo comunitario usado en las facturas de
	// luz y agua (vacío si no hay facturas de esos tipos).
	Referencias []Referencia
}

// Calcular valida el escenario y reparte todas las facturas.
func Calcular(e modelo.Escenario) (Resultado, error) {
	if err := e.Validar(); err != nil {
		return Resultado{}, err
	}
	refs, err := calcularReferencias(e)
	if err != nil {
		return Resultado{}, err
	}

	res := Resultado{Referencias: refs.ordenadas()}
	for _, f := range e.Facturas {
		res.Pagos = append(res.Pagos, repartirFactura(e, f, refs))
	}
	return res, nil
}

// repartirFactura reparte una factura sumando lo que corresponde a cada
// vecino en cada uno de sus componentes.
func repartirFactura(e modelo.Escenario, f modelo.Factura, refs referencias) modelo.Pago {
	vecinos := e.Vecinos()
	dias := f.Ventana.Dias()
	primerDia := e.PrimerDiaReparto()

	porVecino := map[string]float64{}
	for _, componente := range descomponer(vecinos, f, refs, dias, primerDia) {
		for nombre, importe := range componente {
			porVecino[nombre] += importe
		}
	}
	return construirPago(e, f, porVecino)
}

// construirPago añade la parte de la constructora (todo lo no asignado a
// vecinos) y redondea a céntimos.
func construirPago(e modelo.Escenario, f modelo.Factura, porVecino map[string]float64) modelo.Pago {
	total := f.Importe.Total()
	var asignado float64
	for _, v := range porVecino {
		asignado += v
	}

	brutos := make([]float64, len(e.Pagadores))
	for i, p := range e.Pagadores {
		if p.Tipo == modelo.TipoConstructora {
			brutos[i] = total - asignado
		} else {
			brutos[i] = porVecino[p.Nombre]
		}
	}

	pago := modelo.Pago{Factura: f.Nombre}
	for i, importe := range dinero.Repartir(total, brutos) {
		pago.Reparto = append(pago.Reparto, modelo.Reparto{Pagador: e.Pagadores[i].Nombre, Importe: importe})
	}
	return pago
}

// descomponer divide la factura en componentes según su tipo, cada uno ya
// repartido entre los vecinos que lo asumen (nombre -> importe):
//   - Facturas que no son de luz/agua: un único componente (importe total)
//     repartido por propiedad.
//   - Luz/agua: la parte estructural más el consumo comunitario se reparten
//     por propiedad; el consumo individual, entre los propietarios que en
//     algún momento no tuvieron el servicio dado de alta.
func descomponer(vecinos []modelo.Pagador, f modelo.Factura, refs referencias, dias []time.Time, primerDia time.Time) []map[string]float64 {
	if !f.EsSuministro() {
		return []map[string]float64{porPropiedad(f.Importe.Total(), vecinos, dias, primerDia)}
	}
	comunitario := refs[f.Tipo].consumoComunitario(f)
	return []map[string]float64{
		porPropiedad(f.Importe.Otros+comunitario, vecinos, dias, primerDia),
		porConsumoIndividual(f.Importe.Consumo-comunitario, vecinos, f.Tipo, dias, primerDia),
	}
}
