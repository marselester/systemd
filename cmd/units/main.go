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
	defer conn.Close()

	c, err := systemd.New(conn)
	if err != nil {
		log.Print(err)
		return
	}

	if *onlyServices {
		err = c.ListUnits(printServices)
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

func printServices(u *systemd.Unit) {
	if strings.HasSuffix(u.Name, ".service") {
		fmt.Printf("%s %s\n", u.Name, u.ActiveState)
	}
}
