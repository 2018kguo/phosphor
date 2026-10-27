package phosphor

import (
	"strings"
	"unicode/utf8"
)

// Color is a foreground or background color: the terminal's default, one
// of the 256 palette entries, or a 24-bit RGB color. The zero value is
// DefaultColor. Colors are comparable; Palette.Resolve turns one into RGB.
type Color uint32

// DefaultColor is the terminal's default foreground or background.
const DefaultColor Color = 0

const (
	colorIndexed Color = 1 << 24
	colorRGB     Color = 2 << 24
)

// ColorKind says how a Color is specified.
type ColorKind uint8

// The kinds of Color.
const (
	ColorDefault ColorKind = iota // the default foreground or background
	ColorIndexed                  // a palette entry (SGR 30-37, 90-97, 38;5;n...)
	ColorRGB                      // a 24-bit color (SGR 38;2;r;g;b...)
)

// IndexedColor returns palette entry n: 0-7 are the ANSI colors, 8-15
// their bright versions, 16-231 a 6x6x6 color cube and 232-255 a gray
// ramp.
func IndexedColor(n uint8) Color { return colorIndexed | Color(n) }

// RGBColor returns the 24-bit color r, g, b.
func RGBColor(r, g, b uint8) Color {
	return colorRGB | Color(r)<<16 | Color(g)<<8 | Color(b)
}

// Kind reports whether c is the default color, a palette entry or RGB.
func (c Color) Kind() ColorKind {
	switch c >> 24 {
	case 1:
		return ColorIndexed
	case 2:
		return ColorRGB
	}
	return ColorDefault
}

// Index returns the palette entry of an indexed color (0 for other kinds).
func (c Color) Index() uint8 {
	if c.Kind() != ColorIndexed {
		return 0
	}
	return uint8(c)
}

// RGB returns the components of an RGB color (zero for other kinds; see
// Palette.Resolve for those).
func (c Color) RGB() RGB {
	if c.Kind() != ColorRGB {
		return RGB{}
	}
	return RGB{uint8(c >> 16), uint8(c >> 8), uint8(c)}
}

// RGB is a 24-bit color.
type RGB struct{ R, G, B uint8 }

// Attr is a set of text attributes, combined with |.
type Attr uint16

// The text attributes. All but AttrProtected are set with SGR.
const (
	AttrBold          Attr = 1 << iota // SGR 1
	AttrFaint                          // SGR 2
	AttrItalic                         // SGR 3
	AttrBlink                          // SGR 5 or 6
	AttrInverse                        // SGR 7: swap foreground and background
	AttrHidden                         // SGR 8
	AttrStrikethrough                  // SGR 9
	AttrOverline                       // SGR 53
	// AttrProtected marks a character that selective erase (DECSED,
	// DECSEL, and after SPA also ED, EL and ECH) leaves alone. DECSCA and
	// SPA set it.
	AttrProtected
)

// The underline style is kept in the top bits of a cell's attributes.
const (
	ulShift      = 12
	ulMask  Attr = 7 << ulShift
)

// Underline is an underline style (SGR 4, 4:n, 21).
type Underline uint8

// The underline styles.
const (
	UnderlineNone   Underline = iota // SGR 24
	UnderlineSingle                  // SGR 4 or 4:1
	UnderlineDouble                  // SGR 21 or 4:2
	UnderlineCurly                   // SGR 4:3
	UnderlineDotted                  // SGR 4:4
	UnderlineDashed                  // SGR 4:5
)

// Style is how a cell is drawn. The zero value is the default style.
// Styles are comparable. Inverse is not applied: Fg and Bg are as the
// program set them, and AttrInverse (and Modes.ReverseVideo) say to swap
// them.
type Style struct {
	Fg, Bg         Color     // foreground and background
	UnderlineColor Color     // default follows the foreground
	Attrs          Attr      // bold, italic...
	Underline      Underline // the underline style, if underlined
}

func (t *Terminal) publicStyle(c *cell) Style {
	return Style{Fg: c.Fg, Bg: c.Bg, UnderlineColor: t.extraAt(c.E).underline, Attrs: c.A &^ ulMask, Underline: Underline(c.A >> ulShift & 7)}
}

// Hyperlink is an OSC 8 hyperlink. The zero value is no link.
type Hyperlink struct {
	// ID is the link's id parameter, which ties together the cells of one
	// link that are not adjacent ("" if the program gave none).
	ID string
	// URI is the link target.
	URI string
}

// parseLink splits a stored hyperlink target ("params;uri").
func parseLink(target string) Hyperlink {
	params, uri, _ := strings.Cut(target, ";")
	var id string
	for params != "" {
		var kv string
		kv, params, _ = strings.Cut(params, ":")
		if v, ok := strings.CutPrefix(kv, "id="); ok {
			id = v
		}
	}
	return Hyperlink{ID: id, URI: uri}
}

// Cell is the content and style of one screen position, as Terminal.Cell
// returns it. Reading a cell does not allocate.
type Cell struct {
	// Rune is the character, or 0 for an empty cell and for the right half
	// of a wide character.
	Rune rune
	// Combining holds the combining marks, joiners and variation
	// selectors that follow Rune in the cell, usually "".
	Combining string
	// Width is 1, 2 for a wide character (which also covers the cell to its
	// right), or 0 for the right half of a wide character.
	Width int
	// Style is how the cell is drawn (the right half of a wide character
	// has its left half's style).
	Style Style
	// Link is the hyperlink the cell belongs to, if any.
	Link Hyperlink
}

// ascii holds each ASCII character, so Text need not allocate for them.
const ascii = "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f" +
	"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f" +
	" !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~\x7f"

// Text returns the cell's characters: Rune followed by Combining, or ""
// for an empty cell or the right half of a wide character. It allocates
// only for non-ASCII characters; AppendText never does.
func (c Cell) Text() string {
	switch {
	case c.Rune == 0:
		return ""
	case c.Rune < utf8.RuneSelf && c.Combining == "":
		return ascii[c.Rune : c.Rune+1]
	}
	return string(c.Rune) + c.Combining
}

// AppendText appends the cell's characters (as Text returns them) to b.
func (c Cell) AppendText(b []byte) []byte {
	if c.Rune == 0 {
		return b
	}
	return append(utf8.AppendRune(b, c.Rune), c.Combining...)
}

// Palette is the set of colors a terminal reports to programs that ask
// (OSC 4, 10, 11 and 12), and that a renderer uses to draw indexed and
// default colors.
type Palette struct {
	Foreground RGB // the default foreground (OSC 10)
	Background RGB // the default background (OSC 11)
	Cursor     RGB // the cursor (OSC 12)
	// Colors are the 256 indexed colors (OSC 4).
	Colors [256]RGB
}

var defaultPalette = func() Palette {
	p := Palette{
		Foreground: RGB{229, 229, 229},
		Background: RGB{0, 0, 0},
		Cursor:     RGB{229, 229, 229},
	}
	copy(p.Colors[:], []RGB{
		{0, 0, 0}, {205, 0, 0}, {0, 205, 0}, {205, 205, 0},
		{0, 0, 238}, {205, 0, 205}, {0, 205, 205}, {229, 229, 229},
		{127, 127, 127}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
		{92, 92, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
	})
	level := [6]uint8{0, 95, 135, 175, 215, 255}
	for i := range 216 {
		p.Colors[16+i] = RGB{level[i/36], level[i/6%6], level[i%6]}
	}
	for i := range 24 {
		v := uint8(8 + 10*i)
		p.Colors[232+i] = RGB{v, v, v}
	}
	return p
}()

// DefaultPalette returns the palette a Terminal starts with unless
// WithPalette sets another: xterm's 256 colors, with light gray text on
// black.
func DefaultPalette() Palette { return defaultPalette }

// Resolve returns the RGB value of c in this palette. A DefaultColor is
// the default background when background is set, else the default
// foreground.
func (p *Palette) Resolve(c Color, background bool) RGB {
	switch c.Kind() {
	case ColorIndexed:
		return p.Colors[c.Index()]
	case ColorRGB:
		return c.RGB()
	}
	if background {
		return p.Background
	}
	return p.Foreground
}

// CursorShape is the cursor shape a program asked for with DECSCUSR.
type CursorShape uint8

// The cursor shapes.
const (
	CursorDefault   CursorShape = iota // the program did not ask (or asked for the default)
	CursorBlock                        // a block
	CursorUnderline                    // an underline
	CursorBar                          // a vertical bar
)

// Cursor is the cursor's state.
type Cursor struct {
	// X and Y are the 0-based column and row on the visible screen.
	X, Y int
	// Visible is false when the program hid the cursor (DECTCEM).
	Visible bool
	// Shape and Blink are what the program asked for with DECSCUSR.
	Shape CursorShape
	Blink bool
	// PendingWrap is set when a character was just printed in the last
	// column: the cursor stays there, and the next character goes to the
	// next line.
	PendingWrap bool
}

// MouseMode is the mouse tracking a program turned on. Its values are
// the DEC private mode numbers.
type MouseMode uint16

// The mouse tracking modes.
const (
	MouseNone        MouseMode = 0    // no tracking
	MouseX10         MouseMode = 9    // presses only
	MouseNormal      MouseMode = 1000 // presses and releases
	MouseButtonEvent MouseMode = 1002 // ... and motion with a button down
	MouseAnyEvent    MouseMode = 1003 // ... and all motion
)

// MouseEncoding is how mouse reports are encoded. Its values are the DEC
// private mode numbers.
type MouseEncoding uint16

// The mouse report encodings.
const (
	MouseEncodingDefault   MouseEncoding = 0    // CSI M with bytes
	MouseEncodingUTF8      MouseEncoding = 1005 // CSI M with UTF-8
	MouseEncodingSGR       MouseEncoding = 1006 // CSI < ... M/m
	MouseEncodingURXVT     MouseEncoding = 1015 // CSI ... M
	MouseEncodingSGRPixels MouseEncoding = 1016 // CSI < ... M/m with pixel coordinates
)

// MouseTrackingModes records the independently enabled mouse tracking modes.
// Mouse in Modes reports the most recently selected tracking behavior.
type MouseTrackingModes uint8

const (
	MouseTrackingX10 MouseTrackingModes = 1 << iota
	MouseTrackingNormal
	MouseTrackingButtonEvent
	MouseTrackingAnyEvent
)

// MouseEncodingModes records the independently enabled mouse report encodings.
// MouseEncoding in Modes reports the most recently selected encoding.
type MouseEncodingModes uint8

const (
	MouseEncodingModeUTF8 MouseEncodingModes = 1 << iota
	MouseEncodingModeSGR
	MouseEncodingModeURXVT
	MouseEncodingModeSGRPixels
)

// Modes are the terminal modes a program sets, mostly to say how input
// should be encoded for it and how the screen should be drawn.
type Modes struct {
	AltScreen             bool               // the alternate screen is shown (47, 1047, 1049)
	ApplicationCursorKeys bool               // DECCKM: cursor keys send ESC O A...
	ApplicationKeypad     bool               // DECKPAM: the keypad sends application sequences
	BracketedPaste        bool               // 2004: pasted text is wrapped in ESC [200~ ... ESC [201~
	FocusEvents           bool               // 1004: report focus in and out
	Mouse                 MouseMode          // the mouse events to report (9, 1000, 1002, 1003)
	MouseEncoding         MouseEncoding      // how to encode them (1005, 1006, 1015, 1016)
	MouseTrackingModes    MouseTrackingModes // enabled tracking bits, queried and saved independently
	MouseEncodingModes    MouseEncodingModes // enabled encoding bits, queried and saved independently
	// KittyKeyboard holds the kitty keyboard protocol flags in effect
	// (CSI > u), 0 when the protocol is off.
	KittyKeyboard uint16
	Autowrap      bool // DECAWM
	Insert        bool // IRM
	Newline       bool // LNM: line feed also returns the carriage
	Origin        bool // DECOM: cursor positions are relative to the margins
	ReverseVideo  bool // DECSCNM: the whole screen is drawn inverted
	// SyncUpdate is set while the program is drawing a frame it wants
	// shown at once (2026): a renderer may wait for it to clear.
	SyncUpdate bool
	// GraphemeClustering joins Unicode grapheme clusters into cells (2027).
	GraphemeClustering   bool
	Columns132           bool // DECCOLM (3): the program selected 132-column mode
	AllowColumnSwitching bool // mode40 permits DECCOLM to resize the terminal
}
