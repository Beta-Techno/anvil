package profiles

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Profile represents a machine profile configuration.
type Profile struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Tags        []string          `yaml:"tags"`
	Vars        map[string]any    `yaml:"vars,omitempty"`
}

// TagsString returns the tags as a comma-separated string for --tags flag.
func (p *Profile) TagsString() string {
	return strings.Join(p.Tags, ",")
}

// Discover reads all profile YAML files from the profiles/ directory in repoPath.
func Discover(repoPath string) ([]Profile, error) {
	profilesDir := filepath.Join(repoPath, "profiles")

	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No profiles directory = no profiles
		}
		return nil, fmt.Errorf("failed to read profiles directory: %w", err)
	}

	var profiles []Profile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".yml") && !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		profilePath := filepath.Join(profilesDir, entry.Name())
		profile, err := loadProfile(profilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load profile %s: %w", entry.Name(), err)
		}
		profiles = append(profiles, *profile)
	}

	// Sort by name for consistent ordering
	sort.Slice(profiles, func(i, j int) bool {
		// Put common profiles in a specific order
		order := map[string]int{"user": 1, "dev": 2, "server": 3, "agent": 4}
		oi, oki := order[profiles[i].Name]
		oj, okj := order[profiles[j].Name]
		if oki && okj {
			return oi < oj
		}
		if oki {
			return true
		}
		if okj {
			return false
		}
		return profiles[i].Name < profiles[j].Name
	})

	return profiles, nil
}

// Load reads a specific profile by name from the profiles/ directory.
func Load(repoPath, name string) (*Profile, error) {
	profilesDir := filepath.Join(repoPath, "profiles")

	// Try .yml first, then .yaml
	for _, ext := range []string{".yml", ".yaml"} {
		profilePath := filepath.Join(profilesDir, name+ext)
		if _, err := os.Stat(profilePath); err == nil {
			return loadProfile(profilePath)
		}
	}

	return nil, fmt.Errorf("profile %q not found", name)
}

// Names returns just the profile names from a list of profiles.
func Names(profiles []Profile) []string {
	names := make([]string, len(profiles))
	for i, p := range profiles {
		names[i] = p.Name
	}
	return names
}

// FindByName finds a profile by name in a list.
func FindByName(profiles []Profile, name string) *Profile {
	for i := range profiles {
		if profiles[i].Name == name {
			return &profiles[i]
		}
	}
	return nil
}

func loadProfile(path string) (*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var profile Profile
	if err := yaml.Unmarshal(data, &profile); err != nil {
		return nil, err
	}

	// Validate required fields
	if profile.Name == "" {
		return nil, fmt.Errorf("profile missing required field: name")
	}
	if len(profile.Tags) == 0 {
		return nil, fmt.Errorf("profile %q missing required field: tags", profile.Name)
	}

	return &profile, nil
}
