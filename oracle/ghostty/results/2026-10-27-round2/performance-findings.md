# Reuse SGR attribute codes

Keep the shared encoding table and range over it by reference. This removes a 192-byte stack copy for every style change. On Go 1.26.3, darwin/arm64, the compiled appendSGR frame falls from 336 to 144 bytes. The only product change is serialize.go: commit 0d6b7e872bca679be95bbac95e420d000c4450f4, based on 49f49b8.

No per-terminal fields, pointers, history policies, output bytes, or dependencies change. The table is package-private and read-only in use. The reference range matters: ranging over the array value would copy it again.

## Why this seam

A one-second profile on the baseline attributed about 37% of log CPU to appendRow including its ASCII copying, 16% to printASCII, 10% to rotation, and 8% to appendSGR. Disassembly showed the local attribute literal copied to the stack for each appendSGR call. Moving that existing literal is smaller than the earlier rejected hardware, word-packing, and unrolling approaches.

## Measurements and limits

The machine is an Apple M5 Pro running Go 1.26.3. It was shared with native compilation and unrelated work. Timings are therefore noisy, especially small differences. Root must remeasure the integrated head.

Six rotating control/table/guard rounds ran every existing WriteStream and Serialize case plus Replay for 200ms each. Median baseline/table speed ratios, where greater than one means faster:

| Work | Ratio |
| --- | ---: |
| vim | 1.104 |
| less | 0.990 |
| synthetic Claude output | 0.996 |
| scrolling logs | 1.041 |
| progress bars | 1.068 |
| visible repaint | 1.065 |
| repaint with history | 0.980 |
| Lines, unchanged control | 0.964 |
| ScrollbackLines, unchanged control | 1.108 |
| Replay | 1.017 |

The unchanged controls moving shows why these ratios must not become precise speed claims. An earlier five-round run showed larger repaint improvements, which the rotating run did not consistently reproduce.

Fixed-iteration process CPU measurements included startup and fixture construction. Three paired runs gave about 2.4% less log CPU and 2.2% less visible-repaint CPU, but history repaint was about 8.5% worse. That is material counter-evidence. A separate balanced four-pair history repeat after contention eased gave a 3% lower median CPU cost and 5% lower median wall time; all four CPU pairs favored the candidate by 0.2–5.1%. Treat the result as a modest candidate supported by removal of known work, not an established universal improvement.

The extra zero-text-attribute guard was rejected: its rotating median scrolling-log time was about 8% worse. It is absent from the commit. No further hypothesis was kept.

## Verification

- All 524,288 combinations of 16-bit attributes and eight foreground/background color pairs were byte-identical to the frozen 49f49b8 encoder. The comparison helper is an experiment artifact, not a committed implementation mirror.
- Full go test ./... and go vet ./... passed.
- FuzzTokens ran for 20 seconds with two workers and passed 82,906 executions.
- No race/full-platform claim is made here; root owns final integrated verification.

## Retained evidence

The three perf-final-*.txt files hold the rotating control, table and
rejected-guard samples. perf-cpu.jsonl retains the contradictory initial
CPU result, and perf-cpu-repeat.jsonl the four balanced history repeats.
perf-fuzz.txt records the bounded token fuzz run. The integrated report's
macOS and Linux files use the committed comparison driver against the final
runtime. Profiles, compiler listings, the temporary frozen encoder and
rejected implementation sketches were disposable experiment artifacts.
