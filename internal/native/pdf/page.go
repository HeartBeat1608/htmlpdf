package pdf

import (
	"fmt"
	"strings"
)

// Page accumulates the content stream for a single PDF page.
// All coordinates are in PDF user space: origin at bottom-left, Y increases upwards
// Callers work in top-left document space and convert via Page.toY()
type Page struct {
	width  float64 // points
	height float64 // points
	buf    strings.Builder

	// current graphics state - tracked to avoid redundant ops
	curFont          string
	curSize          float64
	curR, curG, curB float64 // fill color

	inText bool // true between BT and ET
}

// newPage create a page of the given size in points
func newPage(width, height float64) *Page {
	return &Page{width: width, height: height}
}

// toY converts a top-left Y coordinate (doc space) to PDF Y (bottom-left origin)
func (p *Page) toY(y float64) float64 {
	return p.height - y
}

// op Writes a raw PDF operator line to the content stream
func (p *Page) op(tokens ...string) {
	p.buf.WriteString(strings.Join(tokens, " "))
	p.buf.WriteByte('\n')
}

// ensureTextMode opens a BT block is not already in one
func (p *Page) ensureTextMode() {
	if !p.inText {
		p.op(opBeginText)
		p.inText = true
	}
}

// endText closes a BT block if open
func (p *Page) endText() {
	if p.inText {
		p.op(opEndText)
		p.inText = false
	}
}

// SetFont sets the active font and size.
// name must be a key in the page resource dictionary (e.g "F1")
func (p *Page) SetFont(name string, size float64) {
	if p.curFont == name && p.curSize == size {
		return
	}
	p.ensureTextMode()
	p.op("/"+name, fmt.Sprintf("%.2f", size), opSetFont)
	p.curFont = name
	p.curSize = size
}

// SetFillColor sets the fill color using normalized RGB (0.0-1.0)
func (p *Page) SetFillColor(r, g, b float64) {
	if p.curR == r && p.curG == g && p.curB == b {
		return
	}
	p.endText() // color ops must be outside text block
	p.op(fmtColor(r, g, b), opSetColor)
	p.curR, p.curG, p.curB = r, g, b
}

// DrawText renders a UTF-8 strings at doc-space (x, y).
// y is the baseline position in top-left document coordinates.
func (p *Page) DrawText(x, y float64, text string) {
	p.ensureTextMode()
	p.op(
		fmt.Sprintf("1 0 0 1 %.4f %.4f", x, p.toY(y)),
		opSetTextMat,
	)
	p.op(escapeText(text), opShowText)
}

// DrawRect draws a filled rectangle in top-left document coordinates.
// (x, y) is the top-left corner.
func (p *Page) DrawRect(x, y, w, h, r, g, b float64) {
	p.endText()
	p.op(fmtColor(r, g, b), opSetColor)
	p.op(
		fmt.Sprintf("%.4f", x),
		fmt.Sprintf("%.4f", p.toY(y+h)),
		fmt.Sprintf("%.4f", w),
		fmt.Sprintf("%.4f", h),
		opRect,
	)
	p.op(opFill)
	// reset fillc color tracking so next SetFillColor isn't skipped
	p.curR, p.curG, p.curB = r, g, b
}

// DrawLine strokes a horizontal or vertical rule
func (p *Page) DrawLine(x1, y1, x2, y2, lineWidth, r, g, b float64) {
	p.endText()
	p.op(fmtColor(r, g, b), opSetStroke)
	p.op(fmt.Sprintf("%.2f", lineWidth), opLineWidth)
	p.op(fmt.Sprintf("%.4f %.4f", x1, p.toY(y1)), opMoveTo)
	p.op(fmt.Sprintf("%.4f %.4f", x2, p.toY(y2)), opLineTo)
	p.op(opStroke)
}

// PlaceImage renders a named XObject (image) at doc-space position.
// width and height are the rendered dimensions in points.
func (p *Page) PlaceImage(name string, x, y, w, h float64) {
	p.endText()
	p.op(opSave)
	// PDF images are 1×1 unit squares; we scale and position with the CTM.
	p.op(
		fmt.Sprintf("%.4f 0 0 %.4f %.4f %.4f", w, h, x, p.toY(y+h)),
		opConcat,
	)
	p.op("/"+name, opDoXObject)
	p.op(opRestore)
}

// bytes finalises the content stream and returns its bytes.
func (p *Page) bytes() []byte {
	p.endText()
	return []byte(p.buf.String())
}
