package workflow

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/db"
	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestShouldFfufEndpoint_TierRules(t *testing.T) {
	sqlDB, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	p := NewPipeline(db.New(sqlDB), nil, nil, t.TempDir()).
		WithConfig(models.PipelineConfig{FfufTier: "small"})

	sc200 := 200
	sc404 := 404
	if !p.shouldFfufEndpoint(&models.WebEndpoint{StatusCode: &sc200}) {
		t.Error("small tier: expected 200 eligible")
	}
	if p.shouldFfufEndpoint(&models.WebEndpoint{StatusCode: &sc404}) {
		t.Error("small tier: expected 404 ineligible")
	}

	p.config.FfufTier = "medium"
	if p.shouldFfufEndpoint(&models.WebEndpoint{Technologies: []string{"nginx"}}) == false {
		t.Error("medium tier: expected fingerprint eligible")
	}
	if p.shouldFfufEndpoint(&models.WebEndpoint{Technologies: nil}) {
		t.Error("medium tier: expected no fingerprint ineligible")
	}

	p.config.FfufTier = "off"
	if p.shouldFfufEndpoint(&models.WebEndpoint{StatusCode: &sc200}) {
		t.Error("off tier: expected ineligible")
	}
}

func TestResolveFfufDictionaryID_TierPick(t *testing.T) {
	dir := t.TempDir()
	sqlDB, err := db.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	q := db.New(sqlDB)
	now := time.Now().UTC()
	for _, d := range []models.Dictionary{
		{ID: "builtin:small", Name: "small", Category: "dirscan", FilePath: "/tmp/s.txt", LineCount: 100, Enabled: true, Builtin: true, CreatedAt: now, UpdatedAt: now},
		{ID: "builtin:large", Name: "large", Category: "dirscan", FilePath: "/tmp/l.txt", LineCount: 5000, Enabled: true, Builtin: true, CreatedAt: now, UpdatedAt: now},
	} {
		if err := q.CreateDictionary(&d); err != nil {
			t.Fatal(err)
		}
	}

	p := NewPipeline(q, nil, nil, dir).WithConfig(models.PipelineConfig{FfufTier: "small"})
	if id := p.resolveFfufDictionaryID(); id != "builtin:small" {
		t.Fatalf("small tier id = %q, want builtin:small", id)
	}
	p.config.FfufTier = "medium"
	if id := p.resolveFfufDictionaryID(); id != "builtin:large" {
		t.Fatalf("medium tier id = %q, want builtin:large", id)
	}
}
