package correction

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"time"
)

type validationOutput struct {
	Status       string `json:"status"`
	ReportCount  int    `json:"reportCount"`
	ExpiredCount int    `json:"expiredCount"`
}

// RunCheckCommand implements the read-only correctioncheck API subcommand.
func RunCheckCommand(arguments []string, stdout io.Writer, stderr io.Writer, now func() time.Time) int {
	flags := flag.NewFlagSet("correctioncheck", flag.ContinueOnError)
	flags.SetOutput(stderr)
	path := flags.String("path", "", "path to a correction JSON Lines file")
	asOfValue := flags.String("as-of", "", "validation time in RFC3339; defaults to now")
	if err := flags.Parse(arguments); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *path == "" {
		_, _ = fmt.Fprintln(stderr, "correction store path is required and positional arguments are not supported")
		return 2
	}
	if now == nil {
		now = time.Now
	}
	asOf := now().UTC()
	if *asOfValue != "" {
		parsed, err := time.Parse(time.RFC3339, *asOfValue)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, "as-of must use RFC3339")
			return 2
		}
		asOf = parsed
	}

	result, err := ValidateFile(*path, asOf)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "correction store is invalid: %v\n", err)
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(validationOutput{
		Status:       "valid",
		ReportCount:  result.ReportCount,
		ExpiredCount: result.ExpiredCount,
	}); err != nil {
		_, _ = fmt.Fprintln(stderr, "could not write validation result")
		return 1
	}
	return 0
}
