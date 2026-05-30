package validate

import "embed"

//go:generate sh -c "cp ../specs/*.json testdata/specs/"

//go:embed testdata/specs/*.json
var schemaFS embed.FS
