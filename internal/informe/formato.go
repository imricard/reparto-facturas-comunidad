package informe

import "strconv"

// formatFloat da la representación más corta y legible de v con hasta 2
// decimales (sin ceros ni punto decimal sobrantes), usando coma como
// separador decimal, como es habitual en español.
func formatFloat(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	for len(s) > 0 && s[len(s)-1] == '0' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			s = s[:i] + "," + s[i+1:]
			break
		}
	}
	return s
}
