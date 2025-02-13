package tracker

// ProcessTracker is implemented by types that can track Docker processes
type ProcessTracker interface {
	StartDockerProcess(name string, containerID string, image string) error
}

// Global process tracker that will be set by the main application
var globalTracker ProcessTracker

// SetTracker sets the global process tracker
func SetTracker(tracker ProcessTracker) {
	globalTracker = tracker
}

// GetTracker returns the global process tracker
func GetTracker() ProcessTracker {
	return globalTracker
}
