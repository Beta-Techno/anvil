package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Beta-Techno/anvil-cli/pkg/config"
	"github.com/Beta-Techno/anvil-cli/pkg/gitssh"
	"github.com/Beta-Techno/anvil-cli/pkg/persona"
	"github.com/Beta-Techno/anvil-cli/pkg/runtime"
	"github.com/Beta-Techno/anvil-cli/pkg/secrets"
	"github.com/Beta-Techno/anvil-cli/pkg/update"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "anvil",
		Short: "Anvil CLI: unlock secrets, pick a persona, run provisioning",
	}

	rootCmd.AddCommand(newUpCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newUpdateCmd())
	rootCmd.AddCommand(newSecretsCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newUpCmd() *cobra.Command {
	var personaFlag, bundleURLFlag, profileFlag, tagsFlag string
	var skipBundleFlag bool
	var repoPathFlag, lockboxURLFlag, lockboxPathFlag, lockboxRefFlag string

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Unlock bundle, select persona, run Anvil provisioning",
		RunE: func(cmd *cobra.Command, args []string) error {
			// First-time setup: prompt for org if not configured
			if config.NeedsSetup() {
				org, err := promptOrg()
				if err != nil {
					return err
				}
				if err := config.SaveOrg(org); err != nil {
					return err
				}
			}

			overrides := map[string]any{
				"persona":     personaFlag,
				"bundle_url":  bundleURLFlag,
				"profile":     profileFlag,
				"tags":        tagsFlag,
				"skip_bundle": skipBundleFlag,
			}
			if repoPathFlag != "" {
				overrides["repo_path"] = repoPathFlag
			}
			if lockboxURLFlag != "" {
				overrides["lockbox_repo_url"] = lockboxURLFlag
			}
			if lockboxPathFlag != "" {
				overrides["lockbox_repo_path"] = lockboxPathFlag
			}
			if lockboxRefFlag != "" {
				overrides["lockbox_repo_ref"] = lockboxRefFlag
			}
			cfg, err := config.Load(overrides)
			if err != nil {
				return err
			}

			if err := runtime.CheckPrereqs(); err != nil {
				return err
			}

			maybeWarnOutdated()

			var bundleData *secrets.BundleData
			if !cfg.SkipBundle {
				if err := secrets.Unlock(cfg.BundleURL, cfg.BundleFile, cfg.AgeKeyFile); err != nil {
					return err
				}
			} else {
				fmt.Println("[anvil] skipping bundle unlock (per config)")
			}

			bundleData, err = secrets.LoadBundleData(cfg.BundleFile)
			if err != nil {
				return err
			}
			if bundleData != nil && bundleData.LockboxAgeKey != "" && cfg.LockboxAgeKeyFile != "" {
				if err := secrets.SaveKeyFile(cfg.LockboxAgeKeyFile, bundleData.LockboxAgeKey); err != nil {
					return err
				}
				fmt.Println("[secrets] Stored lockbox age key at", cfg.LockboxAgeKeyFile)
			}

			lockboxURL := cfg.LockboxRepoURL
			if bundleData != nil && bundleData.LockboxRepoURL != "" {
				lockboxURL = bundleData.LockboxRepoURL
			}
			lockboxRef := cfg.LockboxRepoRef
			if bundleData != nil && bundleData.LockboxRepoRef != "" {
				lockboxRef = bundleData.LockboxRepoRef
			}

			if usesSSH(lockboxURL, cfg.ManiManifests) {
				if bundleData == nil || bundleData.GitHubToken == "" {
					return fmt.Errorf("github token missing in bundle; cannot set up SSH before cloning private repos")
				}
				if err := gitssh.EnsureAccess(gitssh.Config{
					Username: bundleData.GitUsername,
					Email:    bundleData.GitEmail,
					Token:    bundleData.GitHubToken,
				}); err != nil {
					return err
				}
			}
			if lockboxURL != "" && cfg.LockboxRepoPath != "" {
				if err := runtime.EnsureRepoRef(cfg.LockboxRepoPath, lockboxURL, lockboxRef); err != nil {
					return err
				}
			}

			if err := runtime.EnsureRepo(cfg.RepoPath, cfg.RepoURL); err != nil {
				return err
			}
			if err := runtime.EnsureVarsFile(cfg.RepoPath, cfg.VarsFile); err != nil {
				return err
			}
			if err := runtime.EnsurePersonaFile(cfg.RepoPath, cfg.Persona, cfg.PersonaFile); err != nil {
				return err
			}

			if err := persona.LoadOverrides(cfg.PersonaFile); err != nil {
				return err
			}

			ansCfg := runtime.AnsibleConfig{
				RepoPath:        cfg.RepoPath,
				VarsFile:        cfg.VarsFile,
				PersonaFile:     cfg.PersonaFile,
				BundleFile:      cfg.BundleFile,
				Profile:         cfg.Profile,
				Tags:            cfg.Tags,
				Persona:         cfg.Persona,
				LockboxRepoPath: cfg.LockboxRepoPath,
				LockboxAgeKey:   cfg.LockboxAgeKeyFile,
				ManiRepoPath:    cfg.ManiRepoPath,
				ManiRepoURL:     cfg.ManiRepoURL,
				ManiBin:         cfg.ManiBin,
				ManiSyncTags:    cfg.ManiSyncTags,
				ManiRunCommands: cfg.ManiRunCommands,
				ManiManifests:   cfg.ManiManifests,
			}

			if err := runtime.RunAnsible(ansCfg); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&personaFlag, "persona", "dev", "Persona to apply (dev, server, ...)")
	cmd.Flags().StringVar(&bundleURLFlag, "bundle-url", "", "Encrypted bundle URL (derived from org if not set)")
	cmd.Flags().StringVar(&profileFlag, "profile", "devheavy", "Anvil profile to run")
	cmd.Flags().StringVar(&tagsFlag, "tags", "all", "Comma-separated tags override")
	cmd.Flags().StringVar(&repoPathFlag, "repo-path", "", "Override Anvil repo clone path")
	cmd.Flags().StringVar(&lockboxURLFlag, "lockbox-url", "", "Lockbox repo URL override")
	cmd.Flags().StringVar(&lockboxPathFlag, "lockbox-path", "", "Lockbox clone path override")
	cmd.Flags().StringVar(&lockboxRefFlag, "lockbox-ref", "", "Lockbox git ref override")
	cmd.Flags().BoolVar(&skipBundleFlag, "skip-bundle", false, "Skip bundle unlock even if missing")
	return cmd
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check prerequisites (git, curl, sops, age, ansible)",
		Run: func(cmd *cobra.Command, args []string) {
			results := runtime.RunDoctor()
			var ok []string
			var missing []string
			for _, r := range results {
				if r.Err != nil {
					missing = append(missing, fmt.Sprintf("%s (%v)", r.Name, r.Err))
				} else {
					ok = append(ok, fmt.Sprintf("%s -> %s", r.Name, r.Version))
				}
			}
			if len(ok) > 0 {
				fmt.Println("[doctor] OK:")
				for _, entry := range ok {
					fmt.Printf("  - %s\n", entry)
				}
			}
			if len(missing) > 0 {
				fmt.Println("[doctor] Missing or not executable:")
				for _, entry := range missing {
					fmt.Printf("  - %s\n", entry)
				}
			}
			if len(missing) == 0 {
				fmt.Println("[doctor] All required tools detected.")
			}
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Anvil CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version)
		},
	}
}

func newUpdateCmd() *cobra.Command {
	var targetFlag string
	var versionFlag string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Download and install the latest Anvil CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			var target string
			if targetFlag != "" {
				target = targetFlag
			} else {
				execPath, err := os.Executable()
				if err != nil {
					return err
				}
				target = execPath
			}
			target, _ = filepath.Abs(target)

			var desiredVersion, downloadURL string
			var err error
			if versionFlag != "" {
				desiredVersion = versionFlag
				downloadURL = update.BuildDownloadURL(versionFlag)
			} else {
				desiredVersion, downloadURL, err = update.LatestVersionInfo()
				if err != nil {
					return fmt.Errorf("failed to fetch latest release: %w", err)
				}
			}

			checksumURL := downloadURL + ".sha256"
			fmt.Printf("[update] downloading %s\n", downloadURL)
			if err := update.DownloadAndReplace(downloadURL, checksumURL, target); err != nil {
				return fmt.Errorf("update failed: %w", err)
			}
			fmt.Printf("[update] installed %s at %s\n", desiredVersion, target)
			return nil
		},
	}
	cmd.Flags().StringVar(&targetFlag, "target", "", "Binary path to replace (defaults to current executable)")
	cmd.Flags().StringVar(&versionFlag, "version", "", "Specific version tag to install (default latest)")
	return cmd
}

func newSecretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage cached bundle/lockbox secrets",
	}
	cmd.AddCommand(newSecretsResetCmd())
	return cmd
}

func newSecretsResetCmd() *cobra.Command {
	var force bool
	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Delete cached bundle + lockbox key so next run re-downloads them",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(nil)
			if err != nil {
				return err
			}
			bundle := cfg.BundleFile
			lockbox := cfg.LockboxAgeKeyFile
			if !force {
				if proceed := promptYesNo(fmt.Sprintf("Delete %s and %s?", bundle, lockbox)); !proceed {
					fmt.Println("[secrets] reset cancelled")
					return nil
				}
			}
			if err := secrets.Reset(bundle, lockbox); err != nil {
				return err
			}
			fmt.Println("[secrets] Removed cached bundle and lockbox key. Run 'anvil up' to download fresh secrets.")
			return nil
		},
	}
	resetCmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return resetCmd
}

func maybeWarnOutdated() {
	if os.Getenv("ANVIL_NO_UPDATE_CHECK") != "" {
		return
	}
	if version == "" || version == "dev" {
		return
	}
	latest, _, err := update.LatestVersionInfo()
	if err != nil {
		return
	}
	if update.CompareVersions(version, latest) < 0 {
		fmt.Printf("[update] New Anvil CLI version available: %s (current %s). Run `anvil update`.\n", latest, version)
	}
}

func usesSSH(lockboxURL string, manifests []config.ManifestConfig) bool {
	if strings.HasPrefix(lockboxURL, "git@") {
		return true
	}
	for _, m := range manifests {
		if strings.HasPrefix(m.RepoURL, "git@") {
			return true
		}
	}
	return false
}

func promptYesNo(message string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [yes/no]: ", message)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "yes", "y":
			return true
		case "no", "n":
			return false
		}
	}
}

func promptOrg() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("org: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	org := strings.TrimSpace(input)
	if org == "" {
		org = config.DefaultOrg
	}
	return org, nil
}
