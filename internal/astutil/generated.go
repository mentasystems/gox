package astutil

import (
	"bufio"
	"os"
	"regexp"
)

// generatedMarker matches the official Go convention for marking generated
// code: a line `// Code generated ... DO NOT EDIT.` near the top of the file.
// See https://pkg.go.dev/cmd/go#hdr-Generate_Go_files_by_processing_source.
//
// global-ok: compiled regex used read-only by IsGenerated; init once.
var generatedMarker = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.\s*$`)

// IsGenerated reports whether the file at `path` is generated code (and
// therefore should be exempt from analysis). It only reads the first 20
// lines; the convention is that the marker appears near the top.
func IsGenerated(path string) bool {
	f, openErr := os.Open(path)
	if openErr != nil {
		return false
	}
	defer f.Close() // safe-ignore: read-only file
	scanner := bufio.NewScanner(f)
	for i := 0; i < 20 && scanner.Scan(); i++ {
		if generatedMarker.MatchString(scanner.Text()) {
			return true
		}
	}
	return false
}
