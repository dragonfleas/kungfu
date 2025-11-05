package models

type ModulesManifest struct {
	Modules []ModuleEntry `json:"Modules"`
}

type ModuleEntry struct {
	Key     string `json:"Key"`
	Source  string `json:"Source"`
	Version string `json:"Version,omitempty"`
	Dir     string `json:"Dir"`
}
