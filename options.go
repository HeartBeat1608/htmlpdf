package htmlpdf

import "errors"

// ErrNoBrowser is returned by the Chrome backend when no supported browser
// binary can be located on the system PATH or at Options.ChromePath.
var ErrNoBrowser = errors.New("htmlpdf: no supported browser found (install chromium or google-chrome)")

// Backend selects which rendering engine to use.
type Backend int

const (
	// BackendAuto tries Chrome first, falls back to Native when Chrome is not
	// available or cannot render in the current environment.
	BackendAuto Backend = iota
	// BackendChrome requires headless Chrome/Chromium. Returns ErrNoBrowser if absent.
	BackendChrome
	// BackendNative uses the pure-Go renderer. No external process required.
	BackendNative
)

// PageSize is a named paper size.
type PageSize int

const (
	PageA4     PageSize = iota // 595.28 × 841.89 pt
	PageLetter                 // 612 × 792 pt
)

// Orientation controls portrait vs landscape layout.
type Orientation int

const (
	Portrait  Orientation = iota
	Landscape             // swaps width and height
)

// Options controls PDF generation behaviour.
type Options struct {
	// Backend selects the rendering engine. Default: BackendAuto.
	Backend Backend

	// ChromePath overrides the browser binary used by BackendChrome.
	// When empty, the engine searches the system PATH.
	ChromePath string

	// ChromeDisableSandbox adds Chrome's --no-sandbox flag.
	//
	// Leave this false unless you are running in a restricted container or CI
	// environment where the browser sandbox cannot start normally.
	ChromeDisableSandbox bool

	// PageSize sets the paper size. Default: PageA4.
	PageSize PageSize

	// Orientation sets portrait or landscape. Default: Portrait.
	Orientation Orientation

	// MarginTopPt, MarginRightPt, MarginBottomPt, MarginLeftPt set page
	// margins in points (1 pt = 1/72 inch). Zero values use the default
	// 72 pt (1 inch) margins.
	MarginTopPt    float64
	MarginRightPt  float64
	MarginBottomPt float64
	MarginLeftPt   float64

	// Title sets the PDF document title metadata. When empty, the parser
	// uses the HTML <title> element if present.
	Title string
}

// defaults fills zero-value fields with sensible defaults.
func (o Options) defaults() Options {
	if o.MarginTopPt == 0 {
		o.MarginTopPt = 72
	}
	if o.MarginRightPt == 0 {
		o.MarginRightPt = 72
	}
	if o.MarginBottomPt == 0 {
		o.MarginBottomPt = 72
	}
	if o.MarginLeftPt == 0 {
		o.MarginLeftPt = 72
	}
	return o
}

// pageDimensions returns (width, height) in points for the given options,
// applying orientation.
func (o Options) pageDimensions() (w, h float64) {
	switch o.PageSize {
	case PageLetter:
		w, h = 612.0, 792.0
	default: // PageA4
		w, h = 595.28, 841.89
	}
	if o.Orientation == Landscape {
		w, h = h, w
	}
	return
}
