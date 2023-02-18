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
