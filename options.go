package noise

type CompressionType uint8

const (
	CompressionTypeNone CompressionType = iota
	CompressionTypeSnappy
)

type NodeOption func(n *Node)

func MaxMessageSize(maxMessageSize uint32) NodeOption {
	return func(n *Node) {
		n.opts.maxMessageSize = maxMessageSize
	}
}

func Compression(compressType CompressionType) NodeOption {
	return func(n *Node) {
		n.opts.compression = compressType
	}
}

type NodeOptions struct {
	maxMessageSize uint32
	compression    CompressionType
}

// Default, sane node options a node inherits upon instantiation.
//
// By default, nodes may receive messages that are at most 16MB in size.
var DefaultNodeOptions = NodeOptions{
	maxMessageSize: 16777216,
}
