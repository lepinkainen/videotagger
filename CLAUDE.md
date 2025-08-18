# VideoTagger Development Guidelines

## Architecture Overview

VideoTagger is a single-file Go CLI application (`main.go`) built with Kong framework for video file management. It uses a command-based architecture with four main operations:

- **tag**: Renames files with `[resolution][duration][CRC32]` metadata
- **duplicates**: Finds files sharing CRC32 hashes (requires processed files)  
- **verify**: Validates CRC32 checksums against embedded hashes
- **phash**: Compares video frames using perceptual hashing via FFmpeg

## Key Technical Patterns

### Kong CLI Framework Structure

```go
type CLI struct {
    Tag        TagCmd        `cmd:"" help:"Tag video files with metadata and hash"`
    Duplicates DuplicatesCmd `cmd:"" help:"Find duplicate files by hash"`
    Verify     VerifyCmd     `cmd:"" help:"Verify file hash integrity"`
    Phash      PhashCmd      `cmd:"" help:"Find perceptually similar videos"`
}
```

### Filename Processing

- **Tagged format**: `filename_[1920x1080][45min][A1B2C3D4].ext`
- **Regex pattern**: `_\[(\d+x\d+)\]\[(\d+)min\]\[([a-fA-F0-9]{8})\]\.[^\.]*$`
- **Skip logic**: `wasProcessedRegex.MatchString()` detects already tagged files
- **Hash extraction**: `extractHashFromFilename()` parses existing metadata

### Worker Pool Architecture

```go
// Single-file processing for 1 file or 1 worker
if len(cmd.Files) == 1 || workers == 1 {
    // Sequential processing with progress bar
} else {
    // Parallel worker pool with channels
    jobs := make(chan string, len(cmd.Files))
    var wg sync.WaitGroup
    // Default: runtime.NumCPU() workers
}
```

### External Dependencies

- **FFmpeg**: `ffprobe -v quiet -print_format json -show_format` for metadata
- **FFmpeg frame extraction**: `ffmpeg -ss 30 -vframes 1 -q:v 2` for perceptual hashing
- **CRC32**: Standard library `hash/crc32.ChecksumIEEE()` for file integrity
- **Libraries**: Kong (CLI), goimagehash (perceptual hashing), progressbar (UX)

## Development Workflow

### Essential Commands (Build Chain)

- **Build**: `task build` - Runs `test` → `lint` → compile (REQUIRED before completion)
- **Format**: `goimports -w .` - CRITICAL: NEVER use `gofmt`, always use `goimports`
- **Test**: `task test` - Runs `go test -v ./...` (currently no tests exist)
- **Lint**: `task lint` - Uses `golangci-lint run ./...` (no custom config)
- **Install**: `task publish` - Copies `build/videotagger` to `$HOME/bin`

### File Organization

- **Single file**: All code in `main.go` (~500 lines)
- **No tests**: Project currently lacks test coverage (priority for new features)
- **Build output**: `build/videotagger` executable
- **Dependencies**: Go 1.24+, FFmpeg in PATH

## Error Handling Strategy

```go
// Continue processing on errors, don't fail entire batch
if err != nil {
    fmt.Printf("❌ Error processing %s: %v\n", videoFile, err)
    return // Skip this file, continue with others
}
```

- **Non-destructive failures**: Individual file errors don't stop batch processing
- **User feedback**: Immediate error reporting with emoji indicators
- **Graceful degradation**: Missing FFmpeg or corrupted files are handled cleanly

## Code Conventions

- **CLI Output**: Use ❌/✅ emojis for user feedback
- **Progress Bars**: `progressbar.DefaultBytes()` for CRC32 calculations
- **Worker Count**: Defaults to `runtime.NumCPU()` if `--workers` not specified
- **Video Extensions**: Case-insensitive matching via `strings.ToLower()`
- **Formatting**: CRITICAL - Use `goimports -w .` (never `gofmt`)

## Function Structure (main.go:440 lines)

Key functions for testing/modification:

- `main.go:156` - `isVideoFile()` - Extension validation
- `main.go:55` - `extractHashFromFilename()` - Regex parsing
- `main.go:103` - `calculateCRC32()` - File integrity
- `main.go:171` - `getVideoResolution()` - FFprobe wrapper
- `main.go:256` - `processVideoFile()` - Core tagging logic

## Integration Requirements

- **FFmpeg**: `ffprobe` and `ffmpeg` must be in PATH
- **Go 1.24+**: Required for language features
- **Supported formats**: `.mp4, .webm, .mov, .flv, .mkv, .avi, .wmv, .mpg` (case-insensitive)

## Testing Strategy (Priority: Basic Coverage)

Currently no tests exist. When adding tests:

- **Unit tests**: `isVideoFile()`, `extractHashFromFilename()`, regex validation
- **Integration tests**: Mock FFmpeg calls with test fixtures in `test_files/`
- **Error scenarios**: Missing files, corrupted videos, invalid formats
- **Regression tests**: Filename parsing edge cases, worker pool behavior
- **Test command**: `task test` runs before build (dependency chain)

## Development Integration

- **llm-shared**: Follow `llm-shared/languages/go.md` for Go best practices
- **Project rules**: See `.cursor/rules/project-rules.mdc` for constraints
- **Build chain**: `task build` enforces test → lint → compile workflow
