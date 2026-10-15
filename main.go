// Command geoconnex-salmodule implements the SAL Module CLI for harvesting
// JSON-LD directly out of the GeoParquet files in the Geoconnex STAC catalog.
//
// It is invoked as:
//
//	geoconnex-salmodule salmodule ontology
//	geoconnex-salmodule salmodule run
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "salmodule" {
		fmt.Fprintln(os.Stderr, "usage: salmodule ontology | salmodule run")
		os.Exit(2)
	}

	switch os.Args[2] {
	case "ontology":
		if err := runOntology(os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "run":
		if err := runTask(os.Stdout, os.Stderr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			printErrorNode(os.Stdout, err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "usage: salmodule ontology | salmodule run")
		os.Exit(2)
	}
}
