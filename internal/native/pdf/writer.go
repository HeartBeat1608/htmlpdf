package pdf

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"strings"
)

// Standard PDF page sizes in points (1 point = 1/72 inch)
const (
	A4Width      = 595.28
	A4Height     = 841.89
	LetterWidth  = 612.0
	LetterHeight = 792.0
)

// object represents a single PDF indrect object.
type object struct {
	id      int
	content []byte // complete dictionary + stream, ready to write
}

// Writer constructs a PDF 1.4 document
// Usage:
//
// w := NewWriter()
// pg := w.AddPage(A4Width, A4Height)
// pg.SetFont("F1", 12)
// pg.DrawText(72, 72, "Hello, PDF")
// pdf, err := w.Write()
type Writer struct {
	pages   []*Page
	objects []*object // pool of indirect objects (1-indexed)
	nextID  int
}

// NewWriter returns an empty PDF writer.
func NewWriter() *Writer {
	return &Writer{nextID: 1}
}

// AddPage adds a new page of the given dimensions and returns it for drawing
func (w *Writer) AddPage(width, height float64) *Page {
	p := newPage(width, height)
	w.pages = append(w.pages, p)
	return p
}

// alloc reserves the next object ID and appends a placeholder.
func (w *Writer) alloc() int {
	id := w.nextID
	w.nextID++
	w.objects = append(w.objects, &object{id: id})
	return id
}

// set stores content for an already-allocated object ID.
func (w *Writer) set(id int, content []byte) {
	w.objects[id-1].content = content
}

// Write serialises the entier PDF and returns the bytes
func (w *Writer) Write() ([]byte, error) {
	var buf bytes.Buffer

	// ---- Pre-allocate all object IDs so we know them before writing -----

	// Font objects: we include the 4 standard faces we reference.
	// PDF viewers supply the glyphs for the 14 standard Type1 fonts.
	fontNames := []string{
		"Helvetica",
		"Helvetica-Bold",
		"Helvetica-Oblique",
		"Times-Roman",
		"Times-Bold",
		"Courier",
		"Courier-Bold",
	}

	fontKeyMap := map[string]string{
		"Helvetica":         "F1",
		"Helvetica-Bold":    "F2",
		"Helvetica-Oblique": "F3",
		"Times-Roman":       "F4",
		"Times-Bold":        "F5",
		"Courier":           "F6",
		"Courier-Bold":      "F7",
	}
	fontIDs := map[string]int{}
	for _, fn := range fontNames {
		id := w.alloc()
		fontIDs[fn] = id
	}

	// Page content stream objects
	contentIDs := make([]int, len(w.pages))
	for i := range w.pages {
		contentIDs[i] = w.alloc()
	}

	// Page objects
	pageIDs := make([]int, len(w.pages))
	for i := range w.pages {
		pageIDs[i] = w.alloc()
	}

	// Pages (page tree root)
	pagesID := w.alloc()

	// Catalog
	catalogID := w.alloc()

	// --- Build font objects ---
	for _, fn := range fontNames {
		id := fontIDs[fn]
		d := fmt.Sprintf("<< /Type /Font /Subtype /Type1 /BaseFont /%s /Encoding /WinAnsiEncoding >>", fn)
		w.set(id, []byte(d))
	}

	// --- Build font resource string (shared across all pages) ---
	var fontResB strings.Builder
	fontResB.WriteString("<< ")
	for _, fn := range fontNames {
		fmt.Fprintf(&fontResB, "/%s %d 0 R", fontKeyMap[fn], fontIDs[fn])
	}
	fontResB.WriteString(" >>")
	fontRes := fontResB.String()

	// --- Build page content streams and page objects ---
	for i, pg := range w.pages {
		stream := pg.bytes()

		// Content stream object
		cs := fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream)

		w.set(contentIDs[i], []byte(cs))

		// Page object
		pd := fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 %.2f %.2f] /Contents %d 0 R /Resources << /Font %s >> >>",
			pagesID,
			pg.width,
			pg.height,
			contentIDs[i],
			fontRes,
		)
		w.set(pageIDs[i], []byte(pd))
	}

	// --- Page tree ---
	var kidsBuf strings.Builder
	kidsBuf.WriteString("[")
	for i, id := range pageIDs {
		if i > 0 {
			kidsBuf.WriteString(" ")
		}
		fmt.Fprintf(&kidsBuf, "%d 0 R", id)
	}
	kidsBuf.WriteString("]")

	w.set(pagesID, []byte(fmt.Sprintf("<< /Type /Pages /Kids %s /Count %d >>",
		kidsBuf.String(), len(w.pages),
	)))

	// --- Catalog ---
	w.set(catalogID, []byte(fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", pagesID)))

	// -------------------
	// Serialize
	// -------------------
	writeStr := func(s string) { buf.WriteString(s) }

	writeStr("%PDF-1.4\n")
	writeStr("%\xe2\xe3\xcf\xd3\n") // binary comment - signals binary content to tools

	// Cross-reference table offsets
	xrefOffsets := make([]int64, len(w.objects)+1) // 1-indexed; [0] unused

	for _, obj := range w.objects {
		xrefOffsets[obj.id] = int64(buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n", obj.id)
		buf.Write(obj.content)
		buf.WriteString("\nendobj\n")
	}

	xrefOsset := int64(buf.Len())
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(w.objects)+1)

	// free entry for object 0
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(w.objects); i++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", xrefOffsets[i])
	}

	// Trailer
	fmt.Fprintf(&buf,
		"trailer\n<< /Size %d /Root %d 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		len(w.objects)+1, catalogID, xrefOsset,
	)

	return buf.Bytes(), nil
}

// EncodeImageAsXObject encodes an image.Image as a PDF image XObject dictionary+stream.
// Returns the raw object content ready to be embedded via w.set().
// For JPEG source bytes, use EncodeJPEGAsXObject instead to avoid re-encoding.
func EncodeImageAsXObject(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("jpeg encode: %w", err)
	}
	return buildImageStream(buf.Bytes(), img.Bounds().Dx(), img.Bounds().Dy())
}

// EncodeJPEGAsXObject wraps already-encoded JPEG bytes as a PDF image XObject.
func EncodeJPEGAsXObject(jpegBytes []byte, width, height int) ([]byte, error) {
	return buildImageStream(jpegBytes, width, height)
}

func buildImageStream(data []byte, width, height int) ([]byte, error) {
	var b bytes.Buffer
	header := fmt.Sprintf(
		"<< /Type /XObject /Subtype /Image /Width %d /Height %d "+
			"/ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>\nstream\n",
		width, height, len(data),
	)
	b.WriteString(header)
	b.Write(data)
	b.WriteString("\nendstream")
	return b.Bytes(), nil
}

// ReadImage decodes an image from r (JPEG or PNG).
func ReadImage(r io.Reader) (image.Image, string, error) {
	return image.Decode(r)
}
