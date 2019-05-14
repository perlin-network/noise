package noise

type NodeOption func(n *Node) error

func MaxMessageSize(maxMessageSize uint32) NodeOption {
	return func(n *Node) error {
		n.opts.maxMessageSize = maxMessageSize

		return nil
	}
}

type NodeOptions struct {
	maxMessageSize uint32
}

// Default, sane node options a node inherits upon instantiation.
//
// By default, nodes may receive messages that are at most 16MB in size.
var DefaultNodeOptions = NodeOptions{
	maxMessageSize: 16777216,
}
