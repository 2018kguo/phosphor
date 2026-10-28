TEXT github.com/2018kguo/phosphor.(*Terminal).appendRowHistory(SB) /Users/kevinguo/repos/.phosphor-gap-5724_o1l/history-next/serialize.go
  serialize.go:100	0x1001cd9f0		f9400b90		MOVD 16(R28), R16							
  serialize.go:100	0x1001cd9f4		d10503f1		SUB $320, RSP, R17							
  serialize.go:100	0x1001cd9f8		eb10023f		CMP R16, R17								
  serialize.go:100	0x1001cd9fc		54006c49		BLS 866(PC)								
  serialize.go:100	0x1001cda00		d10703f4		SUB $448, RSP, R20							
  serialize.go:100	0x1001cda04		a93ffa9d		STP (R29, R30), -8(R20)							
  serialize.go:100	0x1001cda08		9100029f		MOVD R20, RSP								
  serialize.go:100	0x1001cda0c		d10023fd		SUB $8, RSP, R29							
  serialize.go:100	0x1001cda10		f900ebe1		MOVD R1, 464(RSP)							
  serialize.go:100	0x1001cda14		f900f7e4		MOVD R4, 488(RSP)							
  serialize.go:102	0x1001cda18		36000068		TBZ $0, R8, 3(PC)							
  serialize.go:102	0x1001cda1c		aa0503e8		MOVD R5, R8								
  terminal.go:2665	0x1001cda20		14000348		JMP 840(PC)								
  serialize.go:105	0x1001cda24		a908ffff		STP (ZR, ZR), 136(RSP)							
  serialize.go:109	0x1001cda28		eb0700bf		CMP R7, R5								
  serialize.go:109	0x1001cda2c		5400004b		BLT 2(PC)								
  serialize.go:109	0x1001cda30		aa0703e5		MOVD R7, R5								
  serialize.go:109	0x1001cda34		eb0500df		CMP R5, R6								
  serialize.go:109	0x1001cda38		54006803		BCC 832(PC)								
  serialize.go:110	0x1001cda3c		37006449		TBNZ $0, R9, 802(PC)							
  serialize.go:102	0x1001cda40		f900e7e0		MOVD R0, 456(RSP)							
  serialize.go:102	0x1001cda44		390827e9		MOVB R9, 521(RSP)							
  serialize.go:105	0x1001cda48		f9009fe7		MOVD R7, 312(RSP)							
  serialize.go:102	0x1001cda4c		f900f7e4		MOVD R4, 488(RSP)							
  serialize.go:102	0x1001cda50		f900ffe6		MOVD R6, 504(RSP)							
  serialize.go:117	0x1001cda54		f900a3e5		MOVD R5, 320(RSP)							
  serialize.go:117	0x1001cda58		aa1f03e8		MOVD ZR, R8								
  serialize.go:117	0x1001cda5c		aa1f03ea		MOVD ZR, R10								
  serialize.go:117	0x1001cda60		aa1f03eb		MOVD ZR, R11								
  serialize.go:117	0x1001cda64		14000003		JMP 3(PC)								
  serialize.go:117	0x1001cda68		aa1003e8		MOVD R16, R8								
  serialize.go:117	0x1001cda6c		aa0e03e2		MOVD R14, R2								
  serialize.go:117	0x1001cda70		eb0800bf		CMP R8, R5								
  serialize.go:117	0x1001cda74		540027cd		BLE 318(PC)								
  serialize.go:118	0x1001cda78		54006229		BLS 785(PC)								
  serialize.go:117	0x1001cda7c		f90097e8		MOVD R8, 296(RSP)							
  serialize.go:118	0x1001cda80		8b08108e		ADD R8<<4, R4, R14							
  serialize.go:119	0x1001cda84		b94005cf		MOVWU 4(R14), R15							
  serialize.go:119	0x1001cda88		6b0f015f		CMPW R15, R10								
  serialize.go:119	0x1001cda8c		54000081		BNE 4(PC)								
  serialize.go:119	0x1001cda90		f94005cf		MOVD 8(R14), R15							
  serialize.go:119	0x1001cda94		eb0f017f		CMP R15, R11								
  serialize.go:119	0x1001cda98		54000680		BEQ 52(PC)								
  serialize.go:118	0x1001cda9c		f900c7ee		MOVD R14, 392(RSP)							
  serialize.go:120	0x1001cdaa0		79401dcf		MOVHU 14(R14), R15							
  serialize.go:120	0x1001cdaa4		79412ff0		MOVHU 150(RSP), R16							
  serialize.go:120	0x1001cdaa8		6b1001ff		CMPW R16, R15								
  serialize.go:120	0x1001cdaac		540000a1		BNE 5(PC)								
  serialize.go:120	0x1001cdab0		794019cf		MOVHU 12(R14), R15							
  serialize.go:120	0x1001cdab4		79412bf0		MOVHU 148(RSP), R16							
  serialize.go:120	0x1001cdab8		ca1001ef		EOR R16, R15, R15							
  serialize.go:120	0x1001cdabc		364001cf		TBZ $8, R15, 14(PC)							
  serialize.go:121	0x1001cdac0		a9403dce		LDP (R14), (R14, R15)							
  serialize.go:121	0x1001cdac4		910373fb		ADD $220, RSP, R27							
  serialize.go:121	0x1001cdac8		a9003f6e		STP (R14, R15), (R27)							
  serialize.go:121	0x1001cdacc		b980dfe9		MOVW 220(RSP), R9							
  serialize.go:121	0x1001cdad0		b9808be4		MOVW 136(RSP), R4							
  serialize.go:121	0x1001cdad4		295c2fea		LDPW 224(RSP), (R10, R11)						
  serialize.go:121	0x1001cdad8		29519be5		LDPW 140(RSP), (R5, R6)							
  serialize.go:121	0x1001cdadc		7941d3ec		MOVHU 232(RSP), R12							
  serialize.go:121	0x1001cdae0		79412be7		MOVHU 148(RSP), R7							
  serialize.go:121	0x1001cdae4		79412fe8		MOVHU 150(RSP), R8							
  serialize.go:121	0x1001cdae8		7941d7ed		MOVHU 234(RSP), R13							
  serialize.go:121	0x1001cdaec		94000369		CALL github.com/2018kguo/phosphor.(*Terminal).appendStyle(SB)		
  serialize.go:121	0x1001cdaf0		14000009		JMP 9(PC)								
  serialize.go:123	0x1001cdaf4		a94029c9		LDP (R14), (R9, R10)							
  serialize.go:123	0x1001cdaf8		910333fb		ADD $204, RSP, R27							
  serialize.go:123	0x1001cdafc		a9002b69		STP (R9, R10), (R27)							
  serialize.go:123	0x1001cdb00		b980cfe4		MOVW 204(RSP), R4							
  serialize.go:123	0x1001cdb04		295a1be5		LDPW 208(RSP), (R5, R6)							
  serialize.go:123	0x1001cdb08		7941b3e7		MOVHU 216(RSP), R7							
  serialize.go:123	0x1001cdb0c		7941b7e8		MOVHU 218(RSP), R8							
  serialize.go:123	0x1001cdb10		97fffe10		CALL github.com/2018kguo/phosphor.(*Terminal).appendSGR(SB)		
  serialize.go:125	0x1001cdb14		f940c7ee		MOVD 392(RSP), R14							
  serialize.go:125	0x1001cdb18		a94041cf		LDP (R14), (R15, R16)							
  serialize.go:125	0x1001cdb1c		9102b3fb		ADD $172, RSP, R27							
  serialize.go:125	0x1001cdb20		a900436f		STP (R15, R16), (R27)							
  serialize.go:88	0x1001cdb24		b940b3ef		MOVWU 176(RSP), R15							
  serialize.go:88	0x1001cdb28		f84b43f0		MOVD 180(RSP), R16							
  serialize.go:125	0x1001cdb2c		a908ffff		STP (ZR, ZR), 136(RSP)							
  serialize.go:125	0x1001cdb30		b9008fef		MOVW R15, 140(RSP)							
  serialize.go:125	0x1001cdb34		f9004bf0		MOVD R16, 144(RSP)							
  serialize.go:126	0x1001cdb38		b94005ca		MOVWU 4(R14), R10							
  serialize.go:126	0x1001cdb3c		f94005cb		MOVD 8(R14), R11							
  serialize.go:130	0x1001cdb40		f940f7e4		MOVD 488(RSP), R4							
  serialize.go:130	0x1001cdb44		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:142	0x1001cdb48		f940ffe6		MOVD 504(RSP), R6							
  serialize.go:169	0x1001cdb4c		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:129	0x1001cdb50		f94097e8		MOVD 296(RSP), R8							
  serialize.go:128	0x1001cdb54		394827e9		MOVBU 521(RSP), R9							
  serialize.go:128	0x1001cdb58		aa0203e3		MOVD R2, R3								
  serialize.go:128	0x1001cdb5c		aa0103e2		MOVD R1, R2								
  serialize.go:128	0x1001cdb60		aa0003e1		MOVD R0, R1								
  serialize.go:152	0x1001cdb64		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:128	0x1001cdb68		f9008beb		MOVD R11, 272(RSP)							
  serialize.go:128	0x1001cdb6c		b9009bea		MOVW R10, 152(RSP)							
  serialize.go:128	0x1001cdb70		360000a9		TBZ $0, R9, 5(PC)							
  serialize.go:128	0x1001cdb74		b98001cf		MOVW (R14), R15								
  serialize.go:128	0x1001cdb78		3500006f		CBNZW R15, 3(PC)							
  serialize.go:129	0x1001cdb7c		9100050e		ADD $1, R8, R14								
  serialize.go:130	0x1001cdb80		14000233		JMP 563(PC)								
  serialize.go:140	0x1001cdb84		b98001cf		MOVW (R14), R15								
  serialize.go:140	0x1001cdb88		d10005f0		SUB $1, R15, R16							
  serialize.go:140	0x1001cdb8c		7101fe1f		CMPW $127, R16								
  serialize.go:140	0x1001cdb90		54001503		BCC 168(PC)								
  serialize.go:147	0x1001cdb94		91000510		ADD $1, R8, R16								
  serialize.go:150	0x1001cdb98		310005ff		CMNW $1, R15								
  serialize.go:150	0x1001cdb9c		54000081		BNE 4(PC)								
  serialize.go:150	0x1001cdba0		b27b03f1		ORR $32, ZR, R17							
  serialize.go:150	0x1001cdba4		aa0203ee		MOVD R2, R14								
  serialize.go:150	0x1001cdba8		17ffffb0		JMP -80(PC)								
  serialize.go:147	0x1001cdbac		f900bff0		MOVD R16, 376(RSP)							
  serialize.go:151	0x1001cdbb0		3400006f		CBZW R15, 3(PC)								
  serialize.go:151	0x1001cdbb4		310009ff		CMNW $2, R15								
  serialize.go:151	0x1001cdbb8		54000661		BNE 51(PC)								
  serialize.go:152	0x1001cdbbc		3500030f		CBNZW R15, 24(PC)							
  serialize.go:152	0x1001cdbc0		f940000e		MOVD (R0), R14								
  serialize.go:152	0x1001cdbc4		f10005df		CMP $1, R14								
  serialize.go:152	0x1001cdbc8		540002a1		BNE 21(PC)								
  serialize.go:152	0x1001cdbcc		37000289		TBNZ $0, R9, 20(PC)							
  utf8.go:415		0x1001cdbd0		aa0103e0		MOVD R1, R0								
  utf8.go:415		0x1001cdbd4		aa0203e1		MOVD R2, R1								
  utf8.go:415		0x1001cdbd8		aa0303e2		MOVD R3, R2								
  utf8.go:415		0x1001cdbdc		d28ea983		MOVD $30028, R3								
  utf8.go:415		0x1001cdbe0		97fb5a68		CALL unicode/utf8.appendRuneNonASCII(SB)				
  serialize.go:118	0x1001cdbe4		f940f7e4		MOVD 488(RSP), R4							
  serialize.go:117	0x1001cdbe8		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:142	0x1001cdbec		f940ffe6		MOVD 504(RSP), R6							
  serialize.go:169	0x1001cdbf0		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:128	0x1001cdbf4		394827e9		MOVBU 521(RSP), R9							
  serialize.go:117	0x1001cdbf8		b9409bea		MOVWU 152(RSP), R10							
  serialize.go:117	0x1001cdbfc		f9408beb		MOVD 272(RSP), R11							
  serialize.go:118	0x1001cdc00		f940bff0		MOVD 376(RSP), R16							
  serialize.go:118	0x1001cdc04		b27b03f1		ORR $32, ZR, R17							
  serialize.go:152	0x1001cdc08		aa0203e3		MOVD R2, R3								
  serialize.go:152	0x1001cdc0c		aa0103ee		MOVD R1, R14								
  serialize.go:152	0x1001cdc10		aa0003e1		MOVD R0, R1								
  serialize.go:152	0x1001cdc14		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:155	0x1001cdc18		17ffff94		JMP -108(PC)								
  serialize.go:157	0x1001cdc1c		9100044e		ADD $1, R2, R14								
  serialize.go:157	0x1001cdc20		eb0e007f		CMP R14, R3								
  serialize.go:157	0x1001cdc24		54000282		BCS 20(PC)								
  serialize.go:157	0x1001cdc28		aa0103e0		MOVD R1, R0								
  serialize.go:157	0x1001cdc2c		aa0e03e1		MOVD R14, R1								
  serialize.go:157	0x1001cdc30		aa0303e2		MOVD R3, R2								
  serialize.go:157	0x1001cdc34		b24003e3		ORR $1, ZR, R3								
  serialize.go:157	0x1001cdc38		b0001504		ADRP 2756608(PC), R4							
  serialize.go:157	0x1001cdc3c		91388084		ADD $3616, R4, R4							
  serialize.go:157	0x1001cdc40		97fac360		CALL runtime.growslice(SB)						
  serialize.go:118	0x1001cdc44		f940f7e4		MOVD 488(RSP), R4							
  serialize.go:117	0x1001cdc48		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:142	0x1001cdc4c		f940ffe6		MOVD 504(RSP), R6							
  serialize.go:169	0x1001cdc50		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:128	0x1001cdc54		394827e9		MOVBU 521(RSP), R9							
  serialize.go:117	0x1001cdc58		b9409bea		MOVWU 152(RSP), R10							
  serialize.go:117	0x1001cdc5c		f9408beb		MOVD 272(RSP), R11							
  serialize.go:118	0x1001cdc60		f940bff0		MOVD 376(RSP), R16							
  serialize.go:157	0x1001cdc64		aa0103ee		MOVD R1, R14								
  serialize.go:157	0x1001cdc68		aa0003e1		MOVD R0, R1								
  serialize.go:157	0x1001cdc6c		aa0203e3		MOVD R2, R3								
  serialize.go:152	0x1001cdc70		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:157	0x1001cdc74		d10005cf		SUB $1, R14, R15							
  serialize.go:157	0x1001cdc78		b27b03f1		ORR $32, ZR, R17							
  serialize.go:157	0x1001cdc7c		382f6831		MOVB R17, (R1)(R15)							
  serialize.go:157	0x1001cdc80		17ffff7a		JMP -134(PC)								
  serialize.go:159	0x1001cdc84		d2a80011		MOVD $1073741824, R17							
  serialize.go:159	0x1001cdc88		6b1101ff		CMPW R17, R15								
  serialize.go:159	0x1001cdc8c		5400062a		BGE 49(PC)								
  utf8.go:412		0x1001cdc90		7101fdff		CMPW $127, R15								
  utf8.go:412		0x1001cdc94		54000388		BHI 28(PC)								
  utf8.go:413		0x1001cdc98		9100044e		ADD $1, R2, R14								
  utf8.go:413		0x1001cdc9c		eb0e007f		CMP R14, R3								
  utf8.go:413		0x1001cdca0		540002c2		BCS 22(PC)								
  serialize.go:140	0x1001cdca4		b900efef		MOVW R15, 236(RSP)							
  utf8.go:413		0x1001cdca8		aa0103e0		MOVD R1, R0								
  utf8.go:413		0x1001cdcac		aa0e03e1		MOVD R14, R1								
  utf8.go:413		0x1001cdcb0		aa0303e2		MOVD R3, R2								
  utf8.go:413		0x1001cdcb4		b24003e3		ORR $1, ZR, R3								
  utf8.go:413		0x1001cdcb8		b0001504		ADRP 2756608(PC), R4							
  utf8.go:413		0x1001cdcbc		91388084		ADD $3616, R4, R4							
  utf8.go:413		0x1001cdcc0		97fac340		CALL runtime.growslice(SB)						
  serialize.go:118	0x1001cdcc4		f940f7e4		MOVD 488(RSP), R4							
  serialize.go:117	0x1001cdcc8		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:142	0x1001cdccc		f940ffe6		MOVD 504(RSP), R6							
  serialize.go:169	0x1001cdcd0		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:128	0x1001cdcd4		394827e9		MOVBU 521(RSP), R9							
  serialize.go:117	0x1001cdcd8		b9409bea		MOVWU 152(RSP), R10							
  serialize.go:117	0x1001cdcdc		f9408beb		MOVD 272(RSP), R11							
  utf8.go:413		0x1001cdce0		b980efef		MOVW 236(RSP), R15							
  serialize.go:118	0x1001cdce4		f940bff0		MOVD 376(RSP), R16							
  serialize.go:166	0x1001cdce8		aa0103ee		MOVD R1, R14								
  serialize.go:166	0x1001cdcec		aa0003e1		MOVD R0, R1								
  serialize.go:166	0x1001cdcf0		aa0203e3		MOVD R2, R3								
  serialize.go:152	0x1001cdcf4		f940e7e0		MOVD 456(RSP), R0							
  utf8.go:413		0x1001cdcf8		d10005d1		SUB $1, R14, R17							
  utf8.go:413		0x1001cdcfc		3831682f		MOVB R15, (R1)(R17)							
  serialize.go:166	0x1001cdd00		14000012		JMP 18(PC)								
  utf8.go:415		0x1001cdd04		aa0103e0		MOVD R1, R0								
  utf8.go:415		0x1001cdd08		aa0203e1		MOVD R2, R1								
  utf8.go:415		0x1001cdd0c		aa0303e2		MOVD R3, R2								
  utf8.go:415		0x1001cdd10		aa0f03e3		MOVD R15, R3								
  utf8.go:415		0x1001cdd14		97fb5a1b		CALL unicode/utf8.appendRuneNonASCII(SB)				
  serialize.go:118	0x1001cdd18		f940f7e4		MOVD 488(RSP), R4							
  serialize.go:117	0x1001cdd1c		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:142	0x1001cdd20		f940ffe6		MOVD 504(RSP), R6							
  serialize.go:169	0x1001cdd24		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:128	0x1001cdd28		394827e9		MOVBU 521(RSP), R9							
  serialize.go:117	0x1001cdd2c		b9409bea		MOVWU 152(RSP), R10							
  serialize.go:117	0x1001cdd30		f9408beb		MOVD 272(RSP), R11							
  serialize.go:118	0x1001cdd34		f940bff0		MOVD 376(RSP), R16							
  serialize.go:166	0x1001cdd38		aa0103ee		MOVD R1, R14								
  serialize.go:166	0x1001cdd3c		aa0203e3		MOVD R2, R3								
  serialize.go:166	0x1001cdd40		aa0003e1		MOVD R0, R1								
  serialize.go:152	0x1001cdd44		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:166	0x1001cdd48		b27b03f1		ORR $32, ZR, R17							
  serialize.go:166	0x1001cdd4c		17ffff47		JMP -185(PC)								
  serialize.go:160	0x1001cdd50		a907ffff		STP (ZR, ZR), 120(RSP)							
  serialize.go:161	0x1001cdd54		eb1000bf		CMP R16, R5								
  serialize.go:161	0x1001cdd58		540000ed		BLE 7(PC)								
  serialize.go:161	0x1001cdd5c		8b101093		ADD R16<<4, R4, R19							
  serialize.go:161	0x1001cdd60		b9800274		MOVW (R19), R20								
  serialize.go:161	0x1001cdd64		3100069f		CMNW $1, R20								
  serialize.go:161	0x1001cdd68		54000061		BNE 3(PC)								
  serialize.go:162	0x1001cdd6c		a9405273		LDP (R19), (R19, R20)							
  serialize.go:162	0x1001cdd70		a907d3f3		STP (R19, R20), 120(RSP)						
  serialize.go:164	0x1001cdd74		f9406813		MOVD 208(R0), R19							
  serialize.go:164	0x1001cdd78		a9455273		LDP 80(R19), (R19, R20)							
  serialize.go:164	0x1001cdd7c		cb1101ef		SUB R17, R15, R15							
  serialize.go:164	0x1001cdd80		93407df1		SXTW R15, R17								
  serialize.go:164	0x1001cdd84		eb14023f		CMP R20, R17								
  serialize.go:164	0x1001cdd88		540035a2		BCS 429(PC)								
  serialize.go:164	0x1001cdd8c		937d7def		SBFIZ $3, R15, $32, R15							
  serialize.go:164	0x1001cdd90		8b0f09ef		ADD R15<<2, R15, R15							
  serialize.go:164	0x1001cdd94		8b0f026f		ADD R15, R19, R15							
  serialize.go:164	0x1001cdd98		a94045f0		LDP (R15), (R16, R17)							
  serialize.go:164	0x1001cdd9c		a94151f3		LDP 16(R15), (R19, R20)							
  serialize.go:164	0x1001cdda0		f94011ef		MOVD 32(R15), R15							
  serialize.go:164	0x1001cdda4		a91947f0		STP (R16, R17), 400(RSP)						
  serialize.go:164	0x1001cdda8		a91a53f3		STP (R19, R20), 416(RSP)						
  serialize.go:164	0x1001cddac		f900dbef		MOVD R15, 432(RSP)							
  serialize.go:164	0x1001cddb0		a9403dce		LDP (R14), (R14, R15)							
  serialize.go:164	0x1001cddb4		9102f3fb		ADD $188, RSP, R27							
  serialize.go:164	0x1001cddb8		a9003f6e		STP (R14, R15), (R27)							
  serialize.go:164	0x1001cddbc		a9593fee		LDP 400(RSP), (R14, R15)						
  serialize.go:164	0x1001cddc0		a95a47f0		LDP 416(RSP), (R16, R17)						
  serialize.go:164	0x1001cddc4		b9807be9		MOVW 120(RSP), R9							
  serialize.go:164	0x1001cddc8		b980bfe4		MOVW 188(RSP), R4							
  serialize.go:164	0x1001cddcc		f940dbf3		MOVD 432(RSP), R19							
  serialize.go:164	0x1001cddd0		294fafea		LDPW 124(RSP), (R10, R11)						
  serialize.go:164	0x1001cddd4		29581be5		LDPW 192(RSP), (R5, R6)							
  serialize.go:164	0x1001cddd8		794193e7		MOVHU 200(RSP), R7							
  serialize.go:164	0x1001cdddc		79410bec		MOVHU 132(RSP), R12							
  serialize.go:164	0x1001cdde0		794197e8		MOVHU 202(RSP), R8							
  serialize.go:164	0x1001cdde4		79410fed		MOVHU 134(RSP), R13							
  serialize.go:164	0x1001cdde8		a900bfee		STP (R14, R15), 8(RSP)							
  serialize.go:164	0x1001cddec		a901c7f0		STP (R16, R17), 24(RSP)							
  serialize.go:164	0x1001cddf0		f90017f3		MOVD R19, 40(RSP)							
  serialize.go:164	0x1001cddf4		97ffd23f		CALL github.com/2018kguo/phosphor.(*Terminal).appendCluster(SB)		
  serialize.go:118	0x1001cddf8		f940f7e4		MOVD 488(RSP), R4							
  serialize.go:117	0x1001cddfc		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:142	0x1001cde00		f940ffe6		MOVD 504(RSP), R6							
  serialize.go:169	0x1001cde04		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:128	0x1001cde08		394827e9		MOVBU 521(RSP), R9							
  serialize.go:117	0x1001cde0c		b9409bea		MOVWU 152(RSP), R10							
  serialize.go:117	0x1001cde10		f9408beb		MOVD 272(RSP), R11							
  serialize.go:118	0x1001cde14		f940bff0		MOVD 376(RSP), R16							
  serialize.go:118	0x1001cde18		b27b03f1		ORR $32, ZR, R17							
  serialize.go:118	0x1001cde1c		aa0203e3		MOVD R2, R3								
  serialize.go:118	0x1001cde20		aa0103ee		MOVD R1, R14								
  serialize.go:118	0x1001cde24		aa0003e1		MOVD R0, R1								
  serialize.go:152	0x1001cde28		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:164	0x1001cde2c		17ffff0f		JMP -241(PC)								
  serialize.go:128	0x1001cde30		f90083e2		MOVD R2, 256(RSP)							
  serialize.go:141	0x1001cde34		cb0800ac		SUB R8, R5, R12								
  slices.go:424		0x1001cde38		8b0c004d		ADD R12, R2, R13							
  slices.go:424		0x1001cde3c		cb0301ae		SUB R3, R13, R14							
  slices.go:424		0x1001cde40		f10001df		CMP $0, R14								
  slices.go:424		0x1001cde44		5400052d		BLE 41(PC)								
  serialize.go:141	0x1001cde48		f90093ec		MOVD R12, 288(RSP)							
  slices.go:426		0x1001cde4c		8b0e006f		ADD R14, R3, R15							
  slices.go:426		0x1001cde50		eb0f007f		CMP R15, R3								
  slices.go:426		0x1001cde54		54000063		BCC 3(PC)								
  serialize.go:128	0x1001cde58		aa0303f0		MOVD R3, R16								
  slices.go:426		0x1001cde5c		14000011		JMP 17(PC)								
  slices.go:424		0x1001cde60		f900bbee		MOVD R14, 368(RSP)							
  slices.go:424		0x1001cde64		f900b7ed		MOVD R13, 360(RSP)							
  serialize.go:128	0x1001cde68		f90087e3		MOVD R3, 264(RSP)							
  slices.go:426		0x1001cde6c		aa0103e0		MOVD R1, R0								
  slices.go:426		0x1001cde70		aa0f03e1		MOVD R15, R1								
  slices.go:426		0x1001cde74		aa0303e2		MOVD R3, R2								
  slices.go:426		0x1001cde78		aa0e03e3		MOVD R14, R3								
  slices.go:426		0x1001cde7c		b0001504		ADRP 2756608(PC), R4							
  slices.go:426		0x1001cde80		91388084		ADD $3616, R4, R4							
  slices.go:426		0x1001cde84		97fac2cf		CALL runtime.growslice(SB)						
  slices.go:426		0x1001cde88		f940b7ed		MOVD 360(RSP), R13							
  slices.go:426		0x1001cde8c		f940bbee		MOVD 368(RSP), R14							
  slices.go:426		0x1001cde90		f94087f0		MOVD 264(RSP), R16							
  slices.go:426		0x1001cde94		aa0203e3		MOVD R2, R3								
  slices.go:426		0x1001cde98		aa0103ef		MOVD R1, R15								
  slices.go:426		0x1001cde9c		aa0003e1		MOVD R0, R1								
  slices.go:426		0x1001cdea0		f900bbe3		MOVD R3, 368(RSP)							
  slices.go:426		0x1001cdea4		f900c3e1		MOVD R1, 384(RSP)							
  slices.go:426		0x1001cdea8		8b0f0202		ADD R15, R16, R2							
  slices.go:426		0x1001cdeac		cb0d0042		SUB R13, R2, R2								
  slices.go:426		0x1001cdeb0		8b020020		ADD R2, R1, R0								
  slices.go:426		0x1001cdeb4		aa0e03e1		MOVD R14, R1								
  slices.go:426		0x1001cdeb8		97fad832		CALL runtime.memclrNoHeapPointers(SB)					
  slices.go:426		0x1001cdebc		f940bbe2		MOVD 368(RSP), R2							
  slices.go:426		0x1001cdec0		f94083e3		MOVD 256(RSP), R3							
  slices.go:426		0x1001cdec4		eb03005f		CMP R3, R2								
  slices.go:426		0x1001cdec8		54002b83		BCC 348(PC)								
  serialize.go:142	0x1001cdecc		aa0303e2		MOVD R3, R2								
  serialize.go:142	0x1001cded0		f940f7e4		MOVD 488(RSP), R4							
  serialize.go:142	0x1001cded4		f940ffe6		MOVD 504(RSP), R6							
  serialize.go:142	0x1001cded8		f94097e8		MOVD 296(RSP), R8							
  serialize.go:142	0x1001cdedc		f94093ec		MOVD 288(RSP), R12							
  serialize.go:141	0x1001cdee0		f940bbe3		MOVD 368(RSP), R3							
  serialize.go:141	0x1001cdee4		f940c3e1		MOVD 384(RSP), R1							
  serialize.go:141	0x1001cdee8		f900bbe3		MOVD R3, 368(RSP)							
  serialize.go:141	0x1001cdeec		f900c3e1		MOVD R1, 384(RSP)							
  serialize.go:142	0x1001cdef0		cb030047		SUB R3, R2, R7								
  serialize.go:142	0x1001cdef4		8a87fc47		AND R7->63, R2, R7							
  serialize.go:142	0x1001cdef8		8b070020		ADD R7, R1, R0								
  serialize.go:142	0x1001cdefc		cb020062		SUB R2, R3, R2								
  serialize.go:142	0x1001cdf00		cb0800c5		SUB R8, R6, R5								
  serialize.go:142	0x1001cdf04		cb060106		SUB R6, R8, R6								
  serialize.go:142	0x1001cdf08		8a86fd06		AND R6->63, R8, R6							
  serialize.go:142	0x1001cdf0c		8b061083		ADD R6<<4, R4, R3							
  serialize.go:142	0x1001cdf10		aa0203e1		MOVD R2, R1								
  serialize.go:142	0x1001cdf14		aa0c03e4		MOVD R12, R4								
  serialize.go:142	0x1001cdf18		9400022a		CALL github.com/2018kguo/phosphor.copyASCIICells(SB)			
  serialize.go:143	0x1001cdf1c		f94083e6		MOVD 256(RSP), R6							
  serialize.go:143	0x1001cdf20		8b0000c6		ADD R0, R6, R6								
  serialize.go:143	0x1001cdf24		f940bbe7		MOVD 368(RSP), R7							
  serialize.go:143	0x1001cdf28		eb0600ff		CMP R6, R7								
  serialize.go:143	0x1001cdf2c		54002843		BCC 322(PC)								
  serialize.go:144	0x1001cdf30		f94097ee		MOVD 296(RSP), R14							
  serialize.go:144	0x1001cdf34		8b0001d0		ADD R0, R14, R16							
  serialize.go:152	0x1001cdf38		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:118	0x1001cdf3c		f940f7e4		MOVD 488(RSP), R4							
  serialize.go:117	0x1001cdf40		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:169	0x1001cdf44		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:128	0x1001cdf48		394827e9		MOVBU 521(RSP), R9							
  serialize.go:117	0x1001cdf4c		b9409bea		MOVWU 152(RSP), R10							
  serialize.go:117	0x1001cdf50		f9408beb		MOVD 272(RSP), R11							
  serialize.go:117	0x1001cdf54		b27b03f1		ORR $32, ZR, R17							
  serialize.go:118	0x1001cdf58		f940bbe3		MOVD 368(RSP), R3							
  serialize.go:118	0x1001cdf5c		aa0603ee		MOVD R6, R14								
  serialize.go:118	0x1001cdf60		f940c3e1		MOVD 384(RSP), R1							
  serialize.go:142	0x1001cdf64		f940ffe6		MOVD 504(RSP), R6							
  serialize.go:145	0x1001cdf68		17fffec0		JMP -320(PC)								
  serialize.go:169	0x1001cdf6c		eb0700bf		CMP R7, R5								
  serialize.go:169	0x1001cdf70		5400026a		BGE 19(PC)								
  serialize.go:170	0x1001cdf74		b9808be4		MOVW 136(RSP), R4							
  serialize.go:170	0x1001cdf78		29519be5		LDPW 140(RSP), (R5, R6)							
  serialize.go:170	0x1001cdf7c		79412be7		MOVHU 148(RSP), R7							
  serialize.go:170	0x1001cdf80		79412fe8		MOVHU 150(RSP), R8							
  serialize.go:170	0x1001cdf84		aa1f03e9		MOVD ZR, R9								
  serialize.go:170	0x1001cdf88		aa1f03ea		MOVD ZR, R10								
  serialize.go:170	0x1001cdf8c		aa1f03eb		MOVD ZR, R11								
  serialize.go:170	0x1001cdf90		aa1f03ec		MOVD ZR, R12								
  serialize.go:170	0x1001cdf94		aa1f03ed		MOVD ZR, R13								
  serialize.go:170	0x1001cdf98		9400023e		CALL github.com/2018kguo/phosphor.(*Terminal).appendStyle(SB)		
  serialize.go:171	0x1001cdf9c		a908ffff		STP (ZR, ZR), 136(RSP)							
  serialize.go:182	0x1001cdfa0		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:182	0x1001cdfa4		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:183	0x1001cdfa8		394827e9		MOVBU 521(RSP), R9							
  serialize.go:173	0x1001cdfac		aa0203e3		MOVD R2, R3								
  serialize.go:173	0x1001cdfb0		aa0103e2		MOVD R1, R2								
  serialize.go:173	0x1001cdfb4		aa0003e1		MOVD R0, R1								
  terminal.go:2438	0x1001cdfb8		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:173	0x1001cdfbc		79412be6		MOVHU 148(RSP), R6							
  serialize.go:173	0x1001cdfc0		36400366		TBZ $8, R6, 27(PC)							
  serialize.go:264	0x1001cdfc4		91001446		ADD $5, R2, R6								
  serialize.go:264	0x1001cdfc8		eb06007f		CMP R6, R3								
  serialize.go:264	0x1001cdfcc		54000222		BCS 17(PC)								
  serialize.go:173	0x1001cdfd0		f900abe2		MOVD R2, 336(RSP)							
  serialize.go:264	0x1001cdfd4		aa0103e0		MOVD R1, R0								
  serialize.go:264	0x1001cdfd8		aa0603e1		MOVD R6, R1								
  serialize.go:264	0x1001cdfdc		aa0303e2		MOVD R3, R2								
  serialize.go:264	0x1001cdfe0		d28000a3		MOVD $5, R3								
  serialize.go:264	0x1001cdfe4		b0001504		ADRP 2756608(PC), R4							
  serialize.go:264	0x1001cdfe8		91388084		ADD $3616, R4, R4							
  serialize.go:264	0x1001cdfec		97fac275		CALL runtime.growslice(SB)						
  serialize.go:182	0x1001cdff0		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:182	0x1001cdff4		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:183	0x1001cdff8		394827e9		MOVBU 521(RSP), R9							
  serialize.go:264	0x1001cdffc		aa0103e6		MOVD R1, R6								
  serialize.go:264	0x1001ce000		aa0203e3		MOVD R2, R3								
  serialize.go:264	0x1001ce004		f940abe2		MOVD 336(RSP), R2							
  serialize.go:264	0x1001ce008		aa0003e1		MOVD R0, R1								
  terminal.go:2438	0x1001ce00c		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:264	0x1001ce010		8b020028		ADD R2, R1, R8								
  serialize.go:264	0x1001ce014		d28b636a		MOVD $23323, R10							
  serialize.go:264	0x1001ce018		f2a4460a		MOVK $(8752<<16), R10							
  serialize.go:264	0x1001ce01c		b822682a		MOVW R10, (R1)(R2)							
  serialize.go:264	0x1001ce020		d2800e2a		MOVD $113, R10								
  serialize.go:264	0x1001ce024		3900110a		MOVB R10, 4(R8)								
  serialize.go:174	0x1001ce028		14000002		JMP 2(PC)								
  serialize.go:176	0x1001ce02c		aa0203e6		MOVD R2, R6								
  serialize.go:176	0x1001ce030		79412fe8		MOVHU 150(RSP), R8							
  terminal.go:2435	0x1001ce034		aa0803ea		MOVD R8, R10								
  terminal.go:2435	0x1001ce038		3500006a		CBNZW R10, 3(PC)							
  serialize.go:176	0x1001ce03c		aa1f03e8		MOVD ZR, R8								
  terminal.go:2443	0x1001ce040		1400000b		JMP 11(PC)								
  terminal.go:2438	0x1001ce044		f940680a		MOVD 208(R0), R10							
  terminal.go:2438	0x1001ce048		a9472d4a		LDP 112(R10), (R10, R11)						
  terminal.go:2438	0x1001ce04c		d1000508		SUB $1, R8, R8								
  terminal.go:2438	0x1001ce050		d3403d0c		UBFX $0, R8, $16, R12							
  terminal.go:2438	0x1001ce054		eb0b019f		CMP R11, R12								
  terminal.go:2438	0x1001ce058		54001ec2		BCS 246(PC)								
  terminal.go:2438	0x1001ce05c		d37d3d08		UBFIZ $3, R8, $16, R8							
  terminal.go:2438	0x1001ce060		8b080508		ADD R8<<1, R8, R8							
  terminal.go:2438	0x1001ce064		8b080148		ADD R8, R10, R8								
  terminal.go:2438	0x1001ce068		f9400508		MOVD 8(R8), R8								
  serialize.go:176	0x1001ce06c		b4000848		CBZ R8, 66(PC)								
  serialize.go:250	0x1001ce070		910010c8		ADD $4, R6, R8								
  serialize.go:250	0x1001ce074		eb08007f		CMP R8, R3								
  serialize.go:250	0x1001ce078		54000222		BCS 17(PC)								
  serialize.go:176	0x1001ce07c		f900a7e6		MOVD R6, 328(RSP)							
  serialize.go:250	0x1001ce080		aa0103e0		MOVD R1, R0								
  serialize.go:250	0x1001ce084		aa0803e1		MOVD R8, R1								
  serialize.go:250	0x1001ce088		aa0303e2		MOVD R3, R2								
  serialize.go:250	0x1001ce08c		b27e03e3		ORR $4, ZR, R3								
  serialize.go:250	0x1001ce090		90001504		ADRP 2752512(PC), R4							
  serialize.go:250	0x1001ce094		91388084		ADD $3616, R4, R4							
  serialize.go:250	0x1001ce098		97fac24a		CALL runtime.growslice(SB)						
  serialize.go:182	0x1001ce09c		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:250	0x1001ce0a0		f940a7e6		MOVD 328(RSP), R6							
  serialize.go:182	0x1001ce0a4		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:183	0x1001ce0a8		394827e9		MOVBU 521(RSP), R9							
  serialize.go:250	0x1001ce0ac		aa0203e3		MOVD R2, R3								
  serialize.go:250	0x1001ce0b0		aa0103e8		MOVD R1, R8								
  serialize.go:250	0x1001ce0b4		aa0003e1		MOVD R0, R1								
  terminal.go:2438	0x1001ce0b8		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:250	0x1001ce0bc		d28ba36a		MOVD $23835, R10							
  serialize.go:250	0x1001ce0c0		f2a7670a		MOVK $(15160<<16), R10							
  serialize.go:250	0x1001ce0c4		b826682a		MOVW R10, (R1)(R6)							
  serialize.go:252	0x1001ce0c8		91000506		ADD $1, R8, R6								
  serialize.go:251	0x1001ce0cc		d503201f		NOOP									
  serialize.go:252	0x1001ce0d0		eb06007f		CMP R6, R3								
  terminal.go:2443	0x1001ce0d4		540001e2		BCS 15(PC)								
  serialize.go:252	0x1001ce0d8		aa0103e0		MOVD R1, R0								
  serialize.go:252	0x1001ce0dc		aa0603e1		MOVD R6, R1								
  serialize.go:252	0x1001ce0e0		aa0303e2		MOVD R3, R2								
  serialize.go:252	0x1001ce0e4		b24003e3		ORR $1, ZR, R3								
  serialize.go:252	0x1001ce0e8		90001504		ADRP 2752512(PC), R4							
  serialize.go:252	0x1001ce0ec		91388084		ADD $3616, R4, R4							
  serialize.go:252	0x1001ce0f0		97fac234		CALL runtime.growslice(SB)						
  serialize.go:182	0x1001ce0f4		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:182	0x1001ce0f8		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:183	0x1001ce0fc		394827e9		MOVBU 521(RSP), R9							
  serialize.go:252	0x1001ce100		aa0103e6		MOVD R1, R6								
  serialize.go:252	0x1001ce104		aa0203e3		MOVD R2, R3								
  serialize.go:252	0x1001ce108		aa0003e1		MOVD R0, R1								
  terminal.go:2438	0x1001ce10c		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:252	0x1001ce110		d10004c8		SUB $1, R6, R8								
  serialize.go:252	0x1001ce114		d280076a		MOVD $59, R10								
  serialize.go:252	0x1001ce118		3828682a		MOVB R10, (R1)(R8)							
  serialize.go:256	0x1001ce11c		910008c8		ADD $2, R6, R8								
  serialize.go:256	0x1001ce120		eb08007f		CMP R8, R3								
  serialize.go:256	0x1001ce124		54000222		BCS 17(PC)								
  serialize.go:252	0x1001ce128		f900a7e6		MOVD R6, 328(RSP)							
  serialize.go:256	0x1001ce12c		aa0103e0		MOVD R1, R0								
  serialize.go:256	0x1001ce130		aa0803e1		MOVD R8, R1								
  serialize.go:256	0x1001ce134		aa0303e2		MOVD R3, R2								
  serialize.go:256	0x1001ce138		b27f03e3		ORR $2, ZR, R3								
  serialize.go:256	0x1001ce13c		90001504		ADRP 2752512(PC), R4							
  serialize.go:256	0x1001ce140		91388084		ADD $3616, R4, R4							
  serialize.go:256	0x1001ce144		97fac21f		CALL runtime.growslice(SB)						
  serialize.go:182	0x1001ce148		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:256	0x1001ce14c		f940a7e6		MOVD 328(RSP), R6							
  serialize.go:182	0x1001ce150		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:183	0x1001ce154		394827e9		MOVBU 521(RSP), R9							
  serialize.go:256	0x1001ce158		aa0203e3		MOVD R2, R3								
  serialize.go:256	0x1001ce15c		aa0103e8		MOVD R1, R8								
  serialize.go:256	0x1001ce160		aa0003e1		MOVD R0, R1								
  terminal.go:2438	0x1001ce164		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:256	0x1001ce168		d28b836a		MOVD $23579, R10							
  serialize.go:256	0x1001ce16c		7826682a		MOVH R10, (R1)(R6)							
  serialize.go:177	0x1001ce170		14000002		JMP 2(PC)								
  serialize.go:179	0x1001ce174		aa0603e8		MOVD R6, R8								
  serialize.go:179	0x1001ce178		79412be6		MOVHU 148(RSP), R6							
  serialize.go:179	0x1001ce17c		d29fdfea		MOVD $65279, R10							
  serialize.go:179	0x1001ce180		6a0a00df		TSTW R10, R6								
  serialize.go:179	0x1001ce184		540002a1		BNE 21(PC)								
  serialize.go:179	0x1001ce188		b9408fe6		MOVWU 140(RSP), R6							
  serialize.go:179	0x1001ce18c		35000266		CBNZW R6, 19(PC)							
  serialize.go:179	0x1001ce190		b94093e6		MOVWU 144(RSP), R6							
  serialize.go:179	0x1001ce194		35000226		CBNZW R6, 17(PC)							
  serialize.go:179	0x1001ce198		79412fe6		MOVHU 150(RSP), R6							
  terminal.go:2435	0x1001ce19c		aa0603ea		MOVD R6, R10								
  terminal.go:2435	0x1001ce1a0		3500006a		CBNZW R10, 3(PC)							
  serialize.go:179	0x1001ce1a4		aa1f03e6		MOVD ZR, R6								
  serialize.go:179	0x1001ce1a8		1400000b		JMP 11(PC)								
  terminal.go:2438	0x1001ce1ac		f940680a		MOVD 208(R0), R10							
  terminal.go:2438	0x1001ce1b0		a9472d4a		LDP 112(R10), (R10, R11)						
  terminal.go:2438	0x1001ce1b4		d10004c6		SUB $1, R6, R6								
  terminal.go:2438	0x1001ce1b8		d3403ccc		UBFX $0, R6, $16, R12							
  terminal.go:2438	0x1001ce1bc		eb0b019f		CMP R11, R12								
  terminal.go:2438	0x1001ce1c0		54001362		BCS 155(PC)								
  terminal.go:2438	0x1001ce1c4		d37d3cc6		UBFIZ $3, R6, $16, R6							
  terminal.go:2438	0x1001ce1c8		8b0604c6		ADD R6<<1, R6, R6							
  terminal.go:2438	0x1001ce1cc		8b060146		ADD R6, R10, R6								
  terminal.go:2438	0x1001ce1d0		b94010c6		MOVWU 16(R6), R6							
  serialize.go:179	0x1001ce1d4		34000306		CBZW R6, 24(PC)								
  serialize.go:180	0x1001ce1d8		91001106		ADD $4, R8, R6								
  serialize.go:180	0x1001ce1dc		eb06007f		CMP R6, R3								
  serialize.go:180	0x1001ce1e0		54000222		BCS 17(PC)								
  serialize.go:179	0x1001ce1e4		f900b3e8		MOVD R8, 352(RSP)							
  serialize.go:180	0x1001ce1e8		aa0103e0		MOVD R1, R0								
  serialize.go:180	0x1001ce1ec		aa0603e1		MOVD R6, R1								
  serialize.go:180	0x1001ce1f0		aa0303e2		MOVD R3, R2								
  serialize.go:180	0x1001ce1f4		b27e03e3		ORR $4, ZR, R3								
  serialize.go:180	0x1001ce1f8		90001504		ADRP 2752512(PC), R4							
  serialize.go:180	0x1001ce1fc		91388084		ADD $3616, R4, R4							
  serialize.go:180	0x1001ce200		97fac1f0		CALL runtime.growslice(SB)						
  serialize.go:182	0x1001ce204		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:182	0x1001ce208		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:180	0x1001ce20c		f940b3e8		MOVD 352(RSP), R8							
  serialize.go:183	0x1001ce210		394827e9		MOVBU 521(RSP), R9							
  serialize.go:180	0x1001ce214		aa0203e3		MOVD R2, R3								
  serialize.go:180	0x1001ce218		aa0103e6		MOVD R1, R6								
  serialize.go:180	0x1001ce21c		aa0003e1		MOVD R0, R1								
  serialize.go:187	0x1001ce220		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:180	0x1001ce224		d28b636a		MOVD $23323, R10							
  serialize.go:180	0x1001ce228		f2ada60a		MOVK $(27952<<16), R10							
  serialize.go:180	0x1001ce22c		b828682a		MOVW R10, (R1)(R8)							
  serialize.go:180	0x1001ce230		14000002		JMP 2(PC)								
  serialize.go:182	0x1001ce234		aa0803e6		MOVD R8, R6								
  serialize.go:182	0x1001ce238		cb0500e5		SUB R5, R7, R5								
  serialize.go:182	0x1001ce23c		f10000bf		CMP $0, R5								
  serialize.go:182	0x1001ce240		540002ed		BLE 23(PC)								
  serialize.go:182	0x1001ce244		f9008fe5		MOVD R5, 280(RSP)							
  serialize.go:183	0x1001ce248		36000369		TBZ $0, R9, 27(PC)							
  serialize.go:184	0x1001ce24c		910004c6		ADD $1, R6, R6								
  serialize.go:184	0x1001ce250		eb06007f		CMP R6, R3								
  serialize.go:184	0x1001ce254		540001c2		BCS 14(PC)								
  serialize.go:184	0x1001ce258		aa0103e0		MOVD R1, R0								
  serialize.go:184	0x1001ce25c		aa0603e1		MOVD R6, R1								
  serialize.go:184	0x1001ce260		aa0303e2		MOVD R3, R2								
  serialize.go:184	0x1001ce264		b24003e3		ORR $1, ZR, R3								
  serialize.go:184	0x1001ce268		90001504		ADRP 2752512(PC), R4							
  serialize.go:184	0x1001ce26c		91388084		ADD $3616, R4, R4							
  serialize.go:184	0x1001ce270		97fac1d4		CALL runtime.growslice(SB)						
  varint.go:42		0x1001ce274		f9408fe5		MOVD 280(RSP), R5							
  serialize.go:187	0x1001ce278		394827e9		MOVBU 521(RSP), R9							
  serialize.go:184	0x1001ce27c		aa0103e6		MOVD R1, R6								
  serialize.go:184	0x1001ce280		aa0003e1		MOVD R0, R1								
  serialize.go:184	0x1001ce284		aa0203e3		MOVD R2, R3								
  serialize.go:187	0x1001ce288		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:184	0x1001ce28c		d10004c7		SUB $1, R6, R7								
  serialize.go:184	0x1001ce290		3827683f		MOVB ZR, (R1)(R7)							
  serialize.go:182	0x1001ce294		aa0503e2		MOVD R5, R2								
  varint.go:42		0x1001ce298		1400003b		JMP 59(PC)								
  serialize.go:199	0x1001ce29c		aa0103e0		MOVD R1, R0								
  serialize.go:199	0x1001ce2a0		aa0603e1		MOVD R6, R1								
  serialize.go:199	0x1001ce2a4		aa0303e2		MOVD R3, R2								
  serialize.go:199	0x1001ce2a8		a97ffbfd		LDP -8(RSP), (R29, R30)							
  serialize.go:199	0x1001ce2ac		910703ff		ADD $448, RSP, RSP							
  serialize.go:199	0x1001ce2b0		d65f03c0		RET									
  serialize.go:187	0x1001ce2b4		f9400007		MOVD (R0), R7								
  serialize.go:187	0x1001ce2b8		f10004ff		CMP $1, R7								
  serialize.go:187	0x1001ce2bc		54000081		BNE 4(PC)								
  serialize.go:187	0x1001ce2c0		37000069		TBNZ $0, R9, 3(PC)							
  serialize.go:190	0x1001ce2c4		aa1f03e0		MOVD ZR, R0								
  serialize.go:190	0x1001ce2c8		14000028		JMP 40(PC)								
  serialize.go:194	0x1001ce2cc		aa1f03e0		MOVD ZR, R0								
  serialize.go:194	0x1001ce2d0		14000005		JMP 5(PC)								
  serialize.go:195	0x1001ce2d4		d10004c7		SUB $1, R6, R7								
  serialize.go:195	0x1001ce2d8		b27b03e8		ORR $32, ZR, R8								
  serialize.go:195	0x1001ce2dc		38276828		MOVB R8, (R1)(R7)							
  serialize.go:194	0x1001ce2e0		91000400		ADD $1, R0, R0								
  serialize.go:194	0x1001ce2e4		eb05001f		CMP R5, R0								
  serialize.go:194	0x1001ce2e8		54fffdaa		BGE -19(PC)								
  serialize.go:195	0x1001ce2ec		910004c6		ADD $1, R6, R6								
  serialize.go:195	0x1001ce2f0		eb06007f		CMP R6, R3								
  serialize.go:195	0x1001ce2f4		54ffff02		BCS -8(PC)								
  serialize.go:194	0x1001ce2f8		f900bfe0		MOVD R0, 376(RSP)							
  serialize.go:195	0x1001ce2fc		aa0103e0		MOVD R1, R0								
  serialize.go:195	0x1001ce300		aa0603e1		MOVD R6, R1								
  serialize.go:195	0x1001ce304		aa0303e2		MOVD R3, R2								
  serialize.go:195	0x1001ce308		b24003e3		ORR $1, ZR, R3								
  serialize.go:195	0x1001ce30c		90001504		ADRP 2752512(PC), R4							
  serialize.go:195	0x1001ce310		91388084		ADD $3616, R4, R4							
  serialize.go:195	0x1001ce314		97fac1ab		CALL runtime.growslice(SB)						
  serialize.go:194	0x1001ce318		f9408fe5		MOVD 280(RSP), R5							
  serialize.go:195	0x1001ce31c		aa0103e6		MOVD R1, R6								
  serialize.go:195	0x1001ce320		aa0003e1		MOVD R0, R1								
  serialize.go:195	0x1001ce324		aa0203e3		MOVD R2, R3								
  serialize.go:194	0x1001ce328		f940bfe0		MOVD 376(RSP), R0							
  serialize.go:195	0x1001ce32c		17ffffea		JMP -22(PC)								
  serialize.go:190	0x1001ce330		f900bfe0		MOVD R0, 376(RSP)							
  serialize.go:191	0x1001ce334		d503201f		NOOP									
  utf8.go:415		0x1001ce338		aa0103e0		MOVD R1, R0								
  utf8.go:415		0x1001ce33c		aa0603e1		MOVD R6, R1								
  utf8.go:415		0x1001ce340		aa0303e2		MOVD R3, R2								
  utf8.go:415		0x1001ce344		d28ea983		MOVD $30028, R3								
  utf8.go:415		0x1001ce348		97fb588e		CALL unicode/utf8.appendRuneNonASCII(SB)				
  serialize.go:190	0x1001ce34c		f940bfe4		MOVD 376(RSP), R4							
  serialize.go:190	0x1001ce350		91000484		ADD $1, R4, R4								
  serialize.go:190	0x1001ce354		f9408fe5		MOVD 280(RSP), R5							
  serialize.go:190	0x1001ce358		aa0203e3		MOVD R2, R3								
  serialize.go:190	0x1001ce35c		aa0103e6		MOVD R1, R6								
  serialize.go:190	0x1001ce360		aa0003e1		MOVD R0, R1								
  serialize.go:190	0x1001ce364		aa0403e0		MOVD R4, R0								
  serialize.go:190	0x1001ce368		eb05001f		CMP R5, R0								
  serialize.go:190	0x1001ce36c		54fffe2b		BLT -15(PC)								
  serialize.go:190	0x1001ce370		17ffffcb		JMP -53(PC)								
  varint.go:43		0x1001ce374		d10004c7		SUB $1, R6, R7								
  varint.go:43		0x1001ce378		b279e0a8		ORR $-128, R5, R8							
  varint.go:43		0x1001ce37c		38276828		MOVB R8, (R1)(R7)							
  varint.go:44		0x1001ce380		d347fca5		LSR $7, R5, R5								
  varint.go:42		0x1001ce384		f9007be5		MOVD R5, 240(RSP)							
  varint.go:42		0x1001ce388		f10200bf		CMP $128, R5								
  varint.go:42		0x1001ce38c		54000263		BCC 19(PC)								
  varint.go:43		0x1001ce390		910004c6		ADD $1, R6, R6								
  varint.go:43		0x1001ce394		eb06007f		CMP R6, R3								
  varint.go:43		0x1001ce398		54fffee2		BCS -9(PC)								
  varint.go:43		0x1001ce39c		aa0103e0		MOVD R1, R0								
  varint.go:43		0x1001ce3a0		aa0603e1		MOVD R6, R1								
  varint.go:43		0x1001ce3a4		aa0303e2		MOVD R3, R2								
  varint.go:43		0x1001ce3a8		b24003e3		ORR $1, ZR, R3								
  varint.go:43		0x1001ce3ac		90001504		ADRP 2752512(PC), R4							
  varint.go:43		0x1001ce3b0		91388084		ADD $3616, R4, R4							
  varint.go:43		0x1001ce3b4		97fac183		CALL runtime.growslice(SB)						
  varint.go:43		0x1001ce3b8		f9407be5		MOVD 240(RSP), R5							
  serialize.go:187	0x1001ce3bc		394827e9		MOVBU 521(RSP), R9							
  varint.go:44		0x1001ce3c0		aa0103e6		MOVD R1, R6								
  varint.go:44		0x1001ce3c4		aa0003e1		MOVD R0, R1								
  varint.go:44		0x1001ce3c8		aa0203e3		MOVD R2, R3								
  serialize.go:187	0x1001ce3cc		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:194	0x1001ce3d0		f9408fe2		MOVD 280(RSP), R2							
  varint.go:43		0x1001ce3d4		17ffffe8		JMP -24(PC)								
  varint.go:46		0x1001ce3d8		910004c6		ADD $1, R6, R6								
  varint.go:46		0x1001ce3dc		eb06007f		CMP R6, R3								
  varint.go:46		0x1001ce3e0		540001e2		BCS 15(PC)								
  varint.go:46		0x1001ce3e4		aa0103e0		MOVD R1, R0								
  varint.go:46		0x1001ce3e8		aa0603e1		MOVD R6, R1								
  varint.go:46		0x1001ce3ec		aa0303e2		MOVD R3, R2								
  varint.go:46		0x1001ce3f0		b24003e3		ORR $1, ZR, R3								
  varint.go:46		0x1001ce3f4		90001504		ADRP 2752512(PC), R4							
  varint.go:46		0x1001ce3f8		91388084		ADD $3616, R4, R4							
  varint.go:46		0x1001ce3fc		97fac171		CALL runtime.growslice(SB)						
  varint.go:46		0x1001ce400		f9407be5		MOVD 240(RSP), R5							
  serialize.go:187	0x1001ce404		394827e9		MOVBU 521(RSP), R9							
  serialize.go:185	0x1001ce408		aa0103e6		MOVD R1, R6								
  serialize.go:185	0x1001ce40c		aa0003e1		MOVD R0, R1								
  serialize.go:185	0x1001ce410		aa0203e3		MOVD R2, R3								
  serialize.go:187	0x1001ce414		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:194	0x1001ce418		f9408fe2		MOVD 280(RSP), R2							
  varint.go:46		0x1001ce41c		d10004c7		SUB $1, R6, R7								
  varint.go:46		0x1001ce420		38276825		MOVB R5, (R1)(R7)							
  serialize.go:194	0x1001ce424		aa0203e5		MOVD R2, R5								
  serialize.go:185	0x1001ce428		17ffffa3		JMP -93(PC)								
  terminal.go:2438	0x1001ce42c		97fad6bd		CALL runtime.panicBounds(SB)						
  terminal.go:2438	0x1001ce430		97fad6bc		CALL runtime.panicBounds(SB)						
  serialize.go:143	0x1001ce434		97fad6bb		CALL runtime.panicBounds(SB)						
  slices.go:426		0x1001ce438		97fad6ba		CALL runtime.panicBounds(SB)						
  serialize.go:164	0x1001ce43c		aa1103e0		MOVD R17, R0								
  serialize.go:164	0x1001ce440		aa1403e1		MOVD R20, R1								
  serialize.go:164	0x1001ce444		97fad6b7		CALL runtime.panicBounds(SB)						
  serialize.go:131	0x1001ce448		910005ce		ADD $1, R14, R14							
  serialize.go:130	0x1001ce44c		eb0501df		CMP R5, R14								
  serialize.go:130	0x1001ce450		540003ca		BGE 30(PC)								
  serialize.go:130	0x1001ce454		54001322		BCS 153(PC)								
  serialize.go:130	0x1001ce458		8b0e108f		ADD R14<<4, R4, R15							
  serialize.go:130	0x1001ce45c		b98001f0		MOVW (R15), R16								
  serialize.go:130	0x1001ce460		35000310		CBNZW R16, 24(PC)							
  serialize.go:130	0x1001ce464		a94041ef		LDP (R15), (R15, R16)							
  serialize.go:130	0x1001ce468		910273fb		ADD $156, RSP, R27							
  serialize.go:130	0x1001ce46c		a900436f		STP (R15, R16), (R27)							
  serialize.go:88	0x1001ce470		295443ef		LDPW 160(RSP), (R15, R16)						
  serialize.go:130	0x1001ce474		b9808bf1		MOVW 136(RSP), R17							
  serialize.go:130	0x1001ce478		35000211		CBNZW R17, 16(PC)							
  serialize.go:130	0x1001ce47c		b9408ff1		MOVWU 140(RSP), R17							
  serialize.go:130	0x1001ce480		6b1101ff		CMPW R17, R15								
  serialize.go:130	0x1001ce484		540001a1		BNE 13(PC)								
  serialize.go:130	0x1001ce488		b94093ef		MOVWU 144(RSP), R15							
  serialize.go:130	0x1001ce48c		6b0f021f		CMPW R15, R16								
  serialize.go:130	0x1001ce490		54000141		BNE 10(PC)								
  serialize.go:130	0x1001ce494		79412bef		MOVHU 148(RSP), R15							
  serialize.go:88	0x1001ce498		794153f0		MOVHU 168(RSP), R16							
  serialize.go:130	0x1001ce49c		6b1001ff		CMPW R16, R15								
  serialize.go:130	0x1001ce4a0		540000c1		BNE 6(PC)								
  serialize.go:130	0x1001ce4a4		79412fef		MOVHU 150(RSP), R15							
  serialize.go:88	0x1001ce4a8		794157f0		MOVHU 170(RSP), R16							
  serialize.go:130	0x1001ce4ac		6b1001ff		CMPW R16, R15								
  serialize.go:130	0x1001ce4b0		9a9f17ef		CSET EQ, R15								
  serialize.go:130	0x1001ce4b4		14000006		JMP 6(PC)								
  serialize.go:130	0x1001ce4b8		aa1f03ef		MOVD ZR, R15								
  serialize.go:130	0x1001ce4bc		14000004		JMP 4(PC)								
  serialize.go:130	0x1001ce4c0		aa1f03ef		MOVD ZR, R15								
  serialize.go:130	0x1001ce4c4		14000002		JMP 2(PC)								
  serialize.go:130	0x1001ce4c8		aa1f03ef		MOVD ZR, R15								
  serialize.go:130	0x1001ce4cc		3707fbef		TBNZ $0, R15, -33(PC)							
  serialize.go:130	0x1001ce4d0		f9009bee		MOVD R14, 304(RSP)							
  serialize.go:133	0x1001ce4d4		9100044f		ADD $1, R2, R15								
  serialize.go:133	0x1001ce4d8		eb0f007f		CMP R15, R3								
  serialize.go:133	0x1001ce4dc		540002a2		BCS 21(PC)								
  serialize.go:133	0x1001ce4e0		aa0103e0		MOVD R1, R0								
  serialize.go:133	0x1001ce4e4		aa0f03e1		MOVD R15, R1								
  serialize.go:133	0x1001ce4e8		aa0303e2		MOVD R3, R2								
  serialize.go:133	0x1001ce4ec		b24003e3		ORR $1, ZR, R3								
  serialize.go:133	0x1001ce4f0		90001504		ADRP 2752512(PC), R4							
  serialize.go:133	0x1001ce4f4		91388084		ADD $3616, R4, R4							
  serialize.go:133	0x1001ce4f8		97fac132		CALL runtime.growslice(SB)						
  serialize.go:118	0x1001ce4fc		f940f7e4		MOVD 488(RSP), R4							
  serialize.go:117	0x1001ce500		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:142	0x1001ce504		f940ffe6		MOVD 504(RSP), R6							
  serialize.go:169	0x1001ce508		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:134	0x1001ce50c		f94097e8		MOVD 296(RSP), R8							
  serialize.go:128	0x1001ce510		394827e9		MOVBU 521(RSP), R9							
  serialize.go:117	0x1001ce514		b9409bea		MOVWU 152(RSP), R10							
  serialize.go:117	0x1001ce518		f9408beb		MOVD 272(RSP), R11							
  serialize.go:134	0x1001ce51c		f9409bee		MOVD 304(RSP), R14							
  serialize.go:133	0x1001ce520		aa0103ef		MOVD R1, R15								
  serialize.go:133	0x1001ce524		aa0003e1		MOVD R0, R1								
  serialize.go:133	0x1001ce528		aa0203e3		MOVD R2, R3								
  serialize.go:152	0x1001ce52c		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:133	0x1001ce530		d10005f0		SUB $1, R15, R16							
  serialize.go:133	0x1001ce534		3830683f		MOVB ZR, (R1)(R16)							
  serialize.go:134	0x1001ce538		cb0801d0		SUB R8, R14, R16							
  varint.go:42		0x1001ce53c		14000005		JMP 5(PC)								
  varint.go:43		0x1001ce540		d10005f1		SUB $1, R15, R17							
  varint.go:43		0x1001ce544		b279e213		ORR $-128, R16, R19							
  varint.go:43		0x1001ce548		38316833		MOVB R19, (R1)(R17)							
  varint.go:44		0x1001ce54c		d347fe10		LSR $7, R16, R16							
  varint.go:42		0x1001ce550		f9007ff0		MOVD R16, 248(RSP)							
  varint.go:42		0x1001ce554		f102021f		CMP $128, R16								
  varint.go:42		0x1001ce558		54000343		BCC 26(PC)								
  varint.go:43		0x1001ce55c		910005ef		ADD $1, R15, R15							
  varint.go:43		0x1001ce560		eb0f007f		CMP R15, R3								
  varint.go:43		0x1001ce564		54fffee2		BCS -9(PC)								
  varint.go:43		0x1001ce568		aa0103e0		MOVD R1, R0								
  varint.go:43		0x1001ce56c		aa0f03e1		MOVD R15, R1								
  varint.go:43		0x1001ce570		aa0303e2		MOVD R3, R2								
  varint.go:43		0x1001ce574		b24003e3		ORR $1, ZR, R3								
  varint.go:43		0x1001ce578		90001504		ADRP 2752512(PC), R4							
  varint.go:43		0x1001ce57c		91388084		ADD $3616, R4, R4							
  varint.go:43		0x1001ce580		97fac110		CALL runtime.growslice(SB)						
  serialize.go:118	0x1001ce584		f940f7e4		MOVD 488(RSP), R4							
  serialize.go:117	0x1001ce588		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:142	0x1001ce58c		f940ffe6		MOVD 504(RSP), R6							
  serialize.go:169	0x1001ce590		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:135	0x1001ce594		f94097e8		MOVD 296(RSP), R8							
  serialize.go:128	0x1001ce598		394827e9		MOVBU 521(RSP), R9							
  serialize.go:117	0x1001ce59c		b9409bea		MOVWU 152(RSP), R10							
  serialize.go:117	0x1001ce5a0		f9408beb		MOVD 272(RSP), R11							
  serialize.go:135	0x1001ce5a4		f9409bee		MOVD 304(RSP), R14							
  varint.go:43		0x1001ce5a8		f9407ff0		MOVD 248(RSP), R16							
  varint.go:44		0x1001ce5ac		aa0103ef		MOVD R1, R15								
  varint.go:44		0x1001ce5b0		aa0003e1		MOVD R0, R1								
  varint.go:44		0x1001ce5b4		aa0203e3		MOVD R2, R3								
  serialize.go:152	0x1001ce5b8		f940e7e0		MOVD 456(RSP), R0							
  varint.go:43		0x1001ce5bc		17ffffe1		JMP -31(PC)								
  varint.go:46		0x1001ce5c0		910005ef		ADD $1, R15, R15							
  varint.go:46		0x1001ce5c4		eb0f007f		CMP R15, R3								
  varint.go:46		0x1001ce5c8		540002c2		BCS 22(PC)								
  varint.go:46		0x1001ce5cc		aa0103e0		MOVD R1, R0								
  varint.go:46		0x1001ce5d0		aa0f03e1		MOVD R15, R1								
  varint.go:46		0x1001ce5d4		aa0303e2		MOVD R3, R2								
  varint.go:46		0x1001ce5d8		b24003e3		ORR $1, ZR, R3								
  varint.go:46		0x1001ce5dc		90001504		ADRP 2752512(PC), R4							
  varint.go:46		0x1001ce5e0		91388084		ADD $3616, R4, R4							
  varint.go:46		0x1001ce5e4		97fac0f7		CALL runtime.growslice(SB)						
  serialize.go:118	0x1001ce5e8		f940f7e4		MOVD 488(RSP), R4							
  serialize.go:117	0x1001ce5ec		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:142	0x1001ce5f0		f940ffe6		MOVD 504(RSP), R6							
  serialize.go:169	0x1001ce5f4		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:135	0x1001ce5f8		f94097e8		MOVD 296(RSP), R8							
  serialize.go:128	0x1001ce5fc		394827e9		MOVBU 521(RSP), R9							
  serialize.go:117	0x1001ce600		b9409bea		MOVWU 152(RSP), R10							
  serialize.go:117	0x1001ce604		f9408beb		MOVD 272(RSP), R11							
  serialize.go:135	0x1001ce608		f9409bee		MOVD 304(RSP), R14							
  varint.go:46		0x1001ce60c		f9407ff0		MOVD 248(RSP), R16							
  serialize.go:134	0x1001ce610		aa0103ef		MOVD R1, R15								
  serialize.go:134	0x1001ce614		aa0003e1		MOVD R0, R1								
  serialize.go:134	0x1001ce618		aa0203e3		MOVD R2, R3								
  serialize.go:152	0x1001ce61c		f940e7e0		MOVD 456(RSP), R0							
  varint.go:46		0x1001ce620		d10005f1		SUB $1, R15, R17							
  varint.go:46		0x1001ce624		38316830		MOVB R16, (R1)(R17)							
  serialize.go:135	0x1001ce628		14000005		JMP 5(PC)								
  serialize.go:136	0x1001ce62c		d10005f0		SUB $1, R15, R16							
  serialize.go:136	0x1001ce630		b27b03f1		ORR $32, ZR, R17							
  serialize.go:136	0x1001ce634		38306831		MOVB R17, (R1)(R16)							
  serialize.go:135	0x1001ce638		91000508		ADD $1, R8, R8								
  serialize.go:135	0x1001ce63c		eb0e011f		CMP R14, R8								
  serialize.go:135	0x1001ce640		5400034a		BGE 26(PC)								
  serialize.go:136	0x1001ce644		910005ef		ADD $1, R15, R15							
  serialize.go:136	0x1001ce648		eb0f007f		CMP R15, R3								
  serialize.go:136	0x1001ce64c		54ffff02		BCS -8(PC)								
  serialize.go:135	0x1001ce650		f900afe8		MOVD R8, 344(RSP)							
  serialize.go:136	0x1001ce654		aa0103e0		MOVD R1, R0								
  serialize.go:136	0x1001ce658		aa0f03e1		MOVD R15, R1								
  serialize.go:136	0x1001ce65c		aa0303e2		MOVD R3, R2								
  serialize.go:136	0x1001ce660		b24003e3		ORR $1, ZR, R3								
  serialize.go:136	0x1001ce664		90001504		ADRP 2752512(PC), R4							
  serialize.go:136	0x1001ce668		91388084		ADD $3616, R4, R4							
  serialize.go:136	0x1001ce66c		97fac0d5		CALL runtime.growslice(SB)						
  serialize.go:118	0x1001ce670		f940f7e4		MOVD 488(RSP), R4							
  serialize.go:117	0x1001ce674		f940a3e5		MOVD 320(RSP), R5							
  serialize.go:142	0x1001ce678		f940ffe6		MOVD 504(RSP), R6							
  serialize.go:169	0x1001ce67c		f9409fe7		MOVD 312(RSP), R7							
  serialize.go:135	0x1001ce680		f940afe8		MOVD 344(RSP), R8							
  serialize.go:128	0x1001ce684		394827e9		MOVBU 521(RSP), R9							
  serialize.go:117	0x1001ce688		b9409bea		MOVWU 152(RSP), R10							
  serialize.go:117	0x1001ce68c		f9408beb		MOVD 272(RSP), R11							
  serialize.go:135	0x1001ce690		f9409bee		MOVD 304(RSP), R14							
  serialize.go:136	0x1001ce694		aa0103ef		MOVD R1, R15								
  serialize.go:136	0x1001ce698		aa0003e1		MOVD R0, R1								
  serialize.go:136	0x1001ce69c		aa0203e3		MOVD R2, R3								
  serialize.go:152	0x1001ce6a0		f940e7e0		MOVD 456(RSP), R0							
  serialize.go:136	0x1001ce6a4		17ffffe2		JMP -30(PC)								
  serialize.go:136	0x1001ce6a8		b27b03f1		ORR $32, ZR, R17							
  serialize.go:118	0x1001ce6ac		aa0803f0		MOVD R8, R16								
  serialize.go:118	0x1001ce6b0		aa0f03ee		MOVD R15, R14								
  serialize.go:138	0x1001ce6b4		17fffced		JMP -787(PC)								
  serialize.go:130	0x1001ce6b8		97fad61a		CALL runtime.panicBounds(SB)						
  serialize.go:118	0x1001ce6bc		97fad619		CALL runtime.panicBounds(SB)						
  serialize.go:113	0x1001ce6c0		aa0e03e5		MOVD R14, R5								
  serialize.go:113	0x1001ce6c4		b4ff9be5		CBZ R5, -801(PC)							
  serialize.go:113	0x1001ce6c8		d10004ae		SUB $1, R5, R14								
  serialize.go:113	0x1001ce6cc		8b0e108f		ADD R14<<4, R4, R15							
  serialize.go:113	0x1001ce6d0		b98001f0		MOVW (R15), R16								
  serialize.go:113	0x1001ce6d4		b0001cfb		ADRP 3788800(PC), R27							
  serialize.go:113	0x1001ce6d8		b98da371		MOVW 3488(R27), R17							
  serialize.go:113	0x1001ce6dc		6b10023f		CMPW R16, R17								
  serialize.go:113	0x1001ce6e0		54ff9b01		BNE -808(PC)								
  serialize.go:113	0x1001ce6e4		b94005f0		MOVWU 4(R15), R16							
  serialize.go:113	0x1001ce6e8		b0001cfb		ADRP 3788800(PC), R27							
  serialize.go:113	0x1001ce6ec		b94da771		MOVWU 3492(R27), R17							
  serialize.go:113	0x1001ce6f0		6b10023f		CMPW R16, R17								
  serialize.go:113	0x1001ce6f4		54ff9a61		BNE -813(PC)								
  serialize.go:113	0x1001ce6f8		b94009f0		MOVWU 8(R15), R16							
  serialize.go:113	0x1001ce6fc		b0001cfb		ADRP 3788800(PC), R27							
  serialize.go:113	0x1001ce700		b94dab71		MOVWU 3496(R27), R17							
  serialize.go:113	0x1001ce704		6b10023f		CMPW R16, R17								
  serialize.go:113	0x1001ce708		54ff99c1		BNE -818(PC)								
  serialize.go:113	0x1001ce70c		794019f0		MOVHU 12(R15), R16							
  serialize.go:113	0x1001ce710		b0001cfb		ADRP 3788800(PC), R27							
  serialize.go:113	0x1001ce714		795b5b71		MOVHU 3500(R27), R17							
  serialize.go:113	0x1001ce718		6b11021f		CMPW R17, R16								
  serialize.go:113	0x1001ce71c		54ff9921		BNE -823(PC)								
  serialize.go:113	0x1001ce720		79401def		MOVHU 14(R15), R15							
  serialize.go:113	0x1001ce724		b0001cfb		ADRP 3788800(PC), R27							
  serialize.go:113	0x1001ce728		795b5f70		MOVHU 3502(R27), R16							
  serialize.go:113	0x1001ce72c		6b1001ff		CMPW R16, R15								
  serialize.go:113	0x1001ce730		54fffc80		BEQ -28(PC)								
  serialize.go:113	0x1001ce734		17fffcc3		JMP -829(PC)								
  serialize.go:109	0x1001ce738		97fad5fa		CALL runtime.panicBounds(SB)						
  terminal.go:2665	0x1001ce73c		aa0e03e5		MOVD R14, R5								
  terminal.go:2665	0x1001ce740		f10000bf		CMP $0, R5								
  terminal.go:2665	0x1001ce744		5400014d		BLE 10(PC)								
  terminal.go:2665	0x1001ce748		d10004ae		SUB $1, R5, R14								
  terminal.go:2665	0x1001ce74c		8b0e108f		ADD R14<<4, R4, R15							
  terminal.go:2679	0x1001ce750		a94041ef		LDP (R15), (R15, R16)							
  terminal.go:2680	0x1001ce754		b50000d0		CBNZ R16, 6(PC)								
  terminal.go:2680	0x1001ce758		b4ffff2f		CBZ R15, -7(PC)								
  terminal.go:2680	0x1001ce75c		f0001cdb		ADRP 3780608(PC), R27							
  terminal.go:2680	0x1001ce760		f947e770		MOVD 4040(R27), R16							
  terminal.go:2680	0x1001ce764		eb0f021f		CMP R15, R16								
  terminal.go:2665	0x1001ce768		54fffea0		BEQ -11(PC)								
  serialize.go:103	0x1001ce76c		eb0500ff		CMP R5, R7								
  serialize.go:103	0x1001ce770		5400004c		BGT 2(PC)								
  serialize.go:103	0x1001ce774		aa0703e5		MOVD R7, R5								
  serialize.go:105	0x1001ce778		aa0503e7		MOVD R5, R7								
  serialize.go:109	0x1001ce77c		aa0803e5		MOVD R8, R5								
  serialize.go:103	0x1001ce780		17fffca9		JMP -855(PC)								
  serialize.go:100	0x1001ce784		a90087e0		STP (R0, R1), 8(RSP)							
  serialize.go:100	0x1001ce788		a9018fe2		STP (R2, R3), 24(RSP)							
  serialize.go:100	0x1001ce78c		a90297e4		STP (R4, R5), 40(RSP)							
  serialize.go:100	0x1001ce790		a9039fe6		STP (R6, R7), 56(RSP)							
  serialize.go:100	0x1001ce794		390123e8		MOVB R8, 72(RSP)							
  serialize.go:100	0x1001ce798		390127e9		MOVB R9, 73(RSP)							
  serialize.go:100	0x1001ce79c		aa1e03e3		MOVD R30, R3								
  serialize.go:100	0x1001ce7a0		97facd60		CALL runtime.morestack_noctxt.abi0(SB)					
  serialize.go:100	0x1001ce7a4		a94087e0		LDP 8(RSP), (R0, R1)							
  serialize.go:100	0x1001ce7a8		a9418fe2		LDP 24(RSP), (R2, R3)							
  serialize.go:100	0x1001ce7ac		a94297e4		LDP 40(RSP), (R4, R5)							
  serialize.go:100	0x1001ce7b0		a9439fe6		LDP 56(RSP), (R6, R7)							
  serialize.go:100	0x1001ce7b4		394123e8		MOVBU 72(RSP), R8							
  serialize.go:100	0x1001ce7b8		394127e9		MOVBU 73(RSP), R9							
  serialize.go:100	0x1001ce7bc		17fffc8d		JMP github.com/2018kguo/phosphor.(*Terminal).appendRowHistory(SB)	
