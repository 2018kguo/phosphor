TEXT github.com/2018kguo/phosphor.copyASCIICells(SB) /Users/kevinguo/repos/.phosphor-gap-5724_o1l/history-next-copy/serialize.go
  serialize.go:204	0x1001ce7c0		f9400b90		MOVD 16(R28), R16					
  serialize.go:204	0x1001ce7c4		eb3063ff		CMP R16, RSP						
  serialize.go:204	0x1001ce7c8		54000c69		BLS 99(PC)						
  serialize.go:204	0x1001ce7cc		f81f0ffe		MOVD.W R30, -16(RSP)					
  serialize.go:204	0x1001ce7d0		f81f83fd		MOVD R29, -8(RSP)					
  serialize.go:204	0x1001ce7d4		d10023fd		SUB $8, RSP, R29					
  serialize.go:204	0x1001ce7d8		f9000fe0		MOVD R0, 24(RSP)					
  serialize.go:204	0x1001ce7dc		f9001be3		MOVD R3, 48(RSP)					
  serialize.go:205	0x1001ce7e0		eb04005f		CMP R4, R2						
  serialize.go:205	0x1001ce7e4		54000b43		BCC 90(PC)						
  serialize.go:206	0x1001ce7e8		b4000b04		CBZ R4, 88(PC)						
  serialize.go:206	0x1001ce7ec		b9400461		MOVWU 4(R3), R1						
  serialize.go:211	0x1001ce7f0		d3607c21		LSL $32, R1, R1						
  serialize.go:207	0x1001ce7f4		f9400462		MOVD 8(R3), R2						
  serialize.go:211	0x1001ce7f8		b2400021		ORR $1, R1, R1						
  serialize.go:213	0x1001ce7fc		aa1f03e5		MOVD ZR, R5						
  serialize.go:213	0x1001ce800		14000003		JMP 3(PC)						
  serialize.go:237	0x1001ce804		38266808		MOVB R8, (R0)(R6)					
  serialize.go:213	0x1001ce808		910010a5		ADD $4, R5, R5						
  serialize.go:213	0x1001ce80c		d1000c86		SUB $3, R4, R6						
  serialize.go:213	0x1001ce810		eb0600bf		CMP R6, R5						
  serialize.go:213	0x1001ce814		5400072a		BGE 57(PC)						
  serialize.go:214	0x1001ce818		8b051066		ADD R5<<4, R3, R6					
  serialize.go:215	0x1001ce81c		f94000c7		MOVD (R6), R7						
  serialize.go:216	0x1001ce820		cb0100e8		SUB R1, R7, R8						
  serialize.go:216	0x1001ce824		f101fd1f		CMP $127, R8						
  serialize.go:216	0x1001ce828		540005c2		BCS 46(PC)						
  serialize.go:216	0x1001ce82c		f94004c6		MOVD 8(R6), R6						
  serialize.go:216	0x1001ce830		eb0200df		CMP R2, R6						
  serialize.go:216	0x1001ce834		54000561		BNE 43(PC)						
  serialize.go:219	0x1001ce838		38256807		MOVB R7, (R0)(R5)					
  serialize.go:220	0x1001ce83c		910004a6		ADD $1, R5, R6						
  serialize.go:220	0x1001ce840		8b061067		ADD R6<<4, R3, R7					
  serialize.go:221	0x1001ce844		f94000e8		MOVD (R7), R8						
  serialize.go:222	0x1001ce848		cb010109		SUB R1, R8, R9						
  serialize.go:222	0x1001ce84c		f101fd3f		CMP $127, R9						
  serialize.go:222	0x1001ce850		54000402		BCS 32(PC)						
  serialize.go:222	0x1001ce854		f94004e7		MOVD 8(R7), R7						
  serialize.go:222	0x1001ce858		eb0200ff		CMP R2, R7						
  serialize.go:222	0x1001ce85c		540003a1		BNE 29(PC)						
  serialize.go:225	0x1001ce860		38266808		MOVB R8, (R0)(R6)					
  serialize.go:226	0x1001ce864		910008a6		ADD $2, R5, R6						
  serialize.go:226	0x1001ce868		8b061067		ADD R6<<4, R3, R7					
  serialize.go:227	0x1001ce86c		f94000e8		MOVD (R7), R8						
  serialize.go:228	0x1001ce870		cb010109		SUB R1, R8, R9						
  serialize.go:228	0x1001ce874		f101fd3f		CMP $127, R9						
  serialize.go:228	0x1001ce878		54000242		BCS 18(PC)						
  serialize.go:228	0x1001ce87c		f94004e7		MOVD 8(R7), R7						
  serialize.go:228	0x1001ce880		eb0200ff		CMP R2, R7						
  serialize.go:228	0x1001ce884		540001e1		BNE 15(PC)						
  serialize.go:231	0x1001ce888		38266808		MOVB R8, (R0)(R6)					
  serialize.go:232	0x1001ce88c		91000ca6		ADD $3, R5, R6						
  serialize.go:232	0x1001ce890		8b061067		ADD R6<<4, R3, R7					
  serialize.go:233	0x1001ce894		f94000e8		MOVD (R7), R8						
  serialize.go:234	0x1001ce898		cb010109		SUB R1, R8, R9						
  serialize.go:234	0x1001ce89c		f101fd3f		CMP $127, R9						
  serialize.go:234	0x1001ce8a0		54000082		BCS 4(PC)						
  serialize.go:234	0x1001ce8a4		f94004e7		MOVD 8(R7), R7						
  serialize.go:234	0x1001ce8a8		eb0200ff		CMP R2, R7						
  serialize.go:234	0x1001ce8ac		54fffac0		BEQ -42(PC)						
  serialize.go:235	0x1001ce8b0		aa0603e0		MOVD R6, R0						
  serialize.go:235	0x1001ce8b4		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:235	0x1001ce8b8		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:235	0x1001ce8bc		d65f03c0		RET							
  serialize.go:229	0x1001ce8c0		aa0603e0		MOVD R6, R0						
  serialize.go:229	0x1001ce8c4		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:229	0x1001ce8c8		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:229	0x1001ce8cc		d65f03c0		RET							
  serialize.go:223	0x1001ce8d0		aa0603e0		MOVD R6, R0						
  serialize.go:223	0x1001ce8d4		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:223	0x1001ce8d8		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:223	0x1001ce8dc		d65f03c0		RET							
  serialize.go:217	0x1001ce8e0		aa0503e0		MOVD R5, R0						
  serialize.go:217	0x1001ce8e4		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:217	0x1001ce8e8		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:217	0x1001ce8ec		d65f03c0		RET							
  serialize.go:245	0x1001ce8f0		38256807		MOVB R7, (R0)(R5)					
  serialize.go:239	0x1001ce8f4		910004a5		ADD $1, R5, R5						
  serialize.go:239	0x1001ce8f8		eb05009f		CMP R5, R4						
  serialize.go:239	0x1001ce8fc		540001cd		BLE 14(PC)						
  serialize.go:240	0x1001ce900		54000229		BLS 17(PC)						
  serialize.go:240	0x1001ce904		8b051066		ADD R5<<4, R3, R6					
  serialize.go:241	0x1001ce908		f94000c7		MOVD (R6), R7						
  serialize.go:242	0x1001ce90c		cb0100e8		SUB R1, R7, R8						
  serialize.go:242	0x1001ce910		f101fd1f		CMP $127, R8						
  serialize.go:242	0x1001ce914		54000082		BCS 4(PC)						
  serialize.go:242	0x1001ce918		f94004c6		MOVD 8(R6), R6						
  serialize.go:242	0x1001ce91c		eb0200df		CMP R2, R6						
  serialize.go:242	0x1001ce920		54fffe80		BEQ -12(PC)						
  serialize.go:243	0x1001ce924		aa0503e0		MOVD R5, R0						
  serialize.go:243	0x1001ce928		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:243	0x1001ce92c		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:243	0x1001ce930		d65f03c0		RET							
  serialize.go:247	0x1001ce934		aa0403e0		MOVD R4, R0						
  serialize.go:247	0x1001ce938		f85f83fd		MOVD -8(RSP), R29					
  serialize.go:247	0x1001ce93c		f84107fe		MOVD.P 16(RSP), R30					
  serialize.go:247	0x1001ce940		d65f03c0		RET							
  serialize.go:240	0x1001ce944		97fad577		CALL runtime.panicBounds(SB)				
  serialize.go:206	0x1001ce948		97fad576		CALL runtime.panicBounds(SB)				
  serialize.go:205	0x1001ce94c		97fad575		CALL runtime.panicBounds(SB)				
  serialize.go:205	0x1001ce950		d503201f		NOOP							
  serialize.go:204	0x1001ce954		a90087e0		STP (R0, R1), 8(RSP)					
  serialize.go:204	0x1001ce958		a9018fe2		STP (R2, R3), 24(RSP)					
  serialize.go:204	0x1001ce95c		a90297e4		STP (R4, R5), 40(RSP)					
  serialize.go:204	0x1001ce960		aa1e03e3		MOVD R30, R3						
  serialize.go:204	0x1001ce964		97faccef		CALL runtime.morestack_noctxt.abi0(SB)			
  serialize.go:204	0x1001ce968		a94087e0		LDP 8(RSP), (R0, R1)					
  serialize.go:204	0x1001ce96c		a9418fe2		LDP 24(RSP), (R2, R3)					
  serialize.go:204	0x1001ce970		a94297e4		LDP 40(RSP), (R4, R5)					
  serialize.go:204	0x1001ce974		17ffff93		JMP github.com/2018kguo/phosphor.copyASCIICells(SB)	
  serialize.go:204	0x1001ce978		00000000		?							
  serialize.go:204	0x1001ce97c		00000000		?							
