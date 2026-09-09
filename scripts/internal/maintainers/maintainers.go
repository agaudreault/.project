// Package maintainers loads and models .project/maintainers.yaml, the
// authoritative source for Argo maintainer membership and groups.
package maintainers

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// File is the top-level maintainers.yaml document.
type File struct {
	Maintainers []Project `yaml:"maintainers"`
}

// Project is a single project_id entry (Argo has exactly one: "argo").
type Project struct {
	ProjectID string `yaml:"project_id"`
	Org       string `yaml:"org"`
	Teams     []Team `yaml:"teams"`
}

// Team is a named group of GitHub handles.
type Team struct {
	Name string `yaml:"name"`
	// Managed defaults to true when omitted. A managed team is reconciled by
	// the automation and provisioned to CNCF resources; managed: false (e.g.
	// emeritus) is tracked only.
	Managed *bool    `yaml:"managed"`
	Members []string `yaml:"members"`
}

// IsManaged reports whether the team is reconciled by automation.
func (t Team) IsManaged() bool { return t.Managed == nil || *t.Managed }

// Load reads and parses maintainers.yaml from path.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &f, nil
}

// Project returns the single project entry, erroring if the file does not
// contain exactly one.
func (f *File) Project() (*Project, error) {
	if len(f.Maintainers) != 1 {
		return nil, fmt.Errorf("expected exactly 1 maintainers entry, got %d", len(f.Maintainers))
	}
	return &f.Maintainers[0], nil
}
