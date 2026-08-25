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
	"gopkg.in/yaml.v2"
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

// requireNoReplaceDirective fails when go.mod redirects a module. A replace
// directive overrides the require line, so a guard that reads only the require
// line passes while the build links the replacement version.
func requireNoReplaceDirective(t *testing.T, gomod, module string) {
	t.Helper()
	re := regexp.MustCompile(`(?m)^\s*(?:replace\s+)?` + regexp.QuoteMeta(module) + `\s+(?:v\S+\s+)?=>.*$`)
	line := re.FindString(gomod)
	require.Emptyf(t, line,
		"go.mod holds a replace directive for %s, so the require line does not give the version that the build links: %s",
		module, strings.TrimSpace(line))
}

// effectiveModuleVersion returns the version of a module that the build links.
// A replace directive overrides the require line, so the helper reads the
// replacement version when go.mod holds one. 'go list -m -f {{.Version}}'
// reports the require version even under a replace directive, so a guard that
// uses that template alone cannot see a downgrade.
func effectiveModuleVersion(t *testing.T, module string) string {
	t.Helper()
	const tmpl = "{{if .Replace}}{{.Replace.Version}}{{else}}{{.Version}}{{end}}"
	out, err := exec.Command("go", "list", "-m", "-f", tmpl, module).CombinedOutput()
	require.NoErrorf(t, err, "go list -m failed: %s", out)
	version := strings.TrimSpace(string(out))
	require.NotEmptyf(t, version,
		"go list -m reports no version for %s, so a local directory replaces it", module)
	return version
}

// requireModuleIsPatched checks a version tuple against the patched oauth2
// version of GHSA-6v2p-p543-phr9.
func requireModuleIsPatched(t *testing.T, source, version string) {
	t.Helper()
	major, minor, patch := parseSemver(t, version)
	assert.Truef(t,
		atLeast(major, minor, patch, oauth2MinMajor, oauth2MinMinor, oauth2MinPatch),
		"%s gives golang.org/x/oauth2 %s, which is lower than the patched v%d.%d.%d for GHSA-6v2p-p543-phr9",
		source, version, oauth2MinMajor, oauth2MinMinor, oauth2MinPatch,
	)
}

// Test_Oauth2IsPatched checks that go.mod pins a patched golang.org/x/oauth2.
// This is the primary acceptance criterion of SEC-31. The test also rejects a
// replace directive, because a replace directive hides the linked version.
func Test_Oauth2IsPatched(t *testing.T) {
	gomod := readFile(t, "go.mod")

	requireNoReplaceDirective(t, gomod, "golang.org/x/oauth2")
	requireModuleIsPatched(t, "go.mod", moduleVersionInGoMod(t, gomod, "golang.org/x/oauth2"))
}

// Test_Oauth2EffectiveVersionIsPatched checks the version that the build links.
// go.mod can list one version while minimal version selection picks another,
// and a replace directive can point at a third one. This test reads the
// replacement version when one exists, so a downgrade cannot pass it.
func Test_Oauth2EffectiveVersionIsPatched(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go list in short mode")
	}
	requireNoReplaceDirective(t, readFile(t, "go.mod"), "golang.org/x/oauth2")
	requireModuleIsPatched(t, "go list -m", effectiveModuleVersion(t, "golang.org/x/oauth2"))
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

// goImageStages returns the major and minor Go version of every golang base
// image in a Dockerfile, keyed by the stage name. An unnamed stage gets a
// positional key.
func goImageStages(t *testing.T, dockerfile string) map[string][2]int {
	t.Helper()
	re := regexp.MustCompile(`(?mi)^FROM\s+golang:(\d+)\.(\d+)\S*(?:\s+as\s+(\S+))?`)
	stages := map[string][2]int{}
	for _, m := range re.FindAllStringSubmatch(dockerfile, -1) {
		major, _ := strconv.Atoi(m[1])
		minor, _ := strconv.Atoi(m[2])
		name := strings.ToLower(m[3])
		if name == "" {
			name = "unnamed-" + strconv.Itoa(len(stages))
		}
		stages[name] = [2]int{major, minor}
	}
	return stages
}

// Test_DockerfileGoVersionSatisfiesGoMod checks that every golang stage of the
// Dockerfile uses a Go image new enough for the go directive in go.mod. The
// test names the build stage, because the first FROM line in the file is not
// always the stage that compiles the binary. A mismatch breaks 'docker build'
// but no other unit test would show it.
func Test_DockerfileGoVersionSatisfiesGoMod(t *testing.T) {
	dockerfile := readFile(t, "Dockerfile")
	goMajor, goMinor := goDirective(t, readFile(t, "go.mod"))

	stages := goImageStages(t, dockerfile)
	require.NotEmpty(t, stages,
		"the Dockerfile has no golang base image with a pinned minor version")

	build, ok := stages["build"]
	require.Truef(t, ok,
		"the Dockerfile has no golang stage named build, found %v", stages)
	assert.Truef(t, atLeast(build[0], build[1], 0, goMajor, goMinor, 0),
		"the build stage image golang:%d.%d is older than the go directive %d.%d in go.mod",
		build[0], build[1], goMajor, goMinor)

	// Every other golang stage also runs the compiler, so each one must satisfy
	// the go directive too.
	for name, version := range stages {
		assert.Truef(t, atLeast(version[0], version[1], 0, goMajor, goMinor, 0),
			"the %s stage image golang:%d.%d is older than the go directive %d.%d in go.mod",
			name, version[0], version[1], goMajor, goMinor)
	}
}

// Test_OnPrWorkflowBuildsThePatchedModule traces the acceptance criteria of
// SEC-31 to the only check that runs on a pull request. The build_push job of
// .github/workflows/on_pr.yaml builds the image, the image build runs make, and
// make compiles the module that go.mod describes.
func Test_OnPrWorkflowBuildsThePatchedModule(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping go list in short mode")
	}

	var workflow struct {
		Jobs map[string]struct {
			Name  string `yaml:"name"`
			Steps []struct {
				Name string            `yaml:"name"`
				Uses string            `yaml:"uses"`
				With map[string]string `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	path := filepath.Join(".github", "workflows", "on_pr.yaml")
	require.NoError(t, yaml.Unmarshal([]byte(readFile(t, path)), &workflow),
		"cannot parse %s", path)

	job, ok := workflow.Jobs["build_push"]
	require.Truef(t, ok, "%s has no build_push job", path)

	var built bool
	for _, step := range job.Steps {
		if !strings.Contains(step.Uses, "build-push") {
			continue
		}
		built = true
		assert.Equal(t, "admiral", step.With["name"],
			"the build_push job must build the admiral image")
		assert.Contains(t, step.With["push"], "github.event_name != 'pull_request'",
			"the build_push job must not push an image from a pull request")
	}
	require.Truef(t, built, "the build_push job of %s runs no build-push step", path)

	// The image build is the only check on a pull request, and it runs make.
	assert.Contains(t, readFile(t, "Dockerfile"), "RUN make",
		"the image build must run make, which builds and tests the module")

	// The module that the build_push job compiles must hold the patched oauth2
	// version. This is the acceptance criterion of SEC-31.
	requireModuleIsPatched(t, "the module that build_push compiles",
		effectiveModuleVersion(t, "golang.org/x/oauth2"))
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
