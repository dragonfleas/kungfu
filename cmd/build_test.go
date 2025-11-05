package cmd_test

import (
	"testing"

	"github.com/dragonfleas/kungfu/cmd"
	"github.com/dragonfleas/kungfu/internal/testutil"
)

func TestFindKungfuFiles_Found(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.WriteTestFile(t, tmpDir, "test.kf.hcl", "")

	files, err := cmd.FindKungfuFiles(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}
}

func TestFindKungfuFiles_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	files, err := cmd.FindKungfuFiles(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestFindKungfuFiles_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.WriteTestFile(t, tmpDir, "production.kf.hcl", "")
	testutil.WriteTestFile(t, tmpDir, "staging.kf.hcl", "")

	files, err := cmd.FindKungfuFiles(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}
}

func TestFindTerraformFiles_Found(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.WriteTestFile(t, tmpDir, "main.tf", "")

	files, err := cmd.FindTerraformFiles(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}
}

func TestFindTerraformFiles_ExcludesKungfuFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.WriteTestFile(t, tmpDir, "main.tf", "")
	testutil.WriteTestFile(t, tmpDir, "test.kf.hcl", "")

	files, err := cmd.FindTerraformFiles(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 .tf file, got %d", len(files))
	}
}

func TestFindTerraformFiles_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.WriteTestFile(t, tmpDir, "main.tf", "")
	testutil.WriteTestFile(t, tmpDir, "variables.tf", "")
	testutil.WriteTestFile(t, tmpDir, "outputs.tf", "")

	files, err := cmd.FindTerraformFiles(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d", len(files))
	}
}
