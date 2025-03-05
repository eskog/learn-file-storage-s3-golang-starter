package main

import (
	"os/exec"
)

func processVideoForFastStart(filePath string) (string, error) {
	outFilePath := filePath + ".processing"
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outFilePath)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return outFilePath, nil
}
