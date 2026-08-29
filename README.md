# dotr

Lightweight TUI + CLI for browsing and editing local configs (`$HOME` dots + `$XDG_CONFIG_HOME`), plus a thin GNU Stow layer.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/lum1n/dotr/master/install.sh | sh
```

Installs to `~/.local/bin/dotr` (override with `BINDIR=/usr/local/bin`).

`go install github.com/lum1n/dotr@latest` also works if you have Go 1.26+, but it compiles from source.

From a clone: `make install`.

Needs `$EDITOR`, and optionally GNU `stow`.

## Run

```bash
dotr                 # TUI
dotr list nvim
dotr edit alacritty
dotr backup .zshrc
dotr stow list
dotr doctor
dotr version
```

## CLI

| Command | Action |
|---------|--------|
| `dotr` | Open TUI |
| `dotr list [query]` | List configs (`--json`, `--git`) |
| `dotr edit <query>` | Open best match in `$EDITOR` (`--all` lists matches) |
| `dotr backup <query>` | Snapshot best match |
| `dotr restore <query>` | Restore latest snapshot |
| `dotr config` | Edit `config.yaml` |
| `dotr ignore` | Edit ignore file (`--list`, `--add PATTERN`) |
| `dotr stow` / `stow list` | List packages + link status (`--json`) |
| `dotr stow link [pkg…]` | Stow (`-n` dry-run) |
| `dotr stow unlink [pkg…]` | Unstow (prompts; `-y` skips; `-n` dry-run) |
| `dotr stow restow [pkg…]` | Restow |
| `dotr link <path> <target>` | Create symlink |
| `dotr retarget <path> <target>` | Change symlink target |
| `dotr unlink <path>` | Remove symlink only |
| `dotr doctor` | Check environment |
| `dotr version` | Print version |
| `dotr completion zsh` | Shell completion |

## TUI keys

| Key | Action |
|-----|--------|
| `j` / `k` | Move |
| `tab` | List ↔ preview |
| `/` · `ctrl+f` | Search (fuzzy, live) |
| `n` | New file |
| `l` | Create symlink |
| `t` | Retarget symlink |
| `D` | Delete symlink (keeps target) |
| `e` | Edit |
| `y` / `Y` | Yank path / contents |
| `p` | Paste duplicate |
| `b` / `R` | Backup / restore |
| `s` | Stow packages |
| `i` / `I` / `x` | Ignore / ignore app / manage |
| `,` | Edit dotr config |
| `r` | Rescan |
| `?` | Help |
| `q` | Quit |

Stow mode (`s`): `enter`/`l` link, `u` unlink (confirm), `R` restow, `esc` back.

List marks: `·MAD?` git, `S` stow-owned, `↗` symlink, `🔒` secret. Title shows `⚠cap` when the scan hit its 1500-file soft limit.

With `watch: true`, saving the selected file reloads its preview. The config list does not auto-rescan; press `r`, or it refreshes after `$EDITOR` and mutating actions (ignore, new, paste, symlink, stow).

## Config

`~/.config/dotr/config.yaml`:

```yaml
backup_keep: 20
chroma_style: dracula
confirm_secrets: true
mouse: true
watch: true
git_status: true
# stow_dir: ~/repos          # override ~/.stowrc --dir
# stow_target: ~             # override ~/.stowrc --target
# extra_ignores:
#   - "**/skills/"
```

| What | Where |
|------|--------|
| Config | `~/.config/dotr/config.yaml` |
| Ignores | `~/.config/dotr/ignore` |
| Backups | `$XDG_DATA_HOME/dotr/backups` |
| Stow | `~/.stowrc` (`--dir`, `--target`) |

## Stow

dotr shells out to GNU `stow`. Package status walks each package and checks the target tree (including directory folding when e.g. `~/.config` already exists).

If `~/.stowrc` has `--dir=…/dotfiles` but links were created as package `dotfiles` under the parent (`stow -d ~/repos -t ~ dotfiles`), dotr detects that and uses the parent dir + package name automatically. Override with `stow_dir` / `stow_target` when needed.

## License

MIT
