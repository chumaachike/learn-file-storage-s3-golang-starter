package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
)

func processVideoForFastStart(inputPath string) (string, error) {
	// Create output path (same dir, with .processing suffix)
	ext := filepath.Ext(inputPath)
	outputPath := inputPath[:len(inputPath)-len(ext)] + ".processing" + ext

	// ffmpeg command
	cmd := exec.Command(
		"ffmpeg",
		"-i", inputPath,
		"-c", "copy",
		"-movflags", "faststart",
		"-f", "mp4",
		outputPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return outputPath, nil
}
