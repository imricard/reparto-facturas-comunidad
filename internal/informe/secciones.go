package informe

import (
	"reparto/internal/modelo"
	"reparto/internal/reparto"
)

// cabecera escribe el título del informe y un breve resumen ejecutivo.
func (t *reportTemplate) cabecera(d Datos) {
	pdf := t.pdf
	pdf.SetFont(familiaFuente, "B", 18)
	pdf.CellFormat(0, 10, "Reparto de facturas de la comunidad", "", 1, "L", false, 0, "")
	pdf.SetFont(familiaFuente, "", 10)
	pdf.SetTextColor(90, 90, 90)
	pdf.CellFormat(0, 6, "Reclamación de la constructora por servicios durante la entrega de pisos", "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(2)

	pdf.SetFont(familiaFuente, "", 10)
	resumen := "Este documento detalla el reparto entre la constructora y los vecinos propietarios " +
		"de las facturas de servicios correspondientes al periodo de entrega de los pisos, " +
		"conforme a los criterios descritos en el apartado de supuestos."
	pdf.MultiCell(0, 5.5, resumen, "", "L", false)
}

// supuestos documenta, en texto explicativo, los datos de partida usados en
// el cálculo: fin de obra, coeficientes y fechas de alta de cada vecino.
func (t *reportTemplate) supuestos(d Datos) {
	t.tituloSeccion("Supuestos")
	pdf := t.pdf

	pdf.SetFont(familiaFuente, "B", 10)
	pdf.CellFormat(0, 6, "Fin de las obras: "+d.Escenario.FinObra.Format("02/01/2006"), "", 1, "L", false, 0, "")
	pdf.SetFont(familiaFuente, "", 10)
	pdf.MultiCell(0, 5.5,
		"Hasta esa fecha (incluida) todo el importe de las facturas lo asume la constructora. "+
			"A partir del día siguiente, el importe se reparte entre la constructora y los vecinos "+
			"que ya son propietarios, en la proporción que corresponda a cada factura.", "", "L", false)
	pdf.Ln(2)

	pdf.SetFont(familiaFuente, "B", 10)
	pdf.CellFormat(0, 6, "Vecinos, coeficientes y fechas de alta", "", 1, "L", false, 0, "")
	t.tablaSupuestosVecinos(d.VecinosOrdenados())
	pdf.Ln(1)

	for _, ref := range d.Resultado.Referencias {
		t.parrafoReferencia(ref)
	}
}

func (t *reportTemplate) tablaSupuestosVecinos(vecinos []modelo.Pagador) {
	pdf := t.pdf
	anchos := []float64{40, 30, 30, 30, 30}
	cabeceras := []string{"Vecino", "Coeficiente", "Compra", "Alta luz", "Alta agua"}

	pdf.SetFont(familiaFuente, "B", 9)
	pdf.SetFillColor(235, 235, 235)
	for i, c := range cabeceras {
		pdf.CellFormat(anchos[i], 7, c, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont(familiaFuente, "", 9)
	for _, v := range vecinos {
		pdf.CellFormat(anchos[0], 7, v.Nombre, "1", 0, "L", false, 0, "")
		pdf.CellFormat(anchos[1], 7, formatoPorcentaje(v.Coeficiente), "1", 0, "C", false, 0, "")
		pdf.CellFormat(anchos[2], 7, v.FechaCompra.Format("02/01/2006"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(anchos[3], 7, fechaOTexto(v, modelo.TipoLuz), "1", 0, "C", false, 0, "")
		pdf.CellFormat(anchos[4], 7, fechaOTexto(v, modelo.TipoAgua), "1", 1, "C", false, 0, "")
	}
}

// parrafoReferencia explica cómo se ha estimado el consumo comunitario de un
// servicio de suministro.
func (t *reportTemplate) parrafoReferencia(ref reparto.Referencia) {
	pdf := t.pdf
	pdf.SetFont(familiaFuente, "I", 9.5)
	texto := "Consumo comunitario de " + ref.Servicio + ": estimado en " + euros(ref.EurosPorDia) +
		"/día a partir de las facturas posteriores al " + ref.Desde.Format("02/01/2006") +
		" (cuando todos los vecinos ya tenían el servicio dado de alta): " + listaOVacio(ref.Facturas) + "."
	pdf.MultiCell(0, 5, texto, "", "L", false)
	pdf.SetFont(familiaFuente, "", 10)
}

func listaOVacio(xs []string) string {
	if len(xs) == 0 {
		return "ninguna"
	}
	out := xs[0]
	for _, x := range xs[1:] {
		out += ", " + x
	}
	return out
}

func formatoPorcentaje(v float64) string {
	return trimZeros(v) + " %"
}

func trimZeros(v float64) string {
	s := formatFloat(v)
	return s
}

// resumenPorVecino muestra la tabla resumen: el total que paga cada vecino
// (y, para que cuadre visualmente con las facturas, lo que asume la
// constructora).
func (t *reportTemplate) resumenPorVecino(d Datos) {
	t.tituloSeccion("Resumen: total a pagar por vecino")
	pdf := t.pdf

	anchoNombre, anchoImporte := 90.0, 40.0
	pdf.SetFont(familiaFuente, "B", 9)
	pdf.SetFillColor(235, 235, 235)
	pdf.CellFormat(anchoNombre, 7, "Vecino", "1", 0, "L", true, 0, "")
	pdf.CellFormat(anchoImporte, 7, "Total a pagar", "1", 1, "R", true, 0, "")

	pdf.SetFont(familiaFuente, "", 9)
	for _, v := range d.VecinosOrdenados() {
		pdf.CellFormat(anchoNombre, 7, v.Nombre, "1", 0, "L", false, 0, "")
		pdf.CellFormat(anchoImporte, 7, euros(d.TotalPorVecino[v.Nombre]), "1", 1, "R", false, 0, "")
	}
	pdf.CellFormat(anchoNombre, 7, d.Escenario.Constructora().Nombre+" (constructora)", "1", 0, "L", false, 0, "")
	pdf.CellFormat(anchoImporte, 7, euros(d.TotalPorVecino[d.Escenario.Constructora().Nombre]), "1", 1, "R", false, 0, "")

	pdf.SetFont(familiaFuente, "B", 9)
	pdf.CellFormat(anchoNombre, 7, "Total facturas", "1", 0, "L", false, 0, "")
	pdf.CellFormat(anchoImporte, 7, euros(d.TotalGeneral), "1", 1, "R", false, 0, "")
	pdf.SetFont(familiaFuente, "", 10)
}

// desglosePorFactura detalla, factura a factura, lo que paga cada pagador.
func (t *reportTemplate) desglosePorFactura(d Datos) {
	t.tituloSeccion("Desglose por factura")
	pdf := t.pdf

	for _, f := range d.Escenario.Facturas {
		pago := d.PagoDe(f.Nombre)

		pdf.SetFont(familiaFuente, "B", 10)
		pdf.CellFormat(0, 6, f.Nombre+"  ("+f.Tipo+", "+f.Ventana.String()+")", "", 1, "L", false, 0, "")
		pdf.SetFont(familiaFuente, "", 9)
		pdf.CellFormat(0, 5.5,
			"Importe: consumo "+euros(f.Importe.Consumo)+" + otros "+euros(f.Importe.Otros)+" = total "+euros(f.Importe.Total()),
			"", 1, "L", false, 0, "")
		if f.EsAsignada() {
			pdf.SetFont(familiaFuente, "I", 9)
			pdf.CellFormat(0, 5.5,
				"Factura asignada íntegramente a "+f.AsignadoA+": no se reparte con el resto de vecinos ni con la constructora.",
				"", 1, "L", false, 0, "")
			pdf.SetFont(familiaFuente, "", 9)
		}

		anchoNombre, anchoImporte := 90.0, 40.0
		pdf.SetFont(familiaFuente, "B", 9)
		pdf.SetFillColor(245, 245, 245)
		pdf.CellFormat(anchoNombre, 6.5, "Pagador", "1", 0, "L", true, 0, "")
		pdf.CellFormat(anchoImporte, 6.5, "Importe", "1", 1, "R", true, 0, "")

		pdf.SetFont(familiaFuente, "", 9)
		for _, p := range d.Escenario.Pagadores {
			etiqueta := p.Nombre
			if !p.EsVecino() {
				etiqueta += " (constructora)"
			}
			pdf.CellFormat(anchoNombre, 6.5, etiqueta, "1", 0, "L", false, 0, "")
			pdf.CellFormat(anchoImporte, 6.5, euros(pago.ImporteDe(p.Nombre)), "1", 1, "R", false, 0, "")
		}
		pdf.Ln(3)
	}
}
