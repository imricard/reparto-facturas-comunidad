package reparto

import (
	"time"

	"reparto/internal/modelo"
)

// diasValidos cuenta, de entre dias, cuántos son posteriores (o iguales) al
// fin de obra. Antes de esa fecha nunca hay nada que asignar a los vecinos:
// la constructora lo asume todo.
func diasValidos(dias []time.Time, primerDia time.Time) []time.Time {
	out := make([]time.Time, 0, len(dias))
	for _, d := range dias {
		if !d.Before(primerDia) {
			out = append(out, d)
		}
	}
	return out
}

// diasPropietario cuenta cuántos de esos días v ya es propietario.
func diasPropietario(v modelo.Pagador, dias []time.Time) int {
	n := 0
	for _, d := range dias {
		if v.EsPropietario(d) {
			n++
		}
	}
	return n
}

// diasSinAlta cuenta cuántos de esos días v es propietario pero todavía no
// tiene el servicio dado de alta: son los días en los que su consumo
// individual (no comunitario) queda sin cubrir por su propia alta.
func diasSinAlta(v modelo.Pagador, servicio string, dias []time.Time) int {
	n := 0
	for _, d := range dias {
		if v.EsPropietario(d) && !v.TieneAlta(servicio, d) {
			n++
		}
	}
	return n
}

// porPropiedad reparte importe entre los vecinos según su coeficiente y la
// fracción de la ventana (tras el fin de obra) en que cada uno fue
// propietario. Lo que ningún vecino cubre (porque aún no había comprado, o
// porque la obra no había terminado) queda sin asignar y lo asume la
// constructora.
func porPropiedad(importe float64, vecinos []modelo.Pagador, dias []time.Time, primerDia time.Time) map[string]float64 {
	total := float64(len(dias))
	validos := diasValidos(dias, primerDia)
	out := map[string]float64{}
	for _, v := range vecinos {
		p := diasPropietario(v, validos)
		if p == 0 {
			continue
		}
		out[v.Nombre] = importe * (v.Coeficiente / 100) * (float64(p) / total)
	}
	return out
}

// porConsumoIndividual reparte importe (la parte de consumo que no es
// comunitaria) entre los vecinos que, en algún momento de la ventana (ya con
// la obra terminada y siendo propietarios), tuvieron el servicio sin dar de
// alta. El peso de cada uno es su coeficiente multiplicado por el número de
// días que estuvo sin alta, normalizado para que la suma sea exactamente
// importe: así, quien tarda más en darse de alta asume una porción mayor, y
// si solo hay un rezagado, asume el importe íntegro, sin que nada quede sin
// asignar.
//
// Si nadie estuvo nunca sin alta durante esos días (p. ej. porque la
// estimación de consumo comunitario de referencia no cubre del todo el
// consumo real), no hay a quién atribuir el exceso: se reparte igual que el
// resto de la factura, por propiedad (porPropiedad), para que tampoco quede
// nada sin asignar en la constructora pudiendo repartirse entre vecinos.
func porConsumoIndividual(importe float64, vecinos []modelo.Pagador, servicio string, dias []time.Time, primerDia time.Time) map[string]float64 {
	validos := diasValidos(dias, primerDia)

	pesos := map[string]float64{}
	var pesoTotal float64
	for _, v := range vecinos {
		u := diasSinAlta(v, servicio, validos)
		if u == 0 {
			continue
		}
		peso := v.Coeficiente * float64(u)
		pesos[v.Nombre] = peso
		pesoTotal += peso
	}
	if pesoTotal == 0 {
		return porPropiedad(importe, vecinos, dias, primerDia)
	}

	out := make(map[string]float64, len(pesos))
	for nombre, peso := range pesos {
		out[nombre] = importe * peso / pesoTotal
	}
	return out
}
