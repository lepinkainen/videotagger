# VideoTagger

A comprehensive command-line video management tool that provides tagging, duplicate detection, verification, and similarity analysis for video files.

## Features

VideoTagger offers four powerful commands for video file management:

### 🏷️ **tag** - Smart Video Tagging

Automatically renames video files with embedded metadata including resolution, duration, and CRC32 checksum.

**Format**: `original_filename_[resolution][duration_in_min][CRC32_checksum].extension`

**Example**: `vacation_video.mp4 → vacation_video_[1920x1080][45min][A1B2C3D4].mp4`

### 🔍 **duplicates** - Duplicate Detection

Finds duplicate video files using CRC32 checksums, helping you identify and manage duplicate content efficiently.

### ✅ **verify** - Integrity Verification

Verifies the integrity of previously tagged video files by recalculating and comparing checksums.

### 👁️ **phash** - Visual Similarity Detection  

Finds visually similar videos using perceptual hashing, perfect for identifying near-duplicates, different encodings of the same content, or related clips.

## Requirements

- [Go](https://golang.org/dl/) 1.24 or later
- [FFmpeg](https://ffmpeg.org/download.html) (specifically, the `ffprobe` binary must be available in your PATH)

## Installation

### From Source

1. Clone the repository:

   ```bash
   git clone https://github.com/yourusername/videotagger.git
   cd videotagger
   ```

2. Build using Task (install [Task](https://taskfile.dev/#/installation) if you don't have it):

   ```bash
   task build
   ```

3. Alternatively, build using Go directly:

   ```bash
   go build -o videotagger
   ```

4. Install to your system:

   ```bash
   task publish  # Installs to $HOME/bin
   ```

## Usage

VideoTagger uses a subcommand structure. Each command supports parallel processing with configurable worker counts.

### Tag Videos

Process single or multiple video files:

```bash
# Single file
videotagger tag path/to/video.mp4

# Multiple files
videotagger tag video1.mp4 video2.avi video3.mkv

# With custom worker count
videotagger tag --workers 8 *.mp4

# Process all videos in directory
videotagger tag /path/to/videos/*
```

### Find Duplicates

Detect duplicate videos by comparing checksums:

```bash
# Check for duplicates in current directory
videotagger duplicates *.mp4

# Check specific files
videotagger duplicates video1.mp4 video2.mp4 video3.mp4

# With parallel processing
videotagger duplicates --workers 4 /path/to/videos/*
```

### Verify Integrity

Verify previously tagged video files:

```bash
# Verify tagged files
videotagger verify *_[*].mp4

# Verify specific files
videotagger verify tagged_video_[1920x1080][45min][A1B2C3D4].mp4
```

### Find Similar Videos

Detect visually similar videos using perceptual hashing:

```bash
# Find similar videos in directory
videotagger phash *.mp4

# Compare specific files
videotagger phash video1.mp4 video2.mp4 video3.mp4

# With custom threshold and workers
videotagger phash --workers 6 /path/to/videos/*
```

## Supported Formats

VideoTagger supports the following video formats (case-insensitive):

- .mp4, .webm, .mov, .flv, .mkv, .avi, .wmv, .mpg

## Advanced Features

- **Parallel Processing**: Configurable worker pools for optimal performance
- **Progress Tracking**: Real-time progress bars for long operations
- **Skip Detection**: Automatically skips already processed files
- **Robust Error Handling**: Continues processing remaining files after errors
- **Non-destructive**: Only filenames are changed; video content remains untouched
- **Memory Efficient**: Streams file processing to handle large video collections

## Performance Tips

- Use `--workers` flag to adjust parallelism based on your system
- For large collections, process files in batches
- SSD storage significantly improves CRC32 calculation speed
- FFprobe performance depends on video codec and file size

## Troubleshooting

### Common Issues

**FFprobe not found**:

- Ensure FFmpeg is installed and `ffprobe` is in your PATH
- On macOS: `brew install ffmpeg`
- On Ubuntu/Debian: `apt-get install ffmpeg`

**Permission errors**:

- Ensure write permissions in the target directory
- Check file ownership and permissions

**Memory issues with large files**:

- Reduce worker count with `--workers` flag
- Process files in smaller batches

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests: `task test`
4. Run linter: `task lint`
5. Commit your changes (`git commit -m 'Add some amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## License

[Insert your license information here]
