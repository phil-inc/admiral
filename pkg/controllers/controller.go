package controllers

type Controller interface {
	Watch() chan struct{}
}

type Default struct{}

func (d *Default) Watch() chan struct{} {
	return nil
}
