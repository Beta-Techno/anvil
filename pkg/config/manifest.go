package config

import "github.com/mitchellh/mapstructure"

type ManifestConfig struct {
	RepoURL     string   `mapstructure:"repo_url"`
	RepoPath    string   `mapstructure:"repo_path"`
	RepoRef     string   `mapstructure:"repo_ref"`
	SyncTags    []string `mapstructure:"sync_tags"`
	RunCommands []string `mapstructure:"run_commands"`
}

func decodeManifestList(val any) []ManifestConfig {
	switch typed := val.(type) {
	case []ManifestConfig:
		return normalizeManifests(typed)
	case []map[string]any:
		out := make([]ManifestConfig, 0, len(typed))
		for _, item := range typed {
			var mc ManifestConfig
			if err := mapstructure.Decode(item, &mc); err == nil {
				out = append(out, mc)
			}
		}
		return normalizeManifests(out)
	case nil:
		return nil
	default:
		var mc ManifestConfig
		if err := mapstructure.Decode(typed, &mc); err == nil {
			return normalizeManifests([]ManifestConfig{mc})
		}
	}
	return nil
}

func normalizeManifests(list []ManifestConfig) []ManifestConfig {
	for i := range list {
		list[i].RepoPath = expandPath(list[i].RepoPath)
		if list[i].RepoRef == "" {
			list[i].RepoRef = "main"
		}
		if len(list[i].SyncTags) == 0 {
			list[i].SyncTags = []string{"infra"}
		}
	}
	return list
}

func defaultManifest(cfg *Config) ManifestConfig {
	return ManifestConfig{
		RepoURL:     cfg.ManiRepoURL,
		RepoPath:    cfg.ManiRepoPath,
		RepoRef:     "main",
		SyncTags:    cfg.ManiSyncTags,
		RunCommands: cfg.ManiRunCommands,
	}
}
