// Package dinero encapsula cómo se redondean los importes en euros.
//
// El resto del programa calcula con float64 sin preocuparse de los céntimos;
// solo este paquete sabe que el dinero se paga en múltiplos de 0,01 € y cómo
// evitar que el redondeo haga desaparecer o aparecer céntimos.
package dinero

import (
	"math"
	"sort"
)

// Repartir convierte los importes brutos (en euros, con decimales
// arbitrarios) en importes de 2 decimales cuya suma es exactamente el total
// redondeado a céntimos. Usa el método del resto mayor: cada importe se trunca
// a céntimos y los céntimos sobrantes se asignan a quienes tenían mayor resto.
//
// Los importes negativos (errores de coma flotante) se tratan como cero. El
// resultado conserva el orden de la entrada.
func Repartir(total float64, brutos []float64) []float64 {
	n := len(brutos)
	if n == 0 {
		return nil
	}
	centimos := make([]int64, n)
	resto := make([]float64, n)
	var suma int64
	for i, b := range brutos {
		v := math.Max(b, 0) * 100
		entero := math.Floor(v)
		centimos[i] = int64(entero)
		resto[i] = v - entero
		suma += centimos[i]
	}

	orden := make([]int, n)
	for i := range orden {
		orden[i] = i
	}
	sort.SliceStable(orden, func(a, b int) bool { return resto[orden[a]] > resto[orden[b]] })

	falta := int64(math.Round(total*100)) - suma
	for k := 0; falta > 0; k, falta = k+1, falta-1 {
		centimos[orden[k%n]]++
	}
	// Caso defensivo: si por coma flotante sobrasen céntimos, se quitan a
	// quienes tienen menor resto.
	for k := 0; falta < 0 && k < 10*n; k++ {
		if i := orden[n-1-k%n]; centimos[i] > 0 {
			centimos[i]--
			falta++
		}
	}

	out := make([]float64, n)
	for i, c := range centimos {
		out[i] = float64(c) / 100
	}
	return out
}
