package validate_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BumbleGrid/bgbase/graph"
	"github.com/BumbleGrid/bgbase/validate"
)

func ExampleNewSchemaValidator() {
	body, _ := os.ReadFile(filepath.Join("testdata", "valid_floor_0_only.json"))
	var doc graph.BGSpecDocument
	_ = json.Unmarshal(body, &doc)

	validator, err := validate.NewSchemaValidator()
	if err != nil {
		fmt.Println("compile:", err)
		return
	}
	if err := validator.Validate(&doc); err != nil {
		fmt.Println("invalid:", err)
		return
	}
	fmt.Println("ok")
	// Output: ok
}
