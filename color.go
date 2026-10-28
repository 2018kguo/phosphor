package phosphor

import (
	"math"
	"strconv"
	"strings"
)

// parseColor is the shared X11 color parser for all OSC color protocols.
// rgb: components are scaled; # components occupy the high channel bits.
func parseColor(s string) (RGB, bool) {
	s = strings.Trim(s, " \t")
	if rest, ok := strings.CutPrefix(s, "rgb:"); ok {
		return parseColorComponents(rest, false)
	}
	if rest, ok := strings.CutPrefix(s, "rgbi:"); ok {
		return parseColorComponents(rest, true)
	}
	if rest, ok := strings.CutPrefix(s, "#"); ok && len(rest)%3 == 0 && len(rest) > 0 && len(rest) <= 12 {
		n := len(rest) / 3
		var c [3]uint8
		for i := range c {
			v, err := strconv.ParseUint(rest[i*n:(i+1)*n], 16, 16)
			if err != nil {
				return RGB{}, false
			}
			c[i] = uint8(v << (16 - 4*n) >> 8)
		}
		return RGB{c[0], c[1], c[2]}, true
	}
	// Generated sorted data needs no process-wide heap map. Compare ASCII
	// case-insensitively without allocating a lowercase copy of each query.
	low, high := 0, len(x11ColorNames)
	for low < high {
		mid := low + (high-low)/2
		entry := &x11ColorNames[mid]
		switch compareColorName(s, entry.name) {
		case -1:
			high = mid
		case 0:
			return entry.color, true
		case 1:
			low = mid + 1
		}
	}
	return RGB{}, false
}

func compareColorName(s, lower string) int {
	for i := 0; i < min(len(s), len(lower)); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c < lower[i] {
			return -1
		}
		if c > lower[i] {
			return 1
		}
	}
	if len(s) < len(lower) {
		return -1
	}
	if len(s) > len(lower) {
		return 1
	}
	return 0
}

func parseColorComponents(s string, intensity bool) (RGB, bool) {
	var c [3]uint8
	for i := range c {
		part, rest, more := strings.Cut(s, "/")
		if more != (i < 2) {
			return RGB{}, false
		}
		s = rest
		if intensity {
			// Only decimal float syntax belongs to rgbi:, not Go's hex
			// floats, NaN or Inf. ParseFloat checks ordering and exponents.
			for _, ch := range part {
				if ch < '0' || ch > '9' {
					if ch != '.' && ch != '+' && ch != '-' && ch != 'e' && ch != 'E' {
						return RGB{}, false
					}
				}
			}
			v, err := strconv.ParseFloat(part, 64)
			if err != nil && !math.IsInf(v, 0) || math.IsNaN(v) {
				return RGB{}, false
			}
			c[i] = uint8(min(max(v, 0), 1) * 255)
		} else {
			v, err := strconv.ParseUint(part, 16, 16)
			if err != nil || len(part) == 0 || len(part) > 4 {
				return RGB{}, false
			}
			maxv := uint64(1)<<(4*len(part)) - 1
			c[i] = uint8((v*255 + maxv/2) / maxv)
		}
	}
	return RGB{c[0], c[1], c[2]}, true
}
