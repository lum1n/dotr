# dotr

Lightweight TUI + CLI for browsing and editing local configs (`$HOME` dots + `$XDG_CONFIG_HOME`), plus a thin GNU Stow layer.

## Run

```bash
go run .              # TUI
go build -o dotr .
./dotr list nvim
./dotr edit alacritty
./dotr backup .zshrc
./dotr stow list
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
| `dotr stow unlink [pkg…]` | Unstow |
| `dotr stow restow [pkg…]` | Restow |
| `dotr completion zsh` | Shell completion |

## TUI keys

| Key | Action |
|-----|--------|
| `j` / `k` | Move |
| `tab` | List ↔ preview |
| `/` | Filter |
| `n` | New file |
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

Stow mode (`s`): `enter`/`l` link, `u` unlink, `R` restow, `esc` back.

List marks: `·MAD?` git, `S` stow-owned, `↗` symlink, `🔒` secret.

With `watch: true`, saving the selected file (or changes under watched dirs) reloads preview / rescans automatically.

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

## Roadmap

0. Spike — scan, dual-pane, preview, `$EDITOR`
1. Cockpit — filter, yank/paste, backup/restore, parse badges, help
2. Sharp edges — dotr config, new-file, secret warnings, mouse
3. Optional — CLI verbs, file watchers, git status overlay
4. **Stow** — package status, stow/unstow/restow, TUI + CLI
