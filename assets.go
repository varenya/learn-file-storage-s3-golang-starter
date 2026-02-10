package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
)

func (cfg apiConfig) ensureAssetsDir() error {
	if _, err := os.Stat(cfg.assetsRoot); os.IsNotExist(err) {
		return os.Mkdir(cfg.assetsRoot, 0o755)
	}
	return nil
}

func randomFileName() string {
	randomBytes := make([]byte, 32)
	rand.Read(randomBytes)
	randomFileName := base64.RawURLEncoding.EncodeToString(randomBytes)
	return randomFileName
}

type VideoInfo struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func formatAspectRatio(width, height int) string {
	return fmt.Sprintf("%d:%d", width, height)
}

func getVideoSize(videoInfo VideoInfo) (int, int) {
	width := videoInfo.Streams[0].Width
	height := videoInfo.Streams[0].Height
	gcdAR := gcd(width, height)
	width = width / gcdAR
	height = height / gcdAR
	return width, height
}

func matchRatio(calc, target float64) bool {
	return math.Abs(calc-target) <= 0.5
}

func getVideoAspectRatio(filePath string) (string, error) {
	argString := fmt.Sprintf("-v error -print_format json -show_streams %s", filePath)
	cmdArgs := strings.Split(argString, " ")
	cmd := exec.Command("ffprobe", cmdArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	var videoInfo VideoInfo
	err = json.Unmarshal(out.Bytes(), &videoInfo)
	if err != nil {
		return "", err
	}
	width, height := getVideoSize(videoInfo)
	if matchRatio(float64(width)/float64(height), 16.0/9.0) {
		return formatAspectRatio(16, 9), nil
	}
	if matchRatio(float64(width)/float64(height), 9.0/16.0) {
		return formatAspectRatio(9, 16), nil
	}
	return "other", nil
}

func processVideoForFastStart(filePath string) (string, error) {
	outputFile, err := os.CreateTemp("", "tubely-video")
	outputFileName := outputFile.Name() + ".mp4"
	if err != nil {
		return "", err
	}
	argString := fmt.Sprintf("-i %s -c copy -movflags faststart -f mp4 %s", filePath, outputFileName)
	cmdArgs := strings.Split(argString, " ")
	cmd := exec.Command("ffmpeg", cmdArgs...)
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	return outputFileName, nil
}
