package launchctl

// Launchctl is the narrow interface that the rest of launchdude uses to drive
// launchd. The real implementation shells out; the fake is in-memory for tests.
type Launchctl interface {
	Bootstrap(domain, plistPath string) error
	Bootout(serviceTarget string) error
	Kickstart(serviceTarget string, killExisting bool) error
	Stop(serviceTarget string) error
	Print(serviceTarget string) (*ServiceState, error)
	List(prefix string) ([]ListEntry, error)
}

// ServiceState is what we extract from `launchctl print`.
// Only fields we care about — far from a complete representation.
type ServiceState struct {
	Loaded       bool
	PID          int    // 0 if not running
	LastExitCode int    // 0 if never exited or last exit was clean
	PlistPath    string // absolute path on disk
	Program      string // first argv element from the plist
	State        string // "running", "waiting", "exited", etc.
}

// ListEntry is one row of `launchctl list` output.
type ListEntry struct {
	Label  string
	PID    int // 0 if not running
	Status int // last exit code; 0 if running
}
