package layout

import (
	"testing"

	"github.com/HeartBeat1608/htmlpdf/internal/document"
)

func TestMeasureCellWidthKeepsShortPhraseTogether(t *testing.T) {
	cell := &document.BlockNode{
		Inlines: []document.InlineNode{
			{
				Kind:  document.InlineText,
				Text:  "Unit Price",
				Style: document.DefaultStyle().WithBold(true),
			},
		},
	}

	minW, maxW := measureCellWidth(cell)
	if minW != maxW {
		t.Fatalf("expected short two-word phrase to keep full width together, got min=%.2f max=%.2f", minW, maxW)
	}
}

func TestMeasureCellWidthTracksMultiInlineLineWidth(t *testing.T) {
	cell := &document.BlockNode{
		Inlines: []document.InlineNode{
			{
				Kind:  document.InlineText,
				Text:  "Total",
				Style: document.DefaultStyle().WithBold(true).WithSize(14),
			},
			{
				Kind:  document.InlineText,
				Text:  "Due",
				Style: document.DefaultStyle().WithBold(true).WithSize(14),
			},
		},
	}

	minW, maxW := measureCellWidth(cell)
	if minW != maxW {
		t.Fatalf("expected split inline phrase to measure as a single short label, got min=%.2f max=%.2f", minW, maxW)
	}
}
