package main

import (
	"encoding/json"
	"os"

	"rootry/internal/kspoya"
)

func main() {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	enc.SetEscapeHTML(false)
	enc.Encode(kspoya.Bank)
}
