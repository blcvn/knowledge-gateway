// Package apidocs_test keeps the published API surface and the running one in step.
//
// Route registration and API documentation live in different files and drift silently: a new route
// ships undocumented, or a documented route is renamed and the spec keeps advertising the old path.
// Either way the first person to notice is an integrator whose client 404s.
package apidocs_test

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const (
	routerFile  = "../../internal/bootstrap/app.go"
	openapiFile = "../../docs/api/openapi.yaml"
	readmeFile  = "../../README.md"
)

// routerRoute matches the registration calls in app.go, e.g. router.POST("/v1/kg/...", ...).
var routerRoute = regexp.MustCompile(`router\.(GET|POST|PUT|DELETE|PATCH)\("(/[^"]*)"`)

// openapiPath matches a path key in the spec: two spaces of indent, a leading slash, trailing colon.
var openapiPath = regexp.MustCompile(`(?m)^  (/[^:\s]*(?::[a-z-]+)?):\s*$`)

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func registeredRoutes(t *testing.T) map[string]struct{} {
	t.Helper()
	routes := map[string]struct{}{}
	for _, m := range routerRoute.FindAllStringSubmatch(readFile(t, routerFile), -1) {
		routes[m[2]] = struct{}{}
	}
	if len(routes) == 0 {
		t.Fatalf("no routes found in %s — the extraction pattern is stale", routerFile)
	}
	return routes
}

func documentedPaths(t *testing.T) map[string]struct{} {
	t.Helper()
	paths := map[string]struct{}{}
	for _, m := range openapiPath.FindAllStringSubmatch(readFile(t, openapiFile), -1) {
		paths[m[1]] = struct{}{}
	}
	if len(paths) == 0 {
		t.Fatalf("no paths found in %s — the extraction pattern is stale", openapiFile)
	}
	return paths
}

// normalisePathParams rewrites {id} style parameters so the two sources can be compared even if one
// of them ever switches placeholder spelling.
func normalisePathParams(path string) string {
	return regexp.MustCompile(`\{[^}]+\}`).ReplaceAllString(path, "{param}")
}

func normaliseSet(in map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for path := range in {
		out[normalisePathParams(path)] = struct{}{}
	}
	return out
}

func sortedDiff(have, want map[string]struct{}) []string {
	missing := make([]string, 0)
	for path := range have {
		if _, ok := want[path]; !ok {
			missing = append(missing, path)
		}
	}
	sort.Strings(missing)
	return missing
}

// TestEveryRegisteredRouteIsDocumented catches a route that shipped without a spec entry.
func TestEveryRegisteredRouteIsDocumented(t *testing.T) {
	routes := normaliseSet(registeredRoutes(t))
	docs := normaliseSet(documentedPaths(t))
	if missing := sortedDiff(routes, docs); len(missing) > 0 {
		t.Errorf("routes registered but absent from docs/api/openapi.yaml:\n  %s", strings.Join(missing, "\n  "))
	}
}

// TestEveryDocumentedPathIsRegistered catches the reverse: a spec entry the service does not serve,
// which is worse than an undocumented route because integrators build against it.
func TestEveryDocumentedPathIsRegistered(t *testing.T) {
	routes := normaliseSet(registeredRoutes(t))
	docs := normaliseSet(documentedPaths(t))
	// /healthz is registered outside the versioned surface and is documented; keep it exempt only
	// if it genuinely is registered.
	if missing := sortedDiff(docs, routes); len(missing) > 0 {
		t.Errorf("paths documented in docs/api/openapi.yaml but not registered in the router:\n  %s", strings.Join(missing, "\n  "))
	}
}

// TestReadmeRouteInventoryMatchesRouter keeps the human-facing inventory honest too — it is the
// first thing an integrator reads.
func TestReadmeRouteInventoryMatchesRouter(t *testing.T) {
	readme := readFile(t, readmeFile)
	routes := registeredRoutes(t)

	missing := make([]string, 0)
	for path := range routes {
		// The README lists routes with their query strings sometimes; match on the path prefix.
		if !strings.Contains(readme, path) {
			missing = append(missing, path)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("routes registered but absent from the README route inventory:\n  %s", strings.Join(missing, "\n  "))
	}
}
