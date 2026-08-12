// The suite over the one constant this package holds.
//
// Nothing here writes the version out as a literal. A test source carrying it a
// second time is the thing the row about a second copy refuses, so the suite
// would fail the tree it is meant to judge, which is the same reason the marker
// fixtures next door are base64.
package version

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The shape a tag is made out of. A version with a fourth part, a leading v or
// a trailing space produces a tag that resolves for the run that made it and
// for nothing afterwards, and none of those is visible in a diff of one line.
var threeParts = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

func TestTheVersionIsThreeNumbersAndNothingElse(t *testing.T) {
	if !threeParts.MatchString(Number) {
		t.Errorf("the version is %q, which is not the three parts a tag is made out of", Number)
	}
}

func TestTheTagIsTheVersionWithOnePrefix(t *testing.T) {
	if got, want := Tag(), "v"+Number; got != want {
		t.Errorf("the tag is %q and the version is %q, so the two spell different releases", got, want)
	}
}

// SourceFile is what the row refusing a second copy excludes, so a path that
// stopped naming the file the version is actually in would leave the row
// refusing the original and passing every copy. The tree is walked from here
// because a test runs in its own package directory and the row reads paths from
// the repository root.
func TestTheNamedSourceFileIsTheOneHoldingTheVersion(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(SourceFile)))
	if err != nil {
		t.Fatalf("%s is what the row excludes and it could not be read: %v", SourceFile, err)
	}
	if !strings.Contains(string(body), Number) {
		t.Errorf("%s does not carry the version, so the row excludes a file that holds nothing", SourceFile)
	}
}
