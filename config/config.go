package config

// Config holds global configuration for the operator
type Config struct {
	ProfilesDir string
}

// GlobalConfig is the global configuration instance
var GlobalConfig Config
