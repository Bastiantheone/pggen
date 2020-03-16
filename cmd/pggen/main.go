package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"

	"github.com/opendoor-labs/pggen/gen"
)

func usage(ok bool) {
	usage := `
Usage: pggen [<options>] <config-file>

Args:
 <config-file> A configuration toml file containing a list of database objects
               that pggen should generate code for.

Options:
-h, --help                                   Print this message.

-d, --disable-var <var-pattern>              If <var-pattern> matches against the environment,
                                             don't do anything. The pattern 'VAR' matches if
                                             there are any env vars of that name. The pattern
                                             'VAR=value' matches if there are any env vars
                                             with value 'value'. May be provided more
                                             than once, in which case pggen is disabled if any
                                             match against the environment.

-c, --connection-string <connection-string>  The connection string to use to attach
                                             to the postgres instance we will
                                             generate shims for. May be specified more
                                             than once, in which case the connection
                                             strings are tried in order until one that
                                             works is found. Defaults to $DB_URL.

-o, --output-file <file-name>                The name of the file to write the shims to.
                                             If the file name ends with .go it will be
                                             re-written to end with .gen.go.
                                             Defaults to "./pg_generated.gen.go".
`
	if ok {
		fmt.Print(usage)
		os.Exit(0)
	} else {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
}

func main() {
	var config gen.Config
	config.OutputFileName = "./pg_generated.go"

	func() {
		// While parsing args we will might panic on out-of-bounds array
		// access. This means the arguments are malformed.
		defer func() {
			if x := recover(); x != nil {
				usage(false)
			}
		}()

		args := os.Args[1:]

		if len(args) == 0 {
			usage(false)
		}

		for len(args) > 0 {
			if args[0] == "-c" || args[0] == "--connection-string" {
				config.ConnectionStrings = append(config.ConnectionStrings, args[1])
				args = args[2:]
			} else if args[0] == "-o" || args[0] == "--output-file" {
				config.OutputFileName = args[1]
				args = args[2:]
			} else if args[0] == "-d" || args[0] == "--disable-var" {
				config.DisableVars = append(config.DisableVars, args[1])
				args = args[2:]
			} else if args[0] == "-h" || args[0] == "--help" {
				usage(true)
			} else if len(args) == 1 {
				config.ConfigFilePath = args[0]
				break
			} else {
				usage(false)
			}
		}
	}()

	if len(config.ConnectionStrings) == 0 {
		config.ConnectionStrings = []string{os.Getenv("DB_URL")}
		if len(config.ConnectionStrings[0]) == 0 {
			log.Fatal("No connection string. Either pass '-c' or set DB_URL in the environment.")
		}
	}

	if strings.HasSuffix(config.OutputFileName, ".go") &&
		!strings.HasSuffix(config.OutputFileName, ".gen.go") {
		config.OutputFileName = config.OutputFileName[:len(config.OutputFileName)-3] + ".gen.go"
	}

	//
	// Create the codegenerator and invoke it
	//

	g, err := gen.FromConfig(config)
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}

	err = g.Gen()
	if err != nil {
		fmt.Fprint(os.Stderr, err.Error()+"\n")
		os.Exit(1)
	}
}
