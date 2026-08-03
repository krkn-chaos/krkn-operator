package api

import "strings"

var blockedScenarioPrefixes = []string{"node-scenarios"}
var blockedScenariosExact = []string{"zone-outages", "power-outages"}

func isScenarioBlocked(name string) bool {
	for _, exact := range blockedScenariosExact {
		if name == exact {
			return true
		}
	}
	for _, prefix := range blockedScenarioPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
