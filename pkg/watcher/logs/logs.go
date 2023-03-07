package logs

import (
	"github.com/phil-inc/admiral/pkg/state"
	v1 "k8s.io/api/core/v1"
)

type logs struct {
	state *state.SharedMutable
}

func New(state *state.SharedMutable) *logs {
	return &logs{
		state: state,
	}
}

func Add(obj interface{}) {
	pod := obj.(*v1.Pod)
}

func Update(obj interface{}) {
	pod := obj.(*v1.Pod)
}

func Delete(obj interface{}) {
	pod := obj.(*v1.Pod)
}
