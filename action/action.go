package action

// Action is the action type... helpful, I know
type Action struct {
	// don't need to store the source device since this is linked
	TriggerState   string // On or Off -- or sensor value?
	TargetPlatform string
	TargetDevice   string // IP or name depending on platform
	Verb           string // per-platform specific
	Value          string // per-platform specific
}

// see runner for running actions -- circular imports suck
