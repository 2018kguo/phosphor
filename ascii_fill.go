package phosphor

func fillASCIICellsPlain(dst []cell, src []byte, pen cell) {
	for i, b := range src {
		c := &dst[i]
		c.R, c.Fg, c.Bg, c.A, c.E = rune(b), pen.Fg, pen.Bg, pen.A, pen.E
	}
}
