TEXT github.com/2018kguo/phosphor.copyASCIICells(SB) /Users/kevinguo/repos/.phosphor-gap-5724_o1l/history-next-copy/serialize.go
  serialize.go:204	0x1001ce7c0		f9400b90		MOVD 16(R28), R16					
  serialize.go:204	0x1001ce7c4		eb3063ff		CMP R16, RSP						
  serialize.go:204	0x1001ce7c8		54000889		BLS 68(PC)						
  serialize.go:204	0x1001ce7cc		f81f0ffe		MOVD.W R30, -16(RSP)					
  serialize.go:204	0x1001ce7d0		f81f83fd		MOVD R29, -8(RSP)					
  serialize.go:204	0x1001ce7d4		d10023fd		SUB $8, RSP, R29					
  serialize.go:204	0x1001ce7d8		f9000fe0		MOVD R0, 24(RSP)					
  serialize.go:204	0x1001ce7dc		f9001be3		MOVD R3, 48(RSP)					
  serialize.go:205	0x1001ce7e0		eb04005f		CMP R4, R2						
  serialize.go:205	0x1001ce7e4		54000763		BCC 59(PC)						
  serialize.go:206	0x1001ce7e8		b4000724		CBZ R4, 57(PC)						
  serialize.go:206	0x1001ce7ec		b9400461		MOVWU 4(R3), R1						
  serialize.go:211	0x1001ce7f0		d3607c21		LSL $32, R1, R1						
  serialize.go:207	0x1001ce7f4		f9400462		MOVD 8(R3), R2						
  serialize.go:211	0x1001ce7f8		b2400021		ORR $1, R1, R1						
  serialize.go:213	0x1001ce7fc		d1000485		SUB $1, R4, R5						
  serialize.go:214	0x1001ce800		aa1f03e6		MOVD ZR, R6						
  serialize.go:214	0x1001ce804		14000004		JMP 4(PC)						
  serialize.go:227	0x1001ce808		910004c7		ADD $1, R6, R7						
  serialize.go:227	0x1001ce80c		38276808		MOVB R8, (R0)(R7)					
  serialize.go:214	0x1001ce810		910008c6		ADD $2, R6, R6						
  serialize.go:214	0x1001ce814		eb0500df		CMP R5, R6						
  serialize.go:214	0x1001ce818		5400036a		BGE 27(PC)						
  serialize.go:215	0x1001ce81c		8b061067		ADD R6<<4, R3, R7					
  serialize.go:217	0x1001ce820		f94000e8		MOVD (R7), R8						
  serialize.go:218	0x1001ce824		cb010109		SUB R1, R8, R9						
  serialize.go:218	0x1001ce828		f101fd3f		CMP $127, R9						
  serialize.go:218	0x1001ce82c		54000202		BCS 16(PC)						
  serialize.go:218	0x1001ce830		f94004e9		MOVD 8(R7), R9						
  serialize.go:218	0x1001ce834		eb02013f		CMP R2, R9						
  serialize.go:218	0x1001ce838		540001a1		BNE 13(PC)						
  serialize.go:221	0x1001ce83c		38266808		MOVB R8, (R0)(R6)					
  serialize.go:223	0x1001ce840		f94008e8		MOVD 16(R7), R8						
  serialize.go:224	0x1001ce844		cb010109		SUB R1, R8, R9						
  serialize.go:224	0x1001ce848		f101fd3f		CMP $127, R9						
  serialize.go:224	0x1001ce84c		54000082		BCS 4(PC)						
  serialize.go:224	0x1001ce850		f9400ce7		MOVD 24(R7), R7						
  serialize.go:224	0x1001ce854		eb0200ff		CMP R2, R7						
  serialize.go:224	0x1001ce858		54fffd80		BEQ -20(PC)						
  serialize.go:225	0x1001ce85c		910004c0		ADD $1, R6, R0						
  serialize.go:225	0x1001ce860		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:225	0x1001ce864		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:225	0x1001ce868		d65f03c0		RET							
  serialize.go:219	0x1001ce86c		aa0603e0		MOVD R6, R0						
  serialize.go:219	0x1001ce870		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:219	0x1001ce874		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:219	0x1001ce878		d65f03c0		RET							
  serialize.go:235	0x1001ce87c		38266807		MOVB R7, (R0)(R6)					
  serialize.go:229	0x1001ce880		910004c6		ADD $1, R6, R6						
  serialize.go:229	0x1001ce884		eb06009f		CMP R6, R4						
  serialize.go:229	0x1001ce888		540001ad		BLE 13(PC)						
  serialize.go:230	0x1001ce88c		8b061065		ADD R6<<4, R3, R5					
  serialize.go:231	0x1001ce890		f94000a7		MOVD (R5), R7						
  serialize.go:232	0x1001ce894		cb0100e8		SUB R1, R7, R8						
  serialize.go:232	0x1001ce898		f101fd1f		CMP $127, R8						
  serialize.go:232	0x1001ce89c		54000082		BCS 4(PC)						
  serialize.go:232	0x1001ce8a0		f94004a5		MOVD 8(R5), R5						
  serialize.go:232	0x1001ce8a4		eb0200bf		CMP R2, R5						
  serialize.go:232	0x1001ce8a8		54fffea0		BEQ -11(PC)						
  serialize.go:233	0x1001ce8ac		aa0603e0		MOVD R6, R0						
  serialize.go:233	0x1001ce8b0		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:233	0x1001ce8b4		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:233	0x1001ce8b8		d65f03c0		RET							
  serialize.go:237	0x1001ce8bc		aa0403e0		MOVD R4, R0						
  serialize.go:237	0x1001ce8c0		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:237	0x1001ce8c4		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:237	0x1001ce8c8		d65f03c0		RET							
  serialize.go:206	0x1001ce8cc		97fad595		CALL runtime.panicBounds(SB)				
  serialize.go:205	0x1001ce8d0		97fad594		CALL runtime.panicBounds(SB)				
  serialize.go:205	0x1001ce8d4		d503201f		NOOP							
  serialize.go:204	0x1001ce8d8		a90087e0		STP (R0, R1), 8(RSP)					
  serialize.go:204	0x1001ce8dc		a9018fe2		STP (R2, R3), 24(RSP)					
  serialize.go:204	0x1001ce8e0		a90297e4		STP (R4, R5), 40(RSP)					
  serialize.go:204	0x1001ce8e4		aa1e03e3		MOVD R30, R3						
  serialize.go:204	0x1001ce8e8		97facd0e		CALL runtime.morestack_noctxt.abi0(SB)			
  serialize.go:204	0x1001ce8ec		a94087e0		LDP 8(RSP), (R0, R1)					
  serialize.go:204	0x1001ce8f0		a9418fe2		LDP 24(RSP), (R2, R3)					
  serialize.go:204	0x1001ce8f4		a94297e4		LDP 40(RSP), (R4, R5)					
  serialize.go:204	0x1001ce8f8		17ffffb2		JMP github.com/2018kguo/phosphor.copyASCIICells(SB)	
  serialize.go:204	0x1001ce8fc		00000000		?							
