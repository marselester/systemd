# systemd

[![Documentation](https://godoc.org/github.com/marselester/systemd?status.svg)](https://pkg.go.dev/github.com/marselester/systemd)
[![Go Report Card](https://goreportcard.com/badge/github.com/marselester/systemd)](https://goreportcard.com/report/github.com/marselester/systemd)

**⚠️ This is still a draft.**

This package provides an access to systemd via D-Bus
to list services with a low overhead for a caller.
If you find the API too limiting or missing some of functionality,
perhaps https://github.com/coreos/go-systemd might suit you better.

```go
conn, err := systemd.Dial()
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

c, err := systemd.New(conn)
if err != nil {
    log.Fatal(err)
}

err = c.ListUnits(func(u *systemd.Unit) {
    if strings.HasSuffix(u.Name, ".service") {
        fmt.Printf("%s %s\n", u.Name, u.ActiveState)
    }
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
DecodeString-2       54.6ns ± 2%    54.3ns ± 3%   -0.63%  (p=0.000 n=99+97)
EscapeBusLabel-2     42.9ns ± 2%    47.6ns ± 3%  +10.94%  (p=0.000 n=98+99)
DecodeHeader-2        343ns ± 7%     341ns ± 5%     ~     (p=0.274 n=98+92)
EncodeHeader-2        185ns ± 2%     190ns ± 5%   +2.25%  (p=0.000 n=99+99)
EncodeListUnits-2     232ns ± 3%     233ns ± 3%   +0.32%  (p=0.034 n=97+93)
EncodeMainPID-2       374ns ± 2%     377ns ± 3%   +0.82%  (p=0.000 n=97+93)
DecodeMainPID-2       159ns ±10%     135ns ± 4%  -15.59%  (p=0.000 n=98+98)
DecodeListUnits-2    99.0µs ±10%    93.3µs ± 4%   -5.71%  (p=0.000 n=96+99)

name               old alloc/op   new alloc/op   delta
DecodeString-2        0.00B          0.00B          ~     (all equal)
EscapeBusLabel-2      0.00B          0.00B          ~     (all equal)
DecodeHeader-2        15.0B ± 0%     15.0B ± 0%     ~     (all equal)
EncodeHeader-2        0.00B          0.00B          ~     (all equal)
EncodeListUnits-2     0.00B          0.00B          ~     (all equal)
EncodeMainPID-2       45.0B ± 0%     45.0B ± 0%     ~     (all equal)
DecodeMainPID-2       24.0B ± 0%     24.0B ± 0%     ~     (all equal)
DecodeListUnits-2    25.6kB ± 0%    25.6kB ± 0%     ~     (all equal)

name               old allocs/op  new allocs/op  delta
DecodeString-2         0.00           0.00          ~     (all equal)
EscapeBusLabel-2       0.00           0.00          ~     (all equal)
DecodeHeader-2         0.00           0.00          ~     (all equal)
EncodeHeader-2         0.00           0.00          ~     (all equal)
EncodeListUnits-2      0.00           0.00          ~     (all equal)
EncodeMainPID-2        0.00           0.00          ~     (all equal)
DecodeMainPID-2        1.00 ± 0%      1.00 ± 0%     ~     (all equal)
DecodeListUnits-2      7.00 ± 0%      7.00 ± 0%     ~     (all equal)
```

</details>

When there is a statistically significant improvement,
please update [bench-old.txt](bench-old.txt) and the benchmark results above.

```sh
$ cp bench-new.txt bench-old.txt
```
