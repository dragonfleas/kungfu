package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dragonfleas/kungfu/internal/models"
	"github.com/dragonfleas/kungfu/internal/parser"
)

// WriteTestFile writes a file with the given content to the specified directory.
// It returns the full path to the created file.
func WriteTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	filePath := filepath.Join(dir, name)
	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	return filePath
}

// WriteAndParseKungfuFile writes a Kungfu file with the given content,
// parses it, and returns the parsed config and file path.
func WriteAndParseKungfuFile(t *testing.T, content string) (*models.KungfuConfig, string) {
	t.Helper()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.kf.hcl")

	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	config, err := parser.ParseKungfuFile(filePath)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	return config, filePath
}

// WriteAndParseHCLFile writes a Terraform file with the given content,
// parses it, and returns the parsed HCL file and file path.
func WriteAndParseHCLFile(t *testing.T, content string) (*models.HCLFile, string) {
	t.Helper()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.tf")

	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	hclFile, err := parser.ParseHCLFile(filePath)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	return hclFile, filePath
}

// SetupTerraformFile creates a temporary Terraform file with the given content,
// parses it, and returns a map of HCL files ready for patching.
func SetupTerraformFile(t *testing.T, content string) (map[string]*models.HCLFile, string) {
	t.Helper()
	tmpDir := t.TempDir()
	tfFile := filepath.Join(tmpDir, "main.tf")

	if err := os.WriteFile(tfFile, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	hclFile, _ := parser.ParseHCLFile(tfFile)
	files := map[string]*models.HCLFile{tfFile: hclFile}

	return files, tfFile
}
