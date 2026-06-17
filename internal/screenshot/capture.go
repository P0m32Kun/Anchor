package screenshot

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/P0m32Kun/Anchor/internal/util"
)

// CaptureResult holds the result of a screenshot capture.
type CaptureResult struct {
	Path   string `json:"path"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Size   int64  `json:"size"`
}

// Capture takes a screenshot of the given URL using chromedp (Chrome DevTools Protocol).
// It launches a headless Chrome instance, navigates to the URL, waits for the page to
// render, captures a full-page screenshot as PNG, and saves it to outputDir.
func Capture(ctx context.Context, url, outputDir string, timeout time.Duration) (*CaptureResult, error) {
	if url == "" {
		return nil, fmt.Errorf("empty url")
	}
	if outputDir == "" {
		outputDir = os.TempDir()
	}
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// Build allocator options with Docker-friendly defaults.
	opts := chromedp.DefaultExecAllocatorOptions[:]
	opts = append(opts,
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.WindowSize(1920, 1080),
	)

	// Honor custom browser binary from environment.
	if bin := os.Getenv("ANCHOR_CHROMIUM_BIN"); bin != "" {
		opts = append(opts, chromedp.ExecPath(bin))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	browserCtx, cancel := context.WithTimeout(browserCtx, timeout)
	defer cancel()

	var buf []byte
	if err := chromedp.Run(browserCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.FullScreenshot(&buf, 100),
	); err != nil {
		return nil, fmt.Errorf("chromedp capture %s: %w", url, err)
	}

	filename := fmt.Sprintf("screenshot_%s_%d.png", util.GenerateID(), time.Now().UnixNano())
	outputPath := filepath.Join(outputDir, filename)
	if err := os.WriteFile(outputPath, buf, 0640); err != nil {
		return nil, fmt.Errorf("write screenshot file: %w", err)
	}

	width, height, err := imageDimensions(outputPath)
	if err != nil {
		width, height = 0, 0
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("stat screenshot: %w", err)
	}

	return &CaptureResult{
		Path:   outputPath,
		Width:  width,
		Height: height,
		Size:   info.Size(),
	}, nil
}

// CreateThumbnail creates a thumbnail of the given image at the specified
// max dimensions. It scales down proportionally and writes a JPEG.
func CreateThumbnail(srcPath, dstPath string, maxWidth, maxHeight int) error {
	if maxWidth <= 0 {
		maxWidth = 320
	}
	if maxHeight <= 0 {
		maxHeight = 240
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source image: %w", err)
	}
	defer src.Close()

	img, _, err := image.Decode(src)
	if err != nil {
		return fmt.Errorf("decode image: %w", err)
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// Calculate scale factor
	scaleW := float64(maxWidth) / float64(w)
	scaleH := float64(maxHeight) / float64(h)
	scale := scaleW
	if scaleH < scaleW {
		scale = scaleH
	}
	if scale >= 1.0 {
		// No scaling needed, just copy
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("read source: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(dstPath), 0750); err != nil {
			return fmt.Errorf("mkdir thumbnail dir: %w", err)
		}
		return os.WriteFile(dstPath, data, 0640)
	}

	newW := int(float64(w) * scale)
	newH := int(float64(h) * scale)

	thumb := image.NewRGBA(image.Rect(0, 0, newW, newH))
	for y := 0; y < newH; y++ {
		for x := 0; x < newW; x++ {
			srcX := int(float64(x) / scale)
			srcY := int(float64(y) / scale)
			thumb.Set(x, y, img.At(srcX, srcY))
		}
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0750); err != nil {
		return fmt.Errorf("mkdir thumbnail dir: %w", err)
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create thumbnail file: %w", err)
	}
	defer dst.Close()

	if strings.HasSuffix(strings.ToLower(dstPath), ".jpg") || strings.HasSuffix(strings.ToLower(dstPath), ".jpeg") {
		return jpeg.Encode(dst, thumb, &jpeg.Options{Quality: 80})
	}

	// Default to PNG if extension is not JPEG
	return jpeg.Encode(dst, thumb, &jpeg.Options{Quality: 80})
}

func imageDimensions(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}
