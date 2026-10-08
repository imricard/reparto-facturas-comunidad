package modelo

import (
	"errors"
	"fmt"
	"math"
	"time"

	"reparto/internal/periodo"
)

// toleranciaCoeficientes es el margen admitido al comprobar que la suma de
// los coeficientes no supera el 100 %.
const toleranciaCoeficientes = 1e-6

// Escenario agrupa todos los datos de entrada de un reparto.
type Escenario struct {
	// FinObra es el último día en que la constructora asume todo el importe.
	FinObra   time.Time
	Facturas  []Factura
	Pagadores []Pagador
}

// Vecinos devuelve los pagadores de tipo vecino, en el orden original.
func (e Escenario) Vecinos() []Pagador {
	var vs []Pagador
	for _, p := range e.Pagadores {
		if p.EsVecino() {
			vs = append(vs, p)
		}
	}
	return vs
}

// Constructora devuelve el pagador de tipo constructora (el valor cero si no
// existe; Validar garantiza que existe exactamente uno).
func (e Escenario) Constructora() Pagador {
	for _, p := range e.Pagadores {
		if p.Tipo == TipoConstructora {
			return p
		}
	}
	return Pagador{}
}

// PrimerDiaReparto es el primer día en que los vecinos pueden pagar: el
// siguiente al fin de obra.
func (e Escenario) PrimerDiaReparto() time.Time {
	return periodo.Dia(e.FinObra).AddDate(0, 0, 1)
}

// SumaCoeficientes devuelve la suma de los coeficientes de los vecinos.
func (e Escenario) SumaCoeficientes() float64 {
	var s float64
	for _, v := range e.Vecinos() {
		s += v.Coeficiente
	}
	return s
}

// FechaAltaCompleta devuelve el primer día en que todos los vecinos tienen el
// servicio dado de alta (la mayor de sus fechas de alta). El booleano es falso
// si algún vecino aún no lo ha dado de alta, o si no hay vecinos.
func (e Escenario) FechaAltaCompleta(servicio string) (time.Time, bool) {
	var ultima time.Time
	vecinos := e.Vecinos()
	for _, v := range vecinos {
		alta := v.FechaAlta(servicio)
		if alta.IsZero() {
			return time.Time{}, false
		}
		if alta.After(ultima) {
			ultima = alta
		}
	}
	return ultima, len(vecinos) > 0
}

// Validar comprueba que el escenario es coherente y devuelve todos los
// problemas encontrados en un único error (nil si es válido).
func (e Escenario) Validar() error {
	var errs []error
	fallo := func(formato string, args ...any) { errs = append(errs, fmt.Errorf(formato, args...)) }

	if periodo.Dia(e.FinObra).IsZero() {
		fallo("falta la fecha de fin de obra")
	}
	errs = append(errs, e.validarFacturas()...)
	errs = append(errs, e.validarPagadores()...)
	return errors.Join(errs...)
}

func (e Escenario) validarFacturas() []error {
	var errs []error
	vistos := map[string]bool{}
	for i, f := range e.Facturas {
		etiqueta := fmt.Sprintf("factura #%d (%q)", i+1, f.Nombre)
		switch {
		case f.Nombre == "":
			errs = append(errs, fmt.Errorf("factura #%d: falta el nombre", i+1))
		case vistos[f.Nombre]:
			errs = append(errs, fmt.Errorf("%s: nombre duplicado", etiqueta))
		}
		vistos[f.Nombre] = true
		if f.Tipo == "" {
			errs = append(errs, fmt.Errorf("%s: falta el tipo", etiqueta))
		}
		if f.Ventana.EsCero() {
			errs = append(errs, fmt.Errorf("%s: falta la ventana temporal", etiqueta))
		}
		partes := []struct {
			nombre string
			valor  float64
		}{{"consumo", f.Importe.Consumo}, {"otros", f.Importe.Otros}}
		for _, parte := range partes {
			if v := parte.valor; v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
				errs = append(errs, fmt.Errorf("%s: el importe %q no es válido (%v)", etiqueta, parte.nombre, v))
			}
		}
	}
	return errs
}

func (e Escenario) validarPagadores() []error {
	var errs []error
	vistos := map[string]bool{}
	constructoras := 0
	for i, p := range e.Pagadores {
		etiqueta := fmt.Sprintf("pagador #%d (%q)", i+1, p.Nombre)
		switch {
		case p.Nombre == "":
			errs = append(errs, fmt.Errorf("pagador #%d: falta el nombre", i+1))
		case vistos[p.Nombre]:
			errs = append(errs, fmt.Errorf("%s: nombre duplicado", etiqueta))
		}
		vistos[p.Nombre] = true

		switch p.Tipo {
		case TipoConstructora:
			constructoras++
		case TipoVecino:
			if !(p.Coeficiente > 0 && p.Coeficiente <= 100) {
				errs = append(errs, fmt.Errorf("%s: el coeficiente debe estar en (0, 100], es %v", etiqueta, p.Coeficiente))
			}
		default:
			errs = append(errs, fmt.Errorf("%s: tipo %q desconocido (usa %q o %q)",
				etiqueta, p.Tipo, TipoVecino, TipoConstructora))
		}
	}
	if constructoras != 1 {
		errs = append(errs, fmt.Errorf("debe haber exactamente una constructora, hay %d", constructoras))
	}
	if len(e.Vecinos()) == 0 {
		errs = append(errs, errors.New("no hay ningún vecino"))
	}
	if s := e.SumaCoeficientes(); s > 100+toleranciaCoeficientes {
		errs = append(errs, fmt.Errorf("los coeficientes de los vecinos suman %.4f %%, más del 100 %%", s))
	}
	return errs
}
