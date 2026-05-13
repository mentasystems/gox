package baseline

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ModuleRoot returns the absolute path of the Go module root for the current
// working directory, by asking `go env GOMOD`.
func ModuleRoot() (string, error) {
	cmd := exec.Command("go", "env", "GOMOD")
	out, runErr := cmd.Output()
	if runErr != nil {
		return "", fmt.Errorf("go env GOMOD: %w", runErr)
	}
	gomod := strings.TrimSpace(string(out))
	if gomod == "" || gomod == "/dev/null" {
		return "", errors.New("not inside a Go module")
	}
	return filepath.Dir(gomod), nil
}
