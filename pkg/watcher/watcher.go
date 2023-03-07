package watcher

type Watcher interface {
	Add(o interface{})
	Update(o interface{})
	Delete(o interface{})
}
