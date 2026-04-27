package nuclei

import (
	"testing"

	"github.com/P0m32Kun/Anchor/internal/models"
)

func TestMapPreciseTags_WordPress(t *testing.T) {
	tags := MapPreciseTags([]string{"WordPress"}, "")
	if len(tags) != 1 || tags[0] != "wordpress" {
		t.Fatalf("expected [wordpress], got %v", tags)
	}
}

func TestMapPreciseTags_ApacheDruid(t *testing.T) {
	// Apache Druid should map to the specific tag "apache-druid", not "apache".
	tags := MapPreciseTags([]string{"Apache Druid"}, "")
	if len(tags) != 1 || tags[0] != "apache-druid" {
		t.Fatalf("expected [apache-druid], got %v", tags)
	}
}

func TestMapPreciseTags_NginxVersion(t *testing.T) {
	tags := MapPreciseTags([]string{"nginx/1.18.0"}, "")
	if len(tags) != 1 || tags[0] != "nginx" {
		t.Fatalf("expected [nginx], got %v", tags)
	}
}

func TestMapPreciseTags_Multiple(t *testing.T) {
	tags := MapPreciseTags([]string{"WordPress", "nginx"}, "")
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", tags)
	}
	if tags[0] != "nginx" || tags[1] != "wordpress" {
		t.Fatalf("expected [nginx wordpress], got %v", tags)
	}
}

func TestMapPreciseTags_Duplicate(t *testing.T) {
	tags := MapPreciseTags([]string{"nginx", "nginx/1.18.0"}, "")
	if len(tags) != 1 || tags[0] != "nginx" {
		t.Fatalf("expected [nginx], got %v", tags)
	}
}

func TestMapPreciseTags_Unknown(t *testing.T) {
	tags := MapPreciseTags([]string{"SomeUnknownTech"}, "")
	if len(tags) != 0 {
		t.Fatalf("expected [], got %v", tags)
	}
}

func TestMapPreciseTags_WebServer(t *testing.T) {
	tags := MapPreciseTags([]string{}, "nginx")
	if len(tags) != 1 || tags[0] != "nginx" {
		t.Fatalf("expected [nginx], got %v", tags)
	}
}

func TestGroupEndpointsByTags(t *testing.T) {
	eps := []*models.WebEndpoint{
		{URL: "http://a.com", Technologies: []string{"WordPress"}},
		{URL: "http://b.com", Technologies: []string{"nginx"}},
		{URL: "http://c.com", Technologies: []string{"WordPress", "nginx"}},
		{URL: "http://d.com", Technologies: []string{"UnknownTech"}},
	}
	groups := GroupEndpointsByTags(eps)
	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d: %v", len(groups), groups)
	}
	if len(groups["wordpress"]) != 1 || groups["wordpress"][0] != "http://a.com" {
		t.Fatalf("unexpected wordpress group: %v", groups["wordpress"])
	}
	if len(groups["nginx"]) != 1 || groups["nginx"][0] != "http://b.com" {
		t.Fatalf("unexpected nginx group: %v", groups["nginx"])
	}
	if len(groups["nginx,wordpress"]) != 1 || groups["nginx,wordpress"][0] != "http://c.com" {
		t.Fatalf("unexpected nginx,wordpress group: %v", groups["nginx,wordpress"])
	}
}
