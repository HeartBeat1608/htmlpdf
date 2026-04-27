package parse

import (
	"strings"
	"testing"

	"github.com/HeartBeat1608/htmlpdf/internal/document"
	"github.com/HeartBeat1608/htmlpdf/internal/native/fonts"
)

// ---- helpers ---------------------------------------------------------------

func mustParse(t *testing.T, html string) *document.Document {
	t.Helper()
	doc, err := Parse([]byte(html))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	return doc
}

func blockText(b *document.BlockNode) string {
	var sb strings.Builder
	for _, in := range b.Inlines {
		if in.Kind == document.InlineText {
			sb.WriteString(in.Text)
		}
	}
	return sb.String()
}

func findBlock(blocks []*document.BlockNode, kind document.BlockKind) *document.BlockNode {
	for _, b := range blocks {
		if b.Kind == kind {
			return b
		}
	}
	return nil
}

// ---- document basics -------------------------------------------------------

func TestParseEmpty(t *testing.T) {
	doc := mustParse(t, "")
	if doc == nil {
		t.Fatal("Parse returned nil")
	}
}

func TestParseReturnsDocument(t *testing.T) {
	doc := mustParse(t, "<p>hello</p>")
	if doc.PageWidth == 0 || doc.PageHeight == 0 {
		t.Error("document has zero page dimensions")
	}
	if doc.ContentWidth() <= 0 {
		t.Error("document ContentWidth() <= 0")
	}
}

// ---- heading parsing -------------------------------------------------------

func TestHeadingLevels(t *testing.T) {
	for level := 1; level <= 6; level++ {
		html := `<h` + string(rune('0'+level)) + `>Heading` + string(rune('0'+level)) + `</h` + string(rune('0'+level)) + `>`
		doc := mustParse(t, html)
		var found *document.BlockNode
		for _, b := range doc.Blocks {
			if b.Kind == document.BlockHeading && b.Level == level {
				found = b
				break
			}
		}
		if found == nil {
			t.Errorf("h%d not parsed as BlockHeading level %d", level, level)
			continue
		}
		// Heading text should be bold
		if len(found.Inlines) == 0 {
			t.Errorf("h%d has no inlines", level)
			continue
		}
		if !found.Inlines[0].Style.IsBold() {
			t.Errorf("h%d inline is not bold", level)
		}
	}
}

func TestHeadingFontSizes(t *testing.T) {
	doc := mustParse(t, `<h1>A</h1><h2>B</h2><h3>C</h3>`)
	sizes := map[int]float64{}
	for _, b := range doc.Blocks {
		if b.Kind == document.BlockHeading && len(b.Inlines) > 0 {
			sizes[b.Level] = b.Inlines[0].Style.Size
		}
	}
	if sizes[1] <= sizes[2] {
		t.Errorf("h1 size (%.0f) should be > h2 size (%.0f)", sizes[1], sizes[2])
	}
	if sizes[2] <= sizes[3] {
		t.Errorf("h2 size (%.0f) should be > h3 size (%.0f)", sizes[2], sizes[3])
	}
}

// ---- paragraph / inline ----------------------------------------------------

func TestParagraphText(t *testing.T) {
	doc := mustParse(t, `<p>Hello, world.</p>`)
	b := findBlock(doc.Blocks, document.BlockParagraph)
	if b == nil {
		t.Fatal("no BlockParagraph found")
	}
	if !strings.Contains(blockText(b), "Hello, world.") {
		t.Errorf("paragraph text %q does not contain expected content", blockText(b))
	}
}

func TestBoldInline(t *testing.T) {
	doc := mustParse(t, `<p>normal <strong>bold</strong> normal</p>`)
	b := findBlock(doc.Blocks, document.BlockParagraph)
	if b == nil {
		t.Fatal("no paragraph")
	}
	var boldFound bool
	for _, in := range b.Inlines {
		if in.Kind == document.InlineText && in.Text == "bold" && in.Style.IsBold() {
			boldFound = true
		}
	}
	if !boldFound {
		t.Error("<strong> text not parsed as bold inline")
	}
}

func TestItalicInline(t *testing.T) {
	doc := mustParse(t, `<p>text <em>italic</em> text</p>`)
	b := findBlock(doc.Blocks, document.BlockParagraph)
	if b == nil {
		t.Fatal("no paragraph")
	}
	var itFound bool
	for _, in := range b.Inlines {
		if in.Kind == document.InlineText && in.Text == "italic" && in.Style.IsItalic() {
			itFound = true
		}
	}
	if !itFound {
		t.Error("<em> text not parsed as italic inline")
	}
}

func TestCodeInline(t *testing.T) {
	doc := mustParse(t, `<p>run <code>go test</code> now</p>`)
	b := findBlock(doc.Blocks, document.BlockParagraph)
	if b == nil {
		t.Fatal("no paragraph")
	}
	for _, in := range b.Inlines {
		if in.Kind == document.InlineText && strings.Contains(in.Text, "go test") {
			if in.Style.Face != fonts.Courier && in.Style.Face != fonts.CourierBold {
				t.Errorf("<code> inline has face %d, want Courier family", in.Style.Face)
			}
			return
		}
	}
	t.Error("<code> inline not found")
}

func TestLinkColor(t *testing.T) {
	doc := mustParse(t, `<p><a href="https://example.com">link</a></p>`)
	b := findBlock(doc.Blocks, document.BlockParagraph)
	if b == nil {
		t.Fatal("no paragraph")
	}
	for _, in := range b.Inlines {
		if in.Kind == document.InlineText && in.Text == "link" {
			if in.Style.ColorB < 0.5 {
				t.Errorf("link text not blue: R=%.2f G=%.2f B=%.2f",
					in.Style.ColorR, in.Style.ColorG, in.Style.ColorB)
			}
			return
		}
	}
	t.Error("link inline not found")
}

func TestLineBreak(t *testing.T) {
	doc := mustParse(t, `<p>line one<br>line two</p>`)
	b := findBlock(doc.Blocks, document.BlockParagraph)
	if b == nil {
		t.Fatal("no paragraph")
	}
	var hasBreak bool
	for _, in := range b.Inlines {
		if in.Kind == document.InlineBreak {
			hasBreak = true
		}
	}
	if !hasBreak {
		t.Error("<br> not emitted as InlineBreak")
	}
}

func TestImageInline(t *testing.T) {
	doc := mustParse(t, `<p><img src="photo.jpg" alt="A photo"></p>`)
	b := findBlock(doc.Blocks, document.BlockParagraph)
	if b == nil {
		t.Fatal("no paragraph")
	}
	for _, in := range b.Inlines {
		if in.Kind == document.InlineImage {
			if in.Src != "photo.jpg" {
				t.Errorf("img src = %q, want photo.jpg", in.Src)
			}
			if in.AltText != "A photo" {
				t.Errorf("img alt = %q, want 'A photo'", in.AltText)
			}
			return
		}
	}
	t.Error("InlineImage not found")
}

// ---- lists -----------------------------------------------------------------

func TestUnorderedList(t *testing.T) {
	doc := mustParse(t, `<ul><li>alpha</li><li>beta</li></ul>`)
	var items []*document.BlockNode
	for _, b := range doc.Blocks {
		if b.Kind == document.BlockListItem {
			items = append(items, b)
		}
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 list items, got %d", len(items))
	}
	if items[0].ListMarker != "disc" {
		t.Errorf("ul marker = %q, want disc", items[0].ListMarker)
	}
}

func TestOrderedList(t *testing.T) {
	doc := mustParse(t, `<ol><li>first</li><li>second</li><li>third</li></ol>`)
	var items []*document.BlockNode
	for _, b := range doc.Blocks {
		if b.Kind == document.BlockListItem {
			items = append(items, b)
		}
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 list items, got %d", len(items))
	}
	for i, item := range items {
		want := string(rune('1'+i)) + "."
		if item.ListMarker != want {
			t.Errorf("ol item %d marker = %q, want %q", i, item.ListMarker, want)
		}
	}
}

// ---- horizontal rule -------------------------------------------------------

func TestHRule(t *testing.T) {
	doc := mustParse(t, `<p>before</p><hr><p>after</p>`)
	if findBlock(doc.Blocks, document.BlockHRule) == nil {
		t.Error("<hr> not parsed as BlockHRule")
	}
}

// ---- preformatted ----------------------------------------------------------

func TestPreBlock(t *testing.T) {
	doc := mustParse(t, "<pre>func main() {\n\tfmt.Println()\n}</pre>")
	b := findBlock(doc.Blocks, document.BlockPre)
	if b == nil {
		t.Fatal("no BlockPre found")
	}
	// Should have an InlineBreak between the lines
	var hasBreak bool
	for _, in := range b.Inlines {
		if in.Kind == document.InlineBreak {
			hasBreak = true
		}
	}
	if !hasBreak {
		t.Error("pre block should have InlineBreaks at newlines")
	}
	// Font should be monospace
	for _, in := range b.Inlines {
		if in.Kind == document.InlineText {
			f := in.Style.Face
			if f != fonts.Courier && f != fonts.CourierBold {
				t.Errorf("pre inline has face %d, want Courier family", f)
			}
			break
		}
	}
}

// ---- table -----------------------------------------------------------------

func TestTableStructure(t *testing.T) {
	html := `<table>
		<thead><tr><th>Name</th><th>Age</th></tr></thead>
		<tbody>
			<tr><td>Alice</td><td>30</td></tr>
			<tr><td>Bob</td><td>25</td></tr>
		</tbody>
	</table>`
	doc := mustParse(t, html)
	tbl := findBlock(doc.Blocks, document.BlockTable)
	if tbl == nil {
		t.Fatal("no BlockTable found")
	}
	if len(tbl.Children) != 3 { // 1 header + 2 data rows
		t.Errorf("expected 3 rows, got %d", len(tbl.Children))
	}
	// First row is header — should have background
	if !tbl.Children[0].HasBG {
		t.Error("header row should have background fill")
	}
	// Header row cells should be bold
	if len(tbl.Children[0].Children) != 2 {
		t.Fatalf("header row has %d cells, want 2", len(tbl.Children[0].Children))
	}
	for _, cell := range tbl.Children[0].Children {
		if len(cell.Inlines) == 0 {
			continue
		}
		if !cell.Inlines[0].Style.IsBold() {
			t.Error("header cell text should be bold")
		}
	}
}

func TestTableCellText(t *testing.T) {
	html := `<table><tr><td>hello</td><td>world</td></tr></table>`
	doc := mustParse(t, html)
	tbl := findBlock(doc.Blocks, document.BlockTable)
	if tbl == nil {
		t.Fatal("no BlockTable")
	}
	row := tbl.Children[0]
	if len(row.Children) != 2 {
		t.Fatalf("expected 2 cells, got %d", len(row.Children))
	}
	if blockText(row.Children[0]) != "hello" {
		t.Errorf("cell 0 text = %q, want hello", blockText(row.Children[0]))
	}
}

// ---- CSS parsing -----------------------------------------------------------

func TestCSSFontSizePt(t *testing.T) {
	doc := mustParse(t, `<p style="font-size: 18pt">text</p>`)
	b := findBlock(doc.Blocks, document.BlockParagraph)
	if b == nil || len(b.Inlines) == 0 {
		t.Fatal("no paragraph")
	}
	if b.Inlines[0].Style.Size != 18 {
		t.Errorf("font-size 18pt → %.1f, want 18", b.Inlines[0].Style.Size)
	}
}

func TestCSSFontSizePx(t *testing.T) {
	// 16px × 0.75 = 12pt
	doc := mustParse(t, `<p style="font-size: 16px">text</p>`)
	b := findBlock(doc.Blocks, document.BlockParagraph)
	if b == nil || len(b.Inlines) == 0 {
		t.Fatal("no paragraph")
	}
	if b.Inlines[0].Style.Size != 12 {
		t.Errorf("font-size 16px → %.1f, want 12", b.Inlines[0].Style.Size)
	}
}

func TestCSSColor(t *testing.T) {
	doc := mustParse(t, `<p style="color: #ff0000">red</p>`)
	b := findBlock(doc.Blocks, document.BlockParagraph)
	if b == nil || len(b.Inlines) == 0 {
		t.Fatal("no paragraph")
	}
	in := b.Inlines[0]
	if in.Style.ColorR < 0.99 || in.Style.ColorG > 0.01 || in.Style.ColorB > 0.01 {
		t.Errorf("color #ff0000 → R=%.2f G=%.2f B=%.2f", in.Style.ColorR, in.Style.ColorG, in.Style.ColorB)
	}
}

func TestCSSColorRGB(t *testing.T) {
	doc := mustParse(t, `<p style="color: rgb(0, 128, 255)">blue</p>`)
	b := findBlock(doc.Blocks, document.BlockParagraph)
	if b == nil || len(b.Inlines) == 0 {
		t.Fatal("no paragraph")
	}
	in := b.Inlines[0]
	if in.Style.ColorB < 0.99 || in.Style.ColorR > 0.01 {
		t.Errorf("unexpected colour R=%.2f G=%.2f B=%.2f", in.Style.ColorR, in.Style.ColorG, in.Style.ColorB)
	}
}

func TestCSSShortHex(t *testing.T) {
	// #f00 → #ff0000
	doc := mustParse(t, `<p style="color: #f00">red</p>`)
	b := findBlock(doc.Blocks, document.BlockParagraph)
	if b == nil || len(b.Inlines) == 0 {
		t.Fatal("no paragraph")
	}
	in := b.Inlines[0]
	if in.Style.ColorR < 0.99 || in.Style.ColorG > 0.01 {
		t.Errorf("short hex #f00 → R=%.2f G=%.2f B=%.2f", in.Style.ColorR, in.Style.ColorG, in.Style.ColorB)
	}
}

// ---- whitespace normalisation ----------------------------------------------

func TestNormaliseWhitespace(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"hello world", "hello world"},
		{"  lots   of   space  ", "lots of space"},
		{"line\n  break", "line break"},
		{"\t\ttabs\t\there", "tabs here"},
		{"", ""},
	}
	for _, c := range cases {
		got := normaliseWhitespace(c.in)
		if got != c.want {
			t.Errorf("normaliseWhitespace(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ---- ignored elements ------------------------------------------------------

func TestScriptIgnored(t *testing.T) {
	doc := mustParse(t, `<script>alert("xss")</script><p>visible</p>`)
	for _, b := range doc.Blocks {
		for _, in := range b.Inlines {
			if strings.Contains(in.Text, "alert") {
				t.Error("script content leaked into document")
			}
		}
	}
}

func TestStyleIgnored(t *testing.T) {
	doc := mustParse(t, `<style>body { color: red }</style><p>text</p>`)
	for _, b := range doc.Blocks {
		for _, in := range b.Inlines {
			if strings.Contains(in.Text, "color") {
				t.Error("style element content leaked into document")
			}
		}
	}
}

// ---- parseColor unit tests -------------------------------------------------

func TestParseColor(t *testing.T) {
	cases := []struct {
		input   string
		r, g, b float64
		ok      bool
	}{
		{"#ff0000", 1, 0, 0, true},
		{"#000000", 0, 0, 0, true},
		{"#ffffff", 1, 1, 1, true},
		{"#f00", 1, 0, 0, true},
		{"rgb(255,0,0)", 1, 0, 0, true},
		{"rgb(0, 128, 0)", 0, 128.0 / 255, 0, true},
		{"invalid", 0, 0, 0, false},
		{"#gg0000", 0, 0, 0, false},
	}
	for _, c := range cases {
		r, g, b, ok := parseColor(c.input)
		if ok != c.ok {
			t.Errorf("parseColor(%q) ok=%v, want %v", c.input, ok, c.ok)
			continue
		}
		if ok {
			const eps = 0.005
			if abs(r-c.r) > eps || abs(g-c.g) > eps || abs(b-c.b) > eps {
				t.Errorf("parseColor(%q) = (%.3f,%.3f,%.3f), want (%.3f,%.3f,%.3f)",
					c.input, r, g, b, c.r, c.g, c.b)
			}
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// ---- parseFontSize unit tests ----------------------------------------------

func TestParseFontSize(t *testing.T) {
	cases := []struct {
		input string
		want  float64
	}{
		{"12pt", 12},
		{"16px", 12}, // 16 × 0.75
		{"1em", 12},  // 1 × 12
		{"2rem", 24}, // 2 × 12
		{"medium", 12},
		{"large", 14},
		{"small", 10},
		{"invalid", 0},
	}
	for _, c := range cases {
		got := parseFontSize(c.input)
		if got != c.want {
			t.Errorf("parseFontSize(%q) = %.1f, want %.1f", c.input, got, c.want)
		}
	}
}

// ---- full document integration test ----------------------------------------

func TestFullDocumentStructure(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
  <h1>Title</h1>
  <p>An introductory paragraph with <strong>bold</strong> and <em>italic</em> text.</p>
  <h2>Section</h2>
  <ul>
    <li>Item one</li>
    <li>Item two</li>
  </ul>
  <hr>
  <pre>code block</pre>
  <table>
    <tr><th>Col A</th><th>Col B</th></tr>
    <tr><td>1</td><td>2</td></tr>
  </table>
</body>
</html>`

	doc := mustParse(t, html)

	kinds := map[document.BlockKind]int{}
	for _, b := range doc.Blocks {
		kinds[b.Kind]++
	}

	checks := []struct {
		kind document.BlockKind
		name string
		min  int
	}{
		{document.BlockHeading, "heading", 2},
		{document.BlockParagraph, "paragraph", 1},
		{document.BlockListItem, "list item", 2},
		{document.BlockHRule, "hr", 1},
		{document.BlockPre, "pre", 1},
		{document.BlockTable, "table", 1},
	}
	for _, c := range checks {
		if kinds[c.kind] < c.min {
			t.Errorf("expected ≥%d %s blocks, got %d", c.min, c.name, kinds[c.kind])
		}
	}
}
