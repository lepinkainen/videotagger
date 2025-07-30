# VideoTagger Development Guidelines

## Architecture Overview
VideoTagger is a single-file Go CLI application (`main.go`) built with Kong framework for video file management. It uses a command-based architecture with four main operations:

- **tag**: Renames files with `[resolution][duration][CRC32]` metadata
- **duplicates**: Finds files sharing CRC32 hashes (requires processed files)
- **verify**: Validates CRC32 checksums against embedded hashes
- **phash**: Compares video frames using perceptual hashing via FFmpeg

## Key Technical Patterns

### Filename Processing
- **Tagged format**: `filename_[1920x1080][45min][A1B2C3D4].ext`
- **Regex**: `wasProcessedRegex` extracts resolution, duration, and hash from processed files
- **Skip logic**: Already tagged files are automatically skipped

### Worker Pool Architecture
```go
// Single-file processing for 1 file or 1 worker
if len(cmd.Files) == 1 || workers == 1 {
    // Sequential processing
} else {
    // Parallel worker pool with channels
    jobs := make(chan string, len(cmd.Files))
    var wg sync.WaitGroup
}
```

### External Dependencies
- **FFmpeg**: `ffprobe` extracts video metadata (resolution, duration)
- **FFmpeg frame extraction**: `ffmpeg -ss 30 -vframes 1` for perceptual hashing
- **CRC32**: Standard library `hash/crc32` for file integrity

## Development Workflow

### Essential Commands
- **Build**: `task build` - Runs tests, lint, and compiles (REQUIRED before completion)
- **Format**: `goimports -w .` - NEVER use `gofmt`, always use `goimports`
- **Lint**: `task lint` - Uses golangci-lint (currently no .golangci.yml config)
- **Install**: `task publish` - Copies binary to `$HOME/bin`

### File Organization
- **Single file**: All code in `main.go` (493 lines)
- **No tests**: Project currently lacks test coverage
- **Build output**: `build/videotagger` executable

## Error Handling Strategy
```go
// Continue processing on errors, don't fail entire batch
if err != nil {
    fmt.Printf("❌ Error processing %s: %v\n", videoFile, err)
    return // Skip this file, continue with others
}
```

## Code Conventions
- **CLI Output**: Use ❌/✅ emojis for user feedback
- **Progress Bars**: `progressbar.DefaultBytes()` for file operations
- **Worker Count**: Defaults to `runtime.NumCPU()` if not specified
- **Video Extensions**: Case-insensitive matching via `strings.ToLower()`

## Integration Requirements
- **FFmpeg**: Must be in PATH for video processing
- **Go 1.24+**: Required for language features
- **External Libraries**: Kong (CLI), goimagehash (perceptual hashing), progressbar

## Testing Strategy
Currently no tests exist. When adding tests:
- Focus on `isVideoFile()`, regex matching, and core processing functions
- Mock FFmpeg calls for unit tests
- Test error handling for missing files/corrupt videos