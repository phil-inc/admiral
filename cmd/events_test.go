package main

import (
	"bytes"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewEventsCmd(t *testing.T) {
	success := "Can not get kube config: stat /root/.kube/config: no such file or directory"
	help := "\n\tOpen a watcher that filters events and sends their text to a remote backend.\n\nUsage:\n  events [flags]\n\nFlags:\n  -h, --help   help for events\n"
	failure := "Usage:\n  events [flags]\n\nFlags:\n  -h, --help   help for events\n\n"

	home, err := os.UserHomeDir()
	if err != nil {
		t.Errorf("Failed getting home dir: %s", err)
	}
	mockConfigPath := path.Join(home, ".admiral.yaml")
	_, err = os.Create(mockConfigPath)
	if err != nil {
		t.Errorf("Failed creating mock config: %s", err)
	}
	defer os.Remove(mockConfigPath)

	var tcs = []struct {
		name     string
		args     []string
		succeeds bool
		output   string
	}{
		{"should succeed with no args", []string{}, true, success},
		{"should succeed with -h arg", []string{"-h"}, true, help},
		{"should succeed with --help arg", []string{"--help"}, true, help},
		{"should fail with non-help arg", []string{"test"}, false, failure},
	}

	for _, tt := range tcs {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cmd := NewEventsCmd()
			cmd.SetArgs(tt.args)
			cmd.SetOut(buf)
			err := cmd.Execute()

			if err != nil && tt.succeeds {
				assert.Equal(t, tt.output, err.Error())
			} else {
				assert.Equal(t, tt.output, buf.String())
			}
		})
	}
}

func Test_NewEventsCmdNoConfig(t *testing.T) {
	success := "open : no such file or directory"
	help := "\n\tOpen a watcher that filters events and sends their text to a remote backend.\n\nUsage:\n  events [flags]\n\nFlags:\n  -h, --help   help for events\n"
	failure := "Usage:\n  events [flags]\n\nFlags:\n  -h, --help   help for events\n\n"

	var tcs = []struct {
		name     string
		args     []string
		succeeds bool
		output   string
	}{
		{"should succeed with no args", []string{}, true, success},
		{"should succeed with -h arg", []string{"-h"}, true, help},
		{"should succeed with --help arg", []string{"--help"}, true, help},
		{"should fail with non-help arg", []string{"test"}, false, failure},
	}

	for _, tt := range tcs {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cmd := NewEventsCmd()
			cmd.SetArgs(tt.args)
			cmd.SetOut(buf)
			err := cmd.Execute()

			if err != nil && tt.succeeds {
				assert.Equal(t, tt.output, err.Error())
			} else {
				assert.Equal(t, tt.output, buf.String())
			}
		})
	}
}
