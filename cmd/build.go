package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dragonfleas/kungfu/internal/models"
	"github.com/dragonfleas/kungfu/internal/parser"
	"github.com/dragonfleas/kungfu/internal/patcher"
	"github.com/spf13/cobra"
)

// NewBuildCmd creates the build command.
func NewBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build [root-module-path]",
		Short: "Build patched modules from overlays",
		Long: `Build reads the root module, finds all module declarations, applies patches
from overlay files, and generates patched modules.

Example:
  kungfu build . --overlay overlays/production`,
		Args: cobra.MaximumNArgs(1),
		RunE: runBuild,
	}

	cmd.Flags().String(
		"overlay", "",
		"Specific .kf.hcl file or directory (default: overlays/)")
	cmd.Flags().StringP(
		"output", "o", ".terraform/kungfu/modules",
		"Output directory for patched modules")

	return cmd
}

func runBuild(cmd *cobra.Command, args []string) error {
	absRoot, err := resolveRootPath(args)
	if err != nil {
		return err
	}

	outputDir, _ := cmd.Flags().GetString("output")

	cmd.Printf("Root module: %s\n", absRoot)
	cmd.Printf("Output directory: %s\n", outputDir)

	modules, err := loadAndDisplayModules(cmd, absRoot)
	if err != nil {
		return err
	}

	kfFiles, err := loadOverlayFiles(cmd, absRoot)
	if err != nil {
		return err
	}

	if len(kfFiles) == 0 {
		return nil
	}

	allPatches, err := parseOverlayFiles(cmd, kfFiles)
	if err != nil {
		return err
	}

	patchesBySource := groupPatchesBySource(allPatches)

	if applyErr := applyPatchesToModules(cmd, absRoot, outputDir, modules, patchesBySource); applyErr != nil {
		return applyErr
	}

	if updateErr := updateModulesJSON(absRoot, patchesBySource, modules); updateErr != nil {
		return fmt.Errorf("failed to update modules.json: %w", updateErr)
	}

	cmd.Printf("\nBuild completed successfully!\n")
	cmd.Printf("Patched modules are now active. Run 'terraform plan' to see changes.\n")
	return nil
}

func resolveRootPath(args []string) (string, error) {
	rootPath := "."
	if len(args) > 0 {
		rootPath = args[0]
	}

	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve root path: %w", err)
	}
	return absRoot, nil
}

func loadAndDisplayModules(cmd *cobra.Command, absRoot string) ([]models.ModuleCall, error) {
	modules, err := parser.ParseRootModule(absRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to parse root module: %w", err)
	}

	cmd.Printf("Found %d module(s) in root module\n", len(modules))
	for _, mod := range modules {
		cmd.Printf("  - %s (source: %s)\n", mod.Name, mod.Source)
	}

	return modules, nil
}

func loadOverlayFiles(cmd *cobra.Command, absRoot string) ([]string, error) {
	overlayDir, _ := cmd.Flags().GetString("overlay")
	if overlayDir == "" {
		overlayDir = "overlays"
	}

	overlayPath := overlayDir
	if !filepath.IsAbs(overlayPath) {
		overlayPath = filepath.Join(absRoot, overlayDir)
	}

	fileInfo, err := os.Stat(overlayPath)
	if err != nil {
		return nil, fmt.Errorf("overlay path does not exist: %s", overlayPath)
	}

	var kfFiles []string
	if fileInfo.IsDir() {
		kfFiles, err = FindKungfuFiles(overlayPath)
		if err != nil {
			return nil, fmt.Errorf("failed to find overlay files: %w", err)
		}
		cmd.Printf("Overlay directory: %s\n", overlayPath)
	} else {
		if !strings.HasSuffix(overlayPath, ".kf.hcl") {
			return nil, fmt.Errorf("overlay file must have .kf.hcl extension: %s", overlayPath)
		}
		kfFiles = []string{overlayPath}
		cmd.Printf("Overlay file: %s\n", overlayPath)
	}

	if len(kfFiles) == 0 {
		cmd.Printf("No .kf.hcl files found in %s\n", overlayPath)
	}

	return kfFiles, nil
}

func parseOverlayFiles(cmd *cobra.Command, kfFiles []string) ([]models.Patch, error) {
	cmd.Printf("\nFound %d overlay file(s)\n", len(kfFiles))

	var allPatches []models.Patch
	for _, kfFile := range kfFiles {
		config, parseErr := parser.ParseKungfuFile(kfFile)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", kfFile, parseErr)
		}
		allPatches = append(allPatches, config.Patches...)
		cmd.Printf("  - %s (%d patch(es))\n", filepath.Base(kfFile), len(config.Patches))
	}

	return allPatches, nil
}

func applyPatchesToModules(
	cmd *cobra.Command,
	absRoot string,
	outputDir string,
	modules []models.ModuleCall,
	patchesBySource map[string][]models.Patch,
) error {
	for source, patches := range patchesBySource {
		module := findModuleBySource(modules, source)
		if module == nil {
			cmd.Printf("\nWarning: No module found for source %s, skipping patches\n", source)
			continue
		}

		if err := patchSingleModule(cmd, absRoot, outputDir, module, patches); err != nil {
			return err
		}
	}
	return nil
}

func patchSingleModule(
	cmd *cobra.Command,
	absRoot string,
	outputDir string,
	module *models.ModuleCall,
	patches []models.Patch,
) error {
	cmd.Printf("\nPatching module %s (source: %s)\n", module.Name, module.Source)
	cmd.Printf("  Module path: %s\n", module.Path)

	if _, statErr := os.Stat(module.Path); os.IsNotExist(statErr) {
		return fmt.Errorf("module path does not exist: %s", module.Path)
	}

	tfFiles, findErr := FindTerraformFiles(module.Path)
	if findErr != nil {
		return fmt.Errorf("failed to find terraform files in %s: %w", module.Path, findErr)
	}

	parsedFiles, err := parseModuleFiles(tfFiles)
	if err != nil {
		return err
	}

	patchedFiles, patchErr := patcher.ApplyPatches(parsedFiles, patches)
	if patchErr != nil {
		return fmt.Errorf("failed to apply patches: %w", patchErr)
	}

	return writeModuleFiles(cmd, absRoot, outputDir, module, patchedFiles)
}

func parseModuleFiles(tfFiles []string) (map[string]*models.HCLFile, error) {
	parsedFiles := make(map[string]*models.HCLFile)
	for _, tfFile := range tfFiles {
		parsed, parseErr := parser.ParseHCLFile(tfFile)
		if parseErr != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", tfFile, parseErr)
		}
		parsedFiles[tfFile] = parsed
	}
	return parsedFiles, nil
}

func writeModuleFiles(
	cmd *cobra.Command,
	absRoot string,
	outputDir string,
	module *models.ModuleCall,
	patchedFiles map[string]*models.HCLFile,
) error {
	moduleOutputDir := filepath.Join(absRoot, outputDir, module.Name)
	if mkdirErr := os.MkdirAll(moduleOutputDir, 0750); mkdirErr != nil {
		return fmt.Errorf("failed to create output directory: %w", mkdirErr)
	}

	for originalPath, hclFile := range patchedFiles {
		relPath, relErr := filepath.Rel(module.Path, originalPath)
		if relErr != nil {
			return fmt.Errorf("failed to calculate relative path: %w", relErr)
		}
		outputPath := filepath.Join(moduleOutputDir, relPath)

		if mkdirErr := os.MkdirAll(filepath.Dir(outputPath), 0750); mkdirErr != nil {
			return fmt.Errorf("failed to create output subdirectory: %w", mkdirErr)
		}

		if writeErr := parser.WriteHCLFile(outputPath, hclFile); writeErr != nil {
			return fmt.Errorf("failed to write %s: %w", outputPath, writeErr)
		}
		cmd.Printf("    Wrote: %s\n", relPath)
	}

	return nil
}

func groupPatchesBySource(patches []models.Patch) map[string][]models.Patch {
	result := make(map[string][]models.Patch)
	for _, patch := range patches {
		if patch.Source == "" {
			continue
		}
		result[patch.Source] = append(result[patch.Source], patch)
	}
	return result
}

func findModuleBySource(modules []models.ModuleCall, source string) *models.ModuleCall {
	for _, mod := range modules {
		if mod.Source == source {
			return &mod
		}
	}
	return nil
}

// FindKungfuFiles finds all .kf.hcl files in a directory.
func FindKungfuFiles(dir string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []string{}, nil
	}

	var kfFiles []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".kf.hcl") {
			kfFiles = append(kfFiles, path)
		}
		return nil
	})

	return kfFiles, err
}

// FindTerraformFiles finds all .tf files in a module path.
func FindTerraformFiles(modulePath string) ([]string, error) {
	var tfFiles []string

	err := filepath.Walk(modulePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".tf") && !strings.HasSuffix(path, ".kf.hcl") {
			tfFiles = append(tfFiles, path)
		}
		return nil
	})

	return tfFiles, err
}

func updateModulesJSON(rootPath string, patchedSources map[string][]models.Patch, _ []models.ModuleCall) error {
	modulesJSONPath := filepath.Join(rootPath, ".terraform", "modules", "modules.json")

	if _, statErr := os.Stat(modulesJSONPath); os.IsNotExist(statErr) {
		return errors.New("modules.json not found - run 'terraform init' first")
	}

	data, err := os.ReadFile(modulesJSONPath)
	if err != nil {
		return fmt.Errorf("failed to read modules.json: %w", err)
	}

	var manifest models.ModulesManifest
	if unmarshalErr := json.Unmarshal(data, &manifest); unmarshalErr != nil {
		return fmt.Errorf("failed to parse modules.json: %w", unmarshalErr)
	}

	for i := range manifest.Modules {
		entry := &manifest.Modules[i]

		if entry.Key == "" {
			continue
		}

		// Normalize the source to match patch sources
		// OpenTofu/Terraform adds registry prefix like "registry.opentofu.org/"
		normalizedSource := normalizeModuleSource(entry.Source)

		if _, patched := patchedSources[normalizedSource]; patched {
			newDir := filepath.Join(".terraform", "kungfu", "modules", entry.Key)
			entry.Dir = newDir
		}
	}

	updatedData, marshalErr := json.MarshalIndent(manifest, "", "  ")
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal modules.json: %w", marshalErr)
	}

	if writeErr := os.WriteFile(modulesJSONPath, updatedData, 0600); writeErr != nil {
		return fmt.Errorf("failed to write modules.json: %w", writeErr)
	}

	return nil
}

// normalizeModuleSource removes registry prefixes to allow matching with patch sources.
func normalizeModuleSource(source string) string {
	// Remove common registry prefixes
	prefixes := []string{
		"registry.opentofu.org/",
		"registry.terraform.io/",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(source, prefix) {
			return strings.TrimPrefix(source, prefix)
		}
	}

	return source
}
