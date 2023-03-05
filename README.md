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
DecodeString-2       54.8ns ± 2%    54.7ns ± 2%    -0.35%  (p=0.006 n=98+99)
DecodeHeader-2        345ns ± 8%     341ns ± 7%    -0.96%  (p=0.010 n=95+95)
EncodeHeader-2        305ns ± 2%     177ns ± 2%   -41.91%  (p=0.000 n=96+99)
DecodeMainPID-2       159ns ± 6%     160ns ± 8%      ~     (p=0.095 n=93+98)
DecodeListUnits-2    99.0µs ± 9%    97.9µs ± 8%    -1.05%  (p=0.039 n=96+94)

name               old alloc/op   new alloc/op   delta
DecodeString-2        0.00B          0.00B           ~     (all equal)
DecodeHeader-2        15.0B ± 0%     15.0B ± 0%      ~     (all equal)
EncodeHeader-2        32.0B ± 0%      0.0B       -100.00%  (p=0.000 n=100+100)
DecodeMainPID-2       24.0B ± 0%     24.0B ± 0%      ~     (all equal)
DecodeListUnits-2    25.6kB ± 0%    25.6kB ± 0%      ~     (all equal)

name               old allocs/op  new allocs/op  delta
DecodeString-2         0.00           0.00           ~     (all equal)
DecodeHeader-2         0.00           0.00           ~     (all equal)
EncodeHeader-2         9.00 ± 0%      0.00       -100.00%  (p=0.000 n=100+100)
DecodeMainPID-2        1.00 ± 0%      1.00 ± 0%      ~     (all equal)
DecodeListUnits-2      7.00 ± 0%      7.00 ± 0%      ~     (all equal)
```

</details>

When there is a statistically significant improvement,
please update [bench-old.txt](bench-old.txt) and the benchmark results above.

```sh
$ cp bench-new.txt bench-old.txt
```
