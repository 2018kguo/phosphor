#include <ghostty/vt.h>
typedef struct {
  uint16_t cols, rows, x, y;
  bool visible, pending_wrap, alt, bracketed, application_cursor, origin, autowrap, insert;
} ExpInfo;
GhosttyTerminal exp_new(int cols, int rows);
int exp_info(GhosttyTerminal t, ExpInfo *info);
int exp_format(GhosttyTerminal t, int vt, int history, uint8_t **out, size_t *len);
int exp_restore(const uint8_t *data, size_t len, GhosttyTerminal *out);
uint64_t exp_rss(void);
typedef struct { uint8_t *data; size_t len, cap; int error; } ExpReplies;
ExpReplies *exp_replies_new(GhosttyTerminal t);
void exp_replies_free(ExpReplies *r);
void exp_widths(uint8_t *out, size_t len);
void exp_grapheme_properties(uint8_t *out, size_t len);
typedef struct {
  uint32_t text[1024];
  uint8_t uri[4096];
  uint16_t text_len, uri_len, attrs;
  uint8_t width, underline;
  uint32_t fg, bg, underline_color;
} ExpCell;
typedef struct {
  ExpInfo info;
  ExpCell *cells;
  uint8_t *wraps, *prompts;
  size_t history_rows, total_rows;
  bool modes[24], cursor_blink;
  uint8_t cursor_shape, kitty;
  uint16_t mouse, mouse_encoding;
  GhosttyString title;
} ExpState;
int exp_capture(GhosttyTerminal t, ExpState *out);
void exp_capture_free(ExpState *out);
int exp_mouse_report(GhosttyTerminal t, int action, bool pressed, int tracking, int format,
                     uint8_t *out, size_t capacity, size_t *len);
