package charset

import (
	"unicode/utf8"
)

// Sniff performs a lightweight check to see if the data is valid UTF-8.
// This adheres to the "Extreme Restraint" philosophy by avoiding heavy CGO or complex sniffing.
func Sniff(data []byte) (string, error) {
	if utf8.Valid(data) {
		return "UTF-8", nil
	}
	// Fallback to GBK/Other placeholder for metadata
	return "UNKNOWN/GBK-LIKELY", nil
}
