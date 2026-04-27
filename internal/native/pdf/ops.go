package pdf

import (
	"fmt"
	"strings"
)

// Ops Constants = PDF content stream operators
const (
	opBeginText  = "BT"
	opEndText    = "ET"
	opMoveText   = "Td"
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
	for i := 0; i < len(s); i++ {
		c := s[i]
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
			if c > 127 {
				b.WriteString(fmt.Sprintf("%03o", c))
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte(')')
	return b.String()
}

func fmtColor(r, g, b float64) string {
	return fmt.Sprintf("%.4f %.4f %.4f", r, g, b)
}
