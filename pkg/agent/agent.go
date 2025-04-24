package agent

// Agent interface defines the behavior of AI agents in the system
type Agent interface {
	// Process takes input and generates a response
	Process(input string) (string, error)

	// Close releases any resources held by the agent
	Close() error
}
