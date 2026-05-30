package validate_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/BumbleGrid/bgbase/graph"
	"github.com/BumbleGrid/bgbase/validate"
)

func mustValidator(tb testing.TB) *validate.SchemaValidator {
	tb.Helper()
	validator, err := validate.NewSchemaValidator()
	if err != nil {
		tb.Fatalf("NewSchemaValidator: %v", err)
	}
	return validator
}

func loadFixture(tb testing.TB, name string) *graph.BGSpecDocument {
	tb.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		tb.Fatalf("read fixture %q: %v", name, err)
	}
	var doc graph.BGSpecDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		tb.Fatalf("unmarshal fixture %q: %v", name, err)
	}
	return &doc
}

func mustMultiError(tb testing.TB, err error) *validate.MultiError {
	tb.Helper()
	var multi *validate.MultiError
	if !errors.As(err, &multi) {
		tb.Fatalf("expected *MultiError, got %T: %v", err, err)
	}
	return multi
}

func hasViolation(violations []validate.Violation, path, rule string) bool {
	for _, violation := range violations {
		if violation.Path == path && violation.Rule == rule {
			return true
		}
	}
	return false
}

func TestNewSchemaValidator_compiles(t *testing.T) {
	validator, err := validate.NewSchemaValidator()
	if err != nil {
		t.Fatalf("NewSchemaValidator: %v", err)
	}
	if validator == nil {
		t.Fatal("expected non-nil validator")
	}
}

func TestSchemaValidator_validFloor0Only(t *testing.T) {
	validator := mustValidator(t)
	doc := loadFixture(t, "valid_floor_0_only.json")
	if err := validator.Validate(doc); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestSchemaValidator_validFullExample(t *testing.T) {
	validator := mustValidator(t)
	doc := loadFixture(t, "valid_full_document.json")
	if err := validator.Validate(doc); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestSchemaValidator_invalidMissingCompany(t *testing.T) {
	validator := mustValidator(t)
	doc := loadFixture(t, "invalid_missing_company.json")
	multi := mustMultiError(t, validator.Validate(doc))
	if !hasViolation(multi.Violations(), "/document/company", "minLength") {
		t.Fatalf("violations: %#v", multi.Violations())
	}
}

func TestSchemaValidator_invalidFloor0BadKind(t *testing.T) {
	validator := mustValidator(t)
	doc := loadFixture(t, "invalid_floor0_bad_kind.json")
	multi := mustMultiError(t, validator.Validate(doc))
	if !hasViolation(multi.Violations(), "/floors/0/nodes/0/data/bgKind", "enum") {
		t.Fatalf("violations: %#v", multi.Violations())
	}
}

func TestSchemaValidator_invalidBgspecVersion(t *testing.T) {
	validator := mustValidator(t)
	doc := loadFixture(t, "invalid_bgspec_version_pattern.json")
	multi := mustMultiError(t, validator.Validate(doc))
	if !hasViolation(multi.Violations(), "/bgspec", "pattern") {
		t.Fatalf("violations: %#v", multi.Violations())
	}
}

func TestSchemaValidator_invalidFloorCountOne(t *testing.T) {
	validator := mustValidator(t)
	doc := loadFixture(t, "invalid_floor_count_one.json")
	multi := mustMultiError(t, validator.Validate(doc))
	if !hasViolation(multi.Violations(), "/floors", "minItems") {
		t.Fatalf("violations: %#v", multi.Violations())
	}
}

func TestSchemaValidator_invalidFloor0MissingK8s(t *testing.T) {
	validator := mustValidator(t)
	doc := loadFixture(t, "invalid_floor0_missing_k8s.json")
	multi := mustMultiError(t, validator.Validate(doc))
	violations := multi.Violations()
	found := false
	for _, violation := range violations {
		if violation.Path == "/floors/0/nodes/0/data" && violation.Rule == "required" && strings.Contains(violation.Message, "k8s") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("violations: %#v", violations)
	}
}

func TestSchemaValidator_invalidFloor0ExtraField(t *testing.T) {
	validator := mustValidator(t)
	body, err := os.ReadFile(filepath.Join("testdata", "invalid_floor0_extra_field.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	multi := mustMultiError(t, validator.ValidateJSON(body))
	violations := multi.Violations()
	found := false
	for _, violation := range violations {
		if violation.Path == "/floors/0/nodes/0/data" && violation.Rule == "additionalProperties" && strings.Contains(violation.Message, "extra_field") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("violations: %#v", violations)
	}
}

func TestSchemaValidator_nilDoc(t *testing.T) {
	validator := mustValidator(t)
	err := validator.Validate(nil)
	if err == nil {
		t.Fatal("expected error for nil document")
	}
	var multi *validate.MultiError
	if errors.As(err, &multi) {
		t.Fatalf("expected plain error, got MultiError: %v", err)
	}
	if !strings.Contains(err.Error(), "nil document") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestSchemaValidator_concurrent(t *testing.T) {
	validator := mustValidator(t)
	doc := loadFixture(t, "valid_floor_0_only.json")
	var waitGroup sync.WaitGroup
	for worker := 0; worker < 32; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for run := 0; run < 64; run++ {
				if err := validator.Validate(doc); err != nil {
					t.Errorf("Validate: %v", err)
				}
			}
		}()
	}
	waitGroup.Wait()
}

func TestEmbeddedSchemaMatchesCanonical(t *testing.T) {
	canonical, err := os.ReadFile(filepath.Join("..", "specs", "bgspec.schema.json"))
	if err != nil {
		t.Fatalf("read canonical schema: %v", err)
	}
	embedded, err := os.ReadFile(filepath.Join("testdata", "specs", "bgspec.schema.json"))
	if err != nil {
		t.Fatalf("read embedded schema: %v", err)
	}
	if string(canonical) != string(embedded) {
		t.Fatal("embedded schema drifted from canonical; run: cd validate && go generate ./...")
	}
}

func TestSchemaValidator_multipleViolations(t *testing.T) {
	validator := mustValidator(t)
	doc := loadFixture(t, "invalid_multiple_violations.json")
	multi := mustMultiError(t, validator.Validate(doc))
	violations := multi.Violations()
	if len(violations) < 2 {
		t.Fatalf("expected at least 2 violations, got %d: %#v", len(violations), violations)
	}
}

func TestMultiError_Error_string(t *testing.T) {
	multi := &validate.MultiError{}
	multiErr := collectTestMultiError()
	got := multiErr.Error()
	wantParts := []string{
		"/bgspec: pattern:",
		"/document: required:",
		"/floors: minItems:",
	}
	for _, part := range wantParts {
		if !strings.Contains(got, part) {
			t.Fatalf("Error() = %q, missing %q", got, part)
		}
	}
	if multi.Error() != "bgbase/validate: no violations" {
		t.Fatalf("empty multi error = %q", multi.Error())
	}
}

func collectTestMultiError() *validate.MultiError {
	validator, err := validate.NewSchemaValidator()
	if err != nil {
		panic(err)
	}
	body, err := os.ReadFile(filepath.Join("testdata", "invalid_multiple_violations.json"))
	if err != nil {
		panic(err)
	}
	err = validator.ValidateJSON(body)
	multi, ok := err.(*validate.MultiError)
	if !ok {
		panic(err)
	}
	return multi
}

func TestSchemaValidator_ValidateJSON_validBytes(t *testing.T) {
	validator := mustValidator(t)
	body, err := os.ReadFile(filepath.Join("testdata", "valid_floor_0_only.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := validator.ValidateJSON(body); err != nil {
		t.Fatalf("ValidateJSON: %v", err)
	}
}

func TestSchemaValidator_ValidateJSON_invalidSyntax(t *testing.T) {
	validator := mustValidator(t)
	err := validator.ValidateJSON([]byte("{"))
	if err == nil {
		t.Fatal("expected decode error")
	}
	var multi *validate.MultiError
	if errors.As(err, &multi) {
		t.Fatalf("expected plain error, got MultiError: %v", err)
	}
}

func TestSchemaValidator_floor1IgnoresFloor0Schema(t *testing.T) {
	validator := mustValidator(t)
	body, err := os.ReadFile(filepath.Join("testdata", "valid_floor1_permissive_bgkind.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := validator.ValidateJSON(body); err != nil {
		t.Fatalf("ValidateJSON: %v", err)
	}
}
