// Package document defines the intermediate box tree produced by the HTML
// parser and consumed by the layout engine.
//
// The tree has two levels:
//
//   - Block nodes  — block-level boxes (paragraphs, headings, lists, tables…)
//   - Inline runs  — a contiguous sequence of styled text or an image within
//     a block
//
// This is intentionally simpler than a full CSS box model. The native renderer
// handles a well-defined subset of HTML; anything it cannot represent faithfully
// is approximated or skipped with an ErrUnsupported annotation on the node.
package document

import "github.com/HeartBeat1608/htmlpdf/native/fonts"

// ---------------------------------------------------------------------------
// Text style
// ---------------------------------------------------------------------------

// TextStyle carries the visual properties of an inline run.
// All colour fields are normalised to [0.0, 1.0].
type TextStyle struct {
	Face   fonts.Face
	Size   float64 // points
	ColorR float64
	ColorG float64
	ColorB float64
}

// DefaultStyle returns the base body style: 12pt Helvetica, black.
func DefaultStyle() TextStyle {
	return TextStyle{
		Face:   fonts.Helvetica,
		Size:   12,
		ColorR: 0,
		ColorG: 0,
		ColorB: 0,
	}
}

// WithBold returns a copy of s with the bold flag applied.
func (s TextStyle) WithBold(bold bool) TextStyle {
	s.Face = fonts.FaceFromStyle(familyOf(s.Face), bold, italicOf(s.Face))
	return s
}

// WithItalic returns a copy of s with the italic flag applied.
func (s TextStyle) WithItalic(italic bool) TextStyle {
	s.Face = fonts.FaceFromStyle(familyOf(s.Face), boldOf(s.Face), italic)
	return s
}

// WithFamily returns a copy of s with a new font family.
// family must be "serif", "mono", or "" (sans-serif).
func (s TextStyle) WithFamily(family string) TextStyle {
	s.Face = fonts.FaceFromStyle(family, boldOf(s.Face), italicOf(s.Face))
	return s
}

// WithSize returns a copy of s with a new font size.
func (s TextStyle) WithSize(pt float64) TextStyle {
	s.Size = pt
	return s
}

// WithColor returns a copy of s with a new fill colour.
func (s TextStyle) WithColor(r, g, b float64) TextStyle {
	s.ColorR, s.ColorG, s.ColorB = r, g, b
	return s
}

// IsBold reports whether the face carries bold weight.
func (s TextStyle) IsBold() bool { return boldOf(s.Face) }

// IsItalic reports whether the face carries italic/oblique style.
func (s TextStyle) IsItalic() bool { return italicOf(s.Face) }

// family helpers — derive family/bold/italic back from a Face value.
func familyOf(f fonts.Face) string {
	switch f {
	case fonts.TimesRoman, fonts.TimesBold:
		return "serif"
	case fonts.Courier, fonts.CourierBold:
		return "mono"
	default:
		return ""
	}
}

func boldOf(f fonts.Face) bool {
	switch f {
	case fonts.HelveticaBold, fonts.TimesBold, fonts.CourierBold:
		return true
	}
	return false
}

func italicOf(f fonts.Face) bool {
	return f == fonts.HelveticaOblique
}

// ---------------------------------------------------------------------------
// Inline nodes
// ---------------------------------------------------------------------------

// InlineKind distinguishes the two kinds of inline content.
type InlineKind int

const (
	InlineText  InlineKind = iota // a styled text run
	InlineImage                   // an embedded image
	InlineBreak                   // explicit <br>
)

// InlineNode is one piece of inline content inside a block.
type InlineNode struct {
	Kind InlineKind

	// InlineText fields
	Text  string
	Style TextStyle

	// InlineImage fields
	Src     string  // original src= attribute value
	AltText string  // alt= text — used if image cannot be loaded
	ImgW    float64 // desired render width in points (0 = auto)
	ImgH    float64 // desired render height in points (0 = auto)
}

// ---------------------------------------------------------------------------
// Block nodes
// ---------------------------------------------------------------------------

// BlockKind classifies block-level boxes.
type BlockKind int

const (
	BlockParagraph BlockKind = iota // <p>, <div>, generic block
	BlockHeading                    // <h1>–<h6>
	BlockPre                        // <pre> — preserves whitespace
	BlockQuote                      // <blockquote>
	BlockListItem                   // <li>
	BlockHRule                      // <hr>
	BlockTable                      // <table>
	BlockTableRow                   // <tr>
	BlockTableCell                  // <td> / <th>
	BlockImage                      // <figure> or block-level <img>
	BlockPageBreak                  // synthetic — injected by layout
)

// BlockNode is a block-level box in the document tree.
type BlockNode struct {
	Kind BlockKind

	// Heading level (1–6); only valid when Kind == BlockHeading.
	Level int

	// Inline content — text runs and inline images inside this block.
	// Nil for structural containers (BlockTable, BlockTableRow).
	Inlines []InlineNode

	// Children — nested blocks (list items, table rows/cells, blockquotes).
	Children []*BlockNode

	// List decoration: "disc", "decimal", "none".
	ListMarker string

	// Indentation depth (counts nested blockquotes and list levels).
	Depth int

	// Background fill colour (RGB, 0–1); used for table header rows.
	// Only applied when HasBG is true.
	HasBG         bool
	BGR, BGG, BBB float64

	// Alignment: "left" | "center" | "right" | "justify" — default "left".
	Align string
}

// ---------------------------------------------------------------------------
// Page-level document
// ---------------------------------------------------------------------------

// Margins holds page margins in points.
type Margins struct {
	Top, Right, Bottom, Left float64
}

// DefaultMargins returns 1-inch margins on all sides (72 pt).
func DefaultMargins() Margins {
	return Margins{Top: 72, Right: 72, Bottom: 72, Left: 72}
}

// Document is the complete parsed representation of an HTML document.
type Document struct {
	Title   string
	Blocks  []*BlockNode
	Margins Margins

	// Page dimensions in points.
	PageWidth  float64
	PageHeight float64
}

// NewDocument returns an A4 document with default margins.
func NewDocument() *Document {
	return &Document{
		PageWidth:  595.28,
		PageHeight: 841.89,
		Margins:    DefaultMargins(),
	}
}

// ContentWidth returns the usable text column width (page minus left/right margins).
func (d *Document) ContentWidth() float64 {
	return d.PageWidth - d.Margins.Left - d.Margins.Right
}
