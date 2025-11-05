package parser_test

import (
	"testing"

	"github.com/dragonfleas/kungfu/internal/models"
	"github.com/dragonfleas/kungfu/internal/testutil"
)

func TestParseKungfuFile_ReplaceStrategy(t *testing.T) {
	content := `patch "aws_instance" "web" {
  instance_type = "t3.large"
}`

	config, _ := testutil.WriteAndParseKungfuFile(t, content)

	if len(config.Patches) != 1 {
		t.Errorf("expected 1 patch, got %d", len(config.Patches))
	}
}

func TestParseKungfuFile_MergeStrategy(t *testing.T) {
	content := `patch "aws_instance" "web" {
  tags = merge({
    Owner = "team"
  })
}`

	config, _ := testutil.WriteAndParseKungfuFile(t, content)

	patch := config.Patches[0]
	attr := patch.Attributes["tags"]

	if attr.Strategy != models.StrategyMerge {
		t.Errorf("expected merge strategy, got %d", attr.Strategy)
	}
}

func TestParseKungfuFile_AppendStrategy(t *testing.T) {
	content := `patch "aws_instance" "web" {
  security_groups = append(["sg-123"])
}`

	config, _ := testutil.WriteAndParseKungfuFile(t, content)

	patch := config.Patches[0]
	attr := patch.Attributes["security_groups"]

	if attr.Strategy != models.StrategyAppend {
		t.Errorf("expected append strategy, got %d", attr.Strategy)
	}
}

func TestParseKungfuFile_ResourceLabels(t *testing.T) {
	content := `patch "aws_s3_bucket" "data" {
  versioning = true
}`

	config, _ := testutil.WriteAndParseKungfuFile(t, content)

	patch := config.Patches[0]

	if patch.ResourceType != "aws_s3_bucket" {
		t.Errorf("expected aws_s3_bucket, got %s", patch.ResourceType)
	}
}

func TestParseHCLFile_FindsResources(t *testing.T) {
	content := `resource "aws_instance" "web" {
  instance_type = "t3.micro"
}`

	hclFile, _ := testutil.WriteAndParseHCLFile(t, content)

	if len(hclFile.Resources) != 1 {
		t.Errorf("expected 1 resource, got %d", len(hclFile.Resources))
	}
}

func TestParseHCLFile_ResourceKey(t *testing.T) {
	content := `resource "aws_instance" "api" {
  ami = "ami-123"
}`

	hclFile, _ := testutil.WriteAndParseHCLFile(t, content)

	_, exists := hclFile.Resources["aws_instance.api"]

	if !exists {
		t.Error("expected resource key aws_instance.api to exist")
	}
}
