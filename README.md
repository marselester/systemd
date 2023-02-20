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
and troubleshoot performance issues, e.g., by profiling memory.

```sh
$ go test -run=^$ -bench=^BenchmarkDecodeListUnits$ -benchmem -memprofile list_units.allocs
$ go tool pprof list_units.allocs
```

It is recommended to run benchmarks multiple times and check
how stable they are using [Benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) tool.

```sh
$ go test -bench=. -benchmem -count=10 . | tee bench-new.txt
```

<details>

<summary>

```sh
$ benchstat bench-old.txt
```

</summary>

```
name               time/op
DecodeString-2     53.8ns ± 1%
DecodeListUnits-2   101µs ±22%

name               alloc/op
DecodeString-2      0.00B
DecodeListUnits-2  25.6kB ± 0%

name               allocs/op
DecodeString-2       0.00
DecodeListUnits-2    9.00 ± 0%
```

</details>

The old and new stats are compared as follows.

<details>

<summary>

```sh
$ benchstat bench-old.txt bench-new.txt
```

</summary>

```
name               old time/op    new time/op    delta
DecodeString-2       53.8ns ± 3%    53.8ns ± 1%     ~     (p=0.645 n=10+9)
DecodeListUnits-2     122µs ±10%     101µs ±22%  -17.29%  (p=0.000 n=10+10)

name               old alloc/op   new alloc/op   delta
DecodeString-2        0.00B          0.00B          ~     (all equal)
DecodeListUnits-2    76.0kB ± 0%    25.6kB ± 0%  -66.30%  (p=0.002 n=7+8)

name               old allocs/op  new allocs/op  delta
DecodeString-2         0.00           0.00          ~     (all equal)
DecodeListUnits-2      21.0 ± 0%       9.0 ± 0%  -57.14%  (p=0.000 n=10+10)
```

</details>

When there is a statistically significant improvement,
please update `bench-old.txt` and the benchmark results above.

```sh
$ cp bench-new.txt bench-old.txt
```
