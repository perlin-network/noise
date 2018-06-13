package actor

import (
	"github.com/perlin-network/noise/protobuf"
	"github.com/perlin-network/noise/peer"
)

type ActorTemplate interface {
	Receive(client protobuf.Noise_StreamClient, sender peer.ID, message interface{})
}

type Actor struct {
	ActorTemplate

	Mailbox chan protobuf.Message
	Name    string
}

func (a *Actor) Stop() {
	Registry.DeregisterActor(a.Name)

	// TODO: Send message/stop actor from processing its mailbox.
}

// Send a message to an actor.
func (a *Actor) Tell(message protobuf.Message) {
	a.Mailbox <- message
}

// Creates and registers an actor with a random name and returns its instance.
func CreateActor(template func() ActorTemplate) *Actor {
	return CreateActorNamed(Registry.nextAvailableID(), template)
}

// Creates and registers an actor with a prefixed name and returns its instance.
func CreateActorPrefixed(prefix string, template func() ActorTemplate) *Actor {
	return CreateActorNamed(prefix+"_"+Registry.nextAvailableID(), template)
}

// Creates and registers an actor with a specified name and returns its instance.
func CreateActorNamed(name string, template func() ActorTemplate) *Actor {
	// Create actor.
	actor := &Actor{
		ActorTemplate: template(),
		Name:          name,
		Mailbox:       make(chan protobuf.Message),
	}

	// Register actor to processes.
	err := Registry.RegisterActor(name, actor)
	if err != nil {
		// Actor already exists!!
		return nil
	}

	// TODO: Send message/start having the actor process its mailbox.

	return actor
}
