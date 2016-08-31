#include "textflag.h"

// func crash(b []byte) []byte
TEXT ·crash(SB), NOSPLIT, $0-48
	MOVQ b_base+0(FP), BX
	MOVQ b_len+8(FP), AX
	ADDQ AX, BX
	MOVQ BX, ret+24(FP)
	MOVQ $0, ret+32(FP)
	MOVQ $0, ret+40(FP)

	RET
