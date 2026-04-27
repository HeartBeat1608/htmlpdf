package htmlpdf_test

import (
	"context"
	"fmt"
	"time"

	"github.com/HeartBeat1608/htmlpdf"
)

func ExampleGenerate() {
	html := []byte(`<html><body><h1>Hello</h1><p>Example PDF</p></body></html>`)

	pdf, err := htmlpdf.Generate(html, htmlpdf.Options{
		Backend: htmlpdf.BackendNative,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(len(pdf) > 0)
	// Output: true
}

func ExampleGenerateContext() {
	html := []byte(`<html><body><p>Rendered with context</p></body></html>`)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pdf, err := htmlpdf.GenerateContext(ctx, html, htmlpdf.Options{
		Backend: htmlpdf.BackendNative,
		Title:   "Example",
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(len(pdf) > 0)
	// Output: true
}
