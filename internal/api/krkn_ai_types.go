package api

type KrknAIDiscoveryRequest struct {
	TargetRequestID  string              `json:"targetRequestId"`
	TargetClusters   map[string][]string `json:"targetClusters"`
	NamespacePattern string              `json:"namespacePattern,omitempty"`
	PodLabelPattern  string              `json:"podLabelPattern,omitempty"`
	NodeLabelPattern string              `json:"nodeLabelPattern,omitempty"`
	SkipPodName      string              `json:"skipPodName,omitempty"`
}

type KrknAIConfigRequest struct {
	Name            string              `json:"name"`
	ConfigYAML      string              `json:"configYaml"`
	TargetRequestID string              `json:"targetRequestId"`
	TargetClusters  map[string][]string `json:"targetClusters"`
	Description     string              `json:"description,omitempty"`
	Groups          []string            `json:"groups,omitempty"`
	AvailableToAll  bool                `json:"availableToAll,omitempty"`
}

type KrknAIRunRequest struct {
	Name                     string              `json:"name"`
	ConfigID                 string              `json:"configId"`
	TargetRequestID          string              `json:"targetRequestId"`
	TargetClusters           map[string][]string `json:"targetClusters"`
	PrometheusURL            string              `json:"prometheusUrl,omitempty"`
	PrometheusTokenSecretRef string              `json:"prometheusTokenSecretRef,omitempty"`
	ActiveDeadlineSeconds    int64               `json:"activeDeadlineSeconds,omitempty"`
}
