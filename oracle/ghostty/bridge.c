//go:build ghostty

#include "bridge.h"
#include <stdlib.h>
#ifdef __APPLE__
#include <mach/mach.h>
#else
#include <stdio.h>
#include <unistd.h>
#endif

GhosttyTerminal exp_new(int cols, int rows) {
  GhosttyTerminal t = NULL;
  if (ghostty_terminal_new(NULL, &t, cols, rows)) return NULL;
  size_t bytes = 256 * 1024, lines = 2000, continuation = 64 * 1024;
  if (ghostty_terminal_set(t, GHOSTTY_TERMINAL_OPT_SCROLLBACK_MAX_BYTES, &bytes) ||
      ghostty_terminal_set(t, GHOSTTY_TERMINAL_OPT_SCROLLBACK_MAX_LINES, &lines) ||
      ghostty_terminal_set(t, GHOSTTY_TERMINAL_OPT_CONTINUATION_MAX_BYTES, &continuation)) {
    ghostty_terminal_free(t); return NULL;
  }
  return t;
}

int exp_info(GhosttyTerminal t, ExpInfo *i) {
  GhosttyTerminalScreen screen;
  GhosttyTerminalData keys[] = {
    GHOSTTY_TERMINAL_DATA_COLS, GHOSTTY_TERMINAL_DATA_ROWS,
    GHOSTTY_TERMINAL_DATA_CURSOR_X, GHOSTTY_TERMINAL_DATA_CURSOR_Y,
    GHOSTTY_TERMINAL_DATA_CURSOR_VISIBLE, GHOSTTY_TERMINAL_DATA_CURSOR_PENDING_WRAP,
    GHOSTTY_TERMINAL_DATA_ACTIVE_SCREEN
  };
  void *values[] = {&i->cols, &i->rows, &i->x, &i->y, &i->visible, &i->pending_wrap, &screen};
  int rc = ghostty_terminal_get_multi(t, 7, keys, values, NULL);
  if (rc) return rc;
  i->alt = screen == GHOSTTY_TERMINAL_SCREEN_ALTERNATE;
  GhosttyMode modes[] = {GHOSTTY_MODE_BRACKETED_PASTE, GHOSTTY_MODE_DECCKM,
    GHOSTTY_MODE_ORIGIN, GHOSTTY_MODE_WRAPAROUND, GHOSTTY_MODE_INSERT};
  bool *outs[] = {&i->bracketed, &i->application_cursor, &i->origin, &i->autowrap, &i->insert};
  for (int n = 0; n < 5; n++) {
    GhosttyTerminalModeConfig v = {.mode = modes[n]};
    if ((rc = ghostty_terminal_get(t, GHOSTTY_TERMINAL_DATA_MODE, &v))) return rc;
    *outs[n] = v.value;
  }
  return 0;
}

int exp_format(GhosttyTerminal t, int vt, int history, uint8_t **out, size_t *len) {
  ExpInfo info;
  int rc = exp_info(t, &info);
  if (rc) return rc;
  GhosttyFormatterTerminalOptions opts = GHOSTTY_INIT_SIZED(GhosttyFormatterTerminalOptions);
  opts.emit = vt ? GHOSTTY_FORMATTER_FORMAT_VT : GHOSTTY_FORMATTER_FORMAT_PLAIN;
  opts.trim = false;
  opts.unwrap = false;
  GhosttySelection selection = GHOSTTY_INIT_SIZED(GhosttySelection);
  if (!history) {
    GhosttyPoint first = {.tag = GHOSTTY_POINT_TAG_ACTIVE, .value = {.coordinate = {0, 0}}};
    GhosttyPoint last = {.tag = GHOSTTY_POINT_TAG_ACTIVE, .value = {.coordinate = {info.cols - 1, info.rows - 1}}};
    selection.start = (GhosttyGridRef)GHOSTTY_INIT_SIZED(GhosttyGridRef);
    selection.end = (GhosttyGridRef)GHOSTTY_INIT_SIZED(GhosttyGridRef);
    if ((rc = ghostty_terminal_grid_ref(t, first, &selection.start)) ||
        (rc = ghostty_terminal_grid_ref(t, last, &selection.end))) return rc;
    opts.selection = &selection;
  }
  if (vt) {
    opts.extra.size = sizeof(opts.extra);
    opts.extra.modes = true;
    opts.extra.scrolling_region = true;
    opts.extra.tabstops = true;
    opts.extra.keyboard = true;
    opts.extra.screen.size = sizeof(opts.extra.screen);
    opts.extra.screen.cursor = true;
    opts.extra.screen.style = true;
    opts.extra.screen.hyperlink = true;
    opts.extra.screen.protection = true;
    opts.extra.screen.kitty_keyboard = true;
    opts.extra.screen.charsets = true;
  }
  GhosttyFormatter fmt = NULL;
  rc = ghostty_formatter_terminal_new(NULL, &fmt, t, opts);
  if (rc) return rc;
  rc = ghostty_formatter_format_alloc(fmt, NULL, out, len);
  ghostty_formatter_free(fmt);
  return rc;
}

int exp_restore(const uint8_t *data, size_t len, GhosttyTerminal *out) {
  GhosttySnapshotDecoder decoder = NULL;
  int rc = ghostty_snapshot_decoder_new_buf(NULL, &decoder, data, len);
  if (rc) return rc;
  const bool retain = true;
  const size_t max_continuation = 64 * 1024;
  rc = ghostty_snapshot_decoder_set(decoder, GHOSTTY_SNAPSHOT_DECODER_OPT_MAX_CONTINUATION_BYTES, &max_continuation);
  if (!rc) rc = ghostty_snapshot_decoder_set(decoder, GHOSTTY_SNAPSHOT_DECODER_OPT_RETAIN_CONTINUATION, &retain);
  if (rc) { ghostty_snapshot_decoder_free(decoder); return rc; }
  rc = ghostty_snapshot_decoder_decode(decoder, out);
  ghostty_snapshot_decoder_free(decoder);
  return rc;
}

uint64_t exp_rss(void) {
#ifdef __APPLE__
  struct mach_task_basic_info info;
  mach_msg_type_number_t count = MACH_TASK_BASIC_INFO_COUNT;
  if (task_info(mach_task_self(), MACH_TASK_BASIC_INFO, (task_info_t)&info, &count) != KERN_SUCCESS) return 0;
  return info.resident_size;
#else
  FILE *file = fopen("/proc/self/statm", "r");
  if (!file) return 0;
  unsigned long total = 0, resident = 0;
  int count = fscanf(file, "%lu %lu", &total, &resident);
  fclose(file);
  return count == 2 ? resident * (uint64_t)sysconf(_SC_PAGESIZE) : 0;
#endif
}
