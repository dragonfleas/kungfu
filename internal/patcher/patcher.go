package patcher

import (
	"fmt"

	"github.com/dragonfleas/kungfu/internal/models"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

func ApplyPatches(files map[string]*models.HCLFile, patches []models.Patch) (map[string]*models.HCLFile, error) {
	patchedFiles := make(map[string]*models.HCLFile)
	for path, file := range files {
		patchedFiles[path] = file
	}

	for _, patch := range patches {
		if err := applyPatch(patchedFiles, patch); err != nil {
			return nil, fmt.Errorf("failed to apply patch for %s.%s: %w",
				patch.ResourceType, patch.ResourceName, err)
		}
	}

	return patchedFiles, nil
}

func applyPatch(files map[string]*models.HCLFile, patch models.Patch) error {
	resourceKey := models.ResourceKey(patch.ResourceType, patch.ResourceName)

	var targetResource *models.Resource

	for _, file := range files {
		if resource, exists := file.Resources[resourceKey]; exists {
			targetResource = resource
			break
		}
	}

	if targetResource == nil {
		return fmt.Errorf("resource %s not found in any file", resourceKey)
	}

	resourceBody := targetResource.Block.Body()
	for attrName, patchAttr := range patch.Attributes {
		if err := applyAttribute(resourceBody, attrName, patchAttr); err != nil {
			return fmt.Errorf("failed to apply attribute %s: %w", attrName, err)
		}
	}

	return nil
}

func applyAttribute(body *hclwrite.Body, name string, patchAttr *models.PatchAttribute) error {
	switch patchAttr.Strategy {
	case models.StrategyReplace:
		return replaceAttribute(body, name, patchAttr.Value)
	case models.StrategyMerge:
		return mergeAttribute(body, name, patchAttr.Value)
	case models.StrategyAppend:
		return appendAttribute(body, name, patchAttr.Value)
	default:
		return fmt.Errorf("unknown merge strategy: %d", patchAttr.Strategy)
	}
}

func replaceAttribute(body *hclwrite.Body, name string, value interface{}) error {
	body.RemoveAttribute(name)
	tokens := valueToTokens(value)
	body.SetAttributeRaw(name, tokens)
	return nil
}

func mergeAttribute(body *hclwrite.Body, name string, value interface{}) error {
	existingAttr := body.GetAttribute(name)
	if existingAttr == nil {
		return replaceAttribute(body, name, value)
	}

	existingVal := extractValue(*existingAttr.Expr())
	mergedVal := DeepMerge(existingVal, value)

	body.RemoveAttribute(name)
	tokens := valueToTokens(mergedVal)
	body.SetAttributeRaw(name, tokens)
	return nil
}

func appendAttribute(body *hclwrite.Body, name string, value interface{}) error {
	existingAttr := body.GetAttribute(name)
	if existingAttr == nil {
		return replaceAttribute(body, name, value)
	}

	existingVal := extractValue(*existingAttr.Expr())
	appendedVal := AppendToList(existingVal, value)

	body.RemoveAttribute(name)
	tokens := valueToTokens(appendedVal)
	body.SetAttributeRaw(name, tokens)
	return nil
}

// DeepMerge recursively merges two objects.
func DeepMerge(existing interface{}, patch interface{}) interface{} {
	existingCty, existingIsCty := existing.(cty.Value)
	patchCty, patchIsCty := patch.(cty.Value)

	if !existingIsCty || !patchIsCty {
		return patch
	}

	if !existingCty.Type().IsObjectType() || !patchCty.Type().IsObjectType() {
		return patch
	}

	mergedMap := make(map[string]cty.Value)

	for key, val := range existingCty.AsValueMap() {
		mergedMap[key] = val
	}

	for key, val := range patchCty.AsValueMap() {
		existingVal, exists := mergedMap[key]
		if !exists {
			mergedMap[key] = val
			continue
		}

		if !existingVal.Type().IsObjectType() || !val.Type().IsObjectType() {
			mergedMap[key] = val
			continue
		}

		merged, ok := DeepMerge(existingVal, val).(cty.Value)
		if ok {
			mergedMap[key] = merged
		} else {
			mergedMap[key] = val
		}
	}

	return cty.ObjectVal(mergedMap)
}

// AppendToList appends elements from patch list to existing list.
func AppendToList(existing interface{}, patch interface{}) interface{} {
	existingCty, existingIsCty := existing.(cty.Value)
	patchCty, patchIsCty := patch.(cty.Value)

	if !existingIsCty || !patchIsCty {
		return patch
	}

	if !existingCty.Type().IsTupleType() && !existingCty.Type().IsListType() {
		return patch
	}

	if !patchCty.Type().IsTupleType() && !patchCty.Type().IsListType() {
		return patch
	}

	existingList := existingCty.AsValueSlice()
	patchList := patchCty.AsValueSlice()

	merged := make([]cty.Value, 0, len(existingList)+len(patchList))
	merged = append(merged, existingList...)
	merged = append(merged, patchList...)
	return cty.TupleVal(merged)
}

func extractValue(expr hclwrite.Expression) interface{} {
	tokens := expr.BuildTokens(nil)
	src := string(tokens.Bytes())

	parsed, diags := hclsyntax.ParseExpression([]byte(src), "", hcl.Pos{Line: 1, Column: 1, Byte: 0})
	if diags.HasErrors() {
		return nil
	}

	val, diags := parsed.Value(nil)
	if diags.HasErrors() {
		return nil
	}

	return val
}

func valueToTokens(value interface{}) hclwrite.Tokens {
	switch v := value.(type) {
	case cty.Value:
		return hclwrite.TokensForValue(v)
	case hclsyntax.Expression:
		exprBytes := v.Range().SliceBytes(v.Range().SliceBytes([]byte{}))
		if len(exprBytes) > 0 {
			return hclwrite.TokensForIdentifier(string(exprBytes))
		}
		return hclwrite.TokensForValue(cty.StringVal(""))
	case string:
		return hclwrite.TokensForValue(cty.StringVal(v))
	case int:
		return hclwrite.TokensForValue(cty.NumberIntVal(int64(v)))
	case int32:
		return hclwrite.TokensForValue(cty.NumberIntVal(int64(v)))
	case int64:
		return hclwrite.TokensForValue(cty.NumberIntVal(v))
	case float32:
		return hclwrite.TokensForValue(cty.NumberFloatVal(float64(v)))
	case float64:
		return hclwrite.TokensForValue(cty.NumberFloatVal(v))
	case bool:
		return hclwrite.TokensForValue(cty.BoolVal(v))
	default:
		return hclwrite.TokensForValue(cty.StringVal(fmt.Sprintf("%v", v)))
	}
}
