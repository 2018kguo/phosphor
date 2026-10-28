# Fifth comparison: shell prompts survive resize

This intermediate corpus checked whether shell row annotations and cursor/redraw state survive erase, screen switches and reflow. Direct parity and Phosphor continuation passed; native exporter failures remained visible.

Native `46e9dc9a927b2e7e7cba79e6902c176127dbdf2c`; unmodified public-ABI output for traces.json with [whole](trace-chunk-0.jsonl)/[one-byte](trace-chunk-1.jsonl)/[seven-byte](trace-chunk-7.jsonl) writes. Each run:194 cases/703 parity actions; direct 703/703, Phosphor snapshot/repaint/Compact 703/703, native continuation 618/703 (85 exporter failures retained). Two identity actions and one host-default action keep separate scopes/raw answers. These are finite case counts.

Cases distinguish raw primary/continuation marks, semantic cursor/redraw policy, history metadata, ED/EL, DECSTR pending wrap and live resize mapping. Growing height while ALT is active must use the actual inactive MAIN cursor, independent of DECSC/1049; a non-bottom MAIN cursor must not pull history.

Go/native tests passed; bounded fuzz totals 155,143 snapshot continuation and 87,759 history grammar. Terminal 560 B/cell 16 B; warm ASCII/input-only semantic state allocated zero times. Cursor bytes use existing padding; annotated rows share wrap metadata. [Driver and reproduction](../../README.md); raw files in this directory retain every difference.
