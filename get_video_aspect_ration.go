package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

type FFProbeOutput struct {
	Streams []Dimensions `json:"streams"`
}

type Dimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_streams", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", err
	}

	var result FFProbeOutput
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return "", err
	}

	if len(result.Streams) == 0 {
		return "", fmt.Errorf("no streams found in ffprobe output")
	}

	return aspectRatio(result.Streams[0].Width, result.Streams[0].Height), nil
}

func aspectRatio(width, height int) string {
	ratio := float64(width) / float64(height)

	r16by9 := 16.0 / 9.0
	r9by16 := 9.0 / 16.0

	const tol = 0.01

	switch {
	case math.Abs(ratio-r16by9) < tol:
		return "16:9"
	case math.Abs(ratio-r9by16) < tol:
		return "9:16"
	default:
		return "other"
	}
}
