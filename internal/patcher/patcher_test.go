package patcher_test

import (
	"testing"

	"github.com/dragonfleas/kungfu/internal/models"
	"github.com/dragonfleas/kungfu/internal/patcher"
	"github.com/dragonfleas/kungfu/internal/testutil"
	"github.com/zclconf/go-cty/cty"
)

func TestApplyPatches_ReplaceStrategy(t *testing.T) {
	content := `resource "aws_instance" "web" {
  instance_type = "t3.micro"
}`

	files, _ := testutil.SetupTerraformFile(t, content)

	patch := models.Patch{
		ResourceType: "aws_instance",
		ResourceName: "web",
		Attributes: map[string]*models.PatchAttribute{
			"instance_type": {
				Value:    cty.StringVal("t3.large"),
				Strategy: models.StrategyReplace,
			},
		},
	}

	_, err := patcher.ApplyPatches(files, []models.Patch{patch})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestApplyPatches_MergeStrategy(t *testing.T) {
	content := `resource "aws_instance" "web" {
  tags = {
    Name = "web"
  }
}`

	files, _ := testutil.SetupTerraformFile(t, content)

	newTags := cty.ObjectVal(map[string]cty.Value{
		"Owner": cty.StringVal("team"),
	})

	patch := models.Patch{
		ResourceType: "aws_instance",
		ResourceName: "web",
		Attributes: map[string]*models.PatchAttribute{
			"tags": {
				Value:    newTags,
				Strategy: models.StrategyMerge,
			},
		},
	}

	_, err := patcher.ApplyPatches(files, []models.Patch{patch})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestApplyPatches_AppendStrategy(t *testing.T) {
	content := `resource "aws_instance" "web" {
  security_groups = ["sg-default"]
}`

	files, _ := testutil.SetupTerraformFile(t, content)

	newSGs := cty.TupleVal([]cty.Value{
		cty.StringVal("sg-new"),
	})

	patch := models.Patch{
		ResourceType: "aws_instance",
		ResourceName: "web",
		Attributes: map[string]*models.PatchAttribute{
			"security_groups": {
				Value:    newSGs,
				Strategy: models.StrategyAppend,
			},
		},
	}

	_, err := patcher.ApplyPatches(files, []models.Patch{patch})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestApplyPatches_ResourceNotFound(t *testing.T) {
	content := `resource "aws_instance" "web" {
  instance_type = "t3.micro"
}`

	files, _ := testutil.SetupTerraformFile(t, content)

	patch := models.Patch{
		ResourceType: "aws_instance",
		ResourceName: "nonexistent",
		Attributes:   map[string]*models.PatchAttribute{},
	}

	_, err := patcher.ApplyPatches(files, []models.Patch{patch})

	if err == nil {
		t.Error("expected error for nonexistent resource")
	}
}

func TestDeepMerge_PreservesOriginalKeys(t *testing.T) {
	existing := cty.ObjectVal(map[string]cty.Value{
		"Name": cty.StringVal("web"),
		"Env":  cty.StringVal("prod"),
	})

	patchVal := cty.ObjectVal(map[string]cty.Value{
		"Owner": cty.StringVal("team"),
	})

	result := patcher.DeepMerge(existing, patchVal)
	resultVal := result.(cty.Value)
	resultMap := resultVal.AsValueMap()

	if len(resultMap) != 3 {
		t.Errorf("expected 3 keys, got %d", len(resultMap))
	}
}

func TestDeepMerge_OverridesConflictingKeys(t *testing.T) {
	existing := cty.ObjectVal(map[string]cty.Value{
		"Name": cty.StringVal("old"),
	})

	patchVal := cty.ObjectVal(map[string]cty.Value{
		"Name": cty.StringVal("new"),
	})

	result := patcher.DeepMerge(existing, patchVal)
	resultVal := result.(cty.Value)
	resultMap := resultVal.AsValueMap()

	if resultMap["Name"].AsString() != "new" {
		t.Errorf("expected 'new', got %s", resultMap["Name"].AsString())
	}
}

func TestAppendToList_PreservesOriginal(t *testing.T) {
	existing := cty.TupleVal([]cty.Value{
		cty.StringVal("one"),
	})

	patchVal := cty.TupleVal([]cty.Value{
		cty.StringVal("two"),
	})

	result := patcher.AppendToList(existing, patchVal)
	resultVal := result.(cty.Value)
	resultList := resultVal.AsValueSlice()

	if len(resultList) != 2 {
		t.Errorf("expected 2 items, got %d", len(resultList))
	}
}
