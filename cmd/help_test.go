package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewHelpCmd(t *testing.T) {
	success := "\nFind more information at https://github.com/philinc/admiral\n\nCommands:\nevents\tStream events from a cluster to a backend\nlogs\tStream logs from a cluster to a backend\n\n"
	var tcs = []struct {
		name     string
		args     []string
		succeeds bool
		output   string
	}{
		{"should succeed with no args", []string{}, true, success},
		{"should succeed with args", []string{"test"}, true, success},
	}

	for _, tt := range tcs {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cmd := NewHelpCmd()
			cmd.SetArgs(tt.args)
			cmd.SetOut(buf)
			cmd.Execute()

			assert.Equal(t, tt.output, buf.String())
		})
	}
}
