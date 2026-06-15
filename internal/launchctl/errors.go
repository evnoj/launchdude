package launchctl

import "errors"

// Typed errors mapped from launchctl's notoriously bad exit codes and stderr.
// State-machine code uses errors.Is to recognize these and react idempotently.
var (
	// ErrNotLoaded means the service target is not currently registered with launchd.
	// Returned by Print/Kickstart/Stop/Bootout when the target is unknown.
	ErrNotLoaded = errors.New("service not loaded")

	// ErrAlreadyLoaded means we tried to bootstrap a service that's already registered.
	// Different macOS versions surface this as different errnos (5, 17, 37, etc).
	ErrAlreadyLoaded = errors.New("service already loaded")

	// ErrDisabled means the service is marked disabled via `launchctl disable`.
	// Bootstrap will refuse until `launchctl enable` flips it back.
	ErrDisabled = errors.New("service marked disabled")
)
