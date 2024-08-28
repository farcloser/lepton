package helpers

import (
	"fmt"
	"os/exec"
)

var NetworkDriversToKeep = []string{"host", "none", DefaultNetworkDriver}

func ExtractTarFile(dirPath, tarFilePath string) error {
	cmd := exec.Command("tar", "Cxf", dirPath, tarFilePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to run %v: %q: %w", cmd.Args, string(out), err)
	}
	return nil
}
