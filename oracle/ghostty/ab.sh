#!/usr/bin/env bash
set -euo pipefail

# Binaries built from this same driver against old and new phosphor versions.
if [[ $# != 3 ]]; then
  echo 'usage: ghostty/ab.sh BASELINE CANDIDATE OUTPUT.jsonl' >&2
  exit 2
fi
baseline=$1
candidate=$2
output=$3
: "${PHOSPHOR_DIR:?Set PHOSPHOR_DIR to the phosphor checkout}"
repeats=${COMPARE_REPEATS:-6}
if ! [[ $repeats =~ ^[1-9][0-9]*$ ]] || [[ ${#repeats} -gt 9 ]]; then
  echo 'COMPARE_REPEATS must be an integer from 1 to 999999999' >&2
  exit 2
fi
: > "$output"
for stream in claude-synthetic logs progress vim-recording less-recording rec-nvim-recording rec-nano-recording rec-man-recording graphemes links; do
  for (( repeat=0; repeat<repeats; repeat++ )); do
    labels=(baseline phosphor ghostty)
    if (( repeat % 6 >= 3 )); then labels=(ghostty phosphor baseline); fi
    for position in 0 1 2; do
      label=${labels[$(( (position + repeat % 3) % 3 ))]}
      binary=$candidate
      engine=phosphor
      if [[ $label == baseline ]]; then binary=$baseline; fi
      if [[ $label == ghostty ]]; then engine=ghostty; fi
      COMPARE_LABEL=$label COMPARE_ENGINE=$engine COMPARE_ROUNDS=1 \
        COMPARE_OUTER_REPEAT=$repeat COMPARE_SAMPLE_ORDER=$position \
        COMPARE_STREAM=$stream COMPARE_CHUNK=${COMPARE_CHUNK:-1024} \
        "$binary" -mode throughput -duration "${COMPARE_DURATION:-250ms}" >> "$output"
    done
  done
done
