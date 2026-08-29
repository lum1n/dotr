package tui

const helpText = `dotr — config cockpit

Navigation
  j/k · ↑/↓       move
  g / G           top / bottom
  tab             focus list ↔ preview
  ctrl+d/u        half page
  / · ctrl+f      search (fuzzy; live results; ↑↓/ctrl+n/p move)
  enter           keep search · esc clear
  r               rescan
  mouse           click list / wheel (live reload when watch: true)
  q               quit

Files
  e               edit in $EDITOR
  n               new file (then $EDITOR)
  l               create symlink (path, then target)
  t               retarget selected symlink
  D               delete symlink only (keeps target; confirms)
  y               yank path (clipboard + paste buffer)
  Y               yank file contents (confirms if secret)
  p               paste/duplicate yanked file
  b               backup snapshot (confirms if secret)
  R               restore from backups

Ignore & config
  i               ignore selection (smart dir for nested paths)
  I               ignore whole app
  x               manage ignore list
  ,               edit ~/.config/dotr/config.yaml

Stow
  s               stow packages (link / unlink / restow)
  enter / l       link (stow) selected package
  u               unlink (unstow, confirms)
  R               restow

List marks
  · M A D ?       git status (when git_status: true)
  S               owned by a stow package
  ↗               symlink
  🔒              secret-looking path
  ⚠cap            scan hit 1500-file soft limit

CLI
  dotr list [q]   dotr edit <q>   dotr backup <q>
  dotr restore <q>   dotr config   dotr ignore [--list|--add]
  dotr link <path> <target>   dotr retarget <path> <target>
  dotr unlink <path>
  dotr stow list | link | unlink | restow [pkg…]
  dotr doctor   dotr version

Help
  ?               this screen

Config: ~/.config/dotr/config.yaml
Ignores: ~/.config/dotr/ignore
Backups: ~/.local/share/dotr/backups
Stow:    ~/.stowrc or stow_dir / stow_target in config
`
