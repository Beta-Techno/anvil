package main

import (
	"fmt"
	"os"

	"github.com/Beta-Techno/anvil-cli/pkg/config"
	"github.com/Beta-Techno/anvil-cli/pkg/persona"
	"github.com/Beta-Techno/anvil-cli/pkg/runtime"
	"github.com/Beta-Techno/anvil-cli/pkg/secrets"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "anvil",
		Short: "Anvil CLI: unlock secrets, pick a persona, run provisioning",
	}

	rootCmd.AddCommand(newUpCmd())
	rootCmd.AddCommand(newUnlockCmd())
	rootCmd.AddCommand(newDoctorCmd())

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
