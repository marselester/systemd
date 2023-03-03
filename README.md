# systemd

[![Documentation](https://godoc.org/github.com/marselester/systemd?status.svg)](https://pkg.go.dev/github.com/marselester/systemd)
[![Go Report Card](https://goreportcard.com/badge/github.com/marselester/systemd)](https://goreportcard.com/report/github.com/marselester/systemd)

**⚠️ This is still a draft.**

This package provides an access to systemd via D-Bus
to list services with a low overhead for a caller.

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
DecodeString-2       54.1ns ± 2%    54.8ns ± 2%  +1.43%  (p=0.000 n=98+98)
DecodeListUnits-2     102µs ±14%      99µs ± 9%  -3.37%  (p=0.000 n=96+96)

name               old alloc/op   new alloc/op   delta
DecodeString-2        0.00B          0.00B         ~     (all equal)
DecodeListUnits-2    25.6kB ± 0%    25.6kB ± 0%    ~     (all equal)

name               old allocs/op  new allocs/op  delta
DecodeString-2         0.00           0.00         ~     (all equal)
DecodeListUnits-2      7.00 ± 0%      7.00 ± 0%    ~     (all equal)
```

</details>

When there is a statistically significant improvement,
please update [bench-old.txt](bench-old.txt) and the benchmark results above.

```sh
$ cp bench-new.txt bench-old.txt
```
