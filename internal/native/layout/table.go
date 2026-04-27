package layout

import (
	"github.com/HeartBeat1608/htmlpdf/internal/document"
	"github.com/HeartBeat1608/htmlpdf/internal/native/fonts"
)

// tableColumn describes a single measured column.
type tableColumn struct {
	minWidth float64 // widest single word that cannot wrap
	maxWidth float64 // total width if rendered on one line
}

// measureTable returns the column widths to use for rendering, distributed
// within the available content width using a min-then-proportional algorithm:
//
//  1. Each column gets at least its minWidth (longest unbreakable word).
//  2. Remaining space is distributed proportionally by maxWidth.
//
// This avoids columns that are either impossibly narrow or unnecessarily wide.
func measureTable(tbl *document.BlockNode, availWidth float64) []float64 {
	if len(tbl.Children) == 0 {
		return nil
	}

	// Count columns from the widest row.
	nCols := 0
	for _, row := range tbl.Children {
		if len(row.Children) > nCols {
			nCols = len(row.Children)
		}
	}
	if nCols == 0 {
		return nil
	}

	cols := make([]tableColumn, nCols)

	// Measure every cell in every row.
	for _, row := range tbl.Children {
		for ci, cell := range row.Children {
			if ci >= nCols {
				break
			}
			min, max := measureCellWidth(cell)
			if min > cols[ci].minWidth {
				cols[ci].minWidth = min
			}
			if max > cols[ci].maxWidth {
				cols[ci].maxWidth = max
			}
		}
	}

	// Add a small padding on each side of every cell.
	const cellPadH = 6.0
	for i := range cols {
		cols[i].minWidth += cellPadH * 2
		cols[i].maxWidth += cellPadH * 2
	}

	// Total min and max across all columns.
	totalMin, totalMax := 0.0, 0.0
	for _, c := range cols {
		totalMin += c.minWidth
		totalMax += c.maxWidth
	}

	widths := make([]float64, nCols)

	if totalMin >= availWidth {
		// Constrained: distribute evenly — the table will overflow but that
		// is better than zero-width columns.
		each := availWidth / float64(nCols)
		for i := range widths {
			widths[i] = each
		}
		return widths
	}

	if totalMax <= availWidth {
		// All columns fit at their natural width.
		for i, c := range cols {
			widths[i] = c.maxWidth
		}
		return widths
	}

	// Distribute remaining space (after satisfying minWidths) proportionally
	// by the excess (maxWidth - minWidth) of each column.
	remaining := availWidth - totalMin
	excessTotal := totalMax - totalMin
	for i, c := range cols {
		excess := c.maxWidth - c.minWidth
		if excessTotal > 0 {
			widths[i] = c.minWidth + remaining*(excess/excessTotal)
		} else {
			widths[i] = c.minWidth
		}
	}
	return widths
}

// measureCellWidth returns the (minWidth, maxWidth) of a table cell's inline
// content.  minWidth is the longest word; maxWidth is the full unwrapped line.
func measureCellWidth(cell *document.BlockNode) (minW, maxW float64) {
	for _, in := range cell.Inlines {
		if in.Kind != document.InlineText {
			continue
		}
		face := in.Style.Face
		size := in.Style.Size

		// maxWidth: full string width
		w := fonts.MeasureString(face, size, in.Text)
		if w > maxW {
			maxW = w
		}

		// minWidth: widest individual word
		for _, word := range splitWords(in.Text) {
			ww := fonts.MeasureString(face, size, word)
			if ww > minW {
				minW = ww
			}
		}
	}
	return
}

// cellRowHeight computes the height in points needed to render cell at colWidth.
func cellRowHeight(cell *document.BlockNode, colWidth float64, padding float64) float64 {
	const minHeight = 20.0
	h := wrapInlines(cell.Inlines, colWidth-padding*2) * 1.0
	if h < minHeight {
		h = minHeight
	}
	return h + padding*2
}

// wrapInlines returns the total height (in points) consumed by the inline runs
// when wrapped into the given width, using the greedy line-break algorithm.
func wrapInlines(inlines []document.InlineNode, width float64) float64 {
	if width <= 0 || len(inlines) == 0 {
		return 0
	}

	lineH := 0.0
	totalH := 0.0
	lineW := 0.0

	newLine := func(h float64) {
		if lineH == 0 {
			lineH = h
		}
		totalH += lineH
		lineH = h
		lineW = 0
	}

	for _, in := range inlines {
		if in.Kind == document.InlineBreak {
			newLine(fonts.LineHeight(12))
			continue
		}
		if in.Kind != document.InlineText {
			continue
		}

		lh := fonts.LineHeight(in.Style.Size)
		if lh > lineH {
			lineH = lh
		}

		spW := fonts.SpaceWidth(in.Style.Face, in.Style.Size)
		words := splitWords(in.Text)

		for i, word := range words {
			ww := fonts.MeasureString(in.Style.Face, in.Style.Size, word)
			sep := 0.0
			if i > 0 || lineW > 0 {
				sep = spW
			}
			if lineW > 0 && lineW+sep+ww > width {
				newLine(lh)
				sep = 0
			}
			lineW += sep + ww
		}
	}
	totalH += lineH
	return totalH
}
