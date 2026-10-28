#include "textflag.h"

// Widen eight bytes into runes and interleave Rune, Fg, Bg and Attr/Link words.
TEXT ·fillASCII8(SB), NOSPLIT, $0-64
 MOVD dst_base+0(FP), R0
 MOVD src_base+24(FP), R1
 MOVD src_len+32(FP), R2
 MOVD $pen+48(FP), R8
 MOVWU 4(R8), R3
 MOVWU 8(R8), R4
 MOVWU 12(R8), R5
 VDUP R3, V1.S4
 VDUP R4, V2.S4
 VDUP R5, V3.S4
 LSL $32, R5, R7
 ORR R4, R7, R7
 CMP $8, R2
 BLT tail
bulk:
 VLD1.P 8(R1), [V4.B8]
 VUSHLL $0, V4.B8, V4.H8
 VUSHLL $0, V4.H4, V0.S4
 VST4.P [V0.S4, V1.S4, V2.S4, V3.S4], 64(R0)
 VUSHLL2 $0, V4.H8, V0.S4
 VST4.P [V0.S4, V1.S4, V2.S4, V3.S4], 64(R0)
 SUB $8, R2
 CMP $8, R2
 BGE bulk
tail:
 CBZ R2, done
 MOVBU.P 1(R1), R6
 STPW (R6, R3), (R0)
 MOVD R7, 8(R0)
 ADD $16, R0
 SUB $1, R2
 B tail
done:
 RET
