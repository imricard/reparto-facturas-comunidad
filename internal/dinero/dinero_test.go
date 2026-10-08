package dinero

import (
	"math"
	"math/rand"
	"testing"
)

func sumaCentimos(v []float64) int64 {
	var s int64
	for _, x := range v {
		s += int64(math.Round(x * 100))
	}
	return s
}

func TestRepartirTresPartesIguales(t *testing.T) {
	got := Repartir(100, []float64{100.0 / 3, 100.0 / 3, 100.0 / 3})
	if sumaCentimos(got) != 10000 {
		t.Fatalf("la suma no es 100,00: %v", got)
	}
	for _, x := range got {
		if math.Abs(x-33.33) > 0.011 {
			t.Fatalf("importe fuera de rango: %v", got)
		}
	}
}

func TestRepartirCeroYVacio(t *testing.T) {
	if got := Repartir(0, nil); got != nil {
		t.Fatalf("Repartir(0, nil) = %v", got)
	}
	got := Repartir(0, []float64{0, 0})
	if got[0] != 0 || got[1] != 0 {
		t.Fatalf("Repartir(0, ceros) = %v", got)
	}
}

func TestRepartirTrataNegativosComoCero(t *testing.T) {
	got := Repartir(10, []float64{10, -1e-12})
	if got[0] != 10 || got[1] != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestRepartirPropiedadSumaExacta(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for iter := 0; iter < 2000; iter++ {
		n := 1 + rng.Intn(8)
		pesos := make([]float64, n)
		var sp float64
		for i := range pesos {
			pesos[i] = rng.Float64()
			sp += pesos[i]
		}
		total := math.Round(rng.Float64()*100000) / 100
		brutos := make([]float64, n)
		for i := range brutos {
			brutos[i] = total * pesos[i] / sp
		}
		got := Repartir(total, brutos)
		if want := int64(math.Round(total * 100)); sumaCentimos(got) != want {
			t.Fatalf("iter %d: suma %d céntimos, quería %d (total %v, brutos %v, got %v)",
				iter, sumaCentimos(got), want, total, brutos, got)
		}
		for i := range got {
			if math.Abs(got[i]-brutos[i]) > 0.0101 {
				t.Fatalf("iter %d: el redondeo se aleja más de un céntimo: %v vs %v", iter, got[i], brutos[i])
			}
		}
	}
}
