package backend

type Backend interface {
	Stream()
	Close()
}
