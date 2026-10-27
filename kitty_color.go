package phosphor

import (
	"encoding/hex"
	"strconv"
	"strings"
)

type kittyColorAction uint8

const (
	kittyColorQuery kittyColorAction = iota
	kittyColorSet
	kittyColorReset
)

type kittyColorRequest struct {
	slot   uint16
	action kittyColorAction
	color  RGB
}

const kittyUnsupportedColor = 0xffff
const maxKittyColorRequests = 528 // twice the 256 palette + eight special keys

// oscKittyColor adapts OSC 21 onto the same colors OSC 4/10/11/12 use.
// No color state is owned by the protocol. The headless model has no
// selection, visual bell or transparent background colors.
func (t *Terminal) oscKittyColor(rest string) {
	// Validate the request count before changing colors: an oversized OSC
	// must not leave behind a partially applied batch. Repeating this small
	// bounded parser avoids allocating a second operation list.
	count := 0
	for fields := rest; fields != ""; {
		field, tail, _ := strings.Cut(fields, ";")
		fields = tail
		if _, ok := parseKittyColorRequest(field); ok {
			count++
			if count > maxKittyColorRequests {
				return
			}
		}
	}
	var response []byte
	for fields := rest; fields != ""; {
		field, tail, _ := strings.Cut(fields, ";")
		fields = tail
		op, ok := parseKittyColorRequest(field)
		if !ok || op.slot == kittyUnsupportedColor {
			continue
		}
		switch op.action {
		case kittyColorSet:
			t.setColor(op.slot, op.color)
		case kittyColorReset:
			t.resetColors(func(slot uint16) bool { return slot == op.slot })
		case kittyColorQuery:
			if t.cfg == nil || t.cfg.reply == nil {
				continue
			}
			if response == nil {
				response = append(t.scratch[:0], "\x1b]21"...)
			}
			response = append(response, ';')
			switch op.slot {
			case slotFg:
				response = append(response, "foreground"...)
			case slotBg:
				response = append(response, "background"...)
			case slotCursor:
				response = append(response, "cursor"...)
			default:
				response = strconv.AppendInt(response, int64(op.slot), 10)
			}
			response = append(response, "=rgb:"...)
			c := t.paletteColor(op.slot)
			for i, v := range [3]uint8{c.R, c.G, c.B} {
				if i > 0 {
					response = append(response, '/')
				}
				response = hex.AppendEncode(response, []byte{v})
			}
		}
	}
	if response != nil {
		if t.oscBel {
			response = append(response, '\a')
		} else {
			response = append(response, "\x1b\\"...)
		}
		t.sendReply(response)
	}
}

func parseKittyColorRequest(field string) (kittyColorRequest, bool) {
	key, value, _ := strings.Cut(field, "=")
	var slot uint16
	switch key {
	case "foreground":
		slot = slotFg
	case "background":
		slot = slotBg
	case "cursor":
		slot = slotCursor
	case "selection_foreground", "selection_background", "cursor_text", "visual_bell", "second_transparent_background":
		slot = kittyUnsupportedColor
	default:
		n, err := strconv.ParseUint(key, 10, 8)
		if err != nil {
			return kittyColorRequest{}, false
		}
		slot = uint16(n)
	}
	value = strings.Trim(value, " ")
	op := kittyColorRequest{slot: slot}
	switch value {
	case "":
		op.action = kittyColorReset
	case "?":
		op.action = kittyColorQuery
	default:
		c, ok := parseColor(value)
		if !ok {
			return kittyColorRequest{}, false
		}
		op.action, op.color = kittyColorSet, c
	}
	return op, true
}
