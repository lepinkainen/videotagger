# VideoTagger Development Guidelines

## Commands
- **Build**: `task build` - Compiles the application
- **Lint**: `task lint` - Runs golangci-lint
- **Vet**: `task vet` - Runs Go's built-in code analyzer
- **Clean**: `task clean` - Removes build artifacts 
- **Install**: `task publish` - Installs binary to $HOME/bin
- **Update Dependencies**: `task upgrade-deps`

## Code Style
- **Imports**: Standard Go grouping (stdlib, external, internal)
- **Formatting**: Follow Go standard formatting with `gofmt`
- **Naming**: 
  - Functions/Variables: camelCase
  - Exported constants: PascalCase
- **Error Handling**:
  - Use `fmt.Errorf` with error wrapping (`%w`)
  - Provide detailed error messages
  - Use ❌ for errors, ✅ for success in output
  - Continue processing remaining files after errors

## Project Structure
- Simple CLI tool to rename video files with metadata
- Requires FFmpeg (specifically ffprobe)
- Process files with: `./videotagger [file1] [file2]...` or `tag_videos.sh`