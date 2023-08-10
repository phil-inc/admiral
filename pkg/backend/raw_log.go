package backend

type RawLog struct {
	Log       string            `json:"log"`
	Metadata  map[string]string `json:"metadata"`
	Timestamp string            `json:"timestamp"`
}
