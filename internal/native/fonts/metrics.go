// Package fonts provides character-width metrics for the standard PDF Type1 fonts.
//
// Widths are sourced from Adobe Font Metrics (AFM) files and are in units of
// 1/1000 of a text unit. To convert to points at a given font size:
//
//	widthPt = GlyphWidth(face, codepoint) * fontSize / 1000.0
//
// Only the WinAnsiEncoding range (codepoints 32–255) is covered. Codepoints
// outside that range return the width of a space (index 0 in each table).
// For multi-byte Unicode input, callers should use MeasureString which handles
// the encoding mapping.
//
// The seven faces correspond to the font resource keys used by the PDF writer:
//
//	F1 Helvetica
//	F2 Helvetica-Bold
//	F3 Helvetica-Oblique
//	F4 Times-Roman
//	F5 Times-Bold
//	F6 Courier
//	F7 Courier-Bold
package fonts

// Face identifies one of the seven supported Type1 font faces.
type Face int

const (
	Helvetica        Face = iota // F1 — default body font
	HelveticaBold                // F2
	HelveticaOblique             // F3
	TimesRoman                   // F4
	TimesBold                    // F5
	Courier                      // F6
	CourierBold                  // F7
)

// PDFName returns the PDF BaseFont name for a Face.
func (f Face) PDFName() string {
	switch f {
	case Helvetica:
		return "Helvetica"
	case HelveticaBold:
		return "Helvetica-Bold"
	case HelveticaOblique:
		return "Helvetica-Oblique"
	case TimesRoman:
		return "Times-Roman"
	case TimesBold:
		return "Times-Bold"
	case Courier:
		return "Courier"
	case CourierBold:
		return "Courier-Bold"
	default:
		return "Helvetica"
	}
}

// ResourceKey returns the /Font resource key (F1–F7) used in the PDF resource dict.
func (f Face) ResourceKey() string {
	switch f {
	case Helvetica:
		return "F1"
	case HelveticaBold:
		return "F2"
	case HelveticaOblique:
		return "F3"
	case TimesRoman:
		return "F4"
	case TimesBold:
		return "F5"
	case Courier:
		return "F6"
	case CourierBold:
		return "F7"
	default:
		return "F1"
	}
}

// FaceFromStyle derives a Face from the CSS-style boolean flags the layout
// engine carries. family is "serif" | "mono" | "" (sans-serif default).
func FaceFromStyle(family string, bold, italic bool) Face {
	switch family {
	case "mono":
		if bold {
			return CourierBold
		}
		return Courier
	case "serif":
		if bold {
			return TimesBold
		}
		return TimesRoman
	default: // sans-serif
		if bold && italic {
			return HelveticaBold // no bold-oblique standard face; use bold
		}
		if bold {
			return HelveticaBold
		}
		if italic {
			return HelveticaOblique
		}
		return Helvetica
	}
}

// GlyphWidth returns the AFM width (in 1/1000 text units) for the given
// Unicode codepoint rendered in face f.
//
// Only WinAnsiEncoding (U+0020–U+00FF) is handled. Codepoints outside that
// range return the space width for the face (a safe fallback — the glyph will
// not be present in the standard font anyway).
func GlyphWidth(f Face, r rune) int {
	idx := int(r) - 32
	if idx < 0 || idx >= 224 {
		return widthTables[f][0] // space width as fallback
	}
	return widthTables[f][idx]
}

// MeasureString returns the total width of s in points at the given font size.
// Multi-byte runes outside WinAnsiEncoding are counted at the space width.
func MeasureString(f Face, fontSize float64, s string) float64 {
	total := 0
	for _, r := range s {
		total += GlyphWidth(f, r)
	}
	return float64(total) * fontSize / 1000.0
}

// MeasureRune returns the width of a single rune in points.
func MeasureRune(f Face, fontSize float64, r rune) float64 {
	return float64(GlyphWidth(f, r)) * fontSize / 1000.0
}

// LineHeight returns a comfortable line height for the given font size.
// Uses a 1.2× leading factor, which matches common browser defaults.
func LineHeight(fontSize float64) float64 {
	return fontSize * 1.2
}

// SpaceWidth returns the width of a space character in points.
func SpaceWidth(f Face, fontSize float64) float64 {
	return MeasureRune(f, fontSize, ' ')
}
