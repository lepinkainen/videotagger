#!/bin/bash

# Create test videos script for videotagger testing
echo "📹 Creating 5 test videos in /tmp directory..."

TEST_DIR="/tmp/videotagger-test-$(date +%s)"
mkdir -p "$TEST_DIR"
echo "📁 Created directory: $TEST_DIR"

# Check if FFmpeg is available
if ! command -v ffmpeg >/dev/null 2>&1; then
    echo "❌ FFmpeg not found. Please install FFmpeg to create real test videos."
    exit 1
fi

echo "🎬 Generating test videos with FFmpeg..."

# Video 1: 720p, 3 seconds
ffmpeg -f lavfi -i testsrc=duration=3:size=1280x720:rate=30 -c:v libx264 -t 3 "$TEST_DIR/video1_720p.mp4" -y -loglevel quiet
echo "✅ Created video1_720p.mp4 (1280x720, 3s)"

# Video 2: 1080p, 2 seconds  
ffmpeg -f lavfi -i testsrc=duration=2:size=1920x1080:rate=30 -c:v libx264 -t 2 "$TEST_DIR/video2_1080p.mp4" -y -loglevel quiet
echo "✅ Created video2_1080p.mp4 (1920x1080, 2s)"

# Video 3: 480p, 4 seconds
ffmpeg -f lavfi -i testsrc=duration=4:size=854x480:rate=30 -c:v libx264 -t 4 "$TEST_DIR/video3_480p.mp4" -y -loglevel quiet
echo "✅ Created video3_480p.mp4 (854x480, 4s)"

# Video 4: 4K, 1 second (small duration to keep file size reasonable)
ffmpeg -f lavfi -i testsrc=duration=1:size=3840x2160:rate=30 -c:v libx264 -t 1 "$TEST_DIR/video4_4k.mp4" -y -loglevel quiet
echo "✅ Created video4_4k.mp4 (3840x2160, 1s)"

# Video 5: Duplicate of video1 for duplicate testing
cp "$TEST_DIR/video1_720p.mp4" "$TEST_DIR/video5_duplicate.mp4"
echo "✅ Created video5_duplicate.mp4 (duplicate of video1)"

# Create some non-video files to test filtering
echo "test content" > "$TEST_DIR/readme.txt"
echo "more content" > "$TEST_DIR/document.pdf"
echo "✅ Created non-video files for filtering tests"

echo ""
echo "🎯 Test videos created in: $TEST_DIR"
echo "📋 Files:"
ls -la "$TEST_DIR"

echo ""
echo "🚀 Test commands:"
echo "   ./build/videotagger tag $TEST_DIR/*.mp4"
echo "   ./build/videotagger duplicates $TEST_DIR"
echo "   ./build/videotagger verify $TEST_DIR/*_[*].mp4"
echo "   ./build/videotagger phash $TEST_DIR"

echo ""
echo "🗑️  Cleanup when done: rm -rf $TEST_DIR"