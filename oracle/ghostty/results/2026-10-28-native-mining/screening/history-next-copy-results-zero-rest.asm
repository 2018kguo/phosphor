TEXT github.com/2018kguo/phosphor.copyASCIICells(SB) /Users/kevinguo/repos/.phosphor-gap-5724_o1l/history-next-copy/serialize.go
  serialize.go:204	0x1001ce7c0		f9400b90		MOVD 16(R28), R16					
  serialize.go:204	0x1001ce7c4		eb3063ff		CMP R16, RSP						
  serialize.go:204	0x1001ce7c8		54000789		BLS 60(PC)						
  serialize.go:204	0x1001ce7cc		f81f0ffe		MOVD.W R30, -16(RSP)					
  serialize.go:204	0x1001ce7d0		f81f83fd		MOVD R29, -8(RSP)					
  serialize.go:204	0x1001ce7d4		d10023fd		SUB $8, RSP, R29					
  serialize.go:204	0x1001ce7d8		f9000fe0		MOVD R0, 24(RSP)					
  serialize.go:204	0x1001ce7dc		f9001be3		MOVD R3, 48(RSP)					
  serialize.go:205	0x1001ce7e0		eb04005f		CMP R4, R2						
  serialize.go:205	0x1001ce7e4		54000663		BCC 51(PC)						
  serialize.go:206	0x1001ce7e8		b4000624		CBZ R4, 49(PC)						
  serialize.go:206	0x1001ce7ec		b9400461		MOVWU 4(R3), R1						
  serialize.go:211	0x1001ce7f0		d3607c21		LSL $32, R1, R1						
  serialize.go:211	0x1001ce7f4		b2400021		ORR $1, R1, R1						
  serialize.go:207	0x1001ce7f8		f9400462		MOVD 8(R3), R2						
  serialize.go:212	0x1001ce7fc		b5000062		CBNZ R2, 3(PC)						
  serialize.go:213	0x1001ce800		aa1f03e2		MOVD ZR, R2						
  serialize.go:213	0x1001ce804		14000019		JMP 25(PC)						
  serialize.go:223	0x1001ce808		aa1f03e5		MOVD ZR, R5						
  serialize.go:223	0x1001ce80c		14000003		JMP 3(PC)						
  serialize.go:229	0x1001ce810		38256807		MOVB R7, (R0)(R5)					
  serialize.go:223	0x1001ce814		910004a5		ADD $1, R5, R5						
  serialize.go:223	0x1001ce818		eb05009f		CMP R5, R4						
  serialize.go:223	0x1001ce81c		540001ad		BLE 13(PC)						
  serialize.go:224	0x1001ce820		8b051066		ADD R5<<4, R3, R6					
  serialize.go:225	0x1001ce824		f94000c7		MOVD (R6), R7						
  serialize.go:226	0x1001ce828		cb0100e8		SUB R1, R7, R8						
  serialize.go:226	0x1001ce82c		f101fd1f		CMP $127, R8						
  serialize.go:226	0x1001ce830		54000082		BCS 4(PC)						
  serialize.go:226	0x1001ce834		f94004c6		MOVD 8(R6), R6						
  serialize.go:226	0x1001ce838		eb0200df		CMP R2, R6						
  serialize.go:226	0x1001ce83c		54fffea0		BEQ -11(PC)						
  serialize.go:227	0x1001ce840		aa0503e0		MOVD R5, R0						
  serialize.go:227	0x1001ce844		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:227	0x1001ce848		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:227	0x1001ce84c		d65f03c0		RET							
  serialize.go:231	0x1001ce850		aa0403e0		MOVD R4, R0						
  serialize.go:231	0x1001ce854		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:231	0x1001ce858		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:231	0x1001ce85c		d65f03c0		RET							
  serialize.go:219	0x1001ce860		38226806		MOVB R6, (R0)(R2)					
  serialize.go:213	0x1001ce864		91000442		ADD $1, R2, R2						
  serialize.go:213	0x1001ce868		eb02009f		CMP R2, R4						
  serialize.go:213	0x1001ce86c		5400018d		BLE 12(PC)						
  serialize.go:214	0x1001ce870		8b021065		ADD R2<<4, R3, R5					
  serialize.go:215	0x1001ce874		f94000a6		MOVD (R5), R6						
  serialize.go:216	0x1001ce878		cb0100c7		SUB R1, R6, R7						
  serialize.go:216	0x1001ce87c		f101fcff		CMP $127, R7						
  serialize.go:216	0x1001ce880		54000062		BCS 3(PC)						
  serialize.go:216	0x1001ce884		f94004a5		MOVD 8(R5), R5						
  serialize.go:216	0x1001ce888		b4fffec5		CBZ R5, -10(PC)						
  serialize.go:217	0x1001ce88c		aa0203e0		MOVD R2, R0						
  serialize.go:217	0x1001ce890		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:217	0x1001ce894		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:217	0x1001ce898		d65f03c0		RET							
  serialize.go:221	0x1001ce89c		aa0403e0		MOVD R4, R0						
  serialize.go:221	0x1001ce8a0		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:221	0x1001ce8a4		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:221	0x1001ce8a8		d65f03c0		RET							
  serialize.go:206	0x1001ce8ac		97fad59d		CALL runtime.panicBounds(SB)				
  serialize.go:205	0x1001ce8b0		97fad59c		CALL runtime.panicBounds(SB)				
  serialize.go:205	0x1001ce8b4		d503201f		NOOP							
  serialize.go:204	0x1001ce8b8		a90087e0		STP (R0, R1), 8(RSP)					
  serialize.go:204	0x1001ce8bc		a9018fe2		STP (R2, R3), 24(RSP)					
  serialize.go:204	0x1001ce8c0		a90297e4		STP (R4, R5), 40(RSP)					
  serialize.go:204	0x1001ce8c4		aa1e03e3		MOVD R30, R3						
  serialize.go:204	0x1001ce8c8		97facd16		CALL runtime.morestack_noctxt.abi0(SB)			
  serialize.go:204	0x1001ce8cc		a94087e0		LDP 8(RSP), (R0, R1)					
  serialize.go:204	0x1001ce8d0		a9418fe2		LDP 24(RSP), (R2, R3)					
  serialize.go:204	0x1001ce8d4		a94297e4		LDP 40(RSP), (R4, R5)					
  serialize.go:204	0x1001ce8d8		17ffffba		JMP github.com/2018kguo/phosphor.copyASCIICells(SB)	
  serialize.go:204	0x1001ce8dc		00000000		?							
