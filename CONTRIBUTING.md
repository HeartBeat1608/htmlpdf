# Contributing

Thanks for helping improve `htmlpdf`.

## Getting Started

1. Install a supported Go toolchain.
2. Clone the repository.
3. Run:

```bash
go test ./...
go vet ./...
```

## Scope

Contributions are especially welcome in these areas:

- native renderer correctness and layout edge cases
- Chrome backend reliability across platforms
- documentation, examples, and fixture coverage
- performance profiling and benchmark coverage

## Pull Requests

- Keep changes focused and explain the user-visible impact.
- Add or update tests when behavior changes.
- Document new options or supported features in `README.md`.
- Prefer preserving backwards compatibility in exported APIs.

## Reporting Bugs

Please include:

- a minimal HTML input sample
- the backend used
- the expected behavior
- the actual result
- your Go version and OS

## Code Style

- Run `gofmt` on changed files.
- Keep package boundaries clear: only `htmlpdf` is public API; implementation details live under `internal/`.
