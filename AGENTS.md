# Repository Guidelines

## Project Structure and Module Organization
- `main.go` wires the CLI (kong) and dispatches commands.
- `cmd/` holds command implementations (`tag`, `duplicates`, `verify`, `phash`, `reencode`).
- `video/` contains core video discovery, hashing, metadata, and processing logic.
- `utils/` and `types/` provide shared helpers and data types.
- `ui/` contains terminal UI models used by command output.
- `scripts/` and `tag_videos.sh` are helper utilities; `test_files/` is sample input.
- `build/` is generated output; `llm-shared/` is excluded from lint/test tasks.

## Build, Test, and Development Commands
- `task build` builds the binary after running tests and lint.
- `task build-only` builds without tests/lint (CI use).
- `task build-linux` cross-builds for Linux (GOOS/GOARCH set).
- `task test` runs Go tests (excluding `llm-shared`).
- `task test-ci` runs tests with coverage to `coverage.out` and `-tags=ci`.
- `task lint` runs `goimports`, `go vet`, and `golangci-lint`.
- `go build -o build/videotagger main.go` is the direct Go build path.

## Coding Style and Naming Conventions
- Use standard Go formatting (`gofmt`), with imports normalized by `goimports`.
- Follow Go naming: exported identifiers in `PascalCase`, unexported in `camelCase`.
- Keep files and tests in the same package; test files use `*_test.go`.
- Avoid touching `llm-shared/` unless intentionally updating shared utilities.

## Testing Guidelines
- Framework: Go's `testing` package (no external test runner).
- Place tests alongside packages (examples in `video/`, `utils/`, `ui/`, `main_test.go`).
- Prefer targeted tests for metadata parsing, hashing, and validation behavior.
- Run `task test` locally; CI expects `task test-ci` coverage output.

## Commit and Pull Request Guidelines
- Commit messages mostly follow Conventional Commits (`feat:`, `fix:`, `chore:`).
- Keep subjects short and action-oriented (e.g., `fix: handle empty suffix`).
- PRs should include a clear summary, test results, and any relevant sample outputs.
- Link related issues and note CLI behavior changes or new flags.

## Configuration Notes
- Go version is `1.25` per `go.mod`; ensure it is installed.
- `ffprobe` from FFmpeg must be available in `PATH` for runtime commands.
- Task uses `.env` if present; keep local overrides out of version control.
