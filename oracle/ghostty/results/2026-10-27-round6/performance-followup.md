# Recover the cost of bulk erasure

A semantic-preserving EL 2 change had replaced bulk zeroing with scalar stores. Restoring `clear` after half-wide repairs, plus skipping SGR 0 pool work for E=0, recovered the measured cost without changing damage, wrap/prompt flags, prefix storage, colored erase, layouts or snapshots. The dense-row probe was rejected.

Apple M5 Pro, macOS/arm64, Go 1.26.3; before `8d1511c`, after only those two changes. Same driver/data,1024-byte reads, GOMAXPROCS=1, seven alternating 750 ms pairs. Host speed changed between probes and early wall samples were interrupted: compare adjacent controls, not separate runs. All [accepted-change samples](performance-bulk-clear.jsonl) remain.

| Stream | Before wall MiB/s, seven samples | After wall MiB/s, seven samples | Wall median change | CPU median change |
|---|---|---|---:|---:|
| claude-synthetic | 194.0, 309.3, 314.3, 352.7, 346.2, 323.3, 321.4 | 228.2, 190.3, 337.1, 350.9, 354.8, 350.3, 353.5 | +9.00% | +8.25% |
| progress | 947.5, 980.4, 957.1, 971.5, 971.4, 1015.0, 999.0 | 1017.2, 1022.3, 995.8, 1007.8, 1004.9, 1029.7, 1020.2 | +4.71% | +4.11% |
| vim-recording | 299.7, 288.1, 300.4, 300.8, 300.0, 301.0, 301.4 | 308.5, 302.1, 316.4, 314.1, 309.0, 311.9, 323.1 | +3.82% | +4.26% |
| less-recording | 199.0, 208.9, 209.6, 198.6, 209.0, 203.5, 204.1 | 251.2, 247.8, 246.6, 246.4, 249.8, 247.3, 244.0 | +21.13% | +20.85% |
| logs | 432.2, 432.5, 463.4, 480.9, 463.8, 389.6, 417.1 | 455.9, 445.3, 453.6, 475.3, 489.2, 367.4, 426.7 | +4.86% | +3.52% |

CPU changes within adjacent pairs, in order:

| Stream | Seven paired CPU changes |
|---|---|
| claude-synthetic | +5.96%, +6.09%, +7.17%, +0.24%, +3.61%, +9.04%, +8.97% |
| progress | +4.11%, +4.34%, +5.07%, +3.67%, +3.24%, +1.67%, +1.80% |
| vim-recording | +4.52%, +6.25%, +4.61%, +4.31%, +4.03%, +3.54%, +7.27% |
| less-recording | +22.30%, +19.25%, +19.37%, +23.96%, +21.15%, +21.16%, +20.52% |
| logs | +5.24%, +2.42%, -1.23%, -0.51%, +5.52%, +3.09%, +1.42% |

Matched two-second `BenchmarkWriteStream/claude` profiles used the same 8192-byte harness:8d spent 170/2050 ms in fillCells (140 ms at scalar stores), versus v0.3's 40 ms cumulative EL 2 bulk-clear path. [Profile text](performance-claude-profiles.txt) supports the hotspot, not exact attribution of every percentage point; binary profiles stayed in the experiment directory.

[Row-only probe](performance-row-probe.jsonl): vim+2.7%, Claude-1.4%, less/progress within 0.7%, logs within 0.3%; seven 750 ms pairs except logs five 300 ms pairs. Excluded. [SGR 0-only probe](performance-sgr-probe.jsonl): progress+3.7%, vim+2.5%, Claude flat, less within 0.6%. Retained: E=0 guarantees default underline/no link. SGR 59 stayed unchanged.

Focused damage/prefix/semantic/wide/underline/link/saturation/continuation tests passed; warm forty-row writes stayed zero-allocation. Row-only recording allocations matched production (nvim 1/16 B, nano/man 0). These local measurements do not imply universal speedups.

Rerun each stream seven times, alternating binaries; build via the [driver](../../README.md):

```sh
GOMAXPROCS=1 PHOSPHOR_DIR=/path/to/checkout COMPARE_ENGINE=phosphor COMPARE_ROUNDS=1 COMPARE_STREAM=claude-synthetic COMPARE_CHUNK=1024 BINARY -mode throughput -duration 750ms
GOMAXPROCS=1 go test -run '^$' -bench '^BenchmarkWriteStream/claude$' -benchtime=2s -count=1 -cpuprofile=claude.cpu -o claude.test .
```

Binary SHA256: final-candidate `bb7cd704d558ed4d0c7b374284af079a582623f5a1a6e8f0de75fbd1d94745bb`; inline-row-probe `5cbb788ee2b7ba5c3d66688388bed8ebe5df464cff2a423b6c270da047911728`; sgr-reset-probe `d33f5f890999d87c7fd7c35f543dfaa4725b579bd6801d2b9027282dc8bdf7d0`; bulk-clear-probe `12c214173be5e714cb3798085dacdc13beaef3fd1af3b932d45ff56f89d3f473`.
