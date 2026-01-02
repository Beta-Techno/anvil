package secrets

import (
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// BundleData represents selected fields from the decrypted bootstrap bundle.
type BundleData struct {
	LockboxAgeKey  string `yaml:"lockbox_age_key"`
	LockboxRepoURL string `yaml:"lockbox_repo_url"`
	LockboxRepoRef string `yaml:"lockbox_repo_ref"`
	GitUsername    string `yaml:"git_username"`
	GitEmail       string `yaml:"git_email"`
	GitHubToken    string `yaml:"github_token"`
}

// LoadBundleData loads bundle metadata from disk. If the file is missing it returns (nil, nil).
func LoadBundleData(path string) (*BundleData, error) {
	if path == "" {
		return nil, errors.New("bundle path empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out BundleData
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
