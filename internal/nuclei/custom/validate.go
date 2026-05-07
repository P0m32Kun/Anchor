package custom

import (
	"fmt"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidateYAML performs lightweight structural validation on a Nuclei template
// YAML. It checks:
//   - Valid YAML syntax
//   - Has an "id" field (required by Nuclei)
//   - For workflow-type templates, validates that referenced template paths exist
//     in the source's file tree
func ValidateYAML(data []byte, availableTemplates map[string]bool) *TemplateValidationResult {
	result := &TemplateValidationResult{OK: true}

	// Step 1: Parse YAML
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		result.OK = false
		result.Errors = append(result.Errors, fmt.Sprintf("invalid YAML syntax: %v", err))
		return result
	}

	// yaml.Unmarshal wraps in a Document node; get the actual mapping
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		result.OK = false
		result.Errors = append(result.Errors, "empty YAML document")
		return result
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		result.OK = false
		result.Errors = append(result.Errors, "YAML root must be a mapping")
		return result
	}

	// Step 2: Check for required "id" field
	idValue := getYAMLMapValue(root, "id")
	if idValue == "" {
		result.OK = false
		result.Errors = append(result.Errors, "missing required field 'id'")
	}

	// Step 3: Check for workflow template references
	workflowsNode := getYAMLMapNode(root, "workflows")
	if workflowsNode != nil && workflowsNode.Kind == yaml.SequenceNode {
		for _, wf := range workflowsNode.Content {
			if wf.Kind == yaml.MappingNode {
				tmplPath := getYAMLMapValue(wf, "template")
				if tmplPath != "" {
					result.WorkflowRefs = append(result.WorkflowRefs, tmplPath)
				}
				// Check subtemplates in workflow sequences
				subtemplates := getYAMLMapNode(wf, "subtemplates")
				if subtemplates != nil && subtemplates.Kind == yaml.SequenceNode {
					for _, sub := range subtemplates.Content {
						if sub.Kind == yaml.MappingNode {
							subPath := getYAMLMapValue(sub, "template")
							if subPath != "" {
								result.WorkflowRefs = append(result.WorkflowRefs, subPath)
							}
						}
					}
				}
			}
		}
	}

	// Step 4: Validate workflow template references exist
	if len(result.WorkflowRefs) > 0 && availableTemplates != nil {
		for _, ref := range result.WorkflowRefs {
			// Normalize path - remove leading ./ and /
			cleaned := strings.TrimPrefix(ref, "./")
			cleaned = strings.TrimPrefix(cleaned, "/")
			// Add templates/ prefix if not already present
			if !strings.HasPrefix(cleaned, "templates/") {
				cleaned = path.Join("templates", cleaned)
			}
			if !availableTemplates[cleaned] {
				result.OK = false
				result.Errors = append(result.Errors, fmt.Sprintf("workflow references missing template: %s", ref))
			}
		}
	}

	return result
}

// TemplateValidationResult holds the outcome of validating a single template.
type TemplateValidationResult struct {
	OK            bool     `json:"ok"`
	Errors        []string `json:"errors,omitempty"`
	WorkflowRefs  []string `json:"workflow_refs,omitempty"`
}

// getYAMLMapValue returns the scalar string value of a key in a YAML mapping
// node. Returns "" if key is not found or value is not a scalar.
func getYAMLMapValue(node *yaml.Node, key string) string {
	n := getYAMLMapNode(node, key)
	if n == nil || n.Kind != yaml.ScalarNode {
		return ""
	}
	return n.Value
}

// getYAMLMapNode returns the value node for a key in a YAML mapping node.
// Returns nil if key is not found.
func getYAMLMapNode(node *yaml.Node, key string) *yaml.Node {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Kind == yaml.ScalarNode && node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
