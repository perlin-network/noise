package noise

type Config struct {
	Host            string
	ExternalAddress	string
	Port            int
	PrivateKeyHex   string
	EnableSKademlia bool
	SKademliaC1     int
	SKademliaC2     int
}
