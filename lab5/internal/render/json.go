package render

import (
	"bytes"
	"encoding/json"
)

// JSON pretty-prints a JSON document. If parsing fails the raw bytes are
// returned so the user still sees something useful.
func JSON(data []byte) string {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return string(data)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return string(data)
	}
	return buf.String()
}
