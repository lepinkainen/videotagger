# Project Overview

This project is a command-line tool named **VideoTagger**, written in Go. Its primary purpose is to manage video files by providing functionalities for tagging, finding duplicates, verifying integrity, and identifying visual similarity.

The tool is built using the `kong` library for command-line argument parsing and `bubbletea` for creating interactive terminal user interfaces. It relies on `ffmpeg` (specifically `ffprobe`) for video metadata extraction and processing.

The project is structured into several packages, including:

- `cmd`: Contains the implementation for each command (`tag`, `duplicates`, `verify`, `phash`).
- `video`: Handles video processing tasks like metadata extraction, hashing, and perceptual hashing.
- `ui`: Manages the terminal user interface components.
- `utils`: Provides utility functions, such as dependency validation.

## Building and Running

The project uses `task` as a task runner to simplify common development operations.

### Key Commands

- **Build the project:**

    ```bash
    task build
    ```

    This command runs tests, lints the code, and then builds the executable, placing it in the `build/` directory.

- **Run tests:**

    ```bash
    task test
    ```

- **Lint the code:**

    ```bash
    task lint
    ```

    This uses `golangci-lint` with the configuration defined in `.golangci.yml`.

- **Run the application:**
    After building, the application can be run from the `build` directory:

    ```bash
    ./build/videotagger --help
    ```

    The available commands are:
  - `tag`: Tags video files with metadata.
  - `duplicates`: Finds duplicate video files.
  - `verify`: Verifies the integrity of tagged files.
  - `phash`: Finds visually similar videos.

## Development Conventions

- **Testing:** The project has a suite of tests that can be run with `task test`. New code should be accompanied by corresponding tests.
- **Linting:** The project uses `golangci-lint` to enforce code style and quality. Before committing, run `task lint` to ensure your code adheres to the project's standards.
- **Dependencies:** Dependencies are managed with Go modules (`go.mod` and `go.sum`).
- **Continuous Integration:** The project has a CI pipeline configured in `.github/workflows/go-ci.yml` that runs tests and builds the project on every push.
