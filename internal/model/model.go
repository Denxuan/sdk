package model

import "time"

// Tool is one of the development runtimes managed by sdk.
type Tool string

const (
	Java   Tool = "java"
	NodeJS Tool = "nodejs"
	Maven  Tool = "maven"
	MVND   Tool = "mvnd"
	Go     Tool = "go"
)

func (t Tool) Valid() bool {
	switch t {
	case Java, NodeJS, Maven, MVND, Go:
		return true
	default:
		return false
	}
}

func Tools() []Tool { return []Tool{Java, NodeJS, Maven, MVND, Go} }

type InstalledVersion struct {
	Version     string    `json:"version"`
	Path        string    `json:"path"`
	Managed     bool      `json:"managed"`
	InstalledAt time.Time `json:"installedAt"`
}

type State struct {
	Defaults  map[Tool]string             `json:"defaults"`
	Installed map[Tool][]InstalledVersion `json:"installed"`
}
