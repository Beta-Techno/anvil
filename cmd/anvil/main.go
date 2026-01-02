package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Beta-Techno/anvil-cli/pkg/config"
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
	rootCmd.AddCommand(newUnlockCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newUpdateCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newUpCmd() *cobra.Command {
	var personaFlag, bundleURLFlag, profileFlag, tagsFlag string
	var skipBundleFlag bool

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Unlock bundle, select persona, run Anvil provisioning",
		RunE: func(cmd *cobra.Command, args []string) error {
			overrides := map[string]any{
				"persona":     personaFlag,
				"bundle_url":  bundleURLFlag,
				"profile":     profileFlag,
				"tags":        tagsFlag,
				"skip_bundle": skipBundleFlag,
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
	cmd.Flags().StringVar(&bundleURLFlag, "bundle-url", "https://raw.githubusercontent.com/Beta-Techno/key/main/bundles/default.sops.yaml", "Encrypted bundle URL")
	cmd.Flags().StringVar(&profileFlag, "profile", "devheavy", "Anvil profile to run")
	cmd.Flags().StringVar(&tagsFlag, "tags", "all", "Comma-separated tags override")
	cmd.Flags().BoolVar(&skipBundleFlag, "skip-bundle", false, "Skip bundle unlock even if missing")
	return cmd
}

func newUnlockCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unlock",
		Short: "Download + decrypt bootstrap bundle only",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("[anvil] bundle unlock placeholder")
		},
	}
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check prerequisites (sops, git, ansible, age)",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("[anvil] doctor placeholder")
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

			fmt.Printf("[update] downloading %s\n", downloadURL)
			if err := update.DownloadAndReplace(downloadURL, target); err != nil {
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
