package main

import(
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewRootCmd(t *testing.T) {
	// help is what NewRootCmd prints to stdout when a user asks for help
	help := "\n\tAdmiral: A controller for managing Kubernetes operations\n\t\n\tAdmiral is a series of controllers integrating across a Kubernetes\n\tcluster to do operations on behalf of the operator.\n\nUsage:\n  admiral [flags]\n\nFlags:\n  -h, --help   help for admiral\n"
	// usage is what NewRootCmd prints to stdout when a user uses it wrong
	usage := "Usage:\n  admiral [flags]\n\nFlags:\n  -h, --help   help for admiral\n\n"

	var tcs = []struct{
		name string
		args []string
		succeeds bool
		output string
	}{
		{"should succeed with no args", []string{}, true, ""},
		{"should succeed with a -h flag", []string{"-h"}, true, help},
		{"should succeed with a --help flag", []string{"--help"}, true, help},
		{"should fail with a non-help arg", []string{"test"}, false, usage},
	}

	for _, tt := range tcs {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cmd := NewRootCmd()
			cmd.SetArgs(tt.args)
			cmd.SetOut(buf)
			err := cmd.Execute()

			if tt.succeeds {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}

			assert.Equal(t, tt.output, buf.String())
		})
	}
}
