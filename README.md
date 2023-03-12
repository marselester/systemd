# systemd

[![Documentation](https://godoc.org/github.com/marselester/systemd?status.svg)](https://pkg.go.dev/github.com/marselester/systemd)
[![Go Report Card](https://goreportcard.com/badge/github.com/marselester/systemd)](https://goreportcard.com/report/github.com/marselester/systemd)

This package provides an access to systemd via D-Bus
to list services (think `systemctl list-units`) with a low overhead for a caller.
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

You can get the same results with `dbus-send`.

```sh
$ dbus-send --system --print-reply --dest=org.freedesktop.systemd1 /org/freedesktop/systemd1 org.freedesktop.systemd1.Manager.ListUnits
$ dbus-send --system --print-reply --dest=org.freedesktop.systemd1 /org/freedesktop/systemd1/unit/dbus_2eservice org.freedesktop.DBus.Properties.Get string:'org.freedesktop.systemd1.Service' string:'MainPID'
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
AuthExternal-2        475ns ± 2%     467ns ± 2%    -1.72%  (p=0.000 n=46+45)
DecodeString-2       54.3ns ± 3%    53.4ns ± 4%    -1.67%  (p=0.000 n=49+50)
EscapeBusLabel-2     43.8ns ± 2%    43.2ns ± 2%    -1.35%  (p=0.000 n=48+50)
DecodeHeader-2        340ns ± 4%     340ns ±12%    -0.17%  (p=0.017 n=48+45)
EncodeHeader-2        185ns ± 1%     187ns ± 3%    +1.28%  (p=0.000 n=48+49)
EncodeHello-2         224ns ± 2%     221ns ± 3%    -1.03%  (p=0.000 n=46+50)
DecodeHello-2         122ns ± 4%      84ns ±11%   -31.21%  (p=0.000 n=49+49)
EncodeMainPID-2       383ns ± 4%     383ns ± 3%      ~     (p=0.548 n=43+48)
DecodeMainPID-2       140ns ± 2%      96ns ± 3%   -31.63%  (p=0.000 n=48+50)
EncodeListUnits-2     234ns ± 2%     236ns ± 2%    +0.61%  (p=0.001 n=41+47)
DecodeListUnits-2    92.9µs ± 2%    92.5µs ± 5%    -0.39%  (p=0.028 n=48+45)

name               old alloc/op   new alloc/op   delta
AuthExternal-2        80.0B ± 0%     80.0B ± 0%      ~     (all equal)
DecodeString-2        0.00B          0.00B           ~     (all equal)
EscapeBusLabel-2      0.00B          0.00B           ~     (all equal)
DecodeHeader-2        15.0B ± 0%     15.0B ± 0%      ~     (all equal)
EncodeHeader-2        0.00B          0.00B           ~     (all equal)
EncodeHello-2         0.00B          0.00B           ~     (all equal)
DecodeHello-2         29.0B ± 0%      5.0B ± 0%   -82.76%  (p=0.000 n=50+50)
EncodeMainPID-2       45.0B ± 0%     45.0B ± 0%      ~     (all equal)
DecodeMainPID-2       24.0B ± 0%      0.0B       -100.00%  (p=0.000 n=50+50)
EncodeListUnits-2     0.00B          0.00B           ~     (all equal)
DecodeListUnits-2    21.0kB ± 0%    20.9kB ± 0%    -0.11%  (p=0.000 n=50+50)

name               old allocs/op  new allocs/op  delta
AuthExternal-2         3.00 ± 0%      3.00 ± 0%      ~     (all equal)
DecodeString-2         0.00           0.00           ~     (all equal)
EscapeBusLabel-2       0.00           0.00           ~     (all equal)
DecodeHeader-2         0.00           0.00           ~     (all equal)
EncodeHeader-2         0.00           0.00           ~     (all equal)
EncodeHello-2          0.00           0.00           ~     (all equal)
DecodeHello-2          1.00 ± 0%      0.00       -100.00%  (p=0.000 n=50+50)
EncodeMainPID-2        0.00           0.00           ~     (all equal)
DecodeMainPID-2        1.00 ± 0%      0.00       -100.00%  (p=0.000 n=50+50)
EncodeListUnits-2      0.00           0.00           ~     (all equal)
DecodeListUnits-2      6.00 ± 0%      5.00 ± 0%   -16.67%  (p=0.000 n=50+50)
```

</details>

When there is a statistically significant improvement,
please update [bench-old.txt](bench-old.txt) and the benchmark results above.

```sh
$ cp bench-new.txt bench-old.txt
```
