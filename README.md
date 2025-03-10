# Video Tagger

A command-line tool that automatically renames video files with embedded metadata including resolution, duration, and a checksum.

## Description

Video Tagger scans and processes video files, embedding important metadata directly into the filename. This makes it easy to identify video characteristics without needing to open the file or use specialized software.

Each processed file is renamed with the following format:

```
original_filename_[resolution][duration_in_min][CRC32_checksum].extension
```

For example:

```
vacation_video.mp4 → vacation_video_[1920x1080][45min][A1B2C3D4].mp4
```

## Requirements

- [Go](https://golang.org/dl/) 1.16 or later
- [FFmpeg](https://ffmpeg.org/download.html) (specifically, the `ffprobe` binary must be available in your PATH)

## Installation

### From Source

1. Clone the repository:

   ```
   git clone https://github.com/yourusername/videotagger.git
   cd videotagger
   ```

2. Build using Task (install [Task](https://taskfile.dev/#/installation) if you don't have it):

   ```
   task build
   ```

3. Alternatively, build using Go directly:
   ```
   go build -o videotagger
   ```

## Usage

Process a single video file:

```
videotagger path/to/video.mp4
```

Process multiple video files:

```
videotagger video1.mp4 video2.avi video3.mkv
```

## Supported Formats

Video Tagger supports the following video formats:

- .mp4
- .webm
- .mov
- .flv
- .mkv
- .avi
- .wmv
- .mpg

## Features

- **Non-destructive** - Only the filename is changed; the video content remains untouched
- **Skip Detection** - Already processed files are automatically skipped
- **Progress Bar** - Visual feedback during CRC32 calculation for large files
- **Error Handling** - Clear error messages for troubleshooting

## License

[Insert your license information here]

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
