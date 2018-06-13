package actor

import (
	"github.com/orcaman/concurrent-map"
	"errors"
	"sync/atomic"
)

var (
	ErrAlreadyExists = errors.New("actor with name already exists")
	ErrNotFound = errors.New("actor with name does not exist")
)

var Registry = &ActorRegistry{
	processes: cmap.New(),
	actorCount: 0,
}

type ActorRegistry struct {
	processes cmap.ConcurrentMap
	actorCount uint64
}

func (registry *ActorRegistry) nextAvailableID() string {
	id := hashId(registry.actorCount)
	atomic.AddUint64(&registry.actorCount, 1)

	return id
}

func (registry *ActorRegistry) RegisterActor(name string, actor *Actor) error {
	if registry.processes.Has(name) {
		return ErrAlreadyExists
	}
	registry.processes.Set(name, actor)

	return nil
}

func (registry *ActorRegistry) DeregisterActor(name string) error {
	if !registry.processes.Has(name) {
		return ErrNotFound
	}
	registry.processes.Remove(name)

	return nil
}

func (registry *ActorRegistry) FindActor(id string) (*Actor, error) {
	actor, exists := registry.processes.Get(id)
	if !exists {
		return nil, ErrNotFound
	}

	return actor.(*Actor), nil
}