// Program units prints systemd units
// to show how the package can be configured if needed.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/marselester/systemd"
)

func main() {
	// By default an exit code is set to indicate a failure
	// since there are more failure scenarios to begin with.
	exitCode := 1
	defer func() { os.Exit(exitCode) }()

	addr := flag.String("addr", "", "bus address")
	onlyServices := flag.Bool("svc", false, "show only services")
	checkSerial := flag.Bool("serial", false, "check message serial")
	timeout := flag.Duration("timeout", time.Second, "connection read/write timeout")
	flag.Parse()

	opts := []systemd.Option{
		systemd.WithTimeout(*timeout),
	}
	if *checkSerial {
		opts = append(opts, systemd.WithSerialCheck())
	}
	if *addr != "" {
		opts = append(opts, systemd.WithAddress(*addr))
	}

	c, err := systemd.New(opts...)
	if err != nil {
		log.Print(err)
		return
	}
	defer func() {
		if err = c.Close(); err != nil {
			log.Print(err)
		}
	}()

	if *onlyServices {
		err = printServices(c)
	} else {
		err = c.ListUnits(nil, printAll)
	}
	if err != nil {
		log.Print(err)
		return
	}

	// The program terminates successfully.
	exitCode = 0
}

func printAll(u *systemd.Unit) {
	fmt.Printf("%s %s\n", u.Name, u.ActiveState)
}

// printServices prints service names along with their PIDs.
// It ignores non-service units.
func printServices(c *systemd.Client) error {
	var services []systemd.Unit
	err := c.ListUnits(systemd.IsService, func(u *systemd.Unit) {
		// Must copy a unit,
		// otherwise it will be modified on the next function call.
		services = append(services, *u)
	})
	if err != nil {
		return fmt.Errorf("failed to get systemd units: %w", err)
	}

	var pid uint32
	for _, s := range services {
		if pid, err = c.MainPID(s.Name); err != nil {
			return fmt.Errorf("failed to get PID for service %q: %w", s.Name, err)
		}

		fmt.Printf("%d %s %s\n", pid, s.Name, s.ActiveState)
	}

	return nil
}
