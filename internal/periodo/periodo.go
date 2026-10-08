// Package periodo modela intervalos de días naturales.
//
// Todo el programa razona en días completos (no en horas), así que este
// paquete es el único que sabe cómo se normalizan las fechas y cómo se
// cuentan los días de un intervalo. El resto de paquetes solo pide
// "los días de esta ventana" o "cuántos días tiene".
package periodo

import (
	"errors"
	"fmt"
	"time"
)

// ErrPeriodoInvalido se devuelve al construir un periodo sin fechas o con el
// fin anterior al inicio.
var ErrPeriodoInvalido = errors.New("periodo inválido")

// Dia normaliza t a las 00:00 UTC de su día natural. Sirve para comparar
// fechas sin que influyan la hora ni la zona horaria. La fecha cero se
// conserva como fecha cero.
func Dia(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// Periodo es un intervalo de días naturales cerrado: incluye tanto el día de
// inicio como el de fin. Su valor cero representa "sin periodo".
type Periodo struct {
	inicio, fin time.Time
}

// Nuevo crea un periodo [inicio, fin] (ambos incluidos).
func Nuevo(inicio, fin time.Time) (Periodo, error) {
	i, f := Dia(inicio), Dia(fin)
	switch {
	case i.IsZero() || f.IsZero():
		return Periodo{}, fmt.Errorf("%w: faltan el inicio o el fin", ErrPeriodoInvalido)
	case f.Before(i):
		return Periodo{}, fmt.Errorf("%w: el fin (%s) es anterior al inicio (%s)",
			ErrPeriodoInvalido, f.Format("02/01/2006"), i.Format("02/01/2006"))
	}
	return Periodo{inicio: i, fin: f}, nil
}

// EsCero indica si el periodo no ha sido inicializado.
func (p Periodo) EsCero() bool { return p.inicio.IsZero() }

// Inicio devuelve el primer día del periodo.
func (p Periodo) Inicio() time.Time { return p.inicio }

// Fin devuelve el último día del periodo (incluido).
func (p Periodo) Fin() time.Time { return p.fin }

// NumDias devuelve el número de días naturales del periodo.
func (p Periodo) NumDias() int {
	if p.EsCero() {
		return 0
	}
	return int(p.fin.Sub(p.inicio).Hours()/24) + 1
}

// Dias devuelve, en orden, cada día del periodo.
func (p Periodo) Dias() []time.Time {
	dias := make([]time.Time, 0, p.NumDias())
	for d := p.inicio; !p.EsCero() && !d.After(p.fin); d = d.AddDate(0, 0, 1) {
		dias = append(dias, d)
	}
	return dias
}

// String muestra el periodo como "dd/mm/aaaa - dd/mm/aaaa".
func (p Periodo) String() string {
	return p.inicio.Format("02/01/2006") + " - " + p.fin.Format("02/01/2006")
}
