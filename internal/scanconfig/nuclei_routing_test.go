package scanconfig

import "testing"

func TestNucleiRouter_ResolveKnownTech(t *testing.T) {
	r := DefaultNucleiRouter()
	bucket, tags, skip := r.Resolve([]string{"Nginx/1.18"}, "low")
	if skip || bucket != "nginx" || len(tags) == 0 {
		t.Fatalf("bucket=%q tags=%v skip=%v", bucket, tags, skip)
	}
}

func TestNucleiRouter_SkipNoTechLowNoise(t *testing.T) {
	r := DefaultNucleiRouter()
	_, _, skip := r.Resolve(nil, "low")
	if !skip {
		t.Fatal("expected skip for no tech at low noise")
	}
}

func TestNucleiRouter_FallbackUnknownTech(t *testing.T) {
	r := DefaultNucleiRouter()
	bucket, tags, skip := r.Resolve([]string{"unknown-widget"}, "low")
	if skip || bucket != "_fallback" || len(tags) == 0 {
		t.Fatalf("bucket=%q tags=%v skip=%v", bucket, tags, skip)
	}
}

func TestNucleiRouter_TagsForBucket(t *testing.T) {
	r := DefaultNucleiRouter()
	tags := r.TagsForBucket("nuclei:jenkins")
	if len(tags) != 1 || tags[0] != "jenkins" {
		t.Fatalf("tags = %v", tags)
	}
}
