package config

import _ "embed"

// Default defines a default variable.
//go:embed config.yaml
var Default []byte
