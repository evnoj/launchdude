A nicer way to work with services on macOS.

Services on macOS are called "LaunchAgents" and managed by `launchd`. Creating them by writing XML `.plist` files and working with them via `launchctl` is not very ergonomic. `launchdude` allows you to write services in a simple TOML format, and `launchdude` creates LaunchAgents under the hood. In addition, `launchdude` provides a simple CLI to manage with LaunchAgents that were created by `launchdude`. 

Current limitations:
- limited service customizability
- only supports user agents

<details>
<summary>LLM disclosure</summary>
This project was made with heavy LLM assistance.
</details>
