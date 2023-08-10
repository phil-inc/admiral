package watcher

type Watcher interface {
	Add(o interface{})
	Update(o, n interface{})
	Delete(o interface{})
}
