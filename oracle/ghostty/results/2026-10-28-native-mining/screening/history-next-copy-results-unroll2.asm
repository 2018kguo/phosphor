TEXT github.com/2018kguo/phosphor.copyASCIICells(SB) /Users/kevinguo/repos/.phosphor-gap-5724_o1l/history-next-copy/serialize.go
  serialize.go:204	0x1001ce7c0		f9400b90		MOVD 16(R28), R16					
  serialize.go:204	0x1001ce7c4		eb3063ff		CMP R16, RSP						
  serialize.go:204	0x1001ce7c8		540008a9		BLS 69(PC)						
  serialize.go:204	0x1001ce7cc		f81f0ffe		MOVD.W R30, -16(RSP)					
  serialize.go:204	0x1001ce7d0		f81f83fd		MOVD R29, -8(RSP)					
  serialize.go:204	0x1001ce7d4		d10023fd		SUB $8, RSP, R29					
  serialize.go:204	0x1001ce7d8		f9000fe0		MOVD R0, 24(RSP)					
  serialize.go:204	0x1001ce7dc		f9001be3		MOVD R3, 48(RSP)					
  serialize.go:205	0x1001ce7e0		eb04005f		CMP R4, R2						
  serialize.go:205	0x1001ce7e4		54000783		BCC 60(PC)						
  serialize.go:206	0x1001ce7e8		b4000744		CBZ R4, 58(PC)						
  serialize.go:206	0x1001ce7ec		b9400461		MOVWU 4(R3), R1						
  serialize.go:211	0x1001ce7f0		d3607c21		LSL $32, R1, R1						
  serialize.go:207	0x1001ce7f4		f9400462		MOVD 8(R3), R2						
  serialize.go:211	0x1001ce7f8		b2400021		ORR $1, R1, R1						
  serialize.go:213	0x1001ce7fc		aa1f03e5		MOVD ZR, R5						
  serialize.go:213	0x1001ce800		14000003		JMP 3(PC)						
  serialize.go:225	0x1001ce804		38266808		MOVB R8, (R0)(R6)					
  serialize.go:213	0x1001ce808		910008a5		ADD $2, R5, R5						
  serialize.go:213	0x1001ce80c		d1000486		SUB $1, R4, R6						
  serialize.go:213	0x1001ce810		eb0600bf		CMP R6, R5						
  serialize.go:213	0x1001ce814		540003aa		BGE 29(PC)						
  serialize.go:214	0x1001ce818		8b051066		ADD R5<<4, R3, R6					
  serialize.go:215	0x1001ce81c		f94000c7		MOVD (R6), R7						
  serialize.go:216	0x1001ce820		cb0100e8		SUB R1, R7, R8						
  serialize.go:216	0x1001ce824		f101fd1f		CMP $127, R8						
  serialize.go:216	0x1001ce828		54000242		BCS 18(PC)						
  serialize.go:216	0x1001ce82c		f94004c6		MOVD 8(R6), R6						
  serialize.go:216	0x1001ce830		eb0200df		CMP R2, R6						
  serialize.go:216	0x1001ce834		540001e1		BNE 15(PC)						
  serialize.go:219	0x1001ce838		38256807		MOVB R7, (R0)(R5)					
  serialize.go:220	0x1001ce83c		910004a6		ADD $1, R5, R6						
  serialize.go:220	0x1001ce840		8b061067		ADD R6<<4, R3, R7					
  serialize.go:221	0x1001ce844		f94000e8		MOVD (R7), R8						
  serialize.go:222	0x1001ce848		cb010109		SUB R1, R8, R9						
  serialize.go:222	0x1001ce84c		f101fd3f		CMP $127, R9						
  serialize.go:222	0x1001ce850		54000082		BCS 4(PC)						
  serialize.go:222	0x1001ce854		f94004e7		MOVD 8(R7), R7						
  serialize.go:222	0x1001ce858		eb0200ff		CMP R2, R7						
  serialize.go:222	0x1001ce85c		54fffd40		BEQ -22(PC)						
  serialize.go:223	0x1001ce860		aa0603e0		MOVD R6, R0						
  serialize.go:223	0x1001ce864		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:223	0x1001ce868		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:223	0x1001ce86c		d65f03c0		RET							
  serialize.go:217	0x1001ce870		aa0503e0		MOVD R5, R0						
  serialize.go:217	0x1001ce874		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:217	0x1001ce878		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:217	0x1001ce87c		d65f03c0		RET							
  serialize.go:233	0x1001ce880		38256807		MOVB R7, (R0)(R5)					
  serialize.go:227	0x1001ce884		910004a5		ADD $1, R5, R5						
  serialize.go:227	0x1001ce888		eb05009f		CMP R5, R4						
  serialize.go:227	0x1001ce88c		540001ad		BLE 13(PC)						
  serialize.go:228	0x1001ce890		8b051066		ADD R5<<4, R3, R6					
  serialize.go:229	0x1001ce894		f94000c7		MOVD (R6), R7						
  serialize.go:230	0x1001ce898		cb0100e8		SUB R1, R7, R8						
  serialize.go:230	0x1001ce89c		f101fd1f		CMP $127, R8						
  serialize.go:230	0x1001ce8a0		54000082		BCS 4(PC)						
  serialize.go:230	0x1001ce8a4		f94004c6		MOVD 8(R6), R6						
  serialize.go:230	0x1001ce8a8		eb0200df		CMP R2, R6						
  serialize.go:230	0x1001ce8ac		54fffea0		BEQ -11(PC)						
  serialize.go:231	0x1001ce8b0		aa0503e0		MOVD R5, R0						
  serialize.go:231	0x1001ce8b4		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:231	0x1001ce8b8		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:231	0x1001ce8bc		d65f03c0		RET							
  serialize.go:235	0x1001ce8c0		aa0403e0		MOVD R4, R0						
  serialize.go:235	0x1001ce8c4		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:235	0x1001ce8c8		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:235	0x1001ce8cc		d65f03c0		RET							
  serialize.go:206	0x1001ce8d0		97fad594		CALL runtime.panicBounds(SB)				
  serialize.go:205	0x1001ce8d4		97fad593		CALL runtime.panicBounds(SB)				
  serialize.go:205	0x1001ce8d8		d503201f		NOOP							
  serialize.go:204	0x1001ce8dc		a90087e0		STP (R0, R1), 8(RSP)					
  serialize.go:204	0x1001ce8e0		a9018fe2		STP (R2, R3), 24(RSP)					
  serialize.go:204	0x1001ce8e4		a90297e4		STP (R4, R5), 40(RSP)					
  serialize.go:204	0x1001ce8e8		aa1e03e3		MOVD R30, R3						
  serialize.go:204	0x1001ce8ec		97facd0d		CALL runtime.morestack_noctxt.abi0(SB)			
  serialize.go:204	0x1001ce8f0		a94087e0		LDP 8(RSP), (R0, R1)					
  serialize.go:204	0x1001ce8f4		a9418fe2		LDP 24(RSP), (R2, R3)					
  serialize.go:204	0x1001ce8f8		a94297e4		LDP 40(RSP), (R4, R5)					
  serialize.go:204	0x1001ce8fc		17ffffb1		JMP github.com/2018kguo/phosphor.copyASCIICells(SB)	
