package validate

import (
	"sort"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Violation is one schema failure with its JSON-pointer path, the rule that fired,
// and a human-readable message. Path is always a leading-slash JSON pointer
// (RFC 6901), e.g. "/floors/0/nodes/3/data/k8s/kind".
type Violation struct {
	Path    string
	Rule    string
	Message string
}

// MultiError aggregates all violations found during one Validate call. Implements error.
// Use Violations() to iterate; the error string is a deterministic newline-joined
// "<path>: <rule>: <message>" rendering, sorted by Path then Rule for stable test output.
type MultiError struct {
	violations []Violation
}

func (multi *MultiError) Violations() []Violation {
	out := make([]Violation, len(multi.violations))
	copy(out, multi.violations)
	return out
}

func (multi *MultiError) Error() string {
	if multi == nil || len(multi.violations) == 0 {
		return "bgbase/validate: no violations"
	}
	sorted := append([]Violation(nil), multi.violations...)
	sort.SliceStable(sorted, func(left, right int) bool {
		if sorted[left].Path != sorted[right].Path {
			return sorted[left].Path < sorted[right].Path
		}
		return sorted[left].Rule < sorted[right].Rule
	})
	parts := make([]string, 0, len(sorted))
	for _, violation := range sorted {
		parts = append(parts, violation.Path+": "+violation.Rule+": "+violation.Message)
	}
	return "bgbase/validate: " + strings.Join(parts, "; ")
}

func collectViolations(root *jsonschema.ValidationError) *MultiError {
	multi := &MultiError{}
	walkValidationError(root, multi)
	if len(multi.violations) == 0 {
		multi.violations = append(multi.violations, Violation{
			Path:    root.InstanceLocation,
			Rule:    keywordTail(root.KeywordLocation),
			Message: root.Message,
		})
	}
	return multi
}

func walkValidationError(node *jsonschema.ValidationError, multi *MultiError) {
	if len(node.Causes) == 0 {
		multi.violations = append(multi.violations, Violation{
			Path:    node.InstanceLocation,
			Rule:    keywordTail(node.KeywordLocation),
			Message: node.Message,
		})
		return
	}
	for _, child := range node.Causes {
		walkValidationError(child, multi)
	}
}

func keywordTail(keywordLocation string) string {
	parts := strings.Split(keywordLocation, "/")
	if len(parts) == 0 {
		return keywordLocation
	}
	return parts[len(parts)-1]
}
