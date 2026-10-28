package componentversion

import (
	"fmt"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	for _, version := range []string{"0.0.0", "v1.1.0", "v1.1.0-rc4", "preview/foo"} {
		t.Run(version, func(t *testing.T) {
			data := []byte(fmt.Sprintf(`{"version":%q,"components":[{"component":"api","description":"REST API"}]}`, version))
			manifest, err := Load(data)
			if err != nil || manifest.Version != version {
				t.Fatalf("Load() = %#v, %v; want version %q", manifest, err, version)
			}
		})
	}
}

func TestLoadRejectsInvalidManifest(t *testing.T) {
	for _, data := range []string{
		`{"version":"","components":[{"component":"api","description":"API"}]}`,
		`{"version":"1","components":[]}`,
		`{"version":"1","components":[{"component":"api","description":""}]}`,
		`{"version":"1","components":[{"component":"api","description":"API"},{"component":"api","description":"API"}]}`,
		`{"version":"1","components":[],"unknown":true}`,
		`{"version":"1","components":[]} {}`,
	} {
		if _, err := Load([]byte(data)); err == nil {
			t.Errorf("Load(%q) succeeded unexpectedly", strings.TrimSpace(data))
		}
	}
}
