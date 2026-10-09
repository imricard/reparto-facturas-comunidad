// Package config traduce un fichero de entrada (JSON o YAML) al modelo.Escenario
// que entiende el resto del programa. Es el único paquete que conoce el
// formato del fichero de configuración; el resto del programa solo ve un
// modelo.Escenario.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"

	"reparto/internal/modelo"
	"reparto/internal/periodo"
)

// layoutFecha es el formato admitido para las fechas del fichero: dd/mm/aaaa.
const layoutFecha = "02/01/2006"

// fecha es un time.Time que se (de)serializa como "dd/mm/aaaa" tanto en JSON
// como en YAML, y admite estar vacío (fecha cero) cuando el campo no aplica.
type fecha time.Time

func (f fecha) tiempo() time.Time { return time.Time(f) }

func (f *fecha) UnmarshalJSON(b []byte) error { return f.unmarshal(strings.Trim(string(b), `"`)) }
func (f fecha) MarshalJSON() ([]byte, error)  { return []byte(`"` + f.marshal() + `"`), nil }

func (f *fecha) UnmarshalYAML(b []byte) error { return f.unmarshal(strings.Trim(string(b), `"'`)) }
func (f fecha) MarshalYAML() ([]byte, error)  { return []byte(f.marshal()), nil }

func (f *fecha) unmarshal(s string) error {
	if s == "" || s == "null" {
		*f = fecha{}
		return nil
	}
	t, err := time.Parse(layoutFecha, s)
	if err != nil {
		return fmt.Errorf("fecha %q inválida, se espera dd/mm/aaaa: %w", s, err)
	}
	*f = fecha(t)
	return nil
}

func (f fecha) marshal() string {
	if f.tiempo().IsZero() {
		return ""
	}
	return f.tiempo().Format(layoutFecha)
}

// ventana es el par (desde, hasta) de una factura tal como aparece en el
// fichero de entrada.
type ventana struct {
	Desde fecha `json:"desde" yaml:"desde"`
	Hasta fecha `json:"hasta" yaml:"hasta"`
}

type factura struct {
	Nombre  string  `json:"nombre" yaml:"nombre"`
	Tipo    string  `json:"tipo" yaml:"tipo"`
	Consumo float64 `json:"consumo" yaml:"consumo"`
	Otros   float64 `json:"otros" yaml:"otros"`
	Ventana ventana `json:"ventana-temporal" yaml:"ventana-temporal"`
	// AsignadoA es opcional: el nombre de un vecino que asume él solo el
	// importe íntegro de esta factura, sin repartirla con el resto.
	AsignadoA string `json:"asignado-a,omitempty" yaml:"asignado-a,omitempty"`
}

type pagador struct {
	Nombre        string  `json:"nombre" yaml:"nombre"`
	Tipo          string  `json:"tipo" yaml:"tipo"`
	FechaCompra   fecha   `json:"fecha-compra" yaml:"fecha-compra"`
	FechaAltaLuz  fecha   `json:"fecha-alta-luz" yaml:"fecha-alta-luz"`
	FechaAltaAgua fecha   `json:"fecha-alta-agua" yaml:"fecha-alta-agua"`
	Coeficiente   float64 `json:"coeficiente" yaml:"coeficiente"`
}

// documento es la forma exacta del fichero de entrada.
type documento struct {
	FinObra   fecha     `json:"fin-obra" yaml:"fin-obra"`
	Facturas  []factura `json:"facturas" yaml:"facturas"`
	Pagadores []pagador `json:"pagadores" yaml:"pagadores"`
}

// Leer carga un escenario desde un fichero .json, .yaml o .yml. La extensión
// decide el formato; cualquier otra extensión es un error.
func Leer(ruta string) (modelo.Escenario, error) {
	datos, err := os.ReadFile(ruta)
	if err != nil {
		return modelo.Escenario{}, fmt.Errorf("leyendo %s: %w", ruta, err)
	}
	doc, err := decodificar(ruta, datos)
	if err != nil {
		return modelo.Escenario{}, fmt.Errorf("interpretando %s: %w", ruta, err)
	}
	return aEscenario(doc)
}

func decodificar(ruta string, datos []byte) (documento, error) {
	var doc documento
	var err error
	switch ext := strings.ToLower(filepath.Ext(ruta)); ext {
	case ".json":
		err = jsonUnmarshalEstricto(datos, &doc)
	case ".yaml", ".yml":
		err = yaml.Unmarshal(datos, &doc)
	default:
		err = fmt.Errorf("extensión %q no soportada (usa .json, .yaml o .yml)", ext)
	}
	return doc, err
}

// aEscenario convierte el documento leído en un modelo.Escenario, sin
// realizar más validación semántica que la necesaria para construir las
// ventanas temporales; el resto lo hace modelo.Escenario.Validar.
func aEscenario(doc documento) (modelo.Escenario, error) {
	e := modelo.Escenario{FinObra: doc.FinObra.tiempo()}

	for _, f := range doc.Facturas {
		p, err := periodo.Nuevo(f.Ventana.Desde.tiempo(), f.Ventana.Hasta.tiempo())
		if err != nil {
			return modelo.Escenario{}, fmt.Errorf("factura %q: %w", f.Nombre, err)
		}
		e.Facturas = append(e.Facturas, modelo.Factura{
			Nombre:    f.Nombre,
			Tipo:      f.Tipo,
			Importe:   modelo.Importe{Consumo: f.Consumo, Otros: f.Otros},
			Ventana:   p,
			AsignadoA: f.AsignadoA,
		})
	}

	for _, p := range doc.Pagadores {
		e.Pagadores = append(e.Pagadores, modelo.Pagador{
			Nombre:        p.Nombre,
			Tipo:          p.Tipo,
			FechaCompra:   p.FechaCompra.tiempo(),
			FechaAltaLuz:  p.FechaAltaLuz.tiempo(),
			FechaAltaAgua: p.FechaAltaAgua.tiempo(),
			Coeficiente:   p.Coeficiente,
		})
	}
	return e, nil
}
