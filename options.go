package yaml

// Options holds configuration options for YAML processing
type Options struct {
	// TODO: description for this option
	indent *int
	// TODO: description for this option
	knownFields *bool
	// other parameters
	// ...
}

var (
	// Default values for options
	defaultIndent      = 4
	defaultKnownFields = false

	// defaultOptions is an Option set with all the default values
	defaultOptions = joinOptions(
		OptionIndent(defaultIndent),
		OptionKnownFields(defaultKnownFields),
	)
)

// OptionIndent returns an Options with the specified indent value
func OptionIndent(indent int) Options {
	return Options{indent: &indent}
}

// getIndent returns the Options's indent if set or the default value
func (o Options) getIndent() int {
	if o.indent != nil {
		return *o.indent
	}
	return defaultIndent
}

// OptionKnownFields returns an Options with knownFields enabled
func OptionKnownFields(v bool) Options {
	return Options{knownFields: &v}
}

// getIndent returns the Options's knownFields if set or the default value
func (o Options) getKnownFields() bool {
	if o.knownFields != nil {
		return *o.knownFields
	}
	return defaultKnownFields
}

// joinOptionsDefault combines default options with provided Options into a single Option
func joinOptionsDefault(srcs ...Options) Options {
	return joinOptions(append([]Options{defaultOptions}, srcs...)...)
}

// joinOptions combines Options ignoring unset (nil) fields
func joinOptions(srcs ...Options) Options {
	var result Options
	for _, src := range srcs {
		// Set indent parameter
		if src.indent != nil {
			result.indent = src.indent
		}
		// Set knownFields parameter
		if src.knownFields != nil {
			result.knownFields = src.knownFields
		}
		// Set other parameters
		// ...
	}
	return result
}
