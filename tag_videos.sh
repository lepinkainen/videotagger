#!/bin/bash

# Use a for loop to iterate over all files in the current directory
for file in *
do
    videotagger "$file"
done
