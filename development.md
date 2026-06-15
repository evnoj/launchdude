# Development

Practical commands for working on launchdude. All commands run from the repo root unless noted.

## Tests

```sh
go test ./...                         # all packages, summary
go test -v ./...                      # verbose, lists each test
go test -race ./...                   # data race detector
go test -count=1 ./...                # bypass test cache (force re-run)

go test ./internal/service/           # one package
go test -v ./internal/service/ -run TestEnable_Idempotent   # one test
go test -v ./internal/service/ -run 'TestEnable_'           # regex match
```

Full suite runs in ~3.4s with no integration tests touching real launchctl.

## Build & run

```sh
go build -o launchdude .              # produce the binary at ./launchdude
go run . [args]                       # build and run in one step (no binary left behind)
go install .                          # install to $GOBIN (usually ~/go/bin)
```

## Dependencies

```sh
go mod tidy                           # add missing, remove unused
go mod why <module>                   # explain why a dep is required
go get -u ./...                       # upgrade all deps (use sparingly)
```

## Useful env vars for local testing

| Variable | Effect |
|---|---|
| `LAUNCHDUDE_INTERACTIVE=1` | Forces `ui.Hint()` output on even when stdout isn't a TTY (useful when piping or in test scripts). |
| `CLICOLOR_FORCE=1` | Forces lipgloss to emit ANSI color codes even when stdout isn't a TTY. Combined with `cat -v` you can verify color output. |
| `NO_COLOR=1` | Disables color output (lipgloss honors this automatically). |
| `XDG_CONFIG_HOME=/tmp/xdg` | Override the config directory so testing doesn't touch `~/.config/launchdude/`. |
| `EDITOR` / `VISUAL` | Set during testing (see editor-stub trick below). `VISUAL` takes precedence. |

## Editor-stub trick (testing the create/edit flow)

The `create` and `edit` commands open `$EDITOR`. For non-interactive testing, set `EDITOR` to a command that copies a prepared TOML over the temp file launchdude opened:

```sh
cat > /tmp/want.toml <<'EOF'
name = "demo"
exec_args = ["/bin/sh", "-c", "sleep 60"]
keep_alive = true
EOF

# Must also clear VISUAL since it takes precedence over EDITOR.
VISUAL= EDITOR="/bin/cp /tmp/want.toml" ./launchdude create
```

The "editor" runs `cp /tmp/want.toml <tmpfile>`, which is exactly what a user pressing :wq would have written. To test the recovery prompt, pipe responses to stdin:

```sh
echo "q" | VISUAL= EDITOR="/bin/cp /tmp/invalid.toml" ./launchdude create
```

## Cleaning up after manual testing

Leftover services across all storage layers:

```sh
launchctl list | grep launchdude                              # loaded services
ls ~/Library/LaunchAgents/launchdude.*.plist 2>/dev/null      # installed plists
ls ~/.config/launchdude/services/ 2>/dev/null                 # TOML configs
ls ~/Library/Logs/launchdude/ 2>/dev/null                     # log files
ls ~/.Trash/ | grep -E '\.toml$'                              # trashed configs
```

Full nuke of a single service (when the doctor flow or `launchdude delete` aren't enough):

```sh
launchctl bootout gui/$(id -u)/launchdude.NAME 2>/dev/null
rm -f ~/Library/LaunchAgents/launchdude.NAME.plist
rm -f ~/.config/launchdude/services/NAME.toml
rm -f ~/Library/Logs/launchdude/NAME.{out,err}.log
```

## launchctl quick reference (for debugging launchd interactions)

```sh
launchctl list                                       # all loaded services (TSV)
launchctl print gui/$(id -u)/launchdude.NAME         # detailed state of one service
launchctl bootstrap gui/$(id -u) PATH.plist          # load a plist
launchctl bootout gui/$(id -u)/launchdude.NAME       # unload
launchctl kickstart gui/$(id -u)/launchdude.NAME     # start
launchctl kickstart -k gui/$(id -u)/launchdude.NAME  # kill running process + start fresh
launchctl kill SIGTERM gui/$(id -u)/launchdude.NAME  # stop the running process
```

Exit codes from `launchctl` are notoriously imprecise. When something fails, the actual signal is usually in stderr or in the service's stderr log (`~/Library/Logs/launchdude/NAME.err.log`).

## Where things live

| What | Path |
|---|---|
| Source-of-truth service configs | `$XDG_CONFIG_HOME/launchdude/services/<name>.toml` (default `~/.config/...`) |
| Global launchdude config (optional) | `$XDG_CONFIG_HOME/launchdude/config.toml` |
| User-overridable default template | `$XDG_CONFIG_HOME/launchdude/template.toml` |
| Generated launchagents | `~/Library/LaunchAgents/launchdude.<name>.plist` |
| Auto-routed logs | `~/Library/Logs/launchdude/<name>.{out,err}.log` |
| Trashed configs after `delete` | `~/.Trash/<name>.toml` |

## Package layout

```
cmd/          thin Cobra command wrappers — one file per verb (or grouped in lifecycle.go)
internal/
  config/     TOML schema, validation, XDG path resolution, trash helper
  plist/      ServiceConfig <-> XML plist render & parse, canonical hash
  launchctl/  binary wrapper interface, real (exec) + fake impls, output parsers, typed errors
  service/    Manager — the state machine. All Apply / Enable / Disable / etc. logic
  ui/         lipgloss styles, status/list rendering, interactive editor + recovery prompt
```
