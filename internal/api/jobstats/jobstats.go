// Package jobstats provides aggregate job statistics computation for the krkn-operator API.
package jobstats

// Summary contains aggregate job statistics computed across all runs.
type Summary struct {
	TotalJobs     int `json:"totalJobs"`
	SucceededJobs int `json:"succeededJobs"`
	FailedJobs    int `json:"failedJobs"`
}

// JobCounter provides the per-item counts needed for stats aggregation.
type JobCounter interface {
	JobType() string
	ScenarioSucceeded() int
	ScenarioFailed() int
	ScenarioRunning() int
	ScenarioTotalTargets() int
	GraphTotal() int
	GraphCompleted() int
	GraphFailed() int
}

// Compute aggregates job statistics from a slice of JobCounter items.
func Compute[T JobCounter](jobs []T) Summary {
	var s Summary
	for _, job := range jobs {
		switch job.JobType() {
		case "scenarioRun":
			knownJobs := job.ScenarioSucceeded() + job.ScenarioFailed() + job.ScenarioRunning()
			total := knownJobs
			if tt := job.ScenarioTotalTargets(); tt > total {
				total = tt
			}
			s.TotalJobs += total
			s.SucceededJobs += job.ScenarioSucceeded()
			s.FailedJobs += job.ScenarioFailed()
		case "graphRun":
			s.TotalJobs += job.GraphTotal()
			s.SucceededJobs += job.GraphCompleted()
			s.FailedJobs += job.GraphFailed()
		}
	}
	return s
}
