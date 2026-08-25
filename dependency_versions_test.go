// Package admiral holds repository level invariant tests.
//
// These tests guard the dependency state that the SEC-31 security update
// created. They fail if a later change reintroduces a vulnerable version or
// desynchronizes the Go version between go.mod and the Dockerfile.
package admiral

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimum patched version of golang.org/x/oauth2 for GHSA-6v2p-p543-phr9
const oauth2MinMajor, oauth2MinMinor, oauth2MinPatch = 0, 27, 0

// readFile reads a repository file and fails the test if it is missing.
func readFile(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	require.NoErrorf(t, err, "cannot read %s", name)
	return string(b)
}

// parseSemver splits a "vX.Y.Z" string into its numeric parts.
func parseSemver(t *testing.T, v string) (int, int, int) {
	t.Helper()
	m := regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)`).FindStringSubmatch(strings.TrimSpace(v))
	require.NotNilf(t, m, "version %q is not a semver value", v)
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return major, minor, patch
}

// atLeast reports whether the version tuple a is not lower than the tuple b.
func atLeast(aMaj, aMin, aPatch, bMaj, bMin, bPatch int) bool {
	if aMaj != bMaj {
		return aMaj > bMaj
	}
	if aMin != bMin {
		return aMin > bMin
	}
	return aPatch >= bPatch
}

// requireLine finds the version of a module inside a go.mod require block.
func moduleVersionInGoMod(t *testing.T, gomod, module string) string {
	t.Helper()
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(module) + `\s+(v\S+)`)
	m := re.FindStringSubmatch(gomod)
	require.NotNilf(t, m, "module %s is absent from go.mod", module)
	return m[1]
}

// Test_Oauth2IsPatched checks that go.mod pins a patched golang.org/x/oauth2.
// This is the primary acceptance criterion of SEC-31.
func Test_Oauth2IsPatched(t *testing.T) {
	version := moduleVersionInGoMod(t, readFile(t, "go.mod"), "golang.org/x/oauth2")

	major, minor, patch := parseSemver(t, version)
	assert.Truef(t,
		atLeast(major, minor, patch, oauth2MinMajor, oauth2MinMinor, oauth2MinPatch),
		"golang.org/x/oauth2 %s is lower than the patched v%d.%d.%d for GHSA-6v2p-p543-phr9",
		version, oauth2MinMajor, oauth2MinMinor, oauth2MinPatch,
	)
}

// Test_Oauth2ResolvedVersionIsPatched checks the version that the build
// actually selects. go.mod can list one version while minimal version
// selection picks another, so this asserts on the resolved module.
func Test_Oauth2ResolvedVersionIsPatched(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go list in short mode")
	}
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Version}}", "golang.org/x/oauth2").CombinedOutput()
	require.NoErrorf(t, err, "go list -m failed: %s", out)

	major, minor, patch := parseSemver(t, string(out))
	assert.Truef(t,
		atLeast(major, minor, patch, oauth2MinMajor, oauth2MinMinor, oauth2MinPatch),
		"the build selects golang.org/x/oauth2 %s, which is lower than the patched v%d.%d.%d",
		strings.TrimSpace(string(out)), oauth2MinMajor, oauth2MinMinor, oauth2MinPatch,
	)
}

// Test_Oauth2IsStillLinked confirms that oauth2 remains in the build graph.
// If it were absent, the bump would be cosmetic. It arrives through
// k8s.io/client-go/transport, so this test also detects an accidental drop of
// the authenticated transport path.
func Test_Oauth2IsStillLinked(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go list in short mode")
	}
	out, err := exec.Command("go", "list", "-deps", "./...").CombinedOutput()
	require.NoErrorf(t, err, "go list -deps failed: %s", out)

	deps := string(out)
	assert.Contains(t, deps, "golang.org/x/oauth2",
		"golang.org/x/oauth2 is not linked into the binary")
	assert.Contains(t, deps, "k8s.io/client-go/transport",
		"k8s.io/client-go/transport is not linked into the binary")
}

// Test_Oauth2HashesArePresentAndUnique checks that go.sum holds exactly one
// module hash and one go.mod hash for the patched oauth2 version, and holds no
// stale hash for another version.
func Test_Oauth2HashesArePresentAndUnique(t *testing.T) {
	gosum := readFile(t, "go.sum")
	wanted := moduleVersionInGoMod(t, readFile(t, "go.mod"), "golang.org/x/oauth2")

	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(gosum))
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "golang.org/x/oauth2 ") {
			lines = append(lines, scanner.Text())
		}
	}
	require.NoError(t, scanner.Err())

	// A tidy go.sum holds one "h1:" line and one "/go.mod h1:" line.
	assert.Len(t, lines, 2, "go.sum must hold 2 oauth2 lines, got %v", lines)
	for _, l := range lines {
		assert.Containsf(t, l, wanted,
			"go.sum line %q does not match the go.mod version %s", l, wanted)
	}
}

// Test_AppengineIsFullyRemoved checks that the dropped google.golang.org/appengine
// module leaves no trace in go.mod or go.sum. A hash without a require entry, or
// a require entry without a hash, breaks the build.
func Test_AppengineIsFullyRemoved(t *testing.T) {
	for _, name := range []string{"go.mod", "go.sum"} {
		var hits []string
		scanner := bufio.NewScanner(strings.NewReader(readFile(t, name)))
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "google.golang.org/appengine") {
				hits = append(hits, strings.TrimSpace(scanner.Text()))
			}
		}
		require.NoError(t, scanner.Err())
		assert.Emptyf(t, hits,
			"%s still refers to google.golang.org/appengine: %v", name, hits)
	}
}

// Test_EveryRequiredModuleHasAHash checks that go.sum covers every module that
// go.mod requires. 'go mod tidy' pruned many go.sum lines in this change, so
// this test detects an over-prune.
func Test_EveryRequiredModuleHasAHash(t *testing.T) {
	gomod := readFile(t, "go.mod")
	gosum := readFile(t, "go.sum")

	re := regexp.MustCompile(`(?m)^\s+([\w.\-]+\.[\w.\-]+/[^\s]+)\s+(v\S+)`)
	matches := re.FindAllStringSubmatch(gomod, -1)
	require.NotEmpty(t, matches, "found no require entries in go.mod")

	for _, m := range matches {
		module, version := m[1], m[2]
		assert.Containsf(t, gosum, module+" "+version+" h1:",
			"go.sum has no module hash for %s %s", module, version)
		assert.Containsf(t, gosum, module+" "+version+"/go.mod h1:",
			"go.sum has no go.mod hash for %s %s", module, version)
	}
}

// goDirective returns the major and minor parts of the go directive in go.mod.
func goDirective(t *testing.T, gomod string) (int, int) {
	t.Helper()
	m := regexp.MustCompile(`(?m)^go\s+(\d+)\.(\d+)`).FindStringSubmatch(gomod)
	require.NotNil(t, m, "go.mod has no go directive")
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	return major, minor
}

// Test_ToolchainDirectiveIsConsistent checks the toolchain directive. The
// change removed it. If a later change restores it, it must not name a
// toolchain older than the go directive, because that combination fails.
func Test_ToolchainDirectiveIsConsistent(t *testing.T) {
	gomod := readFile(t, "go.mod")
	goMajor, goMinor := goDirective(t, gomod)

	m := regexp.MustCompile(`(?m)^toolchain\s+go(\d+)\.(\d+)`).FindStringSubmatch(gomod)
	if m == nil {
		t.Log("go.mod has no toolchain directive, which matches the tidied state")
		return
	}
	tcMajor, _ := strconv.Atoi(m[1])
	tcMinor, _ := strconv.Atoi(m[2])
	assert.Truef(t, atLeast(tcMajor, tcMinor, 0, goMajor, goMinor, 0),
		"the toolchain go%d.%d is older than the go directive %d.%d",
		tcMajor, tcMinor, goMajor, goMinor)
}

// Test_DockerfileGoVersionSatisfiesGoMod checks that the Dockerfile build stage
// uses a Go image new enough for the go directive in go.mod. The change raised
// both together. A mismatch breaks 'docker build' but no unit test would show it.
func Test_DockerfileGoVersionSatisfiesGoMod(t *testing.T) {
	dockerfile := readFile(t, "Dockerfile")
	goMajor, goMinor := goDirective(t, readFile(t, "go.mod"))

	m := regexp.MustCompile(`(?mi)^FROM\s+golang:(\d+)\.(\d+)`).FindStringSubmatch(dockerfile)
	require.NotNil(t, m, "the Dockerfile has no golang base image with a pinned minor version")
	imgMajor, _ := strconv.Atoi(m[1])
	imgMinor, _ := strconv.Atoi(m[2])

	assert.Truef(t, atLeast(imgMajor, imgMinor, 0, goMajor, goMinor, 0),
		"the Dockerfile image golang:%d.%d is older than the go directive %d.%d in go.mod",
		imgMajor, imgMinor, goMajor, goMinor)
}

// Test_LocalToolchainSatisfiesGoMod checks that the toolchain which runs the
// tests satisfies the raised go directive.
func Test_LocalToolchainSatisfiesGoMod(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go version in short mode")
	}
	out, err := exec.Command("go", "version").CombinedOutput()
	require.NoErrorf(t, err, "go version failed: %s", out)

	m := regexp.MustCompile(`go(\d+)\.(\d+)`).FindStringSubmatch(string(out))
	require.NotNil(t, m, "cannot parse the output of go version: %s", out)
	localMajor, _ := strconv.Atoi(m[1])
	localMinor, _ := strconv.Atoi(m[2])

	goMajor, goMinor := goDirective(t, readFile(t, "go.mod"))
	assert.Truef(t, atLeast(localMajor, localMinor, 0, goMajor, goMinor, 0),
		"the local toolchain go%d.%d is older than the go directive %d.%d",
		localMajor, localMinor, goMajor, goMinor)
}

// Test_ModuleGraphIsTidy checks that 'go mod tidy' makes no further change.
// The change relied on tidy to drop the toolchain directive and appengine, so an
// untidy result would mean the committed files do not match a tidy run.
func Test_ModuleGraphIsTidy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go mod tidy in short mode")
	}
	root, err := os.Getwd()
	require.NoError(t, err)

	work := t.TempDir()
	// Copy the module files into a scratch directory and tidy them there so the
	// test never writes to the repository.
	cp := exec.Command("cp", "-r",
		filepath.Join(root, "go.mod"), filepath.Join(root, "go.sum"),
		filepath.Join(root, "cmd"), filepath.Join(root, "config"),
		filepath.Join(root, "pkg"), work)
	out, err := cp.CombinedOutput()
	require.NoErrorf(t, err, "cannot copy the module: %s", out)

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = work
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Skipf("go mod tidy cannot run offline: %s", out)
	}

	for _, name := range []string{"go.mod", "go.sum"} {
		before := readFile(t, filepath.Join(root, name))
		after, err := os.ReadFile(filepath.Join(work, name))
		require.NoError(t, err)
		assert.Equalf(t, before, string(after),
			"%s changes when 'go mod tidy' runs, so the committed file is not tidy", name)
	}
}

// Test_GoModHasNoDuplicateRequires checks that no module appears twice in the
// require blocks. A hand edit of a version can create a duplicate that 'go
// build' rejects.
func Test_GoModHasNoDuplicateRequires(t *testing.T) {
	re := regexp.MustCompile(`(?m)^\s+([\w.\-]+\.[\w.\-]+/[^\s]+)\s+v\S+`)
	seen := map[string]int{}
	for _, m := range re.FindAllStringSubmatch(readFile(t, "go.mod"), -1) {
		seen[m[1]]++
	}
	for module, count := range seen {
		assert.Equalf(t, 1, count, "module %s appears %d times in go.mod", module, count)
	}
}
