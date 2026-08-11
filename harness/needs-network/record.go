// What a run leaves behind.
//
// The record is the whole output of this set. It carries the moment, the
// machine and the network the readings were taken from, because a number from a
// real network is a measurement of one moment and printing it without those is
// how a measurement becomes a claim about the site.
package main

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// conditions is what was true of the run rather than of the site. It is a
// separate value from the readings so that a record can never carry one without
// the other.
type conditions struct {
	Taken  time.Time
	Name   string
	Go     string
	OS     string
	Reason string
}

func here(now time.Time, name, reason string) conditions {
	return conditions{
		Taken:  now.UTC(),
		Name:   name,
		Go:     runtime.Version(),
		OS:     runtime.GOOS + "/" + runtime.GOARCH,
		Reason: reason,
	}
}

// stamp is the file name a record is written under. Two colons in a timestamp
// are a path separator on one of the systems this repository is developed on,
// so the time is spelled with hyphens and the format is stated here rather than
// discovered by whoever hits it.
func (c conditions) stamp() string {
	return c.Taken.Format("2006-01-02T15-04-05Z") + ".md"
}

// render writes the record. It takes everything it prints, including the
// moment, so the suite can compare a whole record against an expected one
// rather than against a pattern with holes in it.
func render(c conditions, readings []reading) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# needs-network against %s, %s\n\n", c.Name, c.Taken.Format(time.RFC3339))

	b.WriteString("## The conditions\n\n")
	fmt.Fprintf(&b, "Taken %s, from one machine on one network, at one moment. What is\n",
		c.Taken.Format(time.RFC3339))
	b.WriteString("below is what answered then. It is not a property of the site, and a later\n")
	b.WriteString("reading is the only thing that says whether it still holds.\n\n")
	fmt.Fprintf(&b, "    name       %s\n", c.Name)
	fmt.Fprintf(&b, "    toolchain  %s\n", c.Go)
	fmt.Fprintf(&b, "    machine    %s\n", c.OS)
	fmt.Fprintf(&b, "    run        %s\n\n", c.Reason)

	var taken, could, not []reading
	for _, r := range readings {
		switch {
		case r.Skipped != "":
			not = append(not, r)
		case r.Failed != "":
			could = append(could, r)
		default:
			taken = append(taken, r)
		}
	}

	fmt.Fprintf(&b, "%d reading(s) taken, %d that could not be taken, %d not asked for.\n\n",
		len(taken), len(could), len(not))

	b.WriteString("## What was read\n\n")
	if len(taken) == 0 {
		b.WriteString("Nothing. Every reading below is either one that could not be taken or one\n")
		b.WriteString("that was not asked for, and neither is a result.\n\n")
	}
	for _, r := range taken {
		fmt.Fprintf(&b, "### %s\n\n", r.Name)
		fmt.Fprintf(&b, "    %s\n", r.Asked)
		for _, l := range r.Lines {
			fmt.Fprintf(&b, "    %s\n", l)
		}
		b.WriteString("\n")
	}

	if len(could) > 0 {
		b.WriteString("## What could not be read\n\n")
		b.WriteString("A reading that failed is not a reading that passed, and none of these is\n")
		b.WriteString("reported anywhere else in this record.\n\n")
		for _, r := range could {
			fmt.Fprintf(&b, "### %s\n\n", r.Name)
			fmt.Fprintf(&b, "    %s\n", r.Asked)
			fmt.Fprintf(&b, "    %s\n\n", r.Failed)
		}
	}

	if len(not) > 0 {
		b.WriteString("## What this run did not ask for\n\n")
		for _, r := range not {
			fmt.Fprintf(&b, "### %s\n\n%s\n\n", r.Name, r.Skipped)
		}
	}

	return strings.TrimRight(b.String(), "\n") + "\n"
}
