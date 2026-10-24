// Command jartool is the recorder's side of the stored-repack goldens (../setup.sh builds it): the Go test
// calls the same jarlist functions, so both sides build the input jars and list the output alike.
//
//	jartool build <spec> <jar>   write the jar a jarlist spec spells
//	jartool list <jar>           print the jarlist listing of a jar
package main

import (
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/storedrepack/jarlist"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "jartool:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	switch {
	case len(args) == 3 && args[0] == "build":
		spec, err := os.ReadFile(args[1])
		if err != nil {
			return err
		}
		jar, err := jarlist.Build(spec)
		if err != nil {
			return err
		}
		return os.WriteFile(args[2], jar, 0o644)
	case len(args) == 2 && args[0] == "list":
		l, err := jarlist.List(args[1])
		if err != nil {
			return err
		}
		_, err = fmt.Print(l)
		return err
	}
	return fmt.Errorf("usage: jartool build <spec> <jar> | jartool list <jar>")
}
