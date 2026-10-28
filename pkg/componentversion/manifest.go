// Package componentversion loads embedded component manifests and publishes them
// through KrknComponent resources.
package componentversion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
)

// Load parses a manifest bundled with a component binary. The version comes
// only from the embedded JSON, which tagged builds stamp before compilation.
// It returns an error for malformed JSON, unknown fields, trailing data,
// a blank version, an empty component list, blank component names or
// descriptions, or duplicate component names.
func Load(data []byte) (krknv1alpha1.ComponentManifest, error) {
	var manifest krknv1alpha1.ComponentManifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, fmt.Errorf("decode component manifest: %w", err)
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return manifest, fmt.Errorf("component manifest contains trailing data")
	}
	if strings.TrimSpace(manifest.Version) == "" {
		return manifest, fmt.Errorf("component manifest version is required")
	}
	if len(manifest.Components) == 0 {
		return manifest, fmt.Errorf("component manifest must describe at least one component")
	}
	seen := make(map[string]struct{}, len(manifest.Components))
	for _, component := range manifest.Components {
		if strings.TrimSpace(component.Component) == "" || strings.TrimSpace(component.Description) == "" {
			return manifest, fmt.Errorf("component name and description are required")
		}
		if _, exists := seen[component.Component]; exists {
			return manifest, fmt.Errorf("duplicate component %q", component.Component)
		}
		seen[component.Component] = struct{}{}
	}
	return manifest, nil
}
