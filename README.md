# reparto

Programa en Go que calcula cómo repartir, entre la constructora y los
vecinos, las facturas de servicios (luz, agua y otros) reclamadas durante el
periodo de entrega de los pisos, y genera un informe en PDF con el resultado.

## Compilar y ejecutar

```bash
go build -o reparto ./cmd/reparto
./reparto -entrada escenario.yaml -salida informe.pdf
```

La entrada puede ser un fichero `.json`, `.yaml` o `.yml` (ver
`test/fixtures/escenario_valido.yaml` para un ejemplo completo con el
formato de fechas `dd/mm/aaaa`).

## Estructura del código

- `internal/periodo`  — intervalos de días naturales (ventanas de factura).
- `internal/modelo`   — datos del problema (Factura, Pagador, Pago) y las
  reglas que dependen solo de esos datos (propiedad, altas de servicio,
  validación del escenario).
- `internal/reparto`  — el algoritmo de reparto en sí: cómo se descompone
  cada factura (estructural + comunitario / consumo individual), cómo se
  reparte cada componente en forma cerrada entre los vecinos (`calculo.go`),
  y cómo se estima el consumo comunitario de referencia (`referencia.go`).
- `internal/dinero`   — redondeo a céntimos que conserva el total exacto
  (método del resto mayor).
- `internal/config`   — lectura de JSON/YAML y su traducción a `modelo.Escenario`.
- `internal/informe`  — la plantilla `report-template`: genera el PDF final
  (resumen, supuestos y desglose) a partir del resultado ya calculado.
- `cmd/reparto`        — el ejecutable, que une los paquetes anteriores.
- `test/e2e`           — tests de extremo a extremo, incluidos los exigidos
  en el enunciado (suma de facturas = suma de pagos; mayor coeficiente y alta
  posterior implica pagar más; y el caso inverso).

## Lógica de reparto (resumen)

1. Hasta la fecha de fin de obra (incluida), todo el importe lo asume la
   constructora; ningún día anterior cuenta para ningún vecino.
2. Cada factura se descompone en componentes. Las que no son de luz/agua
   tienen un único componente (el importe total). Las de luz/agua se dividen
   en: estructural (`otros` + una estimación del consumo comunitario) y
   consumo individual (el resto del consumo).
3. **Componente estructural** (`porPropiedad`): cada vecino paga
   `importe × coeficiente × (días_como_propietario / días_totales_factura)`.
   Lo que ningún vecino cubre (por no ser aún propietario, o por estar la
   obra sin terminar) lo asume la constructora.
4. **Consumo individual** (`porConsumoIndividual`): se reparte solo entre los
   vecinos que, en algún momento de la ventana (ya con la obra terminada),
   fueron propietarios sin tener el servicio dado de alta. El peso de cada
   uno es `coeficiente × días_sin_dar_de_alta`, normalizado entre todos los
   rezagados para que la suma sea exactamente el importe del componente —
   así, quien tarda más en darse de alta asume una porción mayor y, si solo
   hay un rezagado en toda la factura, asume el importe íntegro, sin que
   nada quede sin asignar (y por tanto sin que nada caiga, por defecto, en
   la constructora). Si nadie estuvo nunca sin alta, se reparte igual que el
   componente estructural.
5. El consumo comunitario de referencia se estima como la media €/día de las
   facturas cuya ventana empieza en o después de la fecha en que todos los
   vecinos ya tenían el servicio dado de alta.
6. Los importes finales se redondean a céntimos sin que la suma total se
   desvíe del importe real de las facturas (`internal/dinero`).

## Tests

```bash
go test ./...
```

Cobertura por paquete: `modelo` 95%, `reparto` 98%, `dinero` 89%, `periodo`
89%, `config` 74%, `informe` 89% (generación real de PDF, verificada con
`pdftotext`).
