package nuclei

import (
	"sort"
	"strings"

	"github.com/P0m32Kun/Anchor/internal/models"
)

// techToTag maps normalized technology names to the most specific Nuclei tag.
// One fingerprint → one specific tag. No parent/child expansion.
var techToTag = map[string]string{
	// Web servers
	"nginx":              "nginx",
	"apache":             "apache",
	"apache http server": "apache",
	"iis":                "iis",
	"microsoft iis":      "iis",
	"tomcat":             "tomcat",
	"apache tomcat":      "tomcat",

	// Frameworks / CMS
	"wordpress":        "wordpress",
	"wp":               "wordpress",
	"joomla":           "joomla",
	"drupal":           "drupal",
	"django":           "django",
	"flask":            "flask",
	"laravel":          "laravel",
	"rails":            "rails",
	"ruby on rails":    "rails",
	"spring":           "spring",
	"spring boot":      "spring-boot",
	"spring framework": "spring",
	"express":          "express",
	"nodejs":           "nodejs",
	"node.js":          "nodejs",

	// Specific apps
	"phpmyadmin":      "phpmyadmin",
	"jenkins":         "jenkins",
	"gitlab":          "gitlab",
	"grafana":         "grafana",
	"prometheus":      "prometheus",
	"swagger":         "swagger",
	"graphql":         "graphql",
	"couchdb":         "couchdb",
	"rabbitmq":        "rabbitmq",
	"consul":          "consul",
	"vault":           "vault",
	"kibana":          "kibana",
	"nexus":           "nexus",
	"artifactory":     "artifactory",
	"sonarqube":       "sonarqube",
	"weblogic":        "weblogic",
	"websphere":       "websphere",
	"jboss":           "jboss",
	"wildfly":         "jboss",
	"thinkphp":        "thinkphp",
	"struts":          "struts",
	"apache struts":   "struts",

	// Apache projects (specific > generic)
	"apache druid":    "apache-druid",
	"druid":           "apache-druid",
	"apache solr":     "apache-solr",
	"solr":            "apache-solr",
	"apache spark":    "apache-spark",
	"spark":           "apache-spark",
	"apache hadoop":   "apache-hadoop",
	"hadoop":          "apache-hadoop",
	"apache flink":    "apache-flink",
	"flink":           "apache-flink",
	"apache kylin":    "apache-kylin",
	"kylin":           "apache-kylin",
	"apache axis2":    "apache-axis2",
	"axis2":           "apache-axis2",

	// Databases / infra
	"mongodb":         "mongodb",
	"mongo":           "mongodb",
	"mysql":           "mysql",
	"mariadb":         "mariadb",
	"postgresql":      "postgresql",
	"postgres":        "postgresql",
	"redis":           "redis",
	"elasticsearch":   "elasticsearch",
	"elastic":         "elasticsearch",
	"cassandra":       "cassandra",
	"neo4j":           "neo4j",
	"influxdb":        "influxdb",
	"memcached":       "memcached",

	// DevOps / misc
	"docker":          "docker",
	"kubernetes":      "kubernetes",
	"k8s":             "kubernetes",
	"git":             "git",
	"traefik":         "traefik",
	"istio":           "istio",
	"envoy":           "envoy",
}

// MapPreciseTags takes httpx fingerprint data and returns the most specific
// Nuclei tags. One tag per matched fingerprint. No generic fallback.
func MapPreciseTags(technologies []string, webserver string) []string {
	seen := make(map[string]bool)
	var tags []string

	addTag := func(key string) {
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "" {
			return
		}
		// Direct match.
		if tag, ok := techToTag[key]; ok {
			if !seen[tag] {
				seen[tag] = true
				tags = append(tags, tag)
			}
			return
		}
		// Try base name from "tech/version".
		if idx := strings.Index(key, "/"); idx > 0 {
			base := key[:idx]
			if tag, ok := techToTag[base]; ok {
				if !seen[tag] {
					seen[tag] = true
					tags = append(tags, tag)
				}
			}
		}
	}

	for _, t := range technologies {
		addTag(t)
	}

	if webserver != "" {
		addTag(webserver)
	}

	// Sort for stable grouping key.
	sort.Strings(tags)
	return tags
}

// GroupEndpointsByTags groups WebEndpoints by their precise tag sets.
// Returns a map where key is "tag1,tag2" (sorted) and value is list of URLs.
// Endpoints with no tags are omitted.
func GroupEndpointsByTags(endpoints []*models.WebEndpoint) map[string][]string {
	groups := make(map[string][]string)
	for _, ep := range endpoints {
		tags := MapPreciseTags(ep.Technologies, "")
		if len(tags) == 0 {
			continue // Skip endpoints with no fingerprint.
		}
		key := strings.Join(tags, ",")
		groups[key] = append(groups[key], ep.URL)
	}
	return groups
}
