package main

import (
	"bytes"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewLogsCmd(t *testing.T) {
	success := "Can not get kube config: stat /root/.kube/config: no such file or directory"
	help := "\nOpen a watcher that filters pods and sends their logs to a remote logstore.\n\nUsage:\n  logs [flags]\n\nFlags:\n  -h, --help   help for logs\n"
	failure := "Usage:\n  logs [flags]\n\nFlags:\n  -h, --help   help for logs\n\n"

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
			cmd := NewLogsCmd()
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

func Test_NewLogsCmdNoConfig(t *testing.T) {
	success := "open : no such file or directory"
	help := "\nOpen a watcher that filters pods and sends their logs to a remote logstore.\n\nUsage:\n  logs [flags]\n\nFlags:\n  -h, --help   help for logs\n"
	failure := "Usage:\n  logs [flags]\n\nFlags:\n  -h, --help   help for logs\n\n"

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
			cmd := NewLogsCmd()
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
