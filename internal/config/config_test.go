package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLeerJSONYYAMLDanElMismoEscenario(t *testing.T) {
	json, err := Leer(filepath.Join("..", "..", "test", "fixtures", "escenario_valido.json"))
	if err != nil {
		t.Fatal(err)
	}
	yaml, err := Leer(filepath.Join("..", "..", "test", "fixtures", "escenario_valido.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !json.FinObra.Equal(yaml.FinObra) || len(json.Facturas) != len(yaml.Facturas) || len(json.Pagadores) != len(yaml.Pagadores) {
		t.Fatalf("JSON y YAML deben producir el mismo escenario:\nJSON=%+v\nYAML=%+v", json, yaml)
	}
	if err := json.Validar(); err != nil {
		t.Fatalf("el escenario leído no es válido: %v", err)
	}
}

func TestLeerFechasYCampos(t *testing.T) {
	e, err := Leer(filepath.Join("..", "..", "test", "fixtures", "escenario_valido.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !e.FinObra.Equal(time.Date(2025, 6, 12, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("FinObra = %v", e.FinObra)
	}
	ana := e.Vecinos()[0]
	if ana.Nombre != "Ana" || ana.Coeficiente != 60 {
		t.Fatalf("Ana mal leída: %+v", ana)
	}
	if got := e.Facturas[1].Ventana.NumDias(); got != 31 {
		t.Fatalf("Luz-julio debería tener 31 días, tiene %d", got)
	}
}

func TestLeerExtensionNoSoportada(t *testing.T) {
	if _, err := Leer("escenario.txt"); err == nil {
		t.Fatal("quería un error por extensión no soportada")
	}
}

func TestLeerFicheroInexistente(t *testing.T) {
	if _, err := Leer("no-existe.json"); err == nil {
		t.Fatal("quería un error por fichero inexistente")
	}
}

func TestLeerAsignadoA(t *testing.T) {
	e, err := Leer(filepath.Join("..", "..", "test", "fixtures", "escenario_valido.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var encontrada bool
	for _, f := range e.Facturas {
		if f.Nombre == "Residus-piso-Bruno" {
			encontrada = true
			if f.AsignadoA != "Bruno" {
				t.Fatalf("AsignadoA = %q, quería \"Bruno\"", f.AsignadoA)
			}
			if !f.EsAsignada() {
				t.Fatal("la factura debería considerarse asignada")
			}
		}
	}
	if !encontrada {
		t.Fatal("no se encontró la factura Residus-piso-Bruno")
	}
	// El resto de facturas, sin el campo, no deben quedar marcadas como
	// asignadas por error.
	for _, f := range e.Facturas {
		if f.Nombre != "Residus-piso-Bruno" && f.EsAsignada() {
			t.Fatalf("%s no debería estar asignada: AsignadoA=%q", f.Nombre, f.AsignadoA)
		}
	}
	if err := e.Validar(); err != nil {
		t.Fatalf("el escenario con asignado-a debería seguir siendo válido: %v", err)
	}
}
