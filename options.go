package yaml

import (
	"errors"
	"io"
)

// config holds configuration options for YAML processing.
//
// It allows customization of various aspects of YAML parsing and serialization.
type config struct {
	// indent controls the number of spaces to use for indentation
	indent *int
	// knownFields enables strict field checking during unmarshalling
	knownFields *bool

	writer *io.Writer

	// compactSequenceIndent determines if sequence indicators are part of indentation
	compactSequenceIndent *bool
}

func newConfig(opts ...Option) (*config, error) {
	c := &config{}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

type Option func(*config) error

// WithIndent sets the number of spaces to use for indentation when
// marshaling YAML content.
//
// If not set, the default indentation level will be used.
func WithIndent(indent int) Option {
	return func(c *config) error {
		if indent < 0 {
			return errors.New("yaml: cannot indent to a negative number of spaces")
		}

		c.indent = &indent
		return nil
	}
}

// WithCompactSequenceIndent configures whether the sequence indicator '- ' is
// considered part of the indentation when marshaling YAML content.
//
// If compact is true, '- ' is treated as part of the indentation.
// If compact is false, '- ' is not treated as part of the indentation.
func WithCompactSequenceIndent(compact bool) Option {
	return func(c *config) error {
		c.compactSequenceIndent = &compact
		return nil
	}
}

// WithKnownFields enables or disables strict field checking during YAML unmarshalling.
//
// When enabled, unmarshalling will return an error if the YAML input contains fields
// that do not correspond to any fields in the target struct.
func WithKnownFields(knownFields bool) Option {
	return func(c *config) error {
		c.knownFields = &knownFields
		return nil
	}
}

// withWriter sets the output writer for YAML serialization.
func withWriter(w io.Writer) Option {
	return func(c *config) error {
		c.writer = &w
		return nil
	}
}
