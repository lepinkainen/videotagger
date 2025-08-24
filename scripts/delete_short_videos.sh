#!/bin/bash

# This script finds video files with '[0min]' in their name and deletes them
# if their actual duration is less than a specified number of seconds.

# Usage: ./delete_short_videos.sh <max_seconds>

set -euo pipefail

if [ $# -ne 1 ]; then
  echo "Usage: $0 <max_seconds>"
  exit 1
fi

readonly MAX_SECONDS=$1

# Validate MAX_SECONDS is a positive number
if ! [[ "$MAX_SECONDS" =~ ^[0-9]+([.][0-9]+)?$ ]]; then
  echo "Error: max_seconds must be a positive number"
  exit 1
fi

# Use fd to find video files, then filter for [0min] duration tag in proper position
fd --type f \
   --extension mp4 --extension webm --extension mov --extension flv \
   --extension mkv --extension avi --extension wmv --extension mpg \
   --print0 | grep -zE '\]\[0min\]\[' | while IFS= read -r -d '' file; do

   echo "🔍 Checking '$file'..."
  
  # Get duration, suppress ffprobe output completely
  if ! duration=$(ffprobe -v quiet -show_entries format=duration -of csv=p=0 "$file" 2>/dev/null); then
    echo "⚠️  Skipping '$file' (unable to read duration)"
    continue
  fi

  # Skip files with no duration or invalid duration
  if [[ -z "$duration" || "$duration" == "N/A" ]]; then
    echo "⚠️  Skipping '$file' (no duration found)"
    continue
  fi

  # Use awk for floating point comparison (no external bc dependency)
  if awk "BEGIN { exit !($duration < $MAX_SECONDS) }"; then
    # Get file size in megabytes
    file_size_mb=$(stat -f %z "$file" | awk '{printf "%.1f", $1/1024/1024}')
    printf "🗑️  Deleting '%s' (%.1fs, %sMB)\n" "$file" "$duration" "$file_size_mb"
    rm "$file"
  fi
done