# Changelog

## Unreleased

### Added

- `doc.go` and `errors.go` with centralized sentinels (`ErrNotConnected`, resume errors, `ErrStopIteration`)
- `ConfigDefault()` for port, timeout, and buffer defaults (credentials still required via `Validate`)
- English godoc on all exported types and options
- `goleak` in `TestMain`
- `examples/resume` demonstrating path-based resume

### Changed

- `ErrNotConnected` replaces ad-hoc `"ftp: not connected"` strings
- Tests reorganized: one primary `*_test.go` per source file; removed `edge_test.go` and `coverage_test.go`
- README rewritten in English (Chinese quick reference retained at end)

## v0.1.0

### Added

- `Client` composes `Uploader`, `Downloader`, and `Lister`
- Streaming upload/download with `context` cancellation
- Resume via `WithResume` / `WithResumeOffset` (REST/SIZE)
- `ListEach`, `WalkEach`, and `ListIter` for large directories
- Examples under `examples/basic`
