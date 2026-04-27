// Package layout converts a document.Document into a rendered PDF.
//
// The engine flows content top-to-bottom using a simple block layout model:
//
//   - Blocks are stacked vertically, each consuming vertical space.
//   - Inline content within a block is line-wrapped using greedy word-break.
//   - Page overflow triggers a new page; the Y cursor resets to the top margin.
//   - Tables use the column-sizing algorithm in table.go.
//
// All coordinates passed to the PDF writer are in points, in top-left
// document space (the writer handles the flip to PDF's bottom-left origin).
package layout

import (
	"strings"

	"github.com/HeartBeat1608/htmlpdf/internal/document"
	"github.com/HeartBeat1608/htmlpdf/internal/native/fonts"
	"github.com/HeartBeat1608/htmlpdf/internal/native/pdf"
)

// Render lays out doc and returns the PDF bytes.
func Render(doc *document.Document) ([]byte, error) {
	e := &engine{
		doc:    doc,
		writer: pdf.NewWriter(),
	}
	e.newPage()
	for _, block := range doc.Blocks {
		e.renderBlock(block)
	}
	return e.writer.Write()
}

// ---------------------------------------------------------------------------
// engine — carries all mutable layout state
// ---------------------------------------------------------------------------

type engine struct {
	doc    *document.Document
	writer *pdf.Writer
	page   *pdf.Page // current page
	y      float64   // current Y cursor in doc-space (top-left origin, increases downward)
}

// newPage adds a fresh page to the writer and resets the Y cursor to the top margin.
func (e *engine) newPage() {
	e.page = e.writer.AddPage(e.doc.PageWidth, e.doc.PageHeight)
	e.y = e.doc.Margins.Top
}

// x returns the left edge of the content area (adjusted for list/quote depth).
func (e *engine) x(depth int) float64 {
	return e.doc.Margins.Left + float64(depth)*20
}

// contentW returns the usable text width (adjusted for depth indentation).
func (e *engine) contentW(depth int) float64 {
	return e.doc.ContentWidth() - float64(depth)*20
}

// checkPageBreak triggers a new page if adding h points would overflow.
func (e *engine) checkPageBreak(h float64) {
	bottom := e.doc.PageHeight - e.doc.Margins.Bottom
	if e.y+h > bottom {
		e.newPage()
	}
}

// advanceY moves the cursor down by delta, triggering a page break if needed.
func (e *engine) advanceY(delta float64) {
	e.checkPageBreak(delta)
	e.y += delta
}

// ---------------------------------------------------------------------------
// Block dispatch
// ---------------------------------------------------------------------------

func (e *engine) renderBlock(b *document.BlockNode) {
	switch b.Kind {
	case document.BlockHeading:
		e.renderParagraph(b, 8, 6)
	case document.BlockParagraph:
		e.renderParagraph(b, 4, 4)
	case document.BlockPre:
		e.renderPre(b)
	case document.BlockQuote:
		e.renderBlockquote(b)
	case document.BlockListItem:
		e.renderListItem(b)
	case document.BlockHRule:
		e.renderHRule()
	case document.BlockTable:
		e.renderTable(b)
	case document.BlockImage:
		// Image loading is not yet wired — render alt text as italicised paragraph.
		e.renderAltText(b)
	case document.BlockPageBreak:
		e.newPage()
	}
}

// ---------------------------------------------------------------------------
// Paragraph (and heading) rendering
// ---------------------------------------------------------------------------

// renderParagraph renders a block with inline content, adding spaceBefore/After
// points of vertical margin.
func (e *engine) renderParagraph(b *document.BlockNode, spaceBefore, spaceAfter float64) {
	if len(b.Inlines) == 0 {
		return
	}

	w := e.contentW(b.Depth)
	lines := breakLines(b.Inlines, w)
	if len(lines) == 0 {
		return
	}

	// Space before — but not at the very top of a page.
	if e.y > e.doc.Margins.Top+1 {
		e.advanceY(spaceBefore)
	}

	for _, line := range lines {
		lineH := lineHeight(line)
		e.checkPageBreak(lineH)

		xCursor := e.x(b.Depth)
		// Horizontal alignment
		if b.Align == "center" || b.Align == "right" {
			lineW := lineWidth(line)
			switch b.Align {
			case "center":
				xCursor += (w - lineW) / 2
			case "right":
				xCursor += w - lineW
			}
		}

		e.renderLine(line, xCursor, e.y+lineH*0.8) // baseline ≈ 80% of line height
		e.y += lineH
	}

	e.advanceY(spaceAfter)
}

// renderLine draws all runs in a single wrapped line at (x, baseline).
func (e *engine) renderLine(line []inlineRun, x, baseline float64) {
	for _, run := range line {
		e.page.SetFillColor(run.style.ColorR, run.style.ColorG, run.style.ColorB)
		e.page.SetFont(run.style.Face.ResourceKey(), run.style.Size)
		e.page.DrawText(x, baseline, run.text)
		x += fonts.MeasureString(run.style.Face, run.style.Size, run.text)
		if run.trailingSpace {
			x += fonts.SpaceWidth(run.style.Face, run.style.Size)
		}
	}
}

// ---------------------------------------------------------------------------
// Preformatted block
// ---------------------------------------------------------------------------

func (e *engine) renderPre(b *document.BlockNode) {
	const padV = 6.0

	// Draw background box — measure total height first.
	lines := splitPreLines(b.Inlines)
	if len(lines) == 0 {
		return
	}

	lh := fonts.LineHeight(10) // pre uses 10pt mono
	totalH := float64(len(lines))*lh + padV*2

	e.checkPageBreak(totalH)
	e.advanceY(4)

	// Light gray background
	e.page.DrawRect(e.x(b.Depth), e.y, e.contentW(b.Depth), totalH, 0.95, 0.95, 0.95)
	e.y += padV

	for _, line := range lines {
		baseline := e.y + lh*0.8
		e.page.SetFillColor(0.1, 0.1, 0.1)
		if len(line) > 0 {
			e.page.SetFont(line[0].style.Face.ResourceKey(), line[0].style.Size)
			e.page.DrawText(e.x(b.Depth)+4, baseline, line[0].text)
		}
		e.y += lh
	}

	e.y += padV
	e.advanceY(4)
}

// ---------------------------------------------------------------------------
// Blockquote
// ---------------------------------------------------------------------------

func (e *engine) renderBlockquote(b *document.BlockNode) {
	// Draw a left border bar.
	startY := e.y
	for _, child := range b.Children {
		e.renderBlock(child)
	}
	endY := e.y
	if endY > startY {
		e.page.DrawLine(
			e.x(b.Depth)-10, startY,
			e.x(b.Depth)-10, endY,
			2, 0.7, 0.7, 0.7,
		)
	}
}

// ---------------------------------------------------------------------------
// List item
// ---------------------------------------------------------------------------

func (e *engine) renderListItem(b *document.BlockNode) {
	if len(b.Inlines) == 0 {
		return
	}

	w := e.contentW(b.Depth)
	lines := breakLines(b.Inlines, w)
	if len(lines) == 0 {
		return
	}

	// Draw bullet/number before the first line.
	firstLineH := lineHeight(lines[0])
	e.checkPageBreak(firstLineH)

	markerX := e.x(b.Depth) - 16
	baseline := e.y + firstLineH*0.8

	if b.ListMarker == "disc" {
		// Draw a filled circle approximated by text bullet "•"
		e.page.SetFillColor(0, 0, 0)
		e.page.SetFont("F1", 8)
		e.page.DrawText(markerX, baseline, "•")
	} else {
		e.page.SetFillColor(0, 0, 0)
		e.page.SetFont("F1", 10)
		e.page.DrawText(markerX, baseline, b.ListMarker)
	}

	// Render the text lines.
	for i, line := range lines {
		lh := lineHeight(line)
		e.checkPageBreak(lh)
		xCursor := e.x(b.Depth)
		if i == 0 {
			e.renderLine(line, xCursor, e.y+lh*0.8)
		} else {
			e.renderLine(line, xCursor, e.y+lh*0.8)
		}
		e.y += lh
	}

	// Render nested child blocks (sub-lists, etc.)
	for _, child := range b.Children {
		e.renderBlock(child)
	}

	e.advanceY(2)
}

// ---------------------------------------------------------------------------
// Horizontal rule
// ---------------------------------------------------------------------------

func (e *engine) renderHRule() {
	e.advanceY(8)
	e.page.DrawLine(
		e.doc.Margins.Left, e.y,
		e.doc.PageWidth-e.doc.Margins.Right, e.y,
		0.5, 0.75, 0.75, 0.75,
	)
	e.advanceY(8)
}

// ---------------------------------------------------------------------------
// Table
// ---------------------------------------------------------------------------

func (e *engine) renderTable(tbl *document.BlockNode) {
	availW := e.contentW(0)
	colWidths := measureTable(tbl, availW)
	if len(colWidths) == 0 {
		return
	}

	const cellPadV = 4.0
	const cellPadH = 6.0

	e.advanceY(8)

	for _, row := range tbl.Children {
		// Compute row height = max cell height across all cells.
		rowH := 0.0
		for ci, cell := range row.Children {
			if ci >= len(colWidths) {
				break
			}
			h := cellRowHeight(cell, colWidths[ci], cellPadV)
			if h > rowH {
				rowH = h
			}
		}

		e.checkPageBreak(rowH)

		// Background fill for header rows.
		if row.HasBG {
			e.page.DrawRect(e.doc.Margins.Left, e.y, availW, rowH,
				row.BGR, row.BGG, row.BBB)
		}

		// Cell borders (top + bottom of each row).
		e.page.DrawLine(
			e.doc.Margins.Left, e.y,
			e.doc.Margins.Left+availW, e.y,
			0.25, 0.75, 0.75, 0.75,
		)

		// Render cell content.
		xCursor := e.doc.Margins.Left
		for ci, cell := range row.Children {
			if ci >= len(colWidths) {
				break
			}
			cw := colWidths[ci]
			cellLines := breakLines(cell.Inlines, cw-cellPadH*2)

			yCursor := e.y + cellPadV
			for _, line := range cellLines {
				lh := lineHeight(line)
				baseline := yCursor + lh*0.8
				e.renderLine(line, xCursor+cellPadH, baseline)
				yCursor += lh
			}

			// Vertical cell divider.
			if ci < len(colWidths)-1 {
				e.page.DrawLine(
					xCursor+cw, e.y,
					xCursor+cw, e.y+rowH,
					0.25, 0.75, 0.75, 0.75,
				)
			}
			xCursor += cw
		}

		e.y += rowH

		// Bottom border of row.
		e.page.DrawLine(
			e.doc.Margins.Left, e.y,
			e.doc.Margins.Left+availW, e.y,
			0.25, 0.75, 0.75, 0.75,
		)
	}

	e.advanceY(8)
}

// ---------------------------------------------------------------------------
// Image alt-text fallback
// ---------------------------------------------------------------------------

func (e *engine) renderAltText(b *document.BlockNode) {
	for _, in := range b.Inlines {
		if in.Kind == document.InlineImage && in.AltText != "" {
			altStyle := document.DefaultStyle().WithItalic(true).WithColor(0.5, 0.5, 0.5)
			fakeBlock := &document.BlockNode{
				Kind: document.BlockParagraph,
				Inlines: []document.InlineNode{
					{Kind: document.InlineText, Text: "[" + in.AltText + "]", Style: altStyle},
				},
			}
			e.renderParagraph(fakeBlock, 4, 4)
		}
	}
}

// ---------------------------------------------------------------------------
// Line breaking — greedy word-wrap
// ---------------------------------------------------------------------------

// inlineRun is a single styled word or phrase on a line, ready to draw.
type inlineRun struct {
	text          string
	style         document.TextStyle
	trailingSpace bool // whether a space follows this run on the same line
}

// breakLines wraps the inline nodes into lines that fit within width.
// Returns a slice of lines, each line being a slice of inlineRuns.
func breakLines(inlines []document.InlineNode, width float64) [][]inlineRun {
	if width <= 0 || len(inlines) == 0 {
		return nil
	}

	var lines [][]inlineRun
	var currentLine []inlineRun
	lineW := 0.0

	commitLine := func() {
		if len(currentLine) > 0 {
			// Clear trailing space flag on last run of the line.
			currentLine[len(currentLine)-1].trailingSpace = false
			lines = append(lines, currentLine)
			currentLine = nil
			lineW = 0
		}
	}

	for _, in := range inlines {
		if in.Kind == document.InlineBreak {
			commitLine()
			continue
		}
		if in.Kind != document.InlineText {
			continue
		}

		spW := fonts.SpaceWidth(in.Style.Face, in.Style.Size)
		words := splitWords(in.Text)

		for wi, word := range words {
			ww := fonts.MeasureString(in.Style.Face, in.Style.Size, word)

			// Add a space before this word if we're not at the start of a line.
			spaceBefore := 0.0
			if lineW > 0 || wi > 0 {
				spaceBefore = spW
			}
			if lineW > 0 && lineW+spaceBefore+ww > width {
				commitLine()
				spaceBefore = 0
			}

			// Mark trailing space on the previous run.
			if lineW > 0 && spaceBefore > 0 && len(currentLine) > 0 {
				currentLine[len(currentLine)-1].trailingSpace = true
			}

			currentLine = append(currentLine, inlineRun{
				text:  word,
				style: in.Style,
			})
			lineW += spaceBefore + ww
		}
	}
	commitLine()
	return lines
}

// splitWords splits a string on whitespace, returning non-empty tokens.
func splitWords(s string) []string {
	return strings.Fields(s)
}

// lineHeight returns the max line height across all runs on a line.
func lineHeight(line []inlineRun) float64 {
	lh := fonts.LineHeight(12) // minimum
	for _, run := range line {
		if h := fonts.LineHeight(run.style.Size); h > lh {
			lh = h
		}
	}
	return lh
}

// lineWidth returns the total rendered width of a line.
func lineWidth(line []inlineRun) float64 {
	w := 0.0
	for _, run := range line {
		w += fonts.MeasureString(run.style.Face, run.style.Size, run.text)
		if run.trailingSpace {
			w += fonts.SpaceWidth(run.style.Face, run.style.Size)
		}
	}
	return w
}

// splitPreLines converts a pre block's inlines (with InlineBreaks) into
// rows of inlineRuns — one row per line, preserving exact text.
func splitPreLines(inlines []document.InlineNode) [][]inlineRun {
	var lines [][]inlineRun
	var current []inlineRun

	for _, in := range inlines {
		if in.Kind == document.InlineBreak {
			lines = append(lines, current)
			current = nil
			continue
		}
		if in.Kind == document.InlineText {
			current = append(current, inlineRun{text: in.Text, style: in.Style})
		}
	}
	if len(current) > 0 {
		lines = append(lines, current)
	}
	return lines
}
