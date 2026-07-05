package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Result struct {
	Duration int32
	CoverURL string
	PlayURLs map[string]string
}

type Transcoder struct {
	mediaRoot string
	publicURL string
	timeout   time.Duration
}

func New(mediaRoot, publicURL string, timeout time.Duration) *Transcoder {
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	return &Transcoder{mediaRoot: mediaRoot, publicURL: strings.TrimRight(publicURL, "/"), timeout: timeout}
}

func (t *Transcoder) Transcode(ctx context.Context, videoID, sourcePath string) (*Result, error) {
	clean := filepath.ToSlash(filepath.Clean(sourcePath))
	if strings.Contains(clean, "..") {
		return nil, fmt.Errorf("invalid source path")
	}
	absSource := sourcePath
	if !filepath.IsAbs(absSource) {
		absSource = filepath.Join(t.mediaRoot, filepath.FromSlash(clean))
	}
	mediaRootClean := filepath.Clean(t.mediaRoot)
	if !strings.HasPrefix(filepath.Clean(absSource), mediaRootClean+string(filepath.Separator)) && filepath.Clean(absSource) != mediaRootClean {
		return nil, fmt.Errorf("source path outside media root")
	}
	if _, err := os.Stat(absSource); err != nil {
		return nil, fmt.Errorf("source not found: %w", err)
	}

	outDir := filepath.Join(t.mediaRoot, "transcoded", videoID)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}

	duration, err := t.probeDuration(ctx, absSource)
	if err != nil {
		return nil, err
	}

	coverPath := filepath.Join(outDir, "cover.jpg")
	if err := t.extractCover(ctx, absSource, coverPath); err != nil {
		return nil, err
	}

	profiles := []struct {
		quality string
		height  int
	}{
		{"720p", 720},
		{"1080p", 1080},
	}
	playURLs := make(map[string]string)
	for _, p := range profiles {
		profileDir := filepath.Join(outDir, p.quality)
		if err := os.MkdirAll(profileDir, 0o755); err != nil {
			return nil, err
		}
		m3u8 := filepath.Join(profileDir, "index.m3u8")
		if err := t.runHLS(ctx, absSource, p.height, m3u8); err != nil {
			return nil, err
		}
		playURLs[p.quality] = fmt.Sprintf("%s/transcoded/%s/%s/index.m3u8", t.publicURL, videoID, p.quality)
	}

	coverURL := fmt.Sprintf("%s/transcoded/%s/cover.jpg", t.publicURL, videoID)
	return &Result{
		Duration: duration,
		CoverURL: coverURL,
		PlayURLs: playURLs,
	}, nil
}

func (t *Transcoder) probeDuration(ctx context.Context, source string) (int32, error) {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		source,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w", err)
	}
	sec, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, err
	}
	return int32(sec + 0.5), nil
}

func (t *Transcoder) extractCover(ctx context.Context, source, coverPath string) error {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y",
		"-ss", "00:00:01",
		"-i", source,
		"-frames:v", "1",
		coverPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg cover: %w: %s", err, string(out))
	}
	return nil
}

func (t *Transcoder) runHLS(ctx context.Context, source string, height int, m3u8 string) error {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y",
		"-i", source,
		"-vf", fmt.Sprintf("scale=-2:%d", height),
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-c:a", "aac",
		"-hls_time", "4",
		"-hls_list_size", "0",
		"-f", "hls",
		m3u8,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg hls %dp: %w: %s", height, err, string(out))
	}
	return nil
}
