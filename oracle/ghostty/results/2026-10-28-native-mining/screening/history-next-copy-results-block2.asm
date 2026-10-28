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
  serialize.go:213	0x1001ce7fc		aa1f03e5		MOVD ZR, R5						
  serialize.go:213	0x1001ce800		14000004		JMP 4(PC)						
  serialize.go:226	0x1001ce804		910004a6		ADD $1, R5, R6						
  serialize.go:226	0x1001ce808		38266807		MOVB R7, (R0)(R6)					
  serialize.go:213	0x1001ce80c		910008a5		ADD $2, R5, R5						
  serialize.go:213	0x1001ce810		d1000486		SUB $1, R4, R6						
  serialize.go:213	0x1001ce814		eb0600bf		CMP R6, R5						
  serialize.go:213	0x1001ce818		5400036a		BGE 27(PC)						
  serialize.go:214	0x1001ce81c		8b051066		ADD R5<<4, R3, R6					
  serialize.go:216	0x1001ce820		f94000c7		MOVD (R6), R7						
  serialize.go:217	0x1001ce824		cb0100e8		SUB R1, R7, R8						
  serialize.go:217	0x1001ce828		f101fd1f		CMP $127, R8						
  serialize.go:217	0x1001ce82c		54000202		BCS 16(PC)						
  serialize.go:217	0x1001ce830		f94004c8		MOVD 8(R6), R8						
  serialize.go:217	0x1001ce834		eb02011f		CMP R2, R8						
  serialize.go:217	0x1001ce838		540001a1		BNE 13(PC)						
  serialize.go:220	0x1001ce83c		38256807		MOVB R7, (R0)(R5)					
  serialize.go:222	0x1001ce840		f94008c7		MOVD 16(R6), R7						
  serialize.go:223	0x1001ce844		cb0100e8		SUB R1, R7, R8						
  serialize.go:223	0x1001ce848		f101fd1f		CMP $127, R8						
  serialize.go:223	0x1001ce84c		54000082		BCS 4(PC)						
  serialize.go:223	0x1001ce850		f9400cc6		MOVD 24(R6), R6						
  serialize.go:223	0x1001ce854		eb0200df		CMP R2, R6						
  serialize.go:223	0x1001ce858		54fffd60		BEQ -21(PC)						
  serialize.go:224	0x1001ce85c		910004a0		ADD $1, R5, R0						
  serialize.go:224	0x1001ce860		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:224	0x1001ce864		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:224	0x1001ce868		d65f03c0		RET							
  serialize.go:218	0x1001ce86c		aa0503e0		MOVD R5, R0						
  serialize.go:218	0x1001ce870		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:218	0x1001ce874		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:218	0x1001ce878		d65f03c0		RET							
  serialize.go:234	0x1001ce87c		38256807		MOVB R7, (R0)(R5)					
  serialize.go:228	0x1001ce880		910004a5		ADD $1, R5, R5						
  serialize.go:228	0x1001ce884		eb05009f		CMP R5, R4						
  serialize.go:228	0x1001ce888		540001ad		BLE 13(PC)						
  serialize.go:229	0x1001ce88c		8b051066		ADD R5<<4, R3, R6					
  serialize.go:230	0x1001ce890		f94000c7		MOVD (R6), R7						
  serialize.go:231	0x1001ce894		cb0100e8		SUB R1, R7, R8						
  serialize.go:231	0x1001ce898		f101fd1f		CMP $127, R8						
  serialize.go:231	0x1001ce89c		54000082		BCS 4(PC)						
  serialize.go:231	0x1001ce8a0		f94004c6		MOVD 8(R6), R6						
  serialize.go:231	0x1001ce8a4		eb0200df		CMP R2, R6						
  serialize.go:231	0x1001ce8a8		54fffea0		BEQ -11(PC)						
  serialize.go:232	0x1001ce8ac		aa0503e0		MOVD R5, R0						
  serialize.go:232	0x1001ce8b0		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:232	0x1001ce8b4		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:232	0x1001ce8b8		d65f03c0		RET							
  serialize.go:236	0x1001ce8bc		aa0403e0		MOVD R4, R0						
  serialize.go:236	0x1001ce8c0		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:236	0x1001ce8c4		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:236	0x1001ce8c8		d65f03c0		RET							
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
