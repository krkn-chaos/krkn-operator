package api

import "time"

type KrknAIDiscoveryRequest struct {
	TargetRequestID  string              `json:"targetRequestId"`
	TargetClusters   map[string][]string `json:"targetClusters"`
	NamespacePattern string              `json:"namespacePattern,omitempty"`
	PodLabelPattern  string              `json:"podLabelPattern,omitempty"`
	NodeLabelPattern string              `json:"nodeLabelPattern,omitempty"`
	SkipPodName      string              `json:"skipPodName,omitempty"`
}

type KrknAIConfigValidationRequest struct {
	ConfigYAML string `json:"configYaml"`
}

type KrknAIConfigValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type KrknAIConfigValidationResponse struct {
	Valid  bool                          `json:"valid,omitempty"`
	Errors []KrknAIConfigValidationError `json:"errors,omitempty"`
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
	ActiveDeadlineSeconds    *int64              `json:"activeDeadlineSeconds,omitempty"`
}

type KrknAIFitnessProgression struct {
	Generation int      `json:"generation"`
	Best       *float64 `json:"best"`
	Average    *float64 `json:"average"`
}

type KrknAIArtifactSummary struct {
	ArtifactStatus        string                     `json:"artifactStatus"`
	CompletedGenerations  *int                       `json:"completedGenerations"`
	CompletedScenarios    *int                       `json:"completedScenarios"`
	ConfiguredGenerations *int                       `json:"configuredGenerations"`
	PopulationSize        *int                       `json:"populationSize"`
	BestFitness           *float64                   `json:"bestFitness"`
	AverageFitness        *float64                   `json:"averageFitness"`
	BaselineFitness       *float64                   `json:"baselineFitness"`
	FitnessProgression    []KrknAIFitnessProgression `json:"fitnessProgression"`
}

type KrknAIRunSummaryResponse struct {
	Name                  string                     `json:"name"`
	Phase                 string                     `json:"phase"`
	CreatedAt             time.Time                  `json:"createdAt"`
	Cluster               string                     `json:"cluster"`
	OrchestratorPodName   string                     `json:"orchestratorPodName"`
	FailureReason         string                     `json:"failureReason"`
	ArtifactStatus        string                     `json:"artifactStatus"`
	CompletedGenerations  *int                       `json:"completedGenerations"`
	CompletedScenarios    *int                       `json:"completedScenarios"`
	ConfiguredGenerations *int                       `json:"configuredGenerations"`
	PopulationSize        *int                       `json:"populationSize"`
	BestFitness           *float64                   `json:"bestFitness"`
	AverageFitness        *float64                   `json:"averageFitness"`
	BaselineFitness       *float64                   `json:"baselineFitness"`
	FitnessProgression    []KrknAIFitnessProgression `json:"fitnessProgression"`
}

type KrknAIScenarioIndexItem struct {
	Generation      int      `json:"generation"`
	ScenarioID      string   `json:"scenarioId"`
	ScenarioType    string   `json:"scenarioType,omitempty"`
	Outcome         string   `json:"outcome,omitempty"`
	DurationSeconds *float64 `json:"durationSeconds,omitempty"`
	FitnessScore    *float64 `json:"fitnessScore,omitempty"`
	FitnessState    string   `json:"fitnessState,omitempty"`
	ChildRunName    string   `json:"childRunName,omitempty"`
	Phase           string   `json:"phase,omitempty"`
	JobID           string   `json:"jobId,omitempty"`
	PodName         string   `json:"podName,omitempty"`
}

type KrknAIScenarioPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type KrknAIScenarioIndexResponse struct {
	Scenarios  []KrknAIScenarioIndexItem `json:"scenarios"`
	Pagination KrknAIScenarioPagination  `json:"pagination"`
}

type KrknAIScenarioFitnessScore struct {
	ID              int      `json:"id"`
	FitnessScore    *float64 `json:"fitnessScore"`
	WeightedScore   *float64 `json:"weightedScore"`
	NormalizedScore *float64 `json:"normalizedScore"`
}

type KrknAIScenarioFitnessResult struct {
	FitnessScore                 *float64                     `json:"fitnessScore"`
	Scores                       []KrknAIScenarioFitnessScore `json:"scores"`
	HealthCheckFailureScore      *float64                     `json:"healthCheckFailureScore"`
	HealthCheckResponseTimeScore *float64                     `json:"healthCheckResponseTimeScore"`
	KrknFailureScore             *float64                     `json:"krknFailureScore"`
}

type KrknAIScenarioHealthCheck struct {
	Application         string   `json:"application"`
	Timestamp           string   `json:"timestamp"`
	ElapsedSeconds      *float64 `json:"elapsedSeconds,omitempty"`
	ResponseTimeSeconds *float64 `json:"responseTimeSeconds"`
	StatusCode          *int     `json:"statusCode"`
	Success             *bool    `json:"success"`
	Error               string   `json:"error,omitempty"`
}

type KrknAIScenarioDetailResponse struct {
	Generation      int                         `json:"generation"`
	ScenarioID      string                      `json:"scenarioId"`
	ScenarioType    string                      `json:"scenarioType"`
	Parameters      any                         `json:"parameters"`
	Command         string                      `json:"command"`
	Origin          string                      `json:"origin"`
	ParentIDs       []string                    `json:"parentIds"`
	DurationSeconds *float64                    `json:"durationSeconds"`
	ReturnCode      *int                        `json:"returnCode"`
	FitnessResult   KrknAIScenarioFitnessResult `json:"fitnessResult"`
	HealthChecks    []KrknAIScenarioHealthCheck `json:"healthChecks"`
	LogPath         string                      `json:"logPath"`
	FitnessState    string                      `json:"fitnessState"`
}
