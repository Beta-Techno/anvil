package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Beta-Techno/anvil-cli/pkg/config"
	"github.com/Beta-Techno/anvil-cli/pkg/gitssh"
	"github.com/Beta-Techno/anvil-cli/pkg/persona"
	"github.com/Beta-Techno/anvil-cli/pkg/profiles"
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
	rootCmd.AddCommand(newProfilesCmd())

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
		Short: "Unlock bundle, select profile, run Anvil provisioning",
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

			// Load initial config to get repo URL
			initialOverrides := map[string]any{
				"persona":     personaFlag,
				"bundle_url":  bundleURLFlag,
				"skip_bundle": skipBundleFlag,
			}
			if repoPathFlag != "" {
				initialOverrides["repo_path"] = repoPathFlag
			}
			if lockboxURLFlag != "" {
				initialOverrides["lockbox_repo_url"] = lockboxURLFlag
			}
			if lockboxPathFlag != "" {
				initialOverrides["lockbox_repo_path"] = lockboxPathFlag
			}
			if lockboxRefFlag != "" {
				initialOverrides["lockbox_repo_ref"] = lockboxRefFlag
			}
			cfg, err := config.Load(initialOverrides)
			if err != nil {
				return err
			}

			if err := runtime.CheckPrereqs(); err != nil {
				return err
			}

			maybeWarnOutdated()

			// Unlock secrets first (needed for SSH)
			var bundleData *secrets.BundleData
			if !cfg.SkipBundle {
				if err := secrets.Unlock(cfg.BundleURL, cfg.BundleFile, cfg.AgeKeyFile); err != nil {
					return err
				}
			}

			bundleData, err = secrets.LoadBundleData(cfg.BundleFile)
			if err != nil {
				return err
			}
			if bundleData != nil && bundleData.LockboxAgeKey != "" && cfg.LockboxAgeKeyFile != "" {
				if err := secrets.SaveKeyFile(cfg.LockboxAgeKeyFile, bundleData.LockboxAgeKey); err != nil {
					return err
				}
			}

			lockboxURL := cfg.LockboxRepoURL
			if bundleData != nil && bundleData.LockboxRepoURL != "" {
				lockboxURL = bundleData.LockboxRepoURL
			}
			lockboxRef := cfg.LockboxRepoRef
			if bundleData != nil && bundleData.LockboxRepoRef != "" {
				lockboxRef = bundleData.LockboxRepoRef
			}

			// Set up SSH if needed for private repos
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

			// Clone/update lockbox
			if lockboxURL != "" && cfg.LockboxRepoPath != "" {
				if err := runtime.EnsureRepoRef(cfg.LockboxRepoPath, lockboxURL, lockboxRef); err != nil {
					return err
				}
			}

			// Clone/update main repo (needed to discover profiles)
			if err := runtime.EnsureRepo(cfg.RepoPath, cfg.RepoURL); err != nil {
				return err
			}

			// Discover profiles from repo
			availableProfiles, err := profiles.Discover(cfg.RepoPath)
			if err != nil {
				return fmt.Errorf("failed to discover profiles: %w", err)
			}

			// Determine which profile to use
			var selectedProfile *profiles.Profile
			if cmd.Flags().Changed("profile") {
				// Profile specified via flag
				selectedProfile = profiles.FindByName(availableProfiles, profileFlag)
				if selectedProfile == nil {
					return fmt.Errorf("profile %q not found", profileFlag)
				}
			} else if tagsFlag != "all" && cmd.Flags().Changed("tags") {
				// Tags specified directly, skip profile selection
				selectedProfile = nil
			} else if config.NeedsProfileSetup() {
				// Prompt for profile
				if len(availableProfiles) == 0 {
					return fmt.Errorf("no profiles found in %s/profiles/", cfg.RepoPath)
				}
				profileName, err := promptProfileDynamic(availableProfiles)
				if err != nil {
					return err
				}
				if err := config.SaveProfile(profileName); err != nil {
					return err
				}
				selectedProfile = profiles.FindByName(availableProfiles, profileName)
			} else {
				// Use saved profile
				selectedProfile, err = profiles.Load(cfg.RepoPath, cfg.Profile)
				if err != nil {
					// Fallback to dev if saved profile doesn't exist
					selectedProfile = profiles.FindByName(availableProfiles, "dev")
				}
			}

			// Determine tags to use
			effectiveTags := tagsFlag
			if selectedProfile != nil && !cmd.Flags().Changed("tags") {
				effectiveTags = selectedProfile.TagsString()
			}

			// Set up vars and persona files
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
				Tags:            effectiveTags,
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

			// Pass profile vars to Ansible if available
			if selectedProfile != nil && len(selectedProfile.Vars) > 0 {
				ansCfg.ProfileVars = selectedProfile.Vars
			}

			if err := runtime.RunAnsible(ansCfg); err != nil {
				return err
			}
			fmt.Println()
			fmt.Println("✓ Done. Restart your terminal.")
			return nil
		},
	}

	cmd.Flags().StringVar(&personaFlag, "persona", "dev", "Persona to apply (dev, server, ...)")
	cmd.Flags().StringVar(&bundleURLFlag, "bundle-url", "", "Encrypted bundle URL (derived from org if not set)")
	cmd.Flags().StringVar(&profileFlag, "profile", "", "Profile to run (discovered from repo)")
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
			var missing []string
			for _, r := range results {
				if r.Err != nil {
					missing = append(missing, r.Name)
					fmt.Printf("  ✗ %s\n", r.Name)
				} else {
					fmt.Printf("  ✓ %s %s\n", r.Name, r.Version)
				}
			}
			fmt.Println()
			if len(missing) == 0 {
				fmt.Println("All prerequisites installed.")
			} else {
				fmt.Printf("Missing: %s\n", strings.Join(missing, ", "))
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
			fmt.Printf("Downloading %s...\n", desiredVersion)
			if err := update.DownloadAndReplace(downloadURL, checksumURL, target); err != nil {
				return fmt.Errorf("update failed: %w", err)
			}
			fmt.Printf("✓ Updated to %s\n", desiredVersion)
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
					fmt.Println("Cancelled.")
					return nil
				}
			}
			if err := secrets.Reset(bundle, lockbox); err != nil {
				return err
			}
			fmt.Println("✓ Secrets cleared. Run 'anvil up' to re-download.")
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
		fmt.Printf("  Update available: %s → %s (run `anvil update`)\n", version, latest)
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

func promptProfileDynamic(availableProfiles []profiles.Profile) (string, error) {
	reader := bufio.NewReader(os.Stdin)

	// Find default (dev or first)
	defaultIdx := 0
	for i, p := range availableProfiles {
		if p.Name == "dev" {
			defaultIdx = i
			break
		}
	}

	fmt.Println("profile:")
	for i, p := range availableProfiles {
		fmt.Printf("  %d. %s - %s\n", i+1, p.Name, p.Description)
	}
	fmt.Printf("Select [1-%d, default %d]: ", len(availableProfiles), defaultIdx+1)

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)

	// Default
	if input == "" {
		return availableProfiles[defaultIdx].Name, nil
	}

	// Accept number
	if num, err := strconv.Atoi(input); err == nil && num >= 1 && num <= len(availableProfiles) {
		return availableProfiles[num-1].Name, nil
	}

	// Accept name directly
	for _, p := range availableProfiles {
		if strings.EqualFold(input, p.Name) {
			return p.Name, nil
		}
	}

	// Invalid input, use default
	return availableProfiles[defaultIdx].Name, nil
}

func newProfilesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "profiles",
		Short: "List available profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(nil)
			if err != nil {
				return err
			}

			// Check if repo exists
			if _, err := os.Stat(cfg.RepoPath); os.IsNotExist(err) {
				fmt.Printf("Repo not cloned yet. Run 'anvil up' first or clone to %s\n", cfg.RepoPath)
				return nil
			}

			availableProfiles, err := profiles.Discover(cfg.RepoPath)
			if err != nil {
				return err
			}

			if len(availableProfiles) == 0 {
				fmt.Println("No profiles found.")
				return nil
			}

			fmt.Println("Available profiles:")
			for _, p := range availableProfiles {
				fmt.Printf("  %s - %s\n", p.Name, p.Description)
				if len(p.Tags) > 0 {
					fmt.Printf("    tags: %s\n", strings.Join(p.Tags, ", "))
				}
			}
			return nil
		},
	}
}
