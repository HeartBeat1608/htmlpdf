package fonts

import (
	"math"
	"testing"
)

// ---- GlyphWidth ---------------------------------------------------------

func TestGlyphWidthSpaceHelvetica(t *testing.T) {
	// Space (U+0020) should be 278 units for Helvetica per AFM data
	got := GlyphWidth(Helvetica, ' ')
	if got != 278 {
		t.Errorf("Helvetica space width = %d, want 278", got)
	}
}

func TestGlyphWidthCapA(t *testing.T) {
	// 'A' (U+0041) is at index 33 in the table
	// Helvetica A = 667
	got := GlyphWidth(Helvetica, 'A')
	if got != 667 {
		t.Errorf("Helvetica 'A' width = %d, want 667", got)
	}
}

func TestGlyphWidthCourierIsMonospace(t *testing.T) {
	// Every glyph in Courier must be 600 units wide
	testRunes := "AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz0123456789!@#"
	for _, r := range testRunes {
		if w := GlyphWidth(Courier, r); w != 600 {
			t.Errorf("Courier %q width = %d, want 600", r, w)
		}
		if w := GlyphWidth(CourierBold, r); w != 600 {
			t.Errorf("Courier-Bold %q width = %d, want 600", r, w)
		}
	}
}

func TestGlyphWidthOutOfRange(t *testing.T) {
	// Codepoints below 32 and above 255 should return space width
	spaceW := GlyphWidth(Helvetica, ' ')

	if w := GlyphWidth(Helvetica, 0); w != spaceW {
		t.Errorf("codepoint 0 width = %d, want space width %d", w, spaceW)
	}
	if w := GlyphWidth(Helvetica, '\t'); w != spaceW {
		t.Errorf("tab width = %d, want space width %d", w, spaceW)
	}
	if w := GlyphWidth(Helvetica, '→'); w != spaceW { // U+2192, outside WinAnsi
		t.Errorf("arrow width = %d, want space width %d", w, spaceW)
	}
}

func TestGlyphWidthBoldWiderThanRegular(t *testing.T) {
	// Bold faces are generally wider; spot check a few uppercase letters
	boldWider := 0
	for _, r := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		reg := GlyphWidth(Helvetica, r)
		bold := GlyphWidth(HelveticaBold, r)
		if bold >= reg {
			boldWider++
		}
	}
	// At least 20 of 26 uppercase should be wider or equal in bold
	if boldWider < 20 {
		t.Errorf("only %d/26 uppercase letters wider or equal in Helvetica-Bold", boldWider)
	}
}

func TestGlyphWidthObliqueMatchesRegular(t *testing.T) {
	// Helvetica-Oblique has the same widths as Helvetica
	for _, r := range "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz" {
		if GlyphWidth(Helvetica, r) != GlyphWidth(HelveticaOblique, r) {
			t.Errorf("Helvetica vs Oblique width mismatch for %q", r)
		}
	}
}

// ---- MeasureString ------------------------------------------------------

func TestMeasureStringEmpty(t *testing.T) {
	if w := MeasureString(Helvetica, 12, ""); w != 0 {
		t.Errorf("empty string width = %.4f, want 0", w)
	}
}

func TestMeasureStringSingleSpace(t *testing.T) {
	// 278 units * (12/1000) = 3.336 points
	want := float64(278) * 12.0 / 1000.0
	got := MeasureString(Helvetica, 12, " ")
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("MeasureString(Helvetica, 12, \" \") = %.6f, want %.6f", got, want)
	}
}

func TestMeasureStringScalesWithSize(t *testing.T) {
	w12 := MeasureString(Helvetica, 12, "Hello")
	w24 := MeasureString(Helvetica, 24, "Hello")
	if math.Abs(w24-w12*2) > 1e-9 {
		t.Errorf("width at 24pt (%.4f) should be exactly 2× width at 12pt (%.4f)", w24, w12)
	}
}

func TestMeasureStringAdditivity(t *testing.T) {
	// MeasureString("AB") == MeasureString("A") + MeasureString("B")
	ab := MeasureString(Helvetica, 11, "AB")
	a := MeasureString(Helvetica, 11, "A")
	b := MeasureString(Helvetica, 11, "B")
	if math.Abs(ab-(a+b)) > 1e-9 {
		t.Errorf("MeasureString not additive: AB=%.4f, A+B=%.4f", ab, a+b)
	}
}

func TestMeasureStringCourierFixedWidth(t *testing.T) {
	// "Hello" = 5 chars × 600 units × (10/1000) = 30 pt
	want := 5.0 * 600.0 * 10.0 / 1000.0
	got := MeasureString(Courier, 10, "Hello")
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Courier MeasureString = %.4f, want %.4f", got, want)
	}
}

// ---- MeasureRune --------------------------------------------------------

func TestMeasureRuneConsistentWithString(t *testing.T) {
	for _, r := range "ABCabc123!@#" {
		byRune := MeasureRune(Helvetica, 12, r)
		byStr := MeasureString(Helvetica, 12, string(r))
		if math.Abs(byRune-byStr) > 1e-9 {
			t.Errorf("MeasureRune vs MeasureString mismatch for %q: %.6f vs %.6f", r, byRune, byStr)
		}
	}
}

// ---- LineHeight ---------------------------------------------------------

func TestLineHeight(t *testing.T) {
	cases := []struct {
		size float64
		want float64
	}{
		{10, 12},
		{12, 14.4},
		{16, 19.2},
	}
	for _, c := range cases {
		got := LineHeight(c.size)
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("LineHeight(%.0f) = %.4f, want %.4f", c.size, got, c.want)
		}
	}
}

// ---- SpaceWidth ---------------------------------------------------------

func TestSpaceWidthMatchesGlyphWidth(t *testing.T) {
	for _, f := range []Face{Helvetica, HelveticaBold, TimesRoman, Courier} {
		sw := SpaceWidth(f, 12)
		gw := MeasureRune(f, 12, ' ')
		if math.Abs(sw-gw) > 1e-9 {
			t.Errorf("SpaceWidth(face %d, 12) = %.4f, MeasureRune = %.4f", f, sw, gw)
		}
	}
}

// ---- FaceFromStyle ------------------------------------------------------

func TestFaceFromStyleSansDefault(t *testing.T) {
	if got := FaceFromStyle("", false, false); got != Helvetica {
		t.Errorf("default sans = %d, want Helvetica", got)
	}
}

func TestFaceFromStyleSansBold(t *testing.T) {
	if got := FaceFromStyle("", true, false); got != HelveticaBold {
		t.Errorf("sans bold = %d, want HelveticaBold", got)
	}
}

func TestFaceFromStyleSansItalic(t *testing.T) {
	if got := FaceFromStyle("", false, true); got != HelveticaOblique {
		t.Errorf("sans italic = %d, want HelveticaOblique", got)
	}
}

func TestFaceFromStyleSerifRegular(t *testing.T) {
	if got := FaceFromStyle("serif", false, false); got != TimesRoman {
		t.Errorf("serif regular = %d, want TimesRoman", got)
	}
}

func TestFaceFromStyleSerifBold(t *testing.T) {
	if got := FaceFromStyle("serif", true, false); got != TimesBold {
		t.Errorf("serif bold = %d, want TimesBold", got)
	}
}

func TestFaceFromStyleMono(t *testing.T) {
	if got := FaceFromStyle("mono", false, false); got != Courier {
		t.Errorf("mono regular = %d, want Courier", got)
	}
	if got := FaceFromStyle("mono", true, false); got != CourierBold {
		t.Errorf("mono bold = %d, want CourierBold", got)
	}
}

// ---- PDFName / ResourceKey ----------------------------------------------

func TestPDFNames(t *testing.T) {
	cases := []struct {
		face Face
		name string
		key  string
	}{
		{Helvetica, "Helvetica", "F1"},
		{HelveticaBold, "Helvetica-Bold", "F2"},
		{HelveticaOblique, "Helvetica-Oblique", "F3"},
		{TimesRoman, "Times-Roman", "F4"},
		{TimesBold, "Times-Bold", "F5"},
		{Courier, "Courier", "F6"},
		{CourierBold, "Courier-Bold", "F7"},
	}
	for _, c := range cases {
		if got := c.face.PDFName(); got != c.name {
			t.Errorf("Face(%d).PDFName() = %q, want %q", c.face, got, c.name)
		}
		if got := c.face.ResourceKey(); got != c.key {
			t.Errorf("Face(%d).ResourceKey() = %q, want %q", c.face, got, c.key)
		}
	}
}

// ---- Table integrity check ---------------------------------------------

func TestAllTablesHave224Entries(t *testing.T) {
	for i, table := range widthTables {
		if len(table) != 224 {
			t.Errorf("widthTables[%d] has %d entries, want 224", i, len(table))
		}
	}
}

func TestAllWidthsPositive(t *testing.T) {
	for i, table := range widthTables {
		for j, w := range table {
			if w <= 0 {
				t.Errorf("widthTables[%d][%d] = %d, must be positive", i, j, w)
			}
		}
	}
}

func TestKnownWidthSpotChecks(t *testing.T) {
	// Spot-check specific AFM values from the PDF specification
	checks := []struct {
		face Face
		r    rune
		want int
	}{
		{Helvetica, 'i', 222},
		{Helvetica, 'W', 944},
		{Helvetica, 'm', 833},
		{HelveticaBold, 'i', 278},
		{HelveticaBold, 'W', 944},
		{TimesRoman, 'A', 722},
		{TimesRoman, 'i', 278},
		{TimesBold, 'A', 722},
		{TimesBold, 'm', 833},
		{Courier, 'A', 600},
		{Courier, 'i', 600},
	}
	for _, c := range checks {
		got := GlyphWidth(c.face, c.r)
		if got != c.want {
			t.Errorf("GlyphWidth(Face%d, %q) = %d, want %d", c.face, c.r, got, c.want)
		}
	}
}
