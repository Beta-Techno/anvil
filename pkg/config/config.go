package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var (
	v       = viper.New()
	rootDir string
)

func init() {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	rootDir = cwd

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
	v.SetDefault("repo_path", rootDir)
	v.SetDefault("vars_file", filepath.Join(rootDir, "vars", "all.yml"))
	v.SetDefault("persona_file", filepath.Join(rootDir, "vars", "personas", "dev.yml"))
	v.SetDefault("bundle_file", filepath.Join(os.Getenv("HOME"), ".config", "anvil", "key-bundle.yml"))
	v.SetDefault("age_key_file", filepath.Join(os.Getenv("HOME"), ".config", "anvil", "age.key"))

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
		Persona:     v.GetString("persona"),
		BundleURL:   v.GetString("bundle_url"),
		Profile:     v.GetString("profile"),
		Tags:        v.GetString("tags"),
		SkipBundle:  v.GetBool("skip_bundle"),
		RepoPath:    absPath(v.GetString("repo_path")),
		VarsFile:    absPath(v.GetString("vars_file")),
		PersonaFile: absPath(v.GetString("persona_file")),
		BundleFile:  v.GetString("bundle_file"),
		AgeKeyFile:  v.GetString("age_key_file"),
	}

	if cfg.PersonaFile == "" {
		cfg.PersonaFile = filepath.Join(cfg.RepoPath, "vars", "personas", cfg.Persona+".yml")
	}

	if cfg.Persona == "" {
		return nil, fmt.Errorf("persona cannot be empty")
	}
	if _, err := os.Stat(cfg.VarsFile); err != nil {
		return nil, fmt.Errorf("vars file %s missing: %w", cfg.VarsFile, err)
	}
	if _, err := os.Stat(cfg.PersonaFile); err != nil {
		return nil, fmt.Errorf("persona vars file %s missing: %w", cfg.PersonaFile, err)
	}
	return cfg, nil
}

func absPath(p string) string {
	if p == "" {
		return p
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Clean(filepath.Join(rootDir, p))
}
