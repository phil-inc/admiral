package utils

type LogEntry struct {
	Text     string
	Metadata map[string]string
	Err      error
}
