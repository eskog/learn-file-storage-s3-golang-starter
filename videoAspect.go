package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os/exec"
)

func commonDivider(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func aspectRatio(width, height int) (int, int) {
	if width == 0 || height == 0 {
		return 0, 0
	}
	g := commonDivider(width, height)
	return width / g, height / g
}

func getVideoAspectRatio(filePath string) (string, error) {
	type outputStruct struct {
		Streams []struct {
			CodecType         string `json:"codec_type"`
			Width             int    `json:"width"`
			Height            int    `json:"height"`
			SampleAspectRatio string `json:"sample_aspect_ratio"`
		} `json:"streams"`
	}

	command := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var output bytes.Buffer
	command.Stdout = &output

	err := command.Run()
	if err != nil {
		return "", err
	}

	var filteredOutput outputStruct
	err = json.NewDecoder(&output).Decode(&filteredOutput)
	if err != nil {
		log.Printf("error decoding json: %s", err)
		return "", err
	}

	var width, height int
	var sarWidth, sarHeight int = 1, 1 // Default SAR to 1:1

	for _, stream := range filteredOutput.Streams {
		if stream.CodecType == "video" {
			width = stream.Width
			height = stream.Height

			// Parse SAR
			if stream.SampleAspectRatio != "" {
				fmt.Sscanf(stream.SampleAspectRatio, "%d:%d", &sarWidth, &sarHeight)
			}
			break
		}
	}

	if width == 0 || height == 0 {
		return "", errors.New("could not determine video resolution")
	}

	// Apply SAR correction but force rounding
	adjustedWidth := width
	if sarWidth != sarHeight {
		adjustedWidth = int(math.Round(float64(width) * float64(sarWidth) / float64(sarHeight)))
	}

	// Compute reduced aspect ratio
	w, h := aspectRatio(adjustedWidth, height)
	log.Printf("width: %d, height: %d, SAR: %d:%d, adjusted width: %d, aspect ratio: %d:%d",
		width, height, sarWidth, sarHeight, adjustedWidth, w, h)

	finalRatio := fmt.Sprintf("%d:%d", w, h)
	if finalRatio != "16:9" {
		finalRatio = "9:16"
	}
	return finalRatio, nil
}
