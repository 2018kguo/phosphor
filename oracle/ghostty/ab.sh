#!/usr/bin/env bash
set -euo pipefail

# Binaries built from this same driver against old and new phosphor versions.
baseline=$1
candidate=$2
output=$3
: "${PHOSPHOR_DIR:?Set PHOSPHOR_DIR to the phosphor checkout}"
: > "$output"
for stream in claude-synthetic logs progress vim-recording less-recording rec-nvim-recording rec-nano-recording rec-man-recording; do
  for round in 0 1 2 3 4; do
    labels=(baseline phosphor ghostty)
    if (( round % 2 )); then labels=(ghostty phosphor baseline); fi
    for label in "${labels[@]}"; do
      binary=$candidate
      engine=phosphor
      if [[ $label == baseline ]]; then binary=$baseline; fi
      if [[ $label == ghostty ]]; then engine=ghostty; fi
      COMPARE_LABEL=$label COMPARE_ENGINE=$engine COMPARE_ROUNDS=1 \
        COMPARE_STREAM=$stream COMPARE_CHUNK=${COMPARE_CHUNK:-1024} \
        "$binary" -mode throughput -duration "${COMPARE_DURATION:-250ms}" >> "$output"
    done
  done
done
