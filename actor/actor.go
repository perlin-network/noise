package actor

import (
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
)

type MessageReceiver interface {
	Receive(client protobuf.Noise_StreamClient, sender peer.ID, message interface{})
}

type ActorTemplate func() *Actor

type Actor struct {
	MessageReceiver

	Mailbox chan protobuf.Message
	Name string
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
func CreateActor(template ActorTemplate) *Actor {
	return CreateActorNamed(Registry.nextAvailableID(), template)
}

// Creates and registers an actor with a prefixed name and returns its instance.
func CreateActorPrefixed(prefix string, template ActorTemplate) *Actor {
	return CreateActorNamed(prefix + "_" + Registry.nextAvailableID(), template)
}

// Creates and registers an actor with a specified name and returns its instance.
func CreateActorNamed(name string, template ActorTemplate) *Actor {
	// Create actor.
	actor := template()
	actor.Name = name
	actor.Mailbox = make(chan protobuf.Message)

	// Register actor to processes.
	err := Registry.RegisterActor(name, actor)
	if err != nil {
		// Actor already exists!!
		return nil
	}

	// TODO: Send message/start having the actor process its mailbox.

	return actor
}