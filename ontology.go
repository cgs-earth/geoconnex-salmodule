package main

import (
	_ "embed"
	"fmt"
	"io"
)

//go:embed ontology.jsonld
var ontologyJSONLD []byte

func runOntology(w io.Writer) error {
	_, err := fmt.Fprint(w, string(ontologyJSONLD))
	return err
}
