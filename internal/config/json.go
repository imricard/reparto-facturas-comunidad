package config

import (
	"bytes"
	"encoding/json"
)

// jsonUnmarshalEstricto decodifica JSON rechazando campos desconocidos, para
// detectar errores tipográficos en el fichero de entrada cuanto antes.
func jsonUnmarshalEstricto(datos []byte, doc *documento) error {
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.DisallowUnknownFields()
	return dec.Decode(doc)
}
