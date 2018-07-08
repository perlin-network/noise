# Benchmark Example

Receiver:

```
go run receiver/main.go


# pprof
go tool pprof local http://localhost:6060/debug/pprof/profile
> top30 -cum
```

Sender:

```
go run sender/main.go

# pprof
go tool pprof local http://localhost:7070/debug/pprof/profile
> top30 -cum
```