#include "textflag.h"

// func trampoline()
TEXT ·trampoline(SB), NOSPLIT, $0
	// Pick DI because rdi is system V first arg convention.
	LEAQ frame+0(FP), DI
	MOVQ +8(DX), AX
	JMP  AX
