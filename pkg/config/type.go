package config

type BottleStatus struct {
	Name    string
	Running bool
	// Current working directory when lsw shell was executed
	EnteredFrom string
}
