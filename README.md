# systemd

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

Benchmarks help to spot performance changes as the project evolves.
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
DecodeString-2     53.8ns ± 3%
DecodeListUnits-2   122µs ±10%

name               alloc/op
DecodeString-2      0.00B
DecodeListUnits-2  76.0kB ± 0%

name               allocs/op
DecodeString-2       0.00
DecodeListUnits-2    21.0 ± 0%
```

</details>

The old and new stats are compared as follows.

```sh
$ benchstat bench-old.txt bench-new.txt
```
