#include "textflag.h"
#include "go_asm.h"

#define	get_tls(r)	MOVQ TLS, r

// func getCore() int
TEXT ·getCore(SB), NOSPLIT, $0
	// RDTSCP
	BYTE $0x0f; BYTE $0x01; BYTE $0xf9

	// Linux puts core ID in the bottom byte.
	ANDQ $0xff, CX
	MOVQ CX, ret+0(FP)
	RET

// func getPID2() int
TEXT ·getPID2(SB), NOSPLIT, $0
	//get_tls(CX)
	//MOVQ g(CX), AX
	//MOVQ g_m(AX), CX

	//MOVQ CX, ret+0(FP)
