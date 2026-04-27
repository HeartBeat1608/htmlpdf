package pdf

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Ops Constants = PDF content stream operators
const (
	opBeginText  = "BT"
	opEndText    = "ET"
	opMoveText   = "Td"
	opSetTextMat = "Tm"
	opSetFont    = "Tf"
	opShowText   = "Tj"
	opSetLeading = "TL"
	opNextLine   = "T*"
	opSetColor   = "rg"
	opSetStroke  = "RG"
	opRect       = "re"
	opFill       = "f"
	opStroke     = "S"
	opFillStroke = "B"
	opLineWidth  = "w"
	opMoveTo     = "m"
	opLineTo     = "l"
	opConcat     = "cm"
	opSave       = "q"
	opRestore    = "Q"
	opDoXObject  = "Do"
)

func escapeText(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	b.WriteByte('(')
	for _, r := range s {
		c := normalizePDFRune(r)
		switch c {
		case '\\':
			b.WriteString("\\\\")
		case '(':
			b.WriteString("\\(")
		case ')':
			b.WriteString("\\)")
		case '\r':
			b.WriteString("\\r")
		case '\n':
			b.WriteString("\\n")
		default:
			if c < 32 || c > utf8.RuneSelf {
				b.WriteByte('?')
			} else {
				b.WriteRune(c)
			}
		}
	}
	b.WriteByte(')')
	return b.String()
}

func normalizePDFRune(r rune) rune {
	switch r {
	case '\u2012', '\u2013', '\u2014', '\u2015':
		return '-'
	case '\u2018', '\u2019', '\u201A', '\u201B':
		return '\''
	case '\u201C', '\u201D', '\u201E', '\u201F':
		return '"'
	case '\u2026':
		return '.'
	case '\u00A0':
		return ' '
	default:
		return r
	}
}

func fmtColor(r, g, b float64) string {
	return fmt.Sprintf("%.4f %.4f %.4f", r, g, b)
}
