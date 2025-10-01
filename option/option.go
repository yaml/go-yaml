package option

// Config holds configuration options for YAML processing
type Config struct {
	indent      *int
	knownFields *bool
}

const (
	defaultIndent      = 4
	defaultKnownFields = false
)

// Option represents a functional option for configuring YAML processing
type Option func(*Config)

// WithIndent returns an Option that sets the indent value
func WithIndent(indent int) Option {
	return func(c *Config) {
		c.indent = &indent
	}
}

// WithKnownFields returns an Option that enables/disables knownFields validation
func WithKnownFields(enable bool) Option {
	return func(c *Config) {
		c.knownFields = &enable
	}
}

// GetIndent returns the Config's indent if set or the default value
func (c *Config) GetIndent() int {
	if c.indent != nil {
		return *c.indent
	}
	return defaultIndent
}

// GetKnownFields returns the Config's knownFields if set or the default value
func (c *Config) GetKnownFields() bool {
	if c.knownFields != nil {
		return *c.knownFields
	}
	return defaultKnownFields
}

// NewConfig creates a new Config with the provided options
func NewConfig(opts ...Option) *Config {
	cfg := &Config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// Apply applies additional options to an existing Config
func (c *Config) Apply(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}
