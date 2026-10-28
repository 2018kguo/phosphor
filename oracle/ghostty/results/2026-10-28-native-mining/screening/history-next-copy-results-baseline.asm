TEXT github.com/2018kguo/phosphor.copyASCIICells(SB) /Users/kevinguo/repos/.phosphor-gap-5724_o1l/history-next-copy/serialize.go
  serialize.go:204	0x1001ce7c0		f9400b90		MOVD 16(R28), R16					
  serialize.go:204	0x1001ce7c4		eb3063ff		CMP R16, RSP						
  serialize.go:204	0x1001ce7c8		540004c9		BLS 38(PC)						
  serialize.go:204	0x1001ce7cc		f81f0ffe		MOVD.W R30, -16(RSP)					
  serialize.go:204	0x1001ce7d0		f81f83fd		MOVD R29, -8(RSP)					
  serialize.go:204	0x1001ce7d4		d10023fd		SUB $8, RSP, R29					
  serialize.go:204	0x1001ce7d8		f9000fe0		MOVD R0, 24(RSP)					
  serialize.go:204	0x1001ce7dc		f9001be3		MOVD R3, 48(RSP)					
  serialize.go:205	0x1001ce7e0		eb04005f		CMP R4, R2						
  serialize.go:205	0x1001ce7e4		540003a3		BCC 29(PC)						
  serialize.go:206	0x1001ce7e8		b4000364		CBZ R4, 27(PC)						
  serialize.go:206	0x1001ce7ec		b9400461		MOVWU 4(R3), R1						
  serialize.go:211	0x1001ce7f0		d3607c21		LSL $32, R1, R1						
  serialize.go:207	0x1001ce7f4		f9400462		MOVD 8(R3), R2						
  serialize.go:211	0x1001ce7f8		b2400021		ORR $1, R1, R1						
  serialize.go:212	0x1001ce7fc		aa1f03e5		MOVD ZR, R5						
  serialize.go:212	0x1001ce800		14000003		JMP 3(PC)						
  serialize.go:218	0x1001ce804		38256807		MOVB R7, (R0)(R5)					
  serialize.go:212	0x1001ce808		910004a5		ADD $1, R5, R5						
  serialize.go:212	0x1001ce80c		eb05009f		CMP R5, R4						
  serialize.go:212	0x1001ce810		540001ad		BLE 13(PC)						
  serialize.go:213	0x1001ce814		8b051066		ADD R5<<4, R3, R6					
  serialize.go:214	0x1001ce818		f94000c7		MOVD (R6), R7						
  serialize.go:215	0x1001ce81c		cb0100e8		SUB R1, R7, R8						
  serialize.go:215	0x1001ce820		f101fd1f		CMP $127, R8						
  serialize.go:215	0x1001ce824		54000082		BCS 4(PC)						
  serialize.go:215	0x1001ce828		f94004c6		MOVD 8(R6), R6						
  serialize.go:215	0x1001ce82c		eb0200df		CMP R2, R6						
  serialize.go:215	0x1001ce830		54fffea0		BEQ -11(PC)						
  serialize.go:216	0x1001ce834		aa0503e0		MOVD R5, R0						
  serialize.go:216	0x1001ce838		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:216	0x1001ce83c		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:216	0x1001ce840		d65f03c0		RET							
  serialize.go:220	0x1001ce844		aa0403e0		MOVD R4, R0						
  serialize.go:220	0x1001ce848		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:220	0x1001ce84c		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:220	0x1001ce850		d65f03c0		RET							
  serialize.go:206	0x1001ce854		97fad5b3		CALL runtime.panicBounds(SB)				
  serialize.go:205	0x1001ce858		97fad5b2		CALL runtime.panicBounds(SB)				
  serialize.go:205	0x1001ce85c		d503201f		NOOP							
  serialize.go:204	0x1001ce860		a90087e0		STP (R0, R1), 8(RSP)					
  serialize.go:204	0x1001ce864		a9018fe2		STP (R2, R3), 24(RSP)					
  serialize.go:204	0x1001ce868		a90297e4		STP (R4, R5), 40(RSP)					
  serialize.go:204	0x1001ce86c		aa1e03e3		MOVD R30, R3						
  serialize.go:204	0x1001ce870		97facd2c		CALL runtime.morestack_noctxt.abi0(SB)			
  serialize.go:204	0x1001ce874		a94087e0		LDP 8(RSP), (R0, R1)					
  serialize.go:204	0x1001ce878		a9418fe2		LDP 24(RSP), (R2, R3)					
  serialize.go:204	0x1001ce87c		a94297e4		LDP 40(RSP), (R4, R5)					
  serialize.go:204	0x1001ce880		17ffffd0		JMP github.com/2018kguo/phosphor.copyASCIICells(SB)	
  serialize.go:204	0x1001ce884		00000000		?							
  serialize.go:204	0x1001ce888		00000000		?							
  serialize.go:204	0x1001ce88c		00000000		?							
