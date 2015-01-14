#include "textflag.h"

// func Add2(n, m int64) int64
TEXT ·Add2(SB),NOSPLIT,$0
    MOVQ  n+0(FP),AX
    MOVQ  m+8(FP),BX
    ADDQ  AX,BX
    MOVQ  BX,ret+16(FP)
    RET

// func BSF(n int64) int
TEXT ·BSF(SB),NOSPLIT,$0
    BSFQ  n+0(FP),AX
    JEQ   allZero
    MOVQ  AX,ret+8(FP)
    RET
allZero:
    MOVQ  $-1,ret+8(FP)
    RET

// vim: set ft=txt:
