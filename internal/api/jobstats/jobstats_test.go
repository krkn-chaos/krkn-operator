package jobstats

import "testing"

type mockJob struct {
	jobType              string
	scenarioSucc         int
	scenarioFail         int
	scenarioRunning      int
	scenarioTotalTargets int
	graphTotal           int
	graphCompleted       int
	graphFailed          int
}

func (m mockJob) JobType() string            { return m.jobType }
func (m mockJob) ScenarioSucceeded() int     { return m.scenarioSucc }
func (m mockJob) ScenarioFailed() int        { return m.scenarioFail }
func (m mockJob) ScenarioRunning() int       { return m.scenarioRunning }
func (m mockJob) ScenarioTotalTargets() int  { return m.scenarioTotalTargets }
func (m mockJob) GraphTotal() int            { return m.graphTotal }
func (m mockJob) GraphCompleted() int        { return m.graphCompleted }
func (m mockJob) GraphFailed() int           { return m.graphFailed }

func TestCompute(t *testing.T) {
	tests := []struct {
		name     string
		jobs     []mockJob
		expected Summary
	}{
		{
			name:     "empty list",
			jobs:     nil,
			expected: Summary{},
		},
		{
			name: "scenario runs only",
			jobs: []mockJob{
				{jobType: "scenarioRun", scenarioSucc: 3, scenarioFail: 1, scenarioRunning: 2},
				{jobType: "scenarioRun", scenarioSucc: 1, scenarioFail: 0, scenarioRunning: 0},
			},
			expected: Summary{TotalJobs: 7, SucceededJobs: 4, FailedJobs: 1},
		},
		{
			name: "graph runs only",
			jobs: []mockJob{
				{jobType: "graphRun", graphTotal: 5, graphCompleted: 4, graphFailed: 1},
			},
			expected: Summary{TotalJobs: 5, SucceededJobs: 4, FailedJobs: 1},
		},
		{
			name: "mixed",
			jobs: []mockJob{
				{jobType: "scenarioRun", scenarioSucc: 2, scenarioFail: 1, scenarioRunning: 1},
				{jobType: "graphRun", graphTotal: 3, graphCompleted: 2, graphFailed: 1},
			},
			expected: Summary{TotalJobs: 7, SucceededJobs: 4, FailedJobs: 2},
		},
		{
			name: "scenario with pending jobs uses TotalTargets",
			jobs: []mockJob{
				{jobType: "scenarioRun", scenarioSucc: 1, scenarioFail: 0, scenarioRunning: 1, scenarioTotalTargets: 5},
			},
			expected: Summary{TotalJobs: 5, SucceededJobs: 1, FailedJobs: 0},
		},
		{
			name: "TotalTargets not used when less than known jobs",
			jobs: []mockJob{
				{jobType: "scenarioRun", scenarioSucc: 3, scenarioFail: 1, scenarioRunning: 1, scenarioTotalTargets: 3},
			},
			expected: Summary{TotalJobs: 5, SucceededJobs: 3, FailedJobs: 1},
		},
		{
			name: "unknown type ignored",
			jobs: []mockJob{
				{jobType: "unknown", scenarioSucc: 10, graphTotal: 10},
			},
			expected: Summary{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Compute(tc.jobs)
			if got != tc.expected {
				t.Errorf("Compute() = %+v, want %+v", got, tc.expected)
			}
		})
	}
}
