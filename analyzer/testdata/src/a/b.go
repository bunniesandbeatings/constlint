package a

// Config represents configuration with some immutable fields.
type Config struct {
	// +const
	APIKey string

	Timeout int
}

// InitConfig initializes a config.
func InitConfig(apiKey string) *Config {
	return &Config{
		APIKey:  apiKey, // OK: in constructor
		Timeout: 30,
	}
}

// MakeConfig creates a new config.
func MakeConfig() Config {
	c := Config{}
	c.APIKey = "secret" // OK: in constructor
	c.Timeout = 30
	return c
}

// UpdateConfig updates a config.
func UpdateConfig(c *Config) {
	c.APIKey = "new-secret" // want "assignment to const field"
	c.Timeout = 60          // OK: Timeout is not marked as const
}
