package actor

import (
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
)

type ActorTemplate interface {
	Receive(client protobuf.Noise_StreamClient, sender peer.ID, message interface{})
	Start()
	Stop()
}

type Actor struct {
	Props ActorTemplate

	Mailbox chan protobuf.Message
	Name    string
}

func (a *Actor) Stop() {
	a.Props.Stop()
	Registry.DeregisterActor(a.Name)

	// TODO: Send message/stop actor from processing its mailbox.
}

// Send a message to an actor.
func (a *Actor) Tell(message protobuf.Message) {
	a.Mailbox <- message
}

// Creates and registers an actor with a random name and returns its instance.
func CreateActor(template func() ActorTemplate) (*Actor, error) {
	return CreateActorNamed(Registry.nextAvailableID(), template)
}

// Creates and registers an actor with a prefixed name and returns its instance.
func CreateActorPrefixed(prefix string, template func() ActorTemplate) (*Actor, error) {
	return CreateActorNamed(prefix+"_"+Registry.nextAvailableID(), template)
}

// Creates and registers an actor with a specified name and returns its instance.
func CreateActorNamed(name string, template func() ActorTemplate) (*Actor, error) {
	// Create actor.
	actor := &Actor{
		Props:   template(),
		Name:    name,
		Mailbox: make(chan protobuf.Message),
	}

	// Register actor to processes.
	err := Registry.RegisterActor(name, actor)
	if err != nil {
		return nil, err
	}

	// TODO: Send message/start having the actor process its mailbox.

	actor.Props.Start()

	return actor, nil
}
