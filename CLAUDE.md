# VideoTagger Development Guidelines

## Architecture Overview

VideoTagger is a modular Go CLI application built with Kong framework, organized across multiple packages for maintainability. It provides four main video management operations:

- **tag**: Renames files with `[resolution][duration][CRC32]` metadata
- **duplicates**: Finds files sharing CRC32 hashes (requires processed files)  
- **verify**: Validates CRC32 checksums against embedded hashes
- **phash**: Compares video frames using perceptual hashing via FFmpeg

### Package Structure

```plain
main.go (22 lines)          # CLI definition and entry point
cmd/                        # Command implementations (250 lines)
├── tag.go                  # Parallel processing with worker pools
├── duplicates.go           # Hash-based duplicate detection
├── verify.go               # Checksum verification
└── phash.go               # Perceptual hash similarity

ui/                         # TUI components (267 lines)
├── model.go               # Bubble Tea TUI state management
├── styles.go              # Lipgloss styling definitions
└── messages.go            # Worker communication types

utils/                      # Utilities (46 lines)
└── network.go             # Network drive detection

video/ (334 lines)          # Core video processing
├── metadata.go            # FFprobe integration
├── hash.go                # CRC32 + perceptual hashing
├── validation.go          # File type validation
└── processing.go          # Main video processing logic
```

## Key Technical Patterns

### Kong CLI with Package References

```go
type CLI struct {
    Tag        *cmd.TagCmd        `cmd:"" help:"Tag video files..."`
    Duplicates *cmd.DuplicatesCmd `cmd:"" help:"Find duplicate files..."`
    Verify     *cmd.VerifyCmd     `cmd:"" help:"Verify file hash..."`
    Phash      *cmd.PhashCmd      `cmd:"" help:"Find perceptually..."`
}
```

### Filename Processing

- **Tagged format**: `filename_[1920x1080][45min][A1B2C3D4].ext`
- **Regex pattern**: `_\[(\d+x\d+)\]\[(\d+)min\]\[([a-fA-F0-9]{8})\]\.[^\.]*$`
- **Skip logic**: `wasProcessedRegex.MatchString()` detects already tagged files
- **Hash extraction**: `extractHashFromFilename()` parses existing metadata

### Worker Pool Architecture

```go
// Network drive detection influences worker count
if utils.IsNetworkDrive(file) {
    workers = 1 // Single worker for network drives
} else {
    workers = runtime.NumCPU() // Parallel for local drives
}

// Worker pool pattern used in cmd/tag.go
jobs := make(chan string, len(cmd.Files))
var wg sync.WaitGroup
for i := 0; i < workers; i++ {
    wg.Add(1)
    go func(workerID int) {
        defer wg.Done()
        for videoFile := range jobs {
            video.ProcessVideoFile(videoFile)
        }
    }(i)
}
```

### External Dependencies

- **FFmpeg**: `ffprobe -v error -select_streams v:0 -show_entries stream=width,height` for metadata
- **FFmpeg frame extraction**: `ffmpeg -ss 30 -vframes 1 -q:v 2` for perceptual hashing
- **CRC32**: Standard library `hash/crc32.ChecksumIEEE()` for file integrity  
- **Libraries**: Kong (CLI), goimagehash (perceptual hashing), bubbletea/lipgloss (TUI)

## Development Workflow

### Essential Commands (Build Chain)

- **Build**: `task build` - Runs `test` → `lint` → compile (REQUIRED before completion)
- **Format**: `goimports -w .` - CRITICAL: NEVER use `gofmt`, always use `goimports`
- **Test**: `task test` - Runs `go test -v ./...` with comprehensive test suite
- **Lint**: `task lint` - Uses `golangci-lint run ./...` (no custom config)
- **Install**: `task publish` - Copies `build/videotagger` to `$HOME/bin`

### File Organization

- **Modular packages**: Code split across `cmd/`, `ui/`, `utils/`, `video/` packages  
- **Comprehensive tests**: 96 test cases covering core functionality
- **Build output**: `build/videotagger` executable with embedded version
- **Dependencies**: Go 1.24+, FFmpeg in PATH, TUI libraries

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

- **CLI Output**: Use ❌/✅ emojis for user feedback (in `ui.SuccessStyle`, `ui.ErrorStyle`)
- **Progress Bars**: Future TUI integration with bubbletea progress models
- **Worker Count**: Auto-detects network drives, defaults to `runtime.NumCPU()` for local files
- **Video Extensions**: Case-insensitive matching via `strings.ToLower()` in `video.IsVideoFile()`
- **Formatting**: CRITICAL - Use `goimports -w .` (never `gofmt`)

## Modern Shell Tools for Development

Use these faster, more intuitive tools for codebase exploration:

### Code Search with `rg` (ripgrep)
```bash
# Find function definitions across packages
rg "^func " -t go

# Search for specific video processing patterns
rg "video\.(Process|Calculate)" -t go

# Find TODO/FIXME comments across codebase
rg "TODO|FIXME|HACK"

# Search for Kong command patterns
rg "cmd:\"\"" -t go

# Find FFmpeg integration points
rg "ffprobe|ffmpeg" -A 2 -B 1
```

### File Discovery with `fd`
```bash
# Find all Go source files (respects .gitignore)
fd "\.go$"

# Find test files specifically
fd "_test\.go$"

# Find video test fixtures
fd "\.(mp4|avi|mkv)$" test_files/

# Find configuration files
fd "config|\.yml$|\.yaml$"

# Search in specific packages
fd "\.go$" cmd/
fd "\.go$" video/
```

### VideoTagger-Specific Patterns
```bash
# Find video processing functions
rg "ProcessVideo|CalculateCRC32|GetVideoResolution" -t go

# Find CLI command implementations
rg "func.*Run\(\)" -t go cmd/

# Search for network drive handling
rg "IsNetworkDrive|network" -t go

# Find TUI-related code
rg "bubbletea|lipgloss|TUIModel" -t go ui/
```

## Key Functions by Package

### video/ package (core processing)

- `video.IsVideoFile()` - Extension validation with comprehensive format support
- `video.ExtractHashFromFilename()` - Regex parsing of tagged filenames  
- `video.CalculateCRC32()` - File integrity with progress tracking
- `video.GetVideoResolution()` - FFprobe metadata extraction
- `video.ProcessVideoFile()` - Main processing pipeline

### utils/ package  

- `utils.IsNetworkDrive()` - Cross-platform network drive detection (UNC, NFS, SMB)

### ui/ package (future TUI)

- `ui.NewTUIModel()` - Bubble Tea model initialization
- `ui.SuccessStyle`, `ui.ErrorStyle` - Consistent styling across commands

## Integration Requirements

- **FFmpeg**: `ffprobe` and `ffmpeg` must be in PATH
- **Go 1.24+**: Required for language features
- **Supported formats**: `.mp4, .webm, .mov, .flv, .mkv, .avi, .wmv, .mpg` (case-insensitive)

## Testing Strategy

Comprehensive test suite with 96 test cases covering:

- **Unit tests**: All video processing functions, CLI parsing, network detection
- **Integration tests**: Real FFmpeg calls with test fixtures in `test_files/`
- **Error scenarios**: Missing files, corrupted videos, invalid formats, permission issues
- **Regression tests**: Filename parsing edge cases, worker pool behavior, TUI components
- **Test command**: `task test` runs before build (enforced by build chain)

## Development Integration

- **llm-shared**: Follow `llm-shared/languages/go.md` for Go best practices
- **Project rules**: See `.cursor/rules/project-rules.mdc` for constraints
- **Build chain**: `task build` enforces test → lint → compile workflow
