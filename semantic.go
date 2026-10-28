package phosphor

import "strings"

// Row metadata moves with the row, whether it is copied, scrolled or reflowed.
// The existing two slice headers remain the only screen-row authority.
type rowFlags uint8

// PromptKind identifies a row annotated by the shell through OSC133.
type PromptKind uint8

const (
	PromptNone PromptKind = iota
	PromptPrimary
	PromptContinuation
)

// Prompt returns the shell prompt annotation for visible row y.
// Out-of-range rows return PromptNone. It does not allocate.
func (t *Terminal) Prompt(y int) PromptKind {
	return PromptKind(rowFlagsAt(t.wraps(), y) & rowSemanticMask >> 1)
}

// A history parser's temporary annotation at a logical cell offset.
// Persisted row metadata still lives only in the screen's rowFlags slices.
type rowMark struct {
	offset int
	flags  rowFlags
}

const (
	rowWrap               rowFlags = 1
	rowPrompt             rowFlags = 2
	rowPromptContinuation rowFlags = 4
	rowSemanticMask       rowFlags = 6
)

func rowFlagsAt(rows []rowFlags, y int) rowFlags {
	if uint(y) < uint(len(rows)) {
		return rows[y]
	}
	return 0
}

func (t *Terminal) setRowFlags(y int, flags rowFlags) {
	rows := t.wraps()
	if rows == nil {
		if flags == 0 {
			return
		}
		rows = make([]rowFlags, t.rows)
		if t.altActive {
			t.altWrap = rows
		} else {
			t.mainWrap = rows
		}
	}
	rows[y] = flags
}

// Semantic cursor state belongs to the screen, independently of DECSC.
type semanticContent uint8

const (
	semanticOutput semanticContent = iota
	semanticPrompt
	semanticInput
	semanticInputEOL
)

type promptRedraw uint8

const (
	promptRedrawOff promptRedraw = iota // headless hosts do not assume shell redraw
	promptRedrawAll
	promptRedrawLast
)

func (t *Terminal) semanticSlot() int {
	if t.altActive {
		return 1
	}
	return 0
}

func (t *Terminal) semanticContent() semanticContent {
	return t.semantic[t.semanticSlot()]
}

func (t *Terminal) setSemanticContent(content semanticContent) {
	t.semantic[t.semanticSlot()] = content
}

func (t *Terminal) setRowSemantic(y int, flags rowFlags) {
	t.setRowFlags(y, rowFlagsAt(t.wraps(), y)&^rowSemanticMask|flags)
}

func (t *Terminal) semanticCommand(payload []byte) {
	if len(payload) == 0 || len(payload) > 1 && payload[1] != ';' {
		return
	}
	action := payload[0]
	if action == 'L' || action == 'A' || action == 'N' {
		left := 0
		if t.cur.x >= t.left {
			left = t.left
		}
		if t.cur.x != left {
			t.carriageReturn()
			t.index()
		}
	}
	switch action {
	case 'A', 'N', 'P':
		kind := rowPrompt
		if len(payload) > 1 {
			for option := range strings.SplitSeq(string(payload[2:]), ";") {
				key, value, _ := strings.Cut(option, "=")
				switch key {
				case "k":
					if value == "c" || value == "s" {
						kind = rowPromptContinuation
					}
				case "redraw":
					if action != 'P' {
						switch value {
						case "0":
							t.promptRedraw = promptRedrawOff
						case "1":
							t.promptRedraw = promptRedrawAll
						case "last":
							t.promptRedraw = promptRedrawLast
						}
					}
				}
			}
		}
		t.setSemanticContent(semanticPrompt)
		t.setRowSemantic(t.cur.y, kind)
	case 'B':
		t.setSemanticContent(semanticInput)
	case 'I':
		t.setSemanticContent(semanticInputEOL)
	case 'C':
		t.setSemanticContent(semanticOutput)
		if t.cur.x == 0 {
			t.setRowSemantic(t.cur.y, 0)
		}
	case 'D':
		t.setSemanticContent(semanticOutput)
	}
}

func (t *Terminal) semanticIndex() {
	switch t.semanticContent() {
	case semanticPrompt, semanticInput:
		t.setRowSemantic(t.cur.y, rowPromptContinuation)
	case semanticInputEOL:
		t.setSemanticContent(semanticOutput)
	}
}

func (t *Terminal) semanticWrap(content semanticContent) {
	t.setSemanticContent(content)
	if content == semanticPrompt {
		t.setRowSemantic(t.cur.y, rowPromptContinuation)
	}
}

func screenAtPrompt(screen [][]cell, rows []rowFlags) bool {
	for y := len(screen) - 1; y >= 0; y-- {
		if rowFlagsAt(rows, y)&rowSemanticMask != 0 {
			return true
		}
		for _, c := range screen[y] {
			if c.R != 0 && c.R != wideTail && c.R != spacerHead {
				return false
			}
		}
	}
	return false
}

type promptPosition struct {
	line, row int // line=-1 denotes the visible screen
}

func (t *Terminal) clearPromptForRedraw() {
	if t.promptRedraw == promptRedrawOff || t.semantic[0] == semanticOutput {
		return
	}
	y := t.cur.y
	if t.altActive {
		y = t.ext().inactivePrimary.y
	}
	start := promptPosition{line: -1, row: -1}
	history := map[int]*Terminal{}
	if t.promptRedraw == promptRedrawLast {
		start.row = y
	} else {
		inContinuation, found := false, false
		var first, last promptPosition
		visit := func(flags rowFlags, pos promptPosition) bool {
			flags &= rowSemanticMask
			if !inContinuation {
				if flags == 0 {
					return false
				}
				if flags == rowPrompt {
					start, found = pos, true
					return true
				}
				inContinuation = true
				first, last = pos, pos
				return false
			}
			if flags == 0 {
				start, found = last, true
				return true
			}
			if flags == rowPrompt {
				start, found = pos, true
				return true
			}
			last = pos
			return false
		}
		for row := y; row >= 0; row-- {
			if visit(rowFlagsAt(t.mainWrap, row), promptPosition{-1, row}) {
				break
			}
		}
		for line := len(t.sb) - 1; !found && line >= t.sbStart; line-- {
			decoded := replayLine(t.sb[line], t.cols)
			history[line] = decoded
			for row := decoded.cur.y; row >= 0; row-- {
				if visit(rowFlagsAt(decoded.mainWrap, row), promptPosition{line, row}) {
					break
				}
			}
		}
		if !found && inContinuation {
			// Pinned native PromptIterator.nextLeftUp returns the original
			// continuation when its chain reaches absolute history top.
			// Clearing the oldest continuation here would erase extra text.
			start = first
		}
	}
	if start.row >= 0 {
		from := start.row
		if start.line >= 0 {
			for line := start.line; line < len(t.sb); line++ {
				decoded := history[line]
				if decoded == nil {
					decoded = replayLine(t.sb[line], t.cols)
				}
				rowStart := 0
				if line == start.line {
					rowStart = start.row
				}
				for row := rowStart; row <= decoded.cur.y; row++ {
					decoded.main[row] = nil
				}
				encoded := decoded.encodeHistoryRows()
				t.sbBytes += len(encoded) - len(t.sb[line])
				t.sb[line] = encoded
			}
			from = 0
		}
		for row := from; row < t.rows; row++ {
			t.main[row] = nil
		}
	}
}

func (t *Terminal) encodeHistoryRows() []byte {
	var out []byte
	hasMarkers := false
	for _, flags := range t.mainWrap {
		hasMarkers = hasMarkers || flags&rowSemanticMask != 0
	}
	for y, row := range t.main[:t.cur.y+1] {
		if hasMarkers {
			out = append(out, 0, 0, 'P', byte(rowFlagsAt(t.mainWrap, y)&rowSemanticMask>>1))
		}
		trim := y == t.cur.y
		out = t.appendRowHistory(out, row, t.cols, trim, true)
		if trim && contentEnd(row) == 0 && (hasMarkers || y > 0) {
			out = append(out, 0, 1, ' ')
		}
	}
	return out
}

func appendPromptMarker(b []byte, flags rowFlags) []byte {
	if flags&rowSemanticMask == rowPromptContinuation {
		return append(b, "\x1b]133;P;k=s\x1b\\"...)
	}
	return append(b, "\x1b]133;P\x1b\\"...)
}

func (t *Terminal) appendRowSemantics(b []byte, rows []rowFlags) []byte {
	for y, flags := range rows {
		if flags&rowSemanticMask == 0 {
			continue
		}
		b = appendCUP(b, y, 0)
		b = appendPromptMarker(b, flags)
		b = append(b, "\x1b]133;D\x1b\\"...)
	}
	return b
}

func (t *Terminal) appendSemanticCursor(b []byte, slot, row int) []byte {
	switch t.semantic[slot] {
	case semanticPrompt:
		rows := t.mainWrap
		if slot == 1 {
			rows = t.altWrap
		}
		flags := rowFlagsAt(rows, row) & rowSemanticMask
		if flags != 0 {
			b = appendCUP(b, row, 0)
			return appendPromptMarker(b, flags)
		}
		screen := t.main
		if slot == 1 {
			screen = t.alt
		}
		blank := true
		for y, cells := range screen {
			if !zeroRow(cells) || rowFlagsAt(rows, y) != 0 {
				blank = false
				break
			}
		}
		if blank && !t.cur.wrapPending {
			// On a zero screen native P+ED2 removes the row mark, retains
			// prompt cursor state and adds no history because nothing moves.
			return append(b, "\x1b[H\x1b[0m\x1b[0\"q\x1b]133;P\x1b\\\x1b[2J"...)
		}
		// Explicit input reproduces prompt cursor effects without creating
		// a row annotation. Exact snapshots retain the original cursor enum.
		return append(b, "\x1b]133;B\x1b\\"...)
	case semanticInput:
		return append(b, "\x1b]133;B\x1b\\"...)
	case semanticInputEOL:
		return append(b, "\x1b]133;I\x1b\\"...)
	default:
		return append(b, "\x1b]133;D\x1b\\"...)
	}
}

func (t *Terminal) appendPromptRedraw(b []byte) []byte {
	if t.promptRedraw == promptRedrawOff {
		return b
	}
	b = append(b, "\x1b[H\x1b]133;A;redraw="...)
	if t.promptRedraw == promptRedrawLast {
		b = append(b, "last"...)
	} else {
		b = append(b, '1')
	}
	// C at column zero clears the temporary row mark while retaining
	// the redraw policy. It avoids a prompt-triggered ED2 history push.
	return append(b, "\x1b\\\x1b]133;C\x1b\\"...)
}
