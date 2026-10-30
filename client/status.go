package client

import (
	"regexp"
	"strings"
)

// duration matches the human-readable durations Docker puts into a
// container's status, e.g. "Up 2 hours" or "Exited (0) 3 minutes ago".
var duration = regexp.MustCompile(`(Less than a second|About a minute|About an hour|(\d+) (second|minute|hour|day|week|month|year)s?)( ago)?`)

// healthNote matches the health Docker appends to the status of a running
// container; health is reported separately.
var healthNote = regexp.MustCompile(`\((healthy|unhealthy|health: starting)\)`)

var units = map[string]string{
	"second": "s",
	"minute": "m",
	"hour":   "h",
	"day":    "d",
	"week":   "w",
	"month":  "mo",
	"year":   "y",
}

// splitStatus splits a Docker status into the status without its duration
// and health and the duration in short form, e.g. "Up 2 hours (unhealthy)"
// into "Up" and "2h". The age is empty if the status has no duration.
func splitStatus(status string) (string, string) {
	status = healthNote.ReplaceAllString(status, "")
	m := duration.FindStringSubmatchIndex(status)
	if m == nil {
		return strings.Join(strings.Fields(status), " "), ""
	}

	var age string
	switch phrase := status[m[2]:m[3]]; phrase {
	case "Less than a second":
		age = "<1s"
	case "About a minute":
		age = "1m"
	case "About an hour":
		age = "1h"
	default:
		age = status[m[4]:m[5]] + units[status[m[6]:m[7]]]
	}

	rest := strings.Join(strings.Fields(status[:m[0]]+" "+status[m[1]:]), " ")
	return rest, age
}
