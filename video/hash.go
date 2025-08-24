package video

import (
	"fmt"
	"hash/crc32"
	"image"
	_ "image/jpeg"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/corona10/goimagehash"
)

// CalculateCRC32 calculates the CRC32 checksum of a file
func CalculateCRC32(filename string) (uint32, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	h := crc32.NewIEEE()
	if _, err := io.Copy(h, f); err != nil {
		return 0, err
	}

	return h.Sum32(), nil
}

// CalculateVideoPerceptualHash extracts a frame from video and calculates perceptual hash
func CalculateVideoPerceptualHash(videoFile string) (*goimagehash.ImageHash, error) {
	// Create temporary file for extracted frame
	tempFrame := filepath.Join(os.TempDir(), fmt.Sprintf("frame_%d.jpg", os.Getpid()))
	defer func() { _ = os.Remove(tempFrame) }()

	// Extract frame at 30% through the video
	cmd := exec.Command("ffmpeg", "-i", videoFile, "-ss", "00:00:30", "-vframes", "1", "-f", "image2", "-y", tempFrame)
	err := cmd.Run()
	if err != nil {
		// Try extracting at 10 seconds if percentage fails
		cmd = exec.Command("ffmpeg", "-i", videoFile, "-ss", "10", "-vframes", "1", "-f", "image2", "-y", tempFrame)
		if err = cmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to extract frame: %w", err)
		}
	}

	// Calculate perceptual hash of extracted frame
	file, err := os.Open(tempFrame)
	if err != nil {
		return nil, fmt.Errorf("failed to open extracted frame: %w", err)
	}
	defer func() { _ = file.Close() }()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	hash, err := goimagehash.PerceptionHash(img)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate perceptual hash: %w", err)
	}

	return hash, nil
}
