package phosphor

import "unsafe"

const asciiVector = true

// The assembly writes four 32-bit fields. A changed layout must fail to build.
var (
	_ [16]byte = [unsafe.Sizeof(cell{})]byte{}
	_ [4]byte  = [unsafe.Offsetof(cell{}.Fg)]byte{}
	_ [8]byte  = [unsafe.Offsetof(cell{}.Bg)]byte{}
	_ [12]byte = [unsafe.Offsetof(cell{}.A)]byte{}
	_ [14]byte = [unsafe.Offsetof(cell{}.E)]byte{}
)

//go:noescape
func fillASCII8(dst []cell, src []byte, pen cell)
