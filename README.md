# systemd

[![Documentation](https://godoc.org/github.com/marselester/systemd?status.svg)](https://pkg.go.dev/github.com/marselester/systemd)
[![Go Report Card](https://goreportcard.com/badge/github.com/marselester/systemd)](https://goreportcard.com/report/github.com/marselester/systemd)

This package provides an access to systemd via D-Bus
to list services with a low overhead for a caller.
If you find the API too limiting or missing some of functionality,
perhaps https://github.com/coreos/go-systemd might suit you better.

```go
c, err := systemd.New()
if err != nil {
    log.Fatal(err)
}
defer c.Close()

err = c.ListUnits(systemd.IsService, func(u *systemd.Unit) {
    fmt.Printf("%s %s\n", u.Name, u.ActiveState)
})
if err != nil {
    log.Fatal(err)
}
// Output:
// dirmngr.service inactive
// dbus.service active
// snapd.session-agent.service inactive
// gpg-agent.service inactive
// pk-debconf-helper.service inactive
```

Check out [units](cmd/units/main.go) program to see how to get PIDs of services.

```sh
$ go run ./cmd/units -svc
0 dirmngr.service inactive
2375 dbus.service active
0 snapd.session-agent.service inactive
0 gpg-agent.service inactive
0 pk-debconf-helper.service inactive
```

## Testing

Run tests and linters.

```sh
$ go test -v -count=1 .
$ golangci-lint run
```

Benchmarks help to spot performance changes
and troubleshoot performance issues.
For example, you can see where and how much memory gets allocated
when a 35KB D-Bus ListUnits reply is decoded into 157 Unit structs.

```sh
$ go test -run=^$ -bench=^BenchmarkDecodeListUnits$ -benchmem -memprofile list_units.allocs
$ go tool pprof list_units.allocs
```

It is recommended to run benchmarks multiple times and check
how stable they are using [Benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) tool.

```sh
$ go test -timeout 20m -bench=. -benchmem -count=50 . | tee bench-new.txt
$ benchstat bench-new.txt
```

[The old](bench-old.txt) and new stats are compared as follows.

<details>

<summary>

```sh
$ benchstat bench-old.txt bench-new.txt
```

</summary>

```
name               old time/op    new time/op    delta
AuthExternal-2        476ns ± 3%     475ns ± 2%    ~     (p=0.979 n=48+46)
DecodeString-2       54.3ns ± 4%    54.3ns ± 3%    ~     (p=0.878 n=47+49)
EscapeBusLabel-2     43.1ns ± 2%    43.8ns ± 2%  +1.58%  (p=0.000 n=49+48)
DecodeHeader-2        340ns ± 5%     340ns ± 4%    ~     (p=0.626 n=46+48)
EncodeHeader-2        186ns ± 4%     185ns ± 1%    ~     (p=0.312 n=48+48)
EncodeHello-2         222ns ± 3%     224ns ± 2%  +0.57%  (p=0.002 n=44+46)
DecodeHello-2         123ns ± 7%     122ns ± 4%    ~     (p=0.708 n=50+49)
EncodeMainPID-2       377ns ± 3%     383ns ± 4%  +1.62%  (p=0.000 n=47+43)
DecodeMainPID-2       140ns ± 5%     140ns ± 2%    ~     (p=0.644 n=49+48)
EncodeListUnits-2     237ns ± 4%     234ns ± 2%  -1.29%  (p=0.000 n=48+41)
DecodeListUnits-2    93.3µs ± 4%    92.9µs ± 2%    ~     (p=0.500 n=48+48)

name               old alloc/op   new alloc/op   delta
AuthExternal-2        80.0B ± 0%     80.0B ± 0%    ~     (all equal)
DecodeString-2        0.00B          0.00B         ~     (all equal)
EscapeBusLabel-2      0.00B          0.00B         ~     (all equal)
DecodeHeader-2        15.0B ± 0%     15.0B ± 0%    ~     (all equal)
EncodeHeader-2        0.00B          0.00B         ~     (all equal)
EncodeHello-2         0.00B          0.00B         ~     (all equal)
DecodeHello-2         29.0B ± 0%     29.0B ± 0%    ~     (all equal)
EncodeMainPID-2       45.0B ± 0%     45.0B ± 0%    ~     (all equal)
DecodeMainPID-2       24.0B ± 0%     24.0B ± 0%    ~     (all equal)
EncodeListUnits-2     0.00B          0.00B         ~     (all equal)
DecodeListUnits-2    21.0kB ± 0%    21.0kB ± 0%    ~     (p=0.071 n=50+50)

name               old allocs/op  new allocs/op  delta
AuthExternal-2         3.00 ± 0%      3.00 ± 0%    ~     (all equal)
DecodeString-2         0.00           0.00         ~     (all equal)
EscapeBusLabel-2       0.00           0.00         ~     (all equal)
DecodeHeader-2         0.00           0.00         ~     (all equal)
EncodeHeader-2         0.00           0.00         ~     (all equal)
EncodeHello-2          0.00           0.00         ~     (all equal)
DecodeHello-2          1.00 ± 0%      1.00 ± 0%    ~     (all equal)
EncodeMainPID-2        0.00           0.00         ~     (all equal)
DecodeMainPID-2        1.00 ± 0%      1.00 ± 0%    ~     (all equal)
EncodeListUnits-2      0.00           0.00         ~     (all equal)
DecodeListUnits-2      6.00 ± 0%      6.00 ± 0%    ~     (all equal)
```

</details>

When there is a statistically significant improvement,
please update [bench-old.txt](bench-old.txt) and the benchmark results above.

```sh
$ cp bench-new.txt bench-old.txt
```
