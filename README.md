# Anvil — Ubuntu Workstation/Server Bootstrap (Ansible)

One-liner bootstrap for an Ubuntu box with optional workstation extras, IDEs, and dotfiles.

## Quick start (recommended)
```bash
curl -fsSL https://raw.githubusercontent.com/Beta-Techno/anvil/main/install.sh | bash
```
Defaults: profile `devheavy`, tags `essential`, vars file `vars/all.yml` (auto-copied from `vars/all.example.yml` if missing). Override with env: `BRANCH`, `PROFILE`, `TAGS`, `VARS_FILE`, `TARGET_DIR`.

## Local run
```bash
cp vars/all.example.yml vars/all.yml   # edit toggles/checksums
./run.sh                               # uses TAGS=essential, PROFILE=devheavy, VARS_FILE=vars/all.yml
```
Override with `TAGS=... VARS_FILE=... PROFILE=... ./run.sh`.

## Tags (summary)
- **Essential** (`essential`): Core development setup - `base`, `drivers`, `docker`, `git`, `chezmoi` (dotfiles), `langs` (Node/Python/Ruby/Go/Rust/Java)
- **Individual roles**: `tailscale`, `lazyvim`, `cloudflared`, `nginx`, `flatpak_snap`, `fonts`, `terminals`, `chrome`, `zed`, `cursor`, `ghostty`, `toolbox`, `docker_desktop`, `nomachine`, `terminal_extras`, `flatpak_apps`, `snap_apps`, `cleanup`
- **Examples**:
  - `TAGS=essential` — Default; fast core dev setup (recommended)
  - `TAGS=all` — Runs everything (all 27 roles)
  - `TAGS=chrome,cursor` — Install specific apps only
- Install toggles live in `vars/all.yml` (`*_install` flags).

## Vars
- Copy `vars/all.example.yml` → `vars/all.yml` and set `*_install` flags, app lists, dotfiles repo, optional checksums (sha256:…).
- Default chezmoi repo: `https://github.com/Beta-Techno/dotfiles.git`.
- Optional integrity: set `zed_install_checksum`, `ghostty_install_checksum`, `cursor_deb_checksum`, `jetbrains_toolbox_checksum`, `docker_desktop_checksum`.

## Shell Configuration Architecture

Anvil uses a clean separation between system configs (managed by Ansible) and user configs (managed by you or chezmoi):

- **Ansible manages**: `~/.config/shell/*.sh` - Tool-specific configurations (nvm, pyenv, rbenv, PATH)
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
4. Add to `.chezmoiignore` (optional):
   ```
   .config/shell/
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
