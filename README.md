# Anvil — Ubuntu Workstation/Server Bootstrap (Ansible)

One-liner bootstrap for an Ubuntu box with optional workstation extras, IDEs, and dotfiles.

## Quick start (recommended)
```bash
curl -fsSL https://raw.githubusercontent.com/Beta-Techno/anvil/main/install.sh | bash
```
Defaults: profile `devheavy`, tags `all`, vars file `vars/all.yml` (auto-copied from `vars/all.example.yml` if missing). Override with env: `BRANCH`, `PROFILE`, `TAGS`, `VARS_FILE`, `TARGET_DIR`.

## Local run
```bash
cp vars/all.example.yml vars/all.yml   # edit toggles/checksums
./run.sh                               # uses TAGS=all, PROFILE=devheavy, PERSONA=dev
```
Override with `TAGS=... VARS_FILE=... PROFILE=... ./run.sh`.

### Personas (dev vs server)

Anvil now ships with persona presets that toggle opinionated defaults:

- `dev` (default) — full workstation experience with GUIs/languages, but local-only: telemetry agents and remote trackers stay disabled.
- `server` — same base stack plus observability/security agents that report into your fleet tooling.

Pick the persona via `PERSONA` when calling `run.sh`/`bootstrap.sh`:

```bash
PERSONA=server ./run.sh           # renders vars/all.yml + vars/personas/server.yml
PERSONA=dev PROFILE=devheavy ./run.sh --tags essential
```

Persona overrides live under `vars/personas/*.yml`. Drop in your own file and point `PERSONA_FILE` to it if you need custom behavior per org or host type.

### Bootstrap secrets bundle (optional)

`run.sh` looks for a decrypted bundle at `~/.config/anvil/key-bundle.yml`. If it’s missing (and `KEY_BUNDLE_FETCH` isn’t set to `skip`), it automatically runs [`scripts/unlock-key-bundle.sh`](scripts/unlock-key-bundle.sh), which:

- Ensures `sops` is installed
- Prompts for your age secret key (unless `SOPS_AGE_KEY` / `SOPS_AGE_KEY_FILE` already exist)
- Downloads the default encrypted bundle from [`Beta-Techno/key`](../key) (override via `KEY_BUNDLE_URL`)
- Decrypts it to `~/.config/anvil/key-bundle.yml` and stores the age key at `~/.config/anvil/age.key`

If the file exists, `run.sh` automatically includes it (`-e @~/.config/anvil/key-bundle.yml`) so Git credentials, tokens, lockbox age keys, etc. are available to Ansible without manual edits. To bypass the helper entirely (e.g., on air-gapped hosts), set `KEY_BUNDLE_FETCH=skip`.

## Tags (summary)
- **Essential** (`essential`): Core development setup - `base`, `drivers`, `docker`, `git`, `chezmoi` (dotfiles), `langs` (Node/Python/Ruby/Go/Rust/Java)
- **Individual roles**: `tailscale`, `lazyvim`, `cloudflared`, `nginx`, `flatpak_snap`, `fonts`, `terminals`, `chrome`, `zed`, `cursor`, `ghostty`, `toolbox`, `docker_desktop`, `nomachine`, `terminal_extras`, `flatpak_apps`, `snap_apps`, `cleanup`
- **Examples**:
  - `TAGS=all` — Default; runs everything (all 27 roles)
  - `TAGS=essential` — Fast core dev setup only (base, docker, git, chezmoi, langs)
  - `TAGS=chrome,cursor` — Install specific apps only
- Install toggles live in `vars/all.yml` (`*_install` flags).

## Vars
- Copy `vars/all.example.yml` → `vars/all.yml` and set `*_install` flags, app lists, dotfiles repo, optional checksums (sha256:…).
- Default chezmoi repo: `https://github.com/Beta-Techno/dotfiles.git`.
- Optional integrity: set `zed_install_checksum`, `ghostty_install_checksum`, `cursor_deb_checksum`, `jetbrains_toolbox_checksum`, `docker_desktop_checksum`.

## Shell Configuration Architecture

Anvil uses a clean separation between system configs (managed by Ansible) and user configs (managed by you or chezmoi):

- **Ansible manages**:
  - `~/.config/shell/*.sh` - Tool-specific configurations (nvm, pyenv, rbenv, PATH)
  - `~/.ssh/config` - SSH configuration for GitHub and other hosts
  - `~/.gitconfig` - Git global configuration
- **You manage**: `~/.bashrc`, `~/.zshrc` - Your personal dotfiles and customizations

Your `.bashrc`/`.zshrc` gets one minimal block that sources all configs from `~/.config/shell/`:
```bash
# Source all shell configurations from ~/.config/shell/
if [ -d "$HOME/.config/shell" ]; then
  for config in "$HOME/.config/shell"/*.sh; do
    [ -r "$config" ] && . "$config"
  done
fi
```

**Benefits:**
- ✅ No conflicts with dotfile managers (chezmoi, yadm, etc.)
- ✅ Your `.bashrc` stays clean and customizable
- ✅ Tool configs are modular and isolated
- ✅ Adding/removing tools doesn't modify your `.bashrc`

## Working with chezmoi

Anvil includes curated dotfiles from [Beta-Techno/dotfiles](https://github.com/Beta-Techno/dotfiles) by default. The execution order is designed to ensure compatibility:

**Execution order:**
1. Infrastructure roles (base, drivers, docker, git_ssh)
2. **chezmoi** - Applies dotfiles (includes the source loop for ~/.config/shell/)
3. **langs** - Creates tool configs in ~/.config/shell/ (nvm.sh, pyenv.sh, rbenv.sh, path.sh)
4. Everything else

This ensures your `.bashrc` has the source loop **before** tool configs are created.

**If you manage your own dotfiles:**
1. Fork [Beta-Techno/dotfiles](https://github.com/Beta-Techno/dotfiles) or create your own repo
2. Include the source block in your `.bashrc` (see above)
3. Update `chezmoi_dotfiles_repo` in `vars/all.yml`
4. Add to `.chezmoiignore` to prevent conflicts:
   ```
   .config/shell/    # Tool configs managed by Ansible langs role
   .ssh/config       # SSH config managed by Ansible git_ssh role
   ```

## Notes / constraints
- Target: Ubuntu with sudo + network.
- Docker Desktop needs `/dev/kvm` (not LXC); NoMachine is amd64-only. Disable those installs if not applicable.
- GUI/IDE installs assume a desktop environment; fine to skip via flags.

## What’s installed (when enabled)
- Core: updates, base packages, Docker Engine, Tailscale, Git/SSH setup, language runtimes (Node via nvm, Python via pyenv, Ruby via rbenv, Go, Rust, Java), LazyVim, fonts, flatpak/snap, nginx, cloudflared, Alacritty (apt or source)
- Extras: Chrome, Zed, Cursor, Ghostty, JetBrains Toolbox, Docker Desktop, NoMachine, terminal extras (tilix, zsh, oh-my-zsh), Flatpak/Snap apps, chezmoi dotfiles, cleanup

## Repo structure (high level)
- `bootstrap.sh` — entrypoint; installs Ansible if needed, runs `playbook.yml`.
- `run.sh` — local runner: sets `TAGS`/`PROFILE`/`VARS_FILE` (defaults: essential/devheavy/vars/all.yml) then calls `bootstrap.sh`.
- `install.sh` — curlable installer: installs git if needed, clones repo, copies vars template, calls `run.sh`.
- `vars/all.example.yml` — sample vars file with all toggles/checksums/app lists/dotfiles repo.
- `playbook.yml` — applies roles on localhost with tags.
- `roles/` — individual roles for base, drivers, docker, etc.
- `group_vars/all.yml` — shared defaults (apt codename/arch, noninteractive).
