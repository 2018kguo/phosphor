# Restore bulk erasure performance

The retained change uses `clear` for default-background erasure after the existing half-wide repairs, and skips underline-pool resets when the pen has no rare attributes. Damage checks, wrap flags, shell annotations, prefix storage and colored erasure keep their existing behavior. No fixed-layout or snapshot changes.

Measured on Apple M5 Pro, macOS arm64, Go 1.26.3. Before is production `8d1511c`; after adds only these two changes. Same native driver and streams, 1024-byte reads, `GOMAXPROCS=1`, seven alternating before/after pairs of 750 ms. All raw samples are retained. The host slowed between earlier probes and the final run, with interrupted early wall samples; compare each probe only with its adjacent control.

| Stream | Before wall MiB/s, seven samples | After wall MiB/s, seven samples | Wall median change | CPU median change |
|---|---|---|---:|---:|
| claude-synthetic | 194.0, 309.3, 314.3, 352.7, 346.2, 323.3, 321.4 | 228.2, 190.3, 337.1, 350.9, 354.8, 350.3, 353.5 | +9.00% | +8.25% |
| progress | 947.5, 980.4, 957.1, 971.5, 971.4, 1015.0, 999.0 | 1017.2, 1022.3, 995.8, 1007.8, 1004.9, 1029.7, 1020.2 | +4.71% | +4.11% |
| vim-recording | 299.7, 288.1, 300.4, 300.8, 300.0, 301.0, 301.4 | 308.5, 302.1, 316.4, 314.1, 309.0, 311.9, 323.1 | +3.82% | +4.26% |
| less-recording | 199.0, 208.9, 209.6, 198.6, 209.0, 203.5, 204.1 | 251.2, 247.8, 246.6, 246.4, 249.8, 247.3, 244.0 | +21.13% | +20.85% |
| logs | 432.2, 432.5, 463.4, 480.9, 463.8, 389.6, 417.1 | 455.9, 445.3, 453.6, 475.3, 489.2, 367.4, 426.7 | +4.86% | +3.52% |

CPU change within each adjacent pair, in pair order:

| Stream | Seven paired CPU changes |
|---|---|
| claude-synthetic | +5.96%, +6.09%, +7.17%, +0.24%, +3.61%, +9.04%, +8.97% |
| progress | +4.11%, +4.34%, +5.07%, +3.67%, +3.24%, +1.67%, +1.80% |
| vim-recording | +4.52%, +6.25%, +4.61%, +4.31%, +4.03%, +3.54%, +7.27% |
| less-recording | +22.30%, +19.25%, +19.37%, +23.96%, +21.15%, +21.16%, +20.52% |
| logs | +5.24%, +2.42%, -1.23%, -0.51%, +5.52%, +3.09%, +1.42% |

These are local measurements with host scatter. They show a useful recovery on these workloads and do not establish a stable speedup for every terminal workload.

The matched two-second `BenchmarkWriteStream/claude` profiles use the unchanged 8192-byte test harness on v0.3.0 and the 8d production source. In the 8d profile, `fillCells` consumes 170 ms of 2.05 seconds of samples, including 140 ms at the scalar cell assignment. The v0.3 profile routes EL2 through bulk row clearing and spends 40 ms cumulatively in that path. The semantic-preserving EL2 route had retained a scalar zero-fill loop; restoring bulk zeroing recovers this cost without clearing shell annotations. Profiles are sampling evidence, not an exact attribution of every percentage point. Their text is in `performance-claude-profiles.txt`; binary profiles remain in the experiment directory.

Two isolated probes preceded the retained change:

- Sharing the existing inline prefix helper for dense writes removed row-function calls but improved only vim by 2.7%; Claude was -1.4%, less/progress were within 0.7%, and logs within 0.3%. This runtime change was excluded. Its raw file has seven 750 ms pairs except logs, which has five 300 ms pairs.
- Guarding only the two SGR0 branches improved progress by 3.7% and vim by 2.5%; Claude was flat and less within 0.6%. These guards are retained because E=0 guarantees default underline color and no hyperlink. SGR59 is unchanged.

Focused Go damage, cold/prefix storage, semantic, wide, underline/link, pool-saturation and continuation tests pass. Warm forty-row writes retain zero allocations. The excluded row probe also matched production recording allocations (nvim: 1 allocation/16 B; nano/man: zero); this count includes the newer alternate-screen lifetime behavior.

Reproduce using the existing `oracle/ghostty` build instructions, then run the same command once per binary in alternating order, seven times for each stream:

```sh
GOMAXPROCS=1 PHOSPHOR_DIR=/path/to/checkout COMPARE_ENGINE=phosphor COMPARE_ROUNDS=1 COMPARE_STREAM=claude-synthetic COMPARE_CHUNK=1024 BINARY -mode throughput -duration 750ms
GOMAXPROCS=1 go test -run '^$' -bench '^BenchmarkWriteStream/claude$' -benchtime=2s -count=1 -cpuprofile=claude.cpu -o claude.test .
```

Binary SHA-256:

- `final-candidate`: `bb7cd704d558ed4d0c7b374284af079a582623f5a1a6e8f0de75fbd1d94745bb`.
- `inline-row-probe`: `5cbb788ee2b7ba5c3d66688388bed8ebe5df464cff2a423b6c270da047911728`.
- `sgr-reset-probe`: `d33f5f890999d87c7fd7c35f543dfaa4725b579bd6801d2b9027282dc8bdf7d0`.
- `bulk-clear-probe`: `12c214173be5e714cb3798085dacdc13beaef3fd1af3b932d45ff56f89d3f473`.
