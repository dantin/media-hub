package asset

import (
	"os"
	"path/filepath"
	"strconv"
)

// Convert relative filepath to absolute.
func toAbsolutePath(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Clean(filepath.Join(base, path))
}

// createPIDFile creates and writes process ID to PIDFile.
func createPIDFile(PIDFile string) error {
	if PIDFile == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(PIDFile), os.ModePerm); err != nil {
		return err
	}
	file, err := os.Create(PIDFile)
	if err != nil {
		return err
	}
	defer file.Close()

	currentPID := os.Getpid()
	if _, err := file.WriteString(strconv.Itoa(currentPID)); err != nil {
		return nil
	}

	return nil
}
