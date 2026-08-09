//go:build headlessnearmiss

// The near miss for the loopback constraint, for one run. It carries a build
// constraint no route satisfies, so nothing compiles or executes it anywhere:
// the row it is here to trip reads the bytes of a test source, and running this
// on a machine with a person sitting at it is the consent prompt the constraint
// exists to prevent. It is removed in the next commit on this branch.
package site

import (
	"net"
	"testing"
)

func TestServesOnEveryInterface(t *testing.T) {
	ln, err := net.Listen("tcp", "0.0.0.0:8080")
	if err != nil {
		t.Fatalf("listening: %v", err)
	}
	defer ln.Close()
}
