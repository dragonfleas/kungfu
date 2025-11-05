package models

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type MergeStrategy int

const (
	StrategyReplace MergeStrategy = iota
	StrategyMerge
	StrategyAppend
)

type KungfuConfig struct {
	Patches []Patch
}

type Patch struct {
	ResourceType string
	ResourceName string
	Source       string
	Attributes   map[string]*PatchAttribute
	Body         *hclwrite.Body
	Range        hcl.Range
}

type PatchAttribute struct {
	Value    interface{}
	Strategy MergeStrategy
}

type HCLFile struct {
	Path      string
	OrigBytes []byte
	WriteFile *hclwrite.File
	Resources map[string]*Resource
	Variables map[string]*Variable
	Outputs   map[string]*Output
	Locals    map[string]*Local
	Data      map[string]*DataSource
}

type Resource struct {
	Type  string
	Name  string
	Block *hclwrite.Block
	Range hcl.Range
}

type Variable struct {
	Name  string
	Block *hclwrite.Block
	Range hcl.Range
}

type Output struct {
	Name  string
	Block *hclwrite.Block
	Range hcl.Range
}

type Local struct {
	Block *hclwrite.Block
	Range hcl.Range
}

type DataSource struct {
	Type  string
	Name  string
	Block *hclwrite.Block
	Range hcl.Range
}

func ResourceKey(resourceType, resourceName string) string {
	return resourceType + "." + resourceName
}
