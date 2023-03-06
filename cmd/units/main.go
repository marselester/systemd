// Program units prints systemd units.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/marselester/systemd"
)

func main() {
	// By default an exit code is set to indicate a failure
	// since there are more failure scenarios to begin with.
	exitCode := 1
	defer func() { os.Exit(exitCode) }()

	onlyServices := flag.Bool("svc", false, "show only services")
	flag.Parse()

	conn, err := systemd.Dial()
	if err != nil {
		log.Print(err)
		return
	}
	defer func() {
		if err = conn.Close(); err != nil {
			log.Print(err)
		}
	}()

	c, err := systemd.New(conn)
	if err != nil {
		log.Print(err)
		return
	}

	if *onlyServices {
		err = printServices(c)
	} else {
		err = c.ListUnits(printAll)
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
	err := c.ListUnits(func(u *systemd.Unit) {
		if strings.HasSuffix(u.Name, ".service") {
			// Must copy a unit,
			// otherwise it will be modified on the next function call.
			services = append(services, *u)
		}
	})
	if err != nil {
		return err
	}

	var pid uint32
	for _, s := range services {
		if pid, err = c.MainPID(s.Name); err != nil {
			return err
		}

		fmt.Printf("%d %s %s\n", pid, s.Name, s.ActiveState)
	}

	return nil
}
