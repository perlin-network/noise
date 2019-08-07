Noise provides a package to add a port mapping using either NAT-PMP or UPnP IGDv1/IDGv2.

<br />

Add a port mapping for a `net.Listener`:
```go
import "github.com/perlin-network/noise/nat"
import "net"

// Create the listener with random port.
listener, err := net.Listen("tcp", ":0")
if err != nil {
    panic(err)
}

// Get the port from the listener
port := listener.Addr().(*net.TCPAddr).Port

// Using NAT-PMP
resolver := nat.NewPMP()

// or if you want to use UPnP
resolver := nat.NewPMP()

if err := resolver.AddMapping("tcp", uint16(port), uint16(port), 30*time.Minute); err != nil {
    panic(err)
}
```

<br />

To delete the port mapping:
```go
resolver.DeleteMapping()
```