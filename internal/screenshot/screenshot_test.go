package screenshot

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestHasProtocol(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"example.com", false},
		{"ftp://example.com", false},
		{"", false},
	}
	for _, tt := range tests {
		got := hasProtocol(tt.input)
		if got != tt.expected {
			t.Errorf("hasProtocol(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestImageDimensions(t *testing.T) {
	tmpDir := t.TempDir()
	pngPath := filepath.Join(tmpDir, "test.png")

	img := image.NewRGBA(image.Rect(0, 0, 10, 20))
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("create test png: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("encode test png: %v", err)
	}
	f.Close()

	w, h, err := imageDimensions(pngPath)
	if err != nil {
		t.Fatalf("imageDimensions: %v", err)
	}
	if w != 10 || h != 20 {
		t.Errorf("imageDimensions: got %dx%d, want 10x20", w, h)
	}
}

func TestCaptureEmptyURL(t *testing.T) {
	_, err := Capture(t.Context(), "", t.TempDir(), 0)
	if err == nil {
		t.Error("expected error for empty URL")
	}
}

func TestCreateThumbnailInvalidSource(t *testing.T) {
	err := CreateThumbnail("/nonexistent/path.png", filepath.Join(t.TempDir(), "thumb.jpg"), 100, 100)
	if err == nil {
		t.Error("expected error for nonexistent source")
	}
}

func TestNewManager(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(nil, dir)
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	projectDir := mgr.ProjectScreenshotDir("test-project")
	expected := filepath.Join(dir, "projects", "test-project", "screenshots")
	if projectDir != expected {
		t.Errorf("ProjectScreenshotDir = %q, want %q", projectDir, expected)
	}
	thumbDir := mgr.ProjectThumbnailDir("test-project")
	expectedThumb := filepath.Join(dir, "projects", "test-project", "screenshots", "thumbnails")
	if thumbDir != expectedThumb {
		t.Errorf("ProjectThumbnailDir = %q, want %q", thumbDir, expectedThumb)
	}
}
