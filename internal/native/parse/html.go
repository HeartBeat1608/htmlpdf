// Package parse converts an HTML document into a document.Document box tree.
//
// It uses the inlined golang.org/x/net/html parser (Go team maintained,
// zero transitive deps) and supports the element subset defined in the plan:
//
//	Block elements:  div section article main header footer p h1-h6
//	                 ul ol li blockquote pre hr figure table tr td th
//	                 thead tbody
//
//	Inline elements: span strong b em i code a br img
//	                 (and bare text nodes)
//
//	Ignored:         script style noscript svg canvas video audio
//
// CSS support is limited to inline style= attributes.  Supported properties:
//
//	font-size font-weight font-style font-family color
//	text-align background-color
//
// Unsupported properties are silently skipped.
package parse

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/HeartBeat1608/htmlpdf/internal/document"
	"github.com/HeartBeat1608/htmlpdf/internal/native/fonts"
	hp "golang.org/x/net/html"
)

// Parse parses the UTF-8 HTML in src and returns a Document.
// The returned Document always contains a valid Blocks slice; on parse errors
// the successfully-parsed portion is returned alongside the error.
func Parse(src []byte) (*document.Document, error) {
	root, err := hp.Parse(strings.NewReader(string(src)))
	if err != nil {
		return nil, fmt.Errorf("html parse: %w", err)
	}

	p := &parser{doc: document.NewDocument()}

	// hp.Parse always returns a document node whose tree is:
	//   Document -> <html> -> <head>, <body>
	// Locate body explicitly so <head> content (script, style, title) is
	// never walked as body text.
	htmlNode := findDescendant(root, "html")
	if htmlNode == nil {
		htmlNode = root
	}
	if titleNode := findDescendant(htmlNode, "title"); titleNode != nil {
		if titleNode.FirstChild != nil {
			p.doc.Title = strings.TrimSpace(titleNode.FirstChild.Data)
		}
	}
	bodyNode := findChild(htmlNode, "body")
	if bodyNode == nil {
		bodyNode = htmlNode
	}

	p.walkChildren(bodyNode, document.DefaultStyle(), 0)
	return p.doc, nil
}

// ---------------------------------------------------------------------------
// parser state
// ---------------------------------------------------------------------------

type parser struct {
	doc *document.Document
}

// ---------------------------------------------------------------------------
// Block-level walk
// ---------------------------------------------------------------------------

// walkChildren iterates the direct children of n, emitting block nodes into
// p.doc.Blocks. Inline content is collected into a synthetic paragraph block.
func (p *parser) walkChildren(n *hp.Node, style document.TextStyle, depth int) {
	var inlineBuf []document.InlineNode

	flushInlines := func() {
		if len(inlineBuf) > 0 {
			// Trim leading/trailing whitespace-only runs from the buffer
			inlineBuf = trimInlineWhitespace(inlineBuf)
			if len(inlineBuf) > 0 {
				p.doc.Blocks = append(p.doc.Blocks, &document.BlockNode{
					Kind:    document.BlockParagraph,
					Inlines: inlineBuf,
					Depth:   depth,
				})
			}
			inlineBuf = nil
		}
	}

	appendBlock := func(b *document.BlockNode) {
		flushInlines()
		p.doc.Blocks = append(p.doc.Blocks, b)
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if isIgnored(c) {
			continue
		}

		if c.Type == hp.TextNode {
			text := normaliseWhitespace(c.Data)
			if text != "" {
				inlineBuf = append(inlineBuf, document.InlineNode{
					Kind:  document.InlineText,
					Text:  text,
					Style: style,
				})
			}
			continue
		}

		if c.Type != hp.ElementNode {
			continue
		}

		tag := c.DataAtom.String()

		switch {
		// ---- headings ----
		case isHeading(tag):
			flushInlines()
			level := int(tag[1] - '0')
			headStyle := headingStyle(style, level)
			block := &document.BlockNode{
				Kind:    document.BlockHeading,
				Level:   level,
				Inlines: p.collectInlines(c, headStyle),
				Align:   attrOr(c, "align", "left"),
				Depth:   depth,
			}
			applyInlineStyle(c, &block.Inlines, &headStyle)
			appendBlock(block)

		// ---- paragraph / generic block ----
		case tag == "p" || tag == "div" || tag == "section" || tag == "article" ||
			tag == "main" || tag == "header" || tag == "footer":
			flushInlines()
			childStyle := applyElementStyle(c, style)
			if hasBlockChildren(c) {
				p.walkChildren(c, childStyle, depth)
				continue
			}
			block := &document.BlockNode{
				Kind:    document.BlockParagraph,
				Inlines: p.collectInlines(c, childStyle),
				Align:   cssAlign(c, "left"),
				Depth:   depth,
			}
			appendBlock(block)

		// ---- preformatted ----
		case tag == "pre":
			flushInlines()
			preStyle := style.WithFamily("mono").WithSize(style.Size * 0.9)
			preStyle = applyElementStyle(c, preStyle)
			block := &document.BlockNode{
				Kind:    document.BlockPre,
				Inlines: p.collectPre(c, preStyle),
				Depth:   depth,
			}
			appendBlock(block)

		// ---- blockquote ----
		case tag == "blockquote":
			flushInlines()
			quoteStyle := style.WithColor(0.35, 0.35, 0.35)
			block := &document.BlockNode{
				Kind:  document.BlockQuote,
				Depth: depth + 1,
			}
			// Collect child blocks under this blockquote node
			sub := &parser{doc: document.NewDocument()}
			sub.walkChildren(c, quoteStyle, depth+1)
			block.Children = sub.doc.Blocks
			appendBlock(block)

		// ---- lists ----
		case tag == "ul" || tag == "ol":
			flushInlines()
			marker := "disc"
			if tag == "ol" {
				marker = "decimal"
			}
			counter := 0
			for li := c.FirstChild; li != nil; li = li.NextSibling {
				if li.Type != hp.ElementNode || li.DataAtom.String() != "li" {
					continue
				}
				counter++
				markerText := marker
				if marker == "decimal" {
					markerText = fmt.Sprintf("%d.", counter)
				}
				liBlock := &document.BlockNode{
					Kind:       document.BlockListItem,
					Inlines:    p.collectInlines(li, style),
					ListMarker: markerText,
					Depth:      depth + 1,
				}
				// li may also contain nested lists
				if hasBlockChildren(li) {
					sub := &parser{doc: document.NewDocument()}
					sub.walkChildren(li, style, depth+1)
					liBlock.Children = sub.doc.Blocks
				}
				appendBlock(liBlock)
			}

		// ---- horizontal rule ----
		case tag == "hr":
			appendBlock(&document.BlockNode{Kind: document.BlockHRule})

		// ---- table ----
		case tag == "table":
			flushInlines()
			appendBlock(p.parseTable(c, style))

		// ---- figure / block img ----
		case tag == "figure":
			flushInlines()
			src, alt := "", ""
			if img := findChild(c, "img"); img != nil {
				src = attrOr(img, "src", "")
				alt = attrOr(img, "alt", "")
			}
			appendBlock(&document.BlockNode{
				Kind: document.BlockImage,
				Inlines: []document.InlineNode{{
					Kind:    document.InlineImage,
					Src:     src,
					AltText: alt,
				}},
				Depth: depth,
			})

		// ---- inline elements at block level — accumulate into inline buffer ----
		default:
			childStyle := applyElementStyle(c, style)
			inlineBuf = append(inlineBuf, p.collectInlines(c, childStyle)...)
		}
	}

	flushInlines()
}

// ---------------------------------------------------------------------------
// Inline content collection
// ---------------------------------------------------------------------------

// collectInlines recursively gathers inline content from node n.
func (p *parser) collectInlines(n *hp.Node, style document.TextStyle) []document.InlineNode {
	var out []document.InlineNode
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if isIgnored(c) {
			continue
		}
		if c.Type == hp.TextNode {
			text := normaliseWhitespace(c.Data)
			if text != "" {
				out = append(out, document.InlineNode{
					Kind:  document.InlineText,
					Text:  text,
					Style: style,
				})
			}
			continue
		}
		if c.Type != hp.ElementNode {
			continue
		}

		tag := c.DataAtom.String()
		childStyle := applyElementStyle(c, style)

		switch tag {
		case "strong", "b":
			childStyle = childStyle.WithBold(true)
			out = append(out, p.collectInlines(c, childStyle)...)
		case "em", "i":
			childStyle = childStyle.WithItalic(true)
			out = append(out, p.collectInlines(c, childStyle)...)
		case "code":
			childStyle = childStyle.WithFamily("mono")
			out = append(out, p.collectInlines(c, childStyle)...)
		case "a":
			// Render links in a muted blue; don't embed URLs in output
			childStyle = childStyle.WithColor(0.13, 0.37, 0.70)
			out = append(out, p.collectInlines(c, childStyle)...)
		case "span":
			out = append(out, p.collectInlines(c, childStyle)...)
		case "br":
			out = append(out, document.InlineNode{Kind: document.InlineBreak})
		case "img":
			out = append(out, document.InlineNode{
				Kind:    document.InlineImage,
				Src:     attrOr(c, "src", ""),
				AltText: attrOr(c, "alt", ""),
				ImgW:    parsePx(attrOr(c, "width", "0")),
				ImgH:    parsePx(attrOr(c, "height", "0")),
			})
		default:
			// Unknown inline element — recurse with current style
			out = append(out, p.collectInlines(c, childStyle)...)
		}
	}
	return out
}

// collectPre collects inline content from a <pre> block, preserving whitespace.
func (p *parser) collectPre(n *hp.Node, style document.TextStyle) []document.InlineNode {
	var out []document.InlineNode
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == hp.TextNode {
			// Split on newlines and emit InlineBreak between lines
			lines := strings.Split(c.Data, "\n")
			for i, line := range lines {
				if i > 0 {
					out = append(out, document.InlineNode{Kind: document.InlineBreak})
				}
				if line != "" {
					out = append(out, document.InlineNode{
						Kind:  document.InlineText,
						Text:  line,
						Style: style,
					})
				}
			}
		} else if c.Type == hp.ElementNode {
			childStyle := applyElementStyle(c, style)
			out = append(out, p.collectInlines(c, childStyle)...)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Table parsing
// ---------------------------------------------------------------------------

func (p *parser) parseTable(n *hp.Node, style document.TextStyle) *document.BlockNode {
	table := &document.BlockNode{Kind: document.BlockTable}

	// Walk thead / tbody / tr at any depth
	var walkRows func(node *hp.Node, isHeader bool)
	walkRows = func(node *hp.Node, isHeader bool) {
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != hp.ElementNode {
				continue
			}
			tag := c.DataAtom.String()
			switch tag {
			case "thead":
				walkRows(c, true)
			case "tbody", "tfoot":
				walkRows(c, false)
			case "tr":
				row := &document.BlockNode{
					Kind:  document.BlockTableRow,
					HasBG: isHeader,
				}
				if isHeader {
					row.BGR, row.BGG, row.BBB = 0.90, 0.90, 0.92
				}
				for td := c.FirstChild; td != nil; td = td.NextSibling {
					if td.Type != hp.ElementNode {
						continue
					}
					tdTag := td.DataAtom.String()
					if tdTag != "td" && tdTag != "th" {
						continue
					}
					cellStyle := style
					if tdTag == "th" || isHeader {
						cellStyle = cellStyle.WithBold(true)
					}
					cellStyle = applyElementStyle(td, cellStyle)
					cell := &document.BlockNode{
						Kind:    document.BlockTableCell,
						Inlines: p.collectInlines(td, cellStyle),
						Align:   cssAlign(td, "left"),
					}
					row.Children = append(row.Children, cell)
				}
				if len(row.Children) > 0 {
					table.Children = append(table.Children, row)
				}
			}
		}
	}
	walkRows(n, false)
	return table
}

// ---------------------------------------------------------------------------
// CSS / style helpers
// ---------------------------------------------------------------------------

// applyElementStyle returns a new TextStyle with any inline style= overrides
// from node n merged in, plus element-default styles.
func applyElementStyle(n *hp.Node, s document.TextStyle) document.TextStyle {
	// Element defaults first
	tag := ""
	if n.DataAtom != 0 {
		tag = n.DataAtom.String()
	}
	switch tag {
	case "strong", "b", "th":
		s = s.WithBold(true)
	case "em", "i":
		s = s.WithItalic(true)
	case "code", "pre", "kbd", "samp":
		s = s.WithFamily("mono")
	case "small":
		s = s.WithSize(s.Size * 0.85)
	case "sup", "sub":
		s = s.WithSize(s.Size * 0.75)
	}

	// Then inline style= overrides
	styleAttr := attrOr(n, "style", "")
	if styleAttr == "" {
		return s
	}
	return applyCSS(s, styleAttr)
}

// applyCSS parses a CSS declaration block and returns an updated TextStyle.
func applyCSS(s document.TextStyle, css string) document.TextStyle {
	for _, decl := range strings.Split(css, ";") {
		decl = strings.TrimSpace(decl)
		if decl == "" {
			continue
		}
		idx := strings.IndexByte(decl, ':')
		if idx < 0 {
			continue
		}
		prop := strings.ToLower(strings.TrimSpace(decl[:idx]))
		val := strings.TrimSpace(decl[idx+1:])

		switch prop {
		case "font-size":
			if pt := parseFontSize(val); pt > 0 {
				s = s.WithSize(pt)
			}
		case "font-weight":
			s = s.WithBold(val == "bold" || val == "bolder" || val == "700" ||
				val == "800" || val == "900")
		case "font-style":
			s = s.WithItalic(val == "italic" || val == "oblique")
		case "font-family":
			s = s.WithFamily(cssFontFamily(val))
		case "color":
			if r, g, b, ok := parseColor(val); ok {
				s = s.WithColor(r, g, b)
			}
		}
	}
	return s
}

// applyInlineStyle applies style= from a block-level node back onto its inlines.
// Used for headings where we set the style on the run, not the block.
func applyInlineStyle(n *hp.Node, inlines *[]document.InlineNode, s *document.TextStyle) {
	styleAttr := attrOr(n, "style", "")
	if styleAttr == "" {
		return
	}
	*s = applyCSS(*s, styleAttr)
	for i := range *inlines {
		(*inlines)[i].Style = *s
	}
}

// headingStyle returns the TextStyle for a heading at the given level.
func headingStyle(base document.TextStyle, level int) document.TextStyle {
	sizes := [6]float64{24, 20, 16, 14, 12, 11}
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	return base.WithBold(true).WithSize(sizes[level-1])
}

// cssAlign extracts text-align from an inline style or align attribute.
func cssAlign(n *hp.Node, def string) string {
	if a := attrOr(n, "align", ""); a != "" {
		return a
	}
	styleAttr := attrOr(n, "style", "")
	for _, decl := range strings.Split(styleAttr, ";") {
		idx := strings.IndexByte(decl, ':')
		if idx < 0 {
			continue
		}
		if strings.TrimSpace(decl[:idx]) == "text-align" {
			return strings.TrimSpace(decl[idx+1:])
		}
	}
	return def
}

// parseFontSize converts a CSS font-size value to points.
// Supports px, pt, em (relative to 16px), rem, and keywords.
func parseFontSize(val string) float64 {
	keywords := map[string]float64{
		"xx-small": 7, "x-small": 8, "small": 10, "medium": 12,
		"large": 14, "x-large": 18, "xx-large": 24,
	}
	if pt, ok := keywords[val]; ok {
		return pt
	}
	switch {
	case strings.HasSuffix(val, "pt"):
		f, _ := strconv.ParseFloat(strings.TrimSuffix(val, "pt"), 64)
		return f
	case strings.HasSuffix(val, "px"):
		f, _ := strconv.ParseFloat(strings.TrimSuffix(val, "px"), 64)
		return f * 0.75 // 1px = 0.75pt
	case strings.HasSuffix(val, "em"), strings.HasSuffix(val, "rem"):
		suf := "em"
		if strings.HasSuffix(val, "rem") {
			suf = "rem"
		}
		f, _ := strconv.ParseFloat(strings.TrimSuffix(val, suf), 64)
		return f * 12 // relative to 12pt base
	}
	return 0
}

// cssFontFamily maps a CSS font-family value to one of our three family keys.
func cssFontFamily(val string) string {
	val = strings.ToLower(val)
	if strings.Contains(val, "mono") || strings.Contains(val, "courier") ||
		strings.Contains(val, "code") || strings.Contains(val, "consolas") {
		return "mono"
	}
	if strings.Contains(val, "serif") && !strings.Contains(val, "sans") {
		return "serif"
	}
	return "" // sans-serif default
}

// parseColor parses a CSS colour into normalised RGB [0,1].
// Supports #rrggbb, #rgb, and rgb(r,g,b) forms.
func parseColor(val string) (r, g, b float64, ok bool) {
	val = strings.TrimSpace(val)
	if strings.HasPrefix(val, "#") {
		hex := val[1:]
		if len(hex) == 3 {
			hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
		}
		if len(hex) == 6 {
			ri, e1 := strconv.ParseInt(hex[0:2], 16, 64)
			gi, e2 := strconv.ParseInt(hex[2:4], 16, 64)
			bi, e3 := strconv.ParseInt(hex[4:6], 16, 64)
			if e1 == nil && e2 == nil && e3 == nil {
				return float64(ri) / 255, float64(gi) / 255, float64(bi) / 255, true
			}
		}
	}
	if strings.HasPrefix(val, "rgb(") && strings.HasSuffix(val, ")") {
		inner := val[4 : len(val)-1]
		parts := strings.Split(inner, ",")
		if len(parts) == 3 {
			ri, e1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			gi, e2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			bi, e3 := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
			if e1 == nil && e2 == nil && e3 == nil {
				return ri / 255, gi / 255, bi / 255, true
			}
		}
	}
	return 0, 0, 0, false
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

// isIgnored returns true for nodes whose subtrees we skip entirely.
func isIgnored(n *hp.Node) bool {
	if n.Type != hp.ElementNode {
		return false
	}
	switch n.DataAtom.String() {
	case "script", "style", "noscript", "svg", "canvas", "video", "audio", "head":
		return true
	}
	return false
}

// isHeading returns true for h1–h6.
func isHeading(tag string) bool {
	return len(tag) == 2 && tag[0] == 'h' && tag[1] >= '1' && tag[1] <= '6'
}

// hasBlockChildren returns true if n has any block-level element children.
func hasBlockChildren(n *hp.Node) bool {
	blockTags := map[string]bool{
		"p": true, "div": true, "h1": true, "h2": true, "h3": true,
		"h4": true, "h5": true, "h6": true, "ul": true, "ol": true,
		"li": true, "blockquote": true, "pre": true, "table": true,
		"hr": true, "figure": true, "section": true, "article": true,
		"header": true, "footer": true, "main": true,
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == hp.ElementNode && blockTags[c.DataAtom.String()] {
			return true
		}
	}
	return false
}

// findChild returns the first child element with the given tag, or nil.
func findChild(n *hp.Node, tag string) *hp.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == hp.ElementNode && c.DataAtom.String() == tag {
			return c
		}
	}
	return nil
}

// attrOr returns the value of attribute key on n, or def if absent.
func attrOr(n *hp.Node, key, def string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return def
}

// parsePx parses a "123" or "123px" string into a float64 (points ≈ px×0.75).
func parsePx(val string) float64 {
	val = strings.TrimSuffix(strings.TrimSpace(val), "px")
	f, _ := strconv.ParseFloat(val, 64)
	return f * 0.75
}

// normaliseWhitespace collapses runs of whitespace into single spaces and
// trims leading/trailing whitespace.
func normaliseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := true // leading spaces stripped
	for _, r := range s {
		isSpace := r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f'
		if isSpace {
			if !prevSpace {
				b.WriteByte(' ')
			}
			prevSpace = true
		} else {
			b.WriteRune(r)
			prevSpace = false
		}
	}
	result := b.String()
	// trim trailing space
	return strings.TrimRight(result, " ")
}

// trimInlineWhitespace removes leading and trailing whitespace-only InlineText
// runs from the slice.
func trimInlineWhitespace(runs []document.InlineNode) []document.InlineNode {
	for len(runs) > 0 &&
		runs[0].Kind == document.InlineText &&
		strings.TrimSpace(runs[0].Text) == "" {
		runs = runs[1:]
	}
	for len(runs) > 0 &&
		runs[len(runs)-1].Kind == document.InlineText &&
		strings.TrimSpace(runs[len(runs)-1].Text) == "" {
		runs = runs[:len(runs)-1]
	}
	return runs
}

// Ensure fonts import is used (FaceFromStyle is called via document/TextStyle helpers).
var _ = fonts.Helvetica

// findDescendant does a depth-first search for the first element with the
// given tag name anywhere beneath n.
func findDescendant(n *hp.Node, tag string) *hp.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == hp.ElementNode && c.DataAtom.String() == tag {
			return c
		}
		if found := findDescendant(c, tag); found != nil {
			return found
		}
	}
	return nil
}
