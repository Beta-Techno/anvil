# Legacy Ubuntu Bootstrap (Ansible)

Run locally to get a repeatable “blank canvas” for Docker workloads and optional workstation extras.

## Quick start
```bash
./bootstrap.sh            # server profile (base + docker + tailscale + git)
./bootstrap.sh minimal    # base + docker only
./bootstrap.sh workstation # server + terminals/fonts/flatpak_snap
ANSIBLE_TAGS="base,docker" ./bootstrap.sh   # explicit tags override profile
```

Env knobs:
- `ANSIBLE_TAGS` – comma list of tags; overrides profile (e.g., `base,docker,tailscale`)
- `ANSIBLE_EXTRA_VARS` – extra vars string (e.g., `git_user_email=you@example.com`)
- `ANSIBLE_ARGS` – extra ansible-playbook flags (e.g., `-vv`)

## Tags (currently wired)
- `base` – core packages and updates
- `drivers` – best-effort driver install (skips containers)
- `docker` – Docker Engine repo + packages + group membership
- `tailscale` – repo + package (does not auto-login)
- `git` – git config + SSH key + GitHub host stanza
- `langs` – language stacks: nvm+Node LTS, pyenv+Python, rbenv+Ruby, Go, Rust, Java
- `lazyvim` – Neovim AppImage + LazyVim starter config (Node provider via nvm, pynvim via pipx)
- `cloudflared` – Cloudflare tunnel daemon (apt-managed)
- `nginx` – base nginx install + service enable
- `flatpak_snap` – flatpak + Flathub, snapd + snap socket
- `fonts` – JetBrainsMono Nerd font to ~/.local/share/fonts
- `terminals` – Alacritty via apt (source build optional flag)
- `chrome` – Google Chrome via apt repo (disabled by default)
- `zed` – Zed IDE install script (disabled by default)
- `cursor` – Cursor IDE .deb scraper/installer (disabled by default)
- `ghostty` – Ghostty terminal install script (disabled by default)
- `toolbox` – JetBrains Toolbox (disabled by default)
- `docker_desktop` – Docker Desktop .deb (requires /dev/kvm; disabled by default)
- `nomachine` – NoMachine .deb install (disabled by default)
- `terminal_extras` – Tilix, zsh, oh-my-zsh, gnome tweaks, fd/bat aliases (disabled by default)
- `flatpak_apps` – install custom Flatpak apps list (disabled by default)
- `snap_apps` – install snap packages from a list (disabled by default)
- `chezmoi_install` – install chezmoi to ~/.local/bin (disabled by default)
- `chezmoi_apply` – init/update/apply dotfiles repo (disabled by default)
- `cleanup` – apt autoremove/temp cleanup (enabled by default)

Profiles mapped in `bootstrap.sh`:
- `server` (default): base,docker,tailscale,git
- `minimal`: base,docker
- `workstation`: base,docker,tailscale,git,terminals,fonts,flatpak_snap
- `devheavy`: base,docker,tailscale,git,langs,lazyvim,terminals,fonts,flatpak_snap

Optional extras are available via tags (e.g., `ANSIBLE_TAGS="chrome,zed,cursor,ghostty,toolbox,terminal_extras,docker_desktop,nomachine,flatpak_apps,cleanup"`). Most extras default to off; set corresponding `*_install` vars to true if you include the tag.

Installer integrity: many roles accept optional checksum/URL overrides (e.g., `zed_install_checksum`, `ghostty_install_checksum`, `cursor_deb_checksum`, `jetbrains_toolbox_checksum`, `docker_desktop_checksum`) if you want to pin downloads. Leave blank to skip verification.

Chezmoi usage example:
- Install chezmoi: `ANSIBLE_TAGS="chezmoi_install" ANSIBLE_EXTRA_VARS="chezmoi_install=true" ./bootstrap.sh workstation`
- Apply dotfiles: `ANSIBLE_TAGS="chezmoi_apply" ANSIBLE_EXTRA_VARS="chezmoi_apply=true chezmoi_repo=https://github.com/you/dotfiles.git" ./bootstrap.sh workstation`

## Notes
- Install runs on localhost with sudo; keep your password handy for privilege prompts.
- Re-run safe: roles are idempotent; use tags to skip GUI/tooling on servers.
- Git identity: set `GIT_USER_NAME`/`GIT_USER_EMAIL` env vars or pass via `ANSIBLE_EXTRA_VARS`.
