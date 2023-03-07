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
$ go test -timeout 20m -bench=. -benchmem -count=100 . | tee bench-new.txt
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
DecodeString-2       54.3ns ± 3%    53.8ns ± 2%   -0.82%  (p=0.000 n=97+100)
EscapeBusLabel-2     47.6ns ± 3%    47.5ns ± 2%     ~     (p=0.602 n=99+100)
DecodeHeader-2        341ns ± 5%     339ns ± 7%   -0.50%  (p=0.001 n=92+97)
EncodeHeader-2        190ns ± 5%     185ns ± 2%   -2.17%  (p=0.000 n=99+100)
EncodeListUnits-2     233ns ± 3%     232ns ± 3%   -0.34%  (p=0.043 n=93+96)
EncodeMainPID-2       377ns ± 3%     374ns ± 2%   -0.76%  (p=0.000 n=93+86)
DecodeMainPID-2       135ns ± 4%     133ns ± 4%   -1.03%  (p=0.000 n=98+96)
DecodeListUnits-2    93.3µs ± 4%    94.0µs ± 3%   +0.73%  (p=0.001 n=99+95)

name               old alloc/op   new alloc/op   delta
DecodeString-2        0.00B          0.00B          ~     (all equal)
EscapeBusLabel-2      0.00B          0.00B          ~     (all equal)
DecodeHeader-2        15.0B ± 0%     15.0B ± 0%     ~     (all equal)
EncodeHeader-2        0.00B          0.00B          ~     (all equal)
EncodeListUnits-2     0.00B          0.00B          ~     (all equal)
EncodeMainPID-2       45.0B ± 0%     45.0B ± 0%     ~     (all equal)
DecodeMainPID-2       24.0B ± 0%     24.0B ± 0%     ~     (all equal)
DecodeListUnits-2    25.6kB ± 0%    21.0kB ± 0%  -18.01%  (p=0.000 n=100+100)

name               old allocs/op  new allocs/op  delta
DecodeString-2         0.00           0.00          ~     (all equal)
EscapeBusLabel-2       0.00           0.00          ~     (all equal)
DecodeHeader-2         0.00           0.00          ~     (all equal)
EncodeHeader-2         0.00           0.00          ~     (all equal)
EncodeListUnits-2      0.00           0.00          ~     (all equal)
EncodeMainPID-2        0.00           0.00          ~     (all equal)
DecodeMainPID-2        1.00 ± 0%      1.00 ± 0%     ~     (all equal)
DecodeListUnits-2      7.00 ± 0%      6.00 ± 0%  -14.29%  (p=0.000 n=100+100)
```

</details>

When there is a statistically significant improvement,
please update [bench-old.txt](bench-old.txt) and the benchmark results above.

```sh
$ cp bench-new.txt bench-old.txt
```
