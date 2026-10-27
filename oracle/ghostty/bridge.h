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
