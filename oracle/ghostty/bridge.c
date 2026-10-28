//go:build ghostty

#include "bridge.h"
#include <stdlib.h>
#include <string.h>
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

// Probe effective mouse behavior through the public encoder API. Mode bits
// alone cannot identify the last-selected tracking mode or report format.
int exp_mouse_report(GhosttyTerminal t, int action, bool pressed, int tracking, int format,
                     uint8_t *out, size_t capacity, size_t *len) {
  GhosttyMouseEncoder encoder = NULL;
  GhosttyMouseEvent event = NULL;
  int rc = ghostty_mouse_encoder_new(NULL, &encoder);
  if (rc) return rc;
  if ((rc = ghostty_mouse_event_new(NULL, &event))) goto done;
  ghostty_mouse_encoder_setopt_from_terminal(encoder, t);
  GhosttyMouseEncoderSize size = GHOSTTY_INIT_SIZED(GhosttyMouseEncoderSize);
  size.screen_width = 2048; size.screen_height = 4096;
  size.cell_width = 8; size.cell_height = 16;
  ghostty_mouse_encoder_setopt(encoder, GHOSTTY_MOUSE_ENCODER_OPT_SIZE, &size);
  ghostty_mouse_encoder_setopt(encoder, GHOSTTY_MOUSE_ENCODER_OPT_ANY_BUTTON_PRESSED, &pressed);
  if (tracking >= 0) {
    GhosttyMouseTrackingMode v = (GhosttyMouseTrackingMode)tracking;
    ghostty_mouse_encoder_setopt(encoder, GHOSTTY_MOUSE_ENCODER_OPT_EVENT, &v);
  }
  if (format >= 0) {
    GhosttyMouseFormat v = (GhosttyMouseFormat)format;
    ghostty_mouse_encoder_setopt(encoder, GHOSTTY_MOUSE_ENCODER_OPT_FORMAT, &v);
  }
  ghostty_mouse_event_set_action(event, (GhosttyMouseAction)action);
  if (action != GHOSTTY_MOUSE_ACTION_MOTION || pressed)
    ghostty_mouse_event_set_button(event, GHOSTTY_MOUSE_BUTTON_LEFT);
  ghostty_mouse_event_set_position(event, (GhosttyMousePosition){1993, 49});
  rc = ghostty_mouse_encoder_encode(encoder, event, (char *)out, capacity, len);
done:
  ghostty_mouse_event_free(event);
  ghostty_mouse_encoder_free(encoder);
  return rc;
}

static int exp_effective_mouse(GhosttyTerminal t, uint16_t *tracking, uint16_t *format) {
  uint8_t out[64], candidate[64]; size_t len, candidate_len;
  int pattern = 0, rc;
  for (int n = 0; n < 4; n++) {
    int action = n < 2 ? n : GHOSTTY_MOUSE_ACTION_MOTION;
    if ((rc = exp_mouse_report(t, action, n == 2, -1, GHOSTTY_MOUSE_FORMAT_SGR, out, sizeof(out), &len))) return rc;
    if (len) pattern |= 1 << n;
  }
  switch (pattern) {
    case 0: *tracking = 0; break;
    case 1: *tracking = 9; break;
    case 3: *tracking = 1000; break;
    case 7: *tracking = 1002; break;
    case 15: *tracking = 1003; break;
    default: return GHOSTTY_INVALID_VALUE;
  }
  // Keep tracking constant so a disabled mouse does not hide its format.
  if ((rc = exp_mouse_report(t, GHOSTTY_MOUSE_ACTION_PRESS, false, GHOSTTY_MOUSE_TRACKING_NORMAL, -1, out, sizeof(out), &len))) return rc;
  const uint16_t formats[] = {0, 1005, 1006, 1015, 1016};
  for (int n = 0; n < 5; n++) {
    if ((rc = exp_mouse_report(t, GHOSTTY_MOUSE_ACTION_PRESS, false, GHOSTTY_MOUSE_TRACKING_NORMAL, n, candidate, sizeof(candidate), &candidate_len))) return rc;
    if (len == candidate_len && !memcmp(out, candidate, len)) {
      *format = formats[n]; return 0;
    }
  }
  return GHOSTTY_INVALID_VALUE;
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

// A C-owned reply sink never retains a pointer into Go memory. Overflow is
// reported as a capture error instead of silently dropping protocol output.
static void exp_reply(GhosttyTerminal t, void *userdata, const uint8_t *data, size_t len) {
  (void)t;
  ExpReplies *r = userdata;
  if (r->error) return;
  if (len > 1024 * 1024 - r->len) { r->error = GHOSTTY_LIMIT_EXCEEDED; return; }
  size_t required = r->len + len;
  if (required > r->cap) {
    size_t cap = required * 2;
    uint8_t *p = realloc(r->data, cap);
    if (!p) { r->error = GHOSTTY_OUT_OF_MEMORY; return; }
    r->data = p; r->cap = cap;
  }
  if (len) memcpy(r->data + r->len, data, len);
  r->len = required;
}
ExpReplies *exp_replies_new(GhosttyTerminal t) {
  ExpReplies *r = calloc(1, sizeof(*r));
  if (!r) return NULL;
  if (ghostty_terminal_set(t, GHOSTTY_TERMINAL_OPT_USERDATA, r) ||
      ghostty_terminal_set(t, GHOSTTY_TERMINAL_OPT_WRITE_PTY, exp_reply)) {
    free(r); return NULL;
  }
  return r;
}
void exp_replies_free(ExpReplies *r) {
  if (r) { free(r->data); free(r); }
}
void exp_widths(uint8_t *out, size_t len) {
  for (size_t i = 0; i < len; i++) out[i] = ghostty_unicode_codepoint_width((uint32_t)i);
}
void exp_grapheme_properties(uint8_t *out, size_t len) {
  for (size_t i = 0; i < len; i++) {
    uint32_t cps[] = {0x600, (uint32_t)i}; uint8_t width;
    size_t consumed = ghostty_unicode_grapheme_width(cps, 2, &width);
    out[i] = (uint8_t)((consumed << 4) | width);
  }
}
static uint32_t exp_color(GhosttyStyleColor c) {
  switch (c.tag) {
    case GHOSTTY_STYLE_COLOR_PALETTE: return 0x1000000 | c.value.palette;
    case GHOSTTY_STYLE_COLOR_RGB: return 0x2000000 | ((uint32_t)c.value.rgb.r << 16) |
      ((uint32_t)c.value.rgb.g << 8) | c.value.rgb.b;
    default: return 0;
  }
}
void exp_capture_free(ExpState *out) {
  free(out->cells); free(out->wraps);
  out->cells = NULL; out->wraps = NULL; out->prompts = NULL;
}
int exp_capture(GhosttyTerminal t, ExpState *out) {
  memset(out, 0, sizeof(*out));
  int rc = exp_info(t, &out->info);
  if (rc) return rc;
  if ((rc = exp_effective_mouse(t, &out->mouse, &out->mouse_encoding))) return rc;
  if ((rc = ghostty_terminal_get(t, GHOSTTY_TERMINAL_DATA_SCROLLBACK_ROWS, &out->history_rows))) return rc;
  out->total_rows = out->history_rows + out->info.rows;
  // Trace fixtures are bounded independently of production terminal limits.
  if (out->info.cols > 256 || out->total_rows > 256 || out->total_rows * out->info.cols > 32768)
    return GHOSTTY_LIMIT_EXCEEDED;
  out->cells = calloc(out->total_rows * out->info.cols, sizeof(ExpCell));
  out->wraps = calloc(out->total_rows, 2);
  out->prompts = out->wraps ? out->wraps + out->total_rows : NULL;
  if (!out->cells || !out->wraps) { rc = GHOSTTY_OUT_OF_MEMORY; goto fail; }
  for (size_t y = 0; y < out->total_rows; y++) {
    for (size_t x = 0; x < out->info.cols; x++) {
      GhosttyGridRef ref = GHOSTTY_INIT_SIZED(GhosttyGridRef);
      GhosttyPoint point = {.tag = y < out->history_rows ? GHOSTTY_POINT_TAG_HISTORY : GHOSTTY_POINT_TAG_ACTIVE,
        .value = {.coordinate = {(uint16_t)x, (uint32_t)(y < out->history_rows ? y : y - out->history_rows)}}};
      if ((rc = ghostty_terminal_grid_ref(t, point, &ref))) goto fail;
      if (!x) {
        GhosttyRow row; bool wrap; GhosttyRowSemanticPrompt prompt;
        if ((rc = ghostty_grid_ref_row(&ref, &row)) ||
            (rc = ghostty_row_get(row, GHOSTTY_ROW_DATA_WRAP_CONTINUATION, &wrap)) ||
            (rc = ghostty_row_get(row, GHOSTTY_ROW_DATA_SEMANTIC_PROMPT, &prompt))) goto fail;
        out->wraps[y] = wrap;
        out->prompts[y] = (uint8_t)prompt;
      }
      ExpCell *c = &out->cells[y * out->info.cols + x];
      GhosttyCell raw; GhosttyCellWide wide; GhosttyCellContentTag tag; bool protected;
      GhosttyStyle style = GHOSTTY_INIT_SIZED(GhosttyStyle);
      if ((rc = ghostty_grid_ref_cell(&ref, &raw)) ||
          (rc = ghostty_cell_get(raw, GHOSTTY_CELL_DATA_WIDE, &wide)) ||
          (rc = ghostty_cell_get(raw, GHOSTTY_CELL_DATA_CONTENT_TAG, &tag)) ||
          (rc = ghostty_cell_get(raw, GHOSTTY_CELL_DATA_PROTECTED, &protected)) ||
          (rc = ghostty_grid_ref_style(&ref, &style))) goto fail;
      size_t n = 0;
      if ((rc = ghostty_grid_ref_graphemes(&ref, c->text, 1024, &n))) goto fail;
      c->text_len = (uint16_t)n;
      if ((rc = ghostty_grid_ref_hyperlink_uri(&ref, c->uri, 4096, &n))) goto fail;
      c->uri_len = (uint16_t)n;
      c->width = wide == GHOSTTY_CELL_WIDE_WIDE ? 2 : wide == GHOSTTY_CELL_WIDE_SPACER_TAIL ? 0 : 1;
      c->fg = exp_color(style.fg_color); c->bg = exp_color(style.bg_color);
      c->underline_color = exp_color(style.underline_color); c->underline = (uint8_t)style.underline;
      c->attrs = style.bold | (style.faint << 1) | (style.italic << 2) | (style.blink << 3) |
        (style.inverse << 4) | (style.invisible << 5) | (style.strikethrough << 6) |
        (style.overline << 7) | (protected << 8);
      if (tag == GHOSTTY_CELL_CONTENT_BG_COLOR_PALETTE) {
        uint8_t color; if ((rc = ghostty_cell_get(raw, GHOSTTY_CELL_DATA_COLOR_PALETTE, &color))) goto fail;
        c->bg = 0x1000000 | color;
      } else if (tag == GHOSTTY_CELL_CONTENT_BG_COLOR_RGB) {
        GhosttyColorRgb color; if ((rc = ghostty_cell_get(raw, GHOSTTY_CELL_DATA_COLOR_RGB, &color))) goto fail;
        c->bg = 0x2000000 | ((uint32_t)color.r << 16) | ((uint32_t)color.g << 8) | color.b;
      }
    }
  }
  GhosttyMode modes[] = {GHOSTTY_MODE_DECCKM, GHOSTTY_MODE_KEYPAD_KEYS,
    GHOSTTY_MODE_BRACKETED_PASTE, GHOSTTY_MODE_FOCUS_EVENT, GHOSTTY_MODE_WRAPAROUND,
    GHOSTTY_MODE_INSERT, GHOSTTY_MODE_LINEFEED, GHOSTTY_MODE_ORIGIN, GHOSTTY_MODE_REVERSE_COLORS,
    GHOSTTY_MODE_SYNC_OUTPUT, GHOSTTY_MODE_X10_MOUSE, GHOSTTY_MODE_NORMAL_MOUSE,
    GHOSTTY_MODE_BUTTON_MOUSE, GHOSTTY_MODE_ANY_MOUSE, GHOSTTY_MODE_UTF8_MOUSE,
    GHOSTTY_MODE_SGR_MOUSE, GHOSTTY_MODE_URXVT_MOUSE, GHOSTTY_MODE_SGR_PIXELS_MOUSE,
    GHOSTTY_MODE_132_COLUMN, GHOSTTY_MODE_ENABLE_MODE_3, GHOSTTY_MODE_SLOW_SCROLL,
    GHOSTTY_MODE_ALT_SCREEN_LEGACY, GHOSTTY_MODE_ALT_SCREEN, GHOSTTY_MODE_ALT_SCREEN_SAVE};
  for (size_t n = 0; n < sizeof(modes)/sizeof(*modes); n++) {
    GhosttyTerminalModeConfig v = {.mode = modes[n]};
    if ((rc = ghostty_terminal_get(t, GHOSTTY_TERMINAL_DATA_MODE, &v))) goto fail;
    out->modes[n] = v.value;
  }
  if ((rc = ghostty_terminal_get(t, GHOSTTY_TERMINAL_DATA_KITTY_KEYBOARD_FLAGS, &out->kitty)) ||
      (rc = ghostty_terminal_get(t, GHOSTTY_TERMINAL_DATA_TITLE, &out->title))) goto fail;
  GhosttyRenderState render = NULL;
  if ((rc = ghostty_render_state_new(NULL, &render))) goto fail;
  rc = ghostty_render_state_update(render, t);
  GhosttyRenderStateCursorVisualStyle shape;
  if (!rc) rc = ghostty_render_state_get(render, GHOSTTY_RENDER_STATE_DATA_CURSOR_VISUAL_STYLE, &shape);
  if (!rc) rc = ghostty_render_state_get(render, GHOSTTY_RENDER_STATE_DATA_CURSOR_BLINKING, &out->cursor_blink);
  ghostty_render_state_free(render);
  if (rc) goto fail;
  out->cursor_shape = shape == GHOSTTY_RENDER_STATE_CURSOR_VISUAL_STYLE_BAR ? 3 :
    shape == GHOSTTY_RENDER_STATE_CURSOR_VISUAL_STYLE_UNDERLINE ? 2 : 1;
  return 0;
fail:
  exp_capture_free(out); return rc;
}
