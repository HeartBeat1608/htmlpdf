# Basic Example

This example shows how to generate a PDF from HTML and write it to a file.

## Run

```bash
go run ./examples/basic
```

That writes `example.pdf` in the current directory.

## Options

```bash
go run ./examples/basic -backend=native -out=invoice.pdf -title="My First PDF"
```

Flags:

- `-backend`: `auto`, `chrome`, or `native`
- `-out`: output PDF path
- `-title`: PDF document title
