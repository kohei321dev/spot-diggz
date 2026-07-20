package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/facility"
)

const (
	defaultCatalogPath     = "data/facilities.json"
	defaultMinimumValidity = 168 * time.Hour
)

func main() {
	os.Exit(run(os.Args[1:], time.Now, os.Stdout, os.Stderr))
}

func run(args []string, now func() time.Time, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("catalogcheck", flag.ContinueOnError)
	flags.SetOutput(stderr)
	catalogPath := flags.String("path", defaultCatalogPath, "path to the production facility catalog")
	minimumValidity := flags.Duration("minimum-validity", defaultMinimumValidity, "required freshness duration from the current time")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "catalogcheck: positional arguments are not supported")
		return 2
	}
	if strings.TrimSpace(*catalogPath) == "" {
		fmt.Fprintln(stderr, "catalogcheck: path must not be empty")
		return 2
	}
	if *minimumValidity <= 0 {
		fmt.Fprintln(stderr, "catalogcheck: minimum-validity must be greater than zero")
		return 2
	}

	referenceTime := now()
	if referenceTime.IsZero() {
		fmt.Fprintln(stderr, "catalogcheck: current time must not be zero")
		return 1
	}
	catalog, err := facility.LoadCatalogFileAt(*catalogPath, referenceTime)
	if err != nil {
		fmt.Fprintf(stderr, "catalogcheck: load catalog: %v\n", err)
		return 1
	}

	items := catalog.List("")
	if len(items) == 0 {
		fmt.Fprintln(stderr, "catalogcheck: catalog must contain at least one facility")
		return 1
	}

	checkAt := referenceTime.Add(*minimumValidity)
	staleFields := make([]string, 0)
	for _, item := range items {
		if !facility.IsDynamicInformationFresh(item.DynamicVerifiedAt, checkAt) {
			staleFields = append(staleFields, item.ID+"(dynamic)")
		}
		if !facility.IsStableInformationFresh(item.StableVerifiedAt, checkAt) {
			staleFields = append(staleFields, item.ID+"(stable)")
		}
	}
	if len(staleFields) > 0 {
		fmt.Fprintf(stderr, "catalogcheck: catalog is not fresh through %s: %s\n", checkAt.Format(time.RFC3339), strings.Join(staleFields, ", "))
		return 1
	}

	fmt.Fprintf(stdout, "catalogcheck: %d facilities are fresh through %s\n", len(items), checkAt.Format(time.RFC3339))
	return 0
}
