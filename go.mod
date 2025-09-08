module go.yaml.in/yaml/v3

go 1.16

require gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405

// these tags come from gopkg.in/yaml.v3
// they cannot be installed from go.yaml.in/yaml/v3 as it doesn't match
// so they are invalid and are retracted.
retract [v3.0.0, v3.0.1] // v3.0.2 is the first one with go.yaml.in/yaml/v3 module.
