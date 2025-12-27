package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var v = viper.New()

func init() {
	v.SetConfigName("astro")
	v.AddConfigPath("/etc/astro")
	v.AddConfigPath("$HOME/.config/astro")
	v.AddConfigPath("$HOME/.config")
	v.AddConfigPath(".")
	v.SetEnvPrefix("ANVIL")
	v.AutomaticEnv()

	v.SetDefault("persona", "dev")
	v.SetDefault("bundle_url", "https://raw.githubusercontent.com/Beta-Techno/key/main/bundles/default.sops.yaml")
	v.SetDefault("profile", "devheavy")
	v.SetDefault("tags", "all")
	v.SetDefault("skip_bundle", false)

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	v.SetDefault("repo_path", filepath.Join(home, ".local", "share", "anvil"))
	v.SetDefault("repo_url", "https://github.com/Beta-Techno/anvil.git")
	v.SetDefault("bundle_file", filepath.Join(home, ".config", "anvil", "key-bundle.yml"))
	v.SetDefault("age_key_file", filepath.Join(home, ".config", "anvil", "age.key"))

	_ = v.ReadInConfig()
}

// Config represents merged inputs.
type Config struct {
	Persona     string
	BundleURL   string
	Profile     string
	Tags        string
	SkipBundle  bool
	RepoPath    string
	RepoURL     string
	VarsFile    string
	PersonaFile string
	BundleFile  string
	AgeKeyFile  string
}

// Load returns the merged configuration.
func Load(overrides map[string]any) (*Config, error) {
	for k, val := range overrides {
		v.Set(k, val)
	}

	cfg := &Config{
		Persona:    v.GetString("persona"),
		BundleURL:  v.GetString("bundle_url"),
		Profile:    v.GetString("profile"),
		Tags:       v.GetString("tags"),
		SkipBundle: v.GetBool("skip_bundle"),
		RepoPath:   expandPath(v.GetString("repo_path")),
		RepoURL:    v.GetString("repo_url"),
		BundleFile: expandPath(v.GetString("bundle_file")),
		AgeKeyFile: expandPath(v.GetString("age_key_file")),
	}
	if cfg.RepoPath == "" {
		cfg.RepoPath = "."
	}

	varsFile := v.GetString("vars_file")
	if varsFile == "" {
		varsFile = filepath.Join("vars", "all.yml")
	}
	cfg.VarsFile = makeRepoPath(cfg.RepoPath, varsFile)

	personaFile := v.GetString("persona_file")
	if personaFile == "" {
		personaFile = filepath.Join("vars", "personas", cfg.Persona+".yml")
	}
	cfg.PersonaFile = makeRepoPath(cfg.RepoPath, personaFile)

	return cfg, nil
}

func makeRepoPath(repoPath, candidate string) string {
	candidate = expandPath(candidate)
	if filepath.IsAbs(candidate) {
		return candidate
	}
	return filepath.Join(repoPath, candidate)
}

func expandPath(p string) string {
	if p == "" {
		return p
	}
	if len(p) >= 2 && p[:2] == "~/" {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Clean(p)
}
