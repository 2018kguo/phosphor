//go:build !arm64

package phosphor

const asciiVector = false

func fillASCII8(dst []cell, src []byte, pen cell) { fillASCIICellsPlain(dst, src, pen) }
