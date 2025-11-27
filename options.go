package yaml

import (
	"errors"
)

// EncoderOption allows configuring the YAML encoder and marshaler.
type EncoderOption func(*encoder) error

// WithIndent sets the number of spaces to use for indentation when
// marshaling YAML content.
//
// A negative indent value will result in an error.
// 0 can be used to reset the default indentation level.
func WithIndent(indent int) EncoderOption {
	return func(e *encoder) error {
		if indent < 0 {
			return errors.New("yaml: cannot indent to a negative number of spaces")
		}

		e.indent = indent
		return nil
	}
}

// WithCompactSequenceIndent configures whether the sequence indicator '- ' is
// considered part of the indentation when marshaling YAML content.
//
// If compact is true, '- ' is treated as part of the indentation.
// If compact is false, '- ' is not treated as part of the indentation.
func WithCompactSequenceIndent(compact bool) EncoderOption {
	return func(e *encoder) error {
		e.emitter.CompactSequenceIndent = compact
		return nil
	}
}

// DecoderOption allows configuring the YAML decoder and unmarshaler.
type DecoderOption func(*decoder) error

// WithKnownFields enables or disables strict field checking during YAML unmarshalling.
//
// When enabled, unmarshalling will return an error if the YAML input contains fields
// that do not correspond to any fields in the target struct.
func WithKnownFields(knownFields bool) DecoderOption {
	return func(d *decoder) error {
		d.knownFields = knownFields
		return nil
	}
}
