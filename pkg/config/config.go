package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// DefaultOrg is used when no org is configured
const DefaultOrg = "Beta-Techno"

var v = viper.New()

func init() {
	v.SetConfigName("anvil")
	v.AddConfigPath("/etc/anvil")
	v.AddConfigPath("$HOME/.config/anvil")
	v.AddConfigPath("$HOME/.config")
	v.AddConfigPath(".")
	v.SetEnvPrefix("ANVIL")
	v.AutomaticEnv()

	v.SetDefault("org", "")
	v.SetDefault("persona", "dev")
	v.SetDefault("profile", "")
	v.SetDefault("tags", "all")
	v.SetDefault("skip_bundle", false)

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	v.SetDefault("repo_path", filepath.Join(home, ".local", "share", "anvil"))
	v.SetDefault("bundle_file", filepath.Join(home, ".config", "anvil", "key-bundle.yml"))
	v.SetDefault("age_key_file", filepath.Join(home, ".config", "anvil", "age.key"))
	v.SetDefault("lockbox_repo_path", filepath.Join(home, ".local", "share", "lockbox"))
	v.SetDefault("lockbox_repo_ref", "")
	v.SetDefault("lockbox_age_key_file", filepath.Join(home, ".config", "anvil", "lockbox.age"))
	v.SetDefault("mani_repo_path", filepath.Join(home, "code", "infra", "atlas"))
	v.SetDefault("mani_bin", "/usr/local/bin/mani")
	v.SetDefault("mani_sync_tags", []string{"infra"})
	v.SetDefault("mani_run_commands", []string{})

	// URL defaults are not set here - they're derived from org in Load()

	_ = v.ReadInConfig()
}

// OrgURLs contains URLs derived from an org name
type OrgURLs struct {
	BundleURL    string
	RepoURL      string
	LockboxURL   string
	ManiURL      string
}

// DeriveURLs returns conventional URLs for a given org
func DeriveURLs(org string) OrgURLs {
	return OrgURLs{
		BundleURL:  fmt.Sprintf("https://raw.githubusercontent.com/%s/key/main/bundles/default.sops.yaml", org),
		RepoURL:    fmt.Sprintf("https://github.com/%s/anvil.git", org),
		LockboxURL: fmt.Sprintf("git@github.com:%s/lock.git", org),
		ManiURL:    fmt.Sprintf("git@github.com:%s/atlas.git", org),
	}
}

// Config represents merged inputs.
type Config struct {
	Org               string
	Persona           string
	BundleURL         string
	Profile           string
	Tags              string
	SkipBundle        bool
	RepoPath          string
	RepoURL           string
	VarsFile          string
	PersonaFile       string
	BundleFile        string
	AgeKeyFile        string
	LockboxRepoURL    string
	LockboxRepoPath   string
	LockboxRepoRef    string
	LockboxAgeKeyFile string
	ManiRepoURL       string
	ManiRepoPath      string
	ManiBin           string
	ManiSyncTags      []string
	ManiRunCommands   []string
	ManiManifests     []ManifestConfig
}

// Load returns the merged configuration.
func Load(overrides map[string]any) (*Config, error) {
	for k, val := range overrides {
		v.Set(k, val)
	}

	// Determine org - use configured value or default
	org := v.GetString("org")
	if org == "" {
		org = DefaultOrg
	}

	// Derive URLs from org (these serve as defaults)
	urls := DeriveURLs(org)

	// Get explicit URL overrides (empty string means use derived)
	bundleURL := v.GetString("bundle_url")
	if bundleURL == "" {
		bundleURL = urls.BundleURL
	}
	repoURL := v.GetString("repo_url")
	if repoURL == "" {
		repoURL = urls.RepoURL
	}
	lockboxURL := v.GetString("lockbox_repo_url")
	if lockboxURL == "" {
		lockboxURL = urls.LockboxURL
	}
	maniURL := v.GetString("mani_repo_url")
	if maniURL == "" {
		maniURL = urls.ManiURL
	}

	cfg := &Config{
		Org:               org,
		Persona:           v.GetString("persona"),
		BundleURL:         bundleURL,
		Profile:           v.GetString("profile"),
		Tags:              v.GetString("tags"),
		SkipBundle:        v.GetBool("skip_bundle"),
		RepoPath:          expandPath(v.GetString("repo_path")),
		RepoURL:           repoURL,
		BundleFile:        expandPath(v.GetString("bundle_file")),
		AgeKeyFile:        expandPath(v.GetString("age_key_file")),
		LockboxRepoURL:    lockboxURL,
		LockboxRepoPath:   expandPath(v.GetString("lockbox_repo_path")),
		LockboxRepoRef:    v.GetString("lockbox_repo_ref"),
		LockboxAgeKeyFile: expandPath(v.GetString("lockbox_age_key_file")),
		ManiRepoURL:       maniURL,
		ManiRepoPath:      expandPath(v.GetString("mani_repo_path")),
		ManiBin:           expandPath(v.GetString("mani_bin")),
		ManiSyncTags:      toStringSlice(v.Get("mani_sync_tags")),
		ManiRunCommands:   toStringSlice(v.Get("mani_run_commands")),
		ManiManifests:     decodeManifestList(v.Get("mani_manifests")),
	}
	if cfg.RepoPath == "" {
		cfg.RepoPath = "."
	}
	if len(cfg.ManiManifests) == 0 {
		cfg.ManiManifests = []ManifestConfig{defaultManifest(cfg)}
	}
	for i := range cfg.ManiManifests {
		cfg.ManiManifests[i].RepoPath = expandPath(cfg.ManiManifests[i].RepoPath)
		if len(cfg.ManiManifests[i].SyncTags) == 0 {
			cfg.ManiManifests[i].SyncTags = cfg.ManiSyncTags
		}
		if len(cfg.ManiManifests[i].RunCommands) == 0 {
			cfg.ManiManifests[i].RunCommands = cfg.ManiRunCommands
		}
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

func toStringSlice(val any) []string {
	switch v := val.(type) {
	case []string:
		return v
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	default:
		return nil
	}
}

// NeedsSetup returns true if org is not configured (first-time setup needed)
func NeedsSetup() bool {
	return v.GetString("org") == ""
}

// NeedsProfileSetup returns true if profile has not been explicitly chosen
func NeedsProfileSetup() bool {
	return v.GetString("profile") == ""
}

// ValidProfiles returns the list of valid profile names
func ValidProfiles() []string {
	return []string{"user", "dev", "server"}
}

// ProfileDescription returns a short description for a profile
func ProfileDescription(profile string) string {
	switch profile {
	case "user":
		return "Desktop basics"
	case "dev":
		return "Developer workstation"
	case "server":
		return "Headless server"
	default:
		return ""
	}
}

// SaveProfile persists the profile to the config file
func SaveProfile(profile string) error {
	configPath := ConfigFilePath()
	configDir := filepath.Dir(configPath)

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	// Read existing config if present
	existing := make(map[string]any)
	if data, err := os.ReadFile(configPath); err == nil {
		_ = yaml.Unmarshal(data, &existing)
	}

	// Update profile
	existing["profile"] = profile

	// Write back
	data, err := yaml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Update viper in-memory
	v.Set("profile", profile)

	return nil
}

// ConfigFilePath returns the path to the config file
func ConfigFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".config", "anvil", "anvil.yml")
}

// SaveOrg persists the org to the config file
func SaveOrg(org string) error {
	configPath := ConfigFilePath()
	configDir := filepath.Dir(configPath)

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	// Read existing config if present
	existing := make(map[string]any)
	if data, err := os.ReadFile(configPath); err == nil {
		_ = yaml.Unmarshal(data, &existing)
	}

	// Update org
	existing["org"] = org

	// Write back
	data, err := yaml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Update viper in-memory
	v.Set("org", org)

	return nil
}
