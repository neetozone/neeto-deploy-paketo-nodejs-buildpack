package nodejs_test

import (
	"bufio"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

var (
	quotedValue    = regexp.MustCompile(`=\s*"([^"]*)"`)
	dependencyName = regexp.MustCompile(`/([a-z-]+):[0-9.]+$`)
	buildpackID    = regexp.MustCompile(`^(?:neeto-deploy|paketo-buildpacks)/([a-z-]+)$`)
)

// tomlValues returns the quoted values of key inside every table with the
// given header, e.g. the ids of all [[order.group]] tables.
func tomlValues(t *testing.T, path, header, key string) []string {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	var values []string
	inTable := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			inTable = line == header
			continue
		}

		if inTable && strings.HasPrefix(line, key+" ") {
			if match := quotedValue.FindStringSubmatch(line); match != nil {
				values = append(values, match[1])
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	return values
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	var unique []string
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			unique = append(unique, value)
		}
	}
	sort.Strings(unique)

	return unique
}

func TestUnitManifest(t *testing.T) {
	Expect := NewWithT(t).Expect

	var ordered []string
	for _, id := range tomlValues(t, "buildpack.toml", "[[order.group]]", "id") {
		match := buildpackID.FindStringSubmatch(id)
		Expect(match).NotTo(BeNil(), "unexpected buildpack id %q", id)
		ordered = append(ordered, match[1])
	}
	Expect(ordered).NotTo(BeEmpty())

	var packaged []string
	for _, uri := range tomlValues(t, "package.toml", "[[dependencies]]", "uri") {
		match := dependencyName.FindStringSubmatch(uri)
		Expect(match).NotTo(BeNil(), "unexpected dependency uri %q", uri)
		packaged = append(packaged, match[1])
	}

	// Every buildpack an order group refers to must be bundled, and every
	// bundled buildpack must be used by some order group.
	Expect(uniqueSorted(packaged)).To(Equal(uniqueSorted(ordered)))
}
