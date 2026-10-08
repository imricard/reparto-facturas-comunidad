package periodo

import (
	"errors"
	"testing"
	"time"
)

func fecha(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestDiaIgnoraHoraYZona(t *testing.T) {
	madrid := time.FixedZone("CEST", 2*3600)
	got := Dia(time.Date(2025, 6, 12, 23, 59, 0, 0, madrid))
	if !got.Equal(fecha(2025, 6, 12)) {
		t.Fatalf("Dia() = %v, quería 2025-06-12 UTC", got)
	}
}

func TestNuevoValidaLimites(t *testing.T) {
	casos := []struct {
		nombre      string
		inicio, fin time.Time
		ok          bool
	}{
		{"un solo día", fecha(2025, 6, 1), fecha(2025, 6, 1), true},
		{"rango normal", fecha(2025, 6, 1), fecha(2025, 6, 30), true},
		{"fin anterior", fecha(2025, 6, 2), fecha(2025, 6, 1), false},
		{"sin inicio", time.Time{}, fecha(2025, 6, 1), false},
		{"sin fin", fecha(2025, 6, 1), time.Time{}, false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := Nuevo(c.inicio, c.fin)
			if (err == nil) != c.ok {
				t.Fatalf("err = %v, ok esperado = %v", err, c.ok)
			}
			if err != nil && !errors.Is(err, ErrPeriodoInvalido) {
				t.Fatalf("el error no envuelve ErrPeriodoInvalido: %v", err)
			}
		})
	}
}

func TestNumDiasYDiasSonInclusivos(t *testing.T) {
	p, _ := Nuevo(fecha(2025, 2, 27), fecha(2025, 3, 2)) // 27, 28, 1, 2
	if p.NumDias() != 4 {
		t.Fatalf("NumDias = %d, quería 4", p.NumDias())
	}
	dias := p.Dias()
	if len(dias) != 4 || !dias[0].Equal(fecha(2025, 2, 27)) || !dias[3].Equal(fecha(2025, 3, 2)) {
		t.Fatalf("Dias() inesperado: %v", dias)
	}
}

func TestPeriodoCero(t *testing.T) {
	var p Periodo
	if !p.EsCero() || p.NumDias() != 0 || len(p.Dias()) != 0 {
		t.Fatalf("el periodo cero debe estar vacío: %+v", p)
	}
}

func TestString(t *testing.T) {
	p, _ := Nuevo(fecha(2025, 6, 1), fecha(2025, 6, 30))
	if got := p.String(); got != "01/06/2025 - 30/06/2025" {
		t.Fatalf("String() = %q", got)
	}
}
