A nicer way to work with services on macOS.

> "service" here means a program that runs in the background. Services should often be run at startup and kept alive as long as the computer is running.

Services on macOS are called "launch agents" and managed by `launchd` and its CLI, `launchctl`. The native way of managing these services is by writing XML `.plist` files and working with them via `launchctl`. Unfortunately, the XML format and and the `launchctl` CLI are not very easy to use.

`launchdude` allows you to write services in a simple TOML format, and it creates a launch agent `.plist` based on the TOML. In addition, it provides a simple CLI to manage launch agents that were created by `launchdude`. 

see `launchdude --help`:
```
launchdude is a friendly interface to launchctl for managing user-level
launch agents (~/Library/LaunchAgents). Service configs live as TOML in
$XDG_CONFIG_HOME/launchdude/services/ and are rendered to plists on demand.

Usage:
  launchdude [command]

Available Commands:
  apply       Re-render the plist from the TOML config and reload if loaded
  completion  Generate the autocompletion script for the specified shell
  create      Create a new service config and open it in $EDITOR
  delete      Stop, deregister, and remove the service
  disable     Stop and deregister a service (TOML config preserved)
  doctor      Interactively reconcile drift between TOML configs and launchagents
  edit        Open the service's TOML config in $EDITOR
  enable      Register the service with launchd and start it
  help        Help about any command
  import      Adopt an existing launchdude.NAME.plist into a TOML config
  list        List all launchdude services and their state
  logs        Tail the stdout and stderr logs of a service
  restart     Re-apply the config and restart the service with a fresh process
  show        Show a service's state and configured properties
  start       Start a loaded service (no-op if already running)
  status      Show the current state of a service
  stop        Stop a running service (no-op if already stopped)

Flags:
  -h, --help       help for launchdude
      --no-color   disable colored output

Use "launchdude [command] --help" for more information about a command.
```

Current limitations:
- limited service customizability
- only supports user agents

# FAQ
## Will launchdude interfere with my existing launch agents?
No. `launchdude` only manages services that are namespaced via a "launchdude" prefix, meaning the `.plist` file is prefixed with `launchdude.` and the launch agent label is prefixed with `launchdude.`

<details>
<summary>LLM disclosure</summary>
This project was made with heavy LLM assistance.
</details>
