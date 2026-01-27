package adapter

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// FFmpegVideoProcessor implements port.VideoProcessor
type FFmpegVideoProcessor struct {
	ffmpegPath string
}

// NewFFmpegVideoProcessor creates a new video processor
func NewFFmpegVideoProcessor() *FFmpegVideoProcessor {
	ffmpegPath, _ := exec.LookPath("ffmpeg")
	if ffmpegPath == "" {
		commonPaths := []string{
			"/usr/local/bin/ffmpeg",
			"/opt/homebrew/bin/ffmpeg",
		}
		for _, p := range commonPaths {
			if _, err := os.Stat(p); err == nil {
				ffmpegPath = p
				break
			}
		}
	}
	return &FFmpegVideoProcessor{ffmpegPath: ffmpegPath}
}

// IsAvailable checks if ffmpeg is available
func (v *FFmpegVideoProcessor) IsAvailable() bool {
	return v.ffmpegPath != ""
}

// ExtractFrames extracts frames from a video file
func (v *FFmpegVideoProcessor) ExtractFrames(videoPath, outputDir string, interval float64, maxFrames int) ([]string, error) {
	if !v.IsAvailable() {
		return nil, fmt.Errorf("ffmpeg not found")
	}

	if interval <= 0 {
		interval = 1.0
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create frames directory: %w", err)
	}

	videoName := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	framePattern := filepath.Join(outputDir, fmt.Sprintf("%s_frame_%%04d.png", videoName))

	args := []string{
		"-i", videoPath,
		"-vf", fmt.Sprintf("fps=1/%s,scale=640:-1", formatFloat(interval)),
	}

	if maxFrames > 0 {
		args = append(args, "-vframes", strconv.Itoa(maxFrames))
	}

	args = append(args, "-y", framePattern)

	cmd := exec.Command(v.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg error: %v\nOutput: %s", err, string(output))
	}

	frames, err := filepath.Glob(filepath.Join(outputDir, fmt.Sprintf("%s_frame_*.png", videoName)))
	if err != nil {
		return nil, fmt.Errorf("failed to find extracted frames: %w", err)
	}

	return frames, nil
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}
