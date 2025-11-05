package parser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dragonfleas/kungfu/internal/models"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

const (
	// expectedPatchLabels is the number of labels required for a patch block.
	expectedPatchLabels = 2
	// expectedResourceLabels is the number of labels required for resource/data blocks.
	expectedResourceLabels = 2
)

func ParseKungfuFile(path string) (*models.KungfuConfig, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(src, path)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL: %s", diags.Error())
	}

	config := &models.KungfuConfig{
		Patches: make([]models.Patch, 0),
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, errors.New("unexpected body type")
	}
	for _, block := range body.Blocks {
		if block.Type == "patch" {
			patch, patchErr := parsePatchBlock(block)
			if patchErr != nil {
				return nil, fmt.Errorf("failed to parse patch block: %w", patchErr)
			}
			config.Patches = append(config.Patches, patch)
		}
	}

	return config, nil
}

func parsePatchBlock(block *hclsyntax.Block) (models.Patch, error) {
	if len(block.Labels) != expectedPatchLabels {
		return models.Patch{}, fmt.Errorf(
			"patch block requires exactly %d labels (type and name), got %d",
			expectedPatchLabels, len(block.Labels))
	}

	patch := models.Patch{
		ResourceType: block.Labels[0],
		ResourceName: block.Labels[1],
		Attributes:   make(map[string]*models.PatchAttribute),
		Range:        block.Range(),
	}

	for name, attr := range block.Body.Attributes {
		if name == "source" {
			val, diags := attr.Expr.Value(nil)
			if diags.HasErrors() {
				return models.Patch{}, fmt.Errorf("failed to evaluate source attribute: %s", diags.Error())
			}
			patch.Source = val.AsString()
			continue
		}

		patchAttr := &models.PatchAttribute{
			Strategy: models.StrategyReplace,
		}

		strategy, value := detectMergeStrategy(attr.Expr)
		patchAttr.Strategy = strategy

		evalValue, diags := value.Value(nil)
		if diags.HasErrors() {
			patchAttr.Value = value
		} else {
			patchAttr.Value = evalValue
		}

		patch.Attributes[name] = patchAttr
	}

	return patch, nil
}

func detectMergeStrategy(expr hclsyntax.Expression) (models.MergeStrategy, hclsyntax.Expression) {
	callExpr, ok := expr.(*hclsyntax.FunctionCallExpr)
	if !ok {
		return models.StrategyReplace, expr
	}

	switch callExpr.Name {
	case "merge":
		if len(callExpr.Args) == 1 {
			return models.StrategyMerge, callExpr.Args[0]
		}
	case "append":
		if len(callExpr.Args) == 1 {
			return models.StrategyAppend, callExpr.Args[0]
		}
	case "replace":
		if len(callExpr.Args) == 1 {
			return models.StrategyReplace, callExpr.Args[0]
		}
	}

	return models.StrategyReplace, expr
}

func ParseHCLFile(path string) (*models.HCLFile, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	writeFile, diags := hclwrite.ParseConfig(src, path, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse HCL for writing: %s", diags.Error())
	}

	hclFile := &models.HCLFile{
		Path:      path,
		OrigBytes: src,
		WriteFile: writeFile,
		Resources: make(map[string]*models.Resource),
		Variables: make(map[string]*models.Variable),
		Outputs:   make(map[string]*models.Output),
		Locals:    make(map[string]*models.Local),
		Data:      make(map[string]*models.DataSource),
	}

	body := writeFile.Body()
	for _, block := range body.Blocks() {
		switch block.Type() {
		case "resource":
			labels := block.Labels()
			if len(labels) == expectedResourceLabels {
				key := models.ResourceKey(labels[0], labels[1])
				hclFile.Resources[key] = &models.Resource{
					Type:  labels[0],
					Name:  labels[1],
					Block: block,
				}
			}
		case "variable":
			labels := block.Labels()
			if len(labels) == 1 {
				hclFile.Variables[labels[0]] = &models.Variable{
					Name:  labels[0],
					Block: block,
				}
			}
		case "output":
			labels := block.Labels()
			if len(labels) == 1 {
				hclFile.Outputs[labels[0]] = &models.Output{
					Name:  labels[0],
					Block: block,
				}
			}
		case "locals":
			hclFile.Locals["locals"] = &models.Local{
				Block: block,
			}
		case "data":
			labels := block.Labels()
			if len(labels) == expectedResourceLabels {
				key := models.ResourceKey(labels[0], labels[1])
				hclFile.Data[key] = &models.DataSource{
					Type:  labels[0],
					Name:  labels[1],
					Block: block,
				}
			}
		}
	}

	return hclFile, nil
}

func WriteHCLFile(path string, hclFile *models.HCLFile) error {
	data := hclFile.WriteFile.Bytes()
	return os.WriteFile(path, data, 0600)
}

func ParseRootModule(rootPath string) ([]models.ModuleCall, error) {
	tfFiles, err := filepath.Glob(filepath.Join(rootPath, "*.tf"))
	if err != nil {
		return nil, fmt.Errorf("failed to find .tf files: %w", err)
	}

	var modules []models.ModuleCall
	for _, tfFile := range tfFiles {
		fileModules := parseModulesFromFile(tfFile, rootPath)
		modules = append(modules, fileModules...)
	}

	return modules, nil
}

func parseModulesFromFile(tfFile, rootPath string) []models.ModuleCall {
	src, readErr := os.ReadFile(tfFile)
	if readErr != nil {
		return nil
	}

	hclParser := hclparse.NewParser()
	file, diags := hclParser.ParseHCL(src, tfFile)
	if diags.HasErrors() {
		return nil
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil
	}

	var modules []models.ModuleCall
	for _, block := range body.Blocks {
		if block.Type != "module" || len(block.Labels) != 1 {
			continue
		}

		moduleCall := extractModuleCall(block, rootPath)
		if moduleCall.Source != "" {
			modules = append(modules, moduleCall)
		}
	}

	return modules
}

func extractModuleCall(block *hclsyntax.Block, rootPath string) models.ModuleCall {
	moduleCall := models.ModuleCall{
		Name: block.Labels[0],
	}

	for name, attr := range block.Body.Attributes {
		if name == "source" {
			val, evalDiags := attr.Expr.Value(nil)
			if !evalDiags.HasErrors() {
				moduleCall.Source = val.AsString()
			}
		}
	}

	if moduleCall.Source != "" {
		moduleCall.Path = resolveModulePath(rootPath, moduleCall.Name, moduleCall.Source)
	}

	return moduleCall
}

func resolveModulePath(rootPath, moduleName, source string) string {
	if filepath.IsAbs(source) {
		return source
	}
	// Local modules start with ./ or /
	if len(source) > 0 && (source[0] == '.' || source[0] == '/') {
		return filepath.Join(rootPath, source)
	}
	// Remote modules (registry, git, etc.) use the module name as directory
	return filepath.Join(rootPath, ".terraform", "modules", moduleName)
}
