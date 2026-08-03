package api

import "testing"

func TestIsScenarioBlocked(t *testing.T) {
	tests := []struct {
		name     string
		scenario string
		want     bool
	}{
		{"node-scenarios prefix blocks node-scenarios-bm", "node-scenarios-bm", true},
		{"node-scenarios prefix blocks node-scenarios", "node-scenarios", true},
		{"exact match blocks zone-outages", "zone-outages", true},
		{"exact match blocks power-outages", "power-outages", true},
		{"node- alone is now allowed", "node-cpu-hog", false},
		{"node-drain is now allowed", "node-drain", false},
		{"bare 'node' is not blocked", "node", false},
		{"pod-disruption is allowed", "pod-disruption", false},
		{"network-chaos is allowed", "network-chaos", false},
		{"container-kill is allowed", "container-kill", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isScenarioBlocked(tt.scenario)
			if got != tt.want {
				t.Errorf("isScenarioBlocked(%q) = %v, want %v", tt.scenario, got, tt.want)
			}
		})
	}
}
