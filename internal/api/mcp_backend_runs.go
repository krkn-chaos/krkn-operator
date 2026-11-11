/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/mcpserver"
	"github.com/krkn-chaos/krkn-operator/pkg/cloudcreds"
	"github.com/krkn-chaos/krkn-operator/pkg/files"
	"github.com/krkn-chaos/krkn-operator/pkg/groupauth"
	"github.com/krkn-chaos/krkn-operator/pkg/registry"
)

// Cluster discovery polling, overridable in tests.
var (
	mcpTargetPollInterval = time.Second
	mcpTargetPollTimeout  = 20 * time.Second
)

// mcpTargetRequestMaxAge bounds how old a reused robot-owned target request
// may be.
const mcpTargetRequestMaxAge = 10 * time.Minute

// job log bounds.
const (
	mcpLogDefaultTail = 200
	mcpLogMaxTail     = 1000
	mcpLogLimitBytes  = 262144
)

// errMCPNotFound builds a client-safe not_found error.
func errMCPNotFound(format string, args ...any) error {
	return &mcpserver.BackendError{Code: mcpserver.CodeNotFound, Message: fmt.Sprintf(format, args...)}
}

// errMCPInvalidArg builds a client-safe invalid_argument error.
func errMCPInvalidArg(format string, args ...any) error {
	return &mcpserver.BackendError{Code: mcpserver.CodeInvalidArg, Message: fmt.Sprintf(format, args...)}
}

// errMCPInternal builds a client-safe internal error.
func errMCPInternal() error {
	return &mcpserver.BackendError{Code: mcpserver.CodeInternal, Message: "internal error"}
}

// ListClusters discovers target clusters for the robot.
func (b *mcpBackend) ListClusters(ctx context.Context, p mcpserver.Principal, in mcpserver.ListClustersInput) (mcpserver.ListClustersOutput, error) {
	h := b.h
	trID := in.TargetRequestID

	if trID == "" {
		reused, err := b.recentRobotTargetRequest(ctx, p.RobotName)
		if err != nil {
			return mcpserver.ListClustersOutput{}, errMCPInternal()
		}
		if reused != nil {
			trID = reused.Name
		} else {
			created, err := b.createRobotTargetRequest(ctx, p.RobotName)
			if err != nil {
				return mcpserver.ListClustersOutput{}, err
			}
			trID = created
		}
	}

	tr := &krknv1alpha1.KrknTargetRequest{}
	if err := h.client.Get(ctx, client.ObjectKey{Name: trID, Namespace: h.namespace}, tr); err != nil {
		if apierrors.IsNotFound(err) {
			return mcpserver.ListClustersOutput{}, errMCPNotFound("target request %q not found", trID)
		}
		return mcpserver.ListClustersOutput{}, errMCPInternal()
	}

	if tr.Status.Status != "Completed" {
		completed := b.pollTargetRequest(ctx, trID)
		if completed == nil {
			return mcpserver.ListClustersOutput{TargetRequestID: trID, Status: "Pending"}, nil
		}
		// Use the fetched object: the local tr predates completion and has no
		// TargetData, which would yield an empty cluster list.
		tr = completed
	}

	return b.buildClusterList(ctx, p, tr)
}

// recentRobotTargetRequest returns the newest completed robot-owned target
// request within the reuse window, or nil.
func (b *mcpBackend) recentRobotTargetRequest(ctx context.Context, robotName string) (*krknv1alpha1.KrknTargetRequest, error) {
	ownerLabel, err := robotOwnerLabel(robotName)
	if err != nil {
		return nil, err
	}
	list := &krknv1alpha1.KrknTargetRequestList{}
	if err := b.h.client.List(ctx, list,
		client.InNamespace(b.h.namespace),
		client.MatchingLabels{"krkn.krkn-chaos.dev/owner-user": ownerLabel},
	); err != nil {
		return nil, err
	}

	var newest *krknv1alpha1.KrknTargetRequest
	for i := range list.Items {
		tr := &list.Items[i]
		if tr.Status.Status != "Completed" {
			continue
		}
		if time.Since(tr.CreationTimestamp.Time) > mcpTargetRequestMaxAge {
			continue
		}
		if newest == nil || tr.CreationTimestamp.After(newest.CreationTimestamp.Time) {
			newest = tr
		}
	}
	return newest, nil
}

// createRobotTargetRequest dispatches POST /api/v1/targets as the robot.
func (b *mcpBackend) createRobotTargetRequest(ctx context.Context, robotName string) (string, error) {
	res, err := b.h.dispatchAsRobot(ctx, robotName, "POST", TargetsPath, nil, nil, nil, b.h.PostTarget)
	if err != nil {
		return "", errMCPInternal()
	}
	if err := backendErrorFromDispatch(res); err != nil {
		return "", err
	}
	var created struct {
		UUID string `json:"uuid"`
	}
	if err := decodeJSON(res.Body, &created); err != nil || created.UUID == "" {
		return "", errMCPInternal()
	}
	return created.UUID, nil
}

// pollTargetRequest waits for a target request to complete, respecting ctx.
// It returns the completed KrknTargetRequest (including its TargetData), or
// nil if the deadline or context expired first.
func (b *mcpBackend) pollTargetRequest(ctx context.Context, trID string) *krknv1alpha1.KrknTargetRequest {
	deadline := time.Now().Add(mcpTargetPollTimeout)
	for {
		tr := &krknv1alpha1.KrknTargetRequest{}
		if err := b.h.client.Get(ctx, client.ObjectKey{Name: trID, Namespace: b.h.namespace}, tr); err == nil && tr.Status.Status == "Completed" {
			return tr
		}
		if time.Now().After(deadline) {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(mcpTargetPollInterval):
		}
	}
}

// buildClusterList filters target request clusters by the robot's actions.
func (b *mcpBackend) buildClusterList(ctx context.Context, p mcpserver.Principal, tr *krknv1alpha1.KrknTargetRequest) (mcpserver.ListClustersOutput, error) {
	perms, err := b.h.robotClusterPerms(ctx, p.RobotName)
	if err != nil {
		return mcpserver.ListClustersOutput{}, errMCPInternal()
	}

	out := mcpserver.ListClustersOutput{TargetRequestID: tr.Name, Status: "Completed"}
	for providerName, clusters := range tr.Status.TargetData {
		for _, cluster := range clusters {
			actions := visibleActions(perms, cluster.ClusterAPIURL)
			if len(actions) == 0 {
				continue
			}
			online := cluster.Online
			out.Clusters = append(out.Clusters, mcpserver.ClusterOutput{
				ProviderName:  providerName,
				ClusterName:   cluster.ClusterName,
				ClusterAPIURL: cluster.ClusterAPIURL,
				Online:        online,
				Actions:       actions,
			})
		}
	}
	sort.Slice(out.Clusters, func(i, j int) bool {
		if out.Clusters[i].ClusterName != out.Clusters[j].ClusterName {
			return out.Clusters[i].ClusterName < out.Clusters[j].ClusterName
		}
		return out.Clusters[i].ProviderName < out.Clusters[j].ProviderName
	})
	return out, nil
}

// visibleActions returns the sorted action names the robot has on a URL.
func visibleActions(perms clusterPerms, url string) []string {
	actions := perms[url]
	if len(actions) == 0 {
		return nil
	}
	names := make([]string, 0, len(actions))
	for action := range actions {
		names = append(names, string(action))
	}
	sort.Strings(names)
	return names
}

// robotOwnerLabel builds the owner label value for a robot, with validation.
func robotOwnerLabel(robotName string) (string, error) {
	return groupauth.SanitizeUserIDForLabel("robot:" + robotName)
}

// ListScenarioRuns lists visible runs with filters and pagination.
func (b *mcpBackend) ListScenarioRuns(ctx context.Context, p mcpserver.Principal, in mcpserver.ListScenarioRunsInput) (mcpserver.ListScenarioRunsOutput, error) {
	h := b.h
	perms, err := h.robotClusterPerms(ctx, p.RobotName)
	if err != nil {
		return mcpserver.ListScenarioRunsOutput{}, errMCPInternal()
	}

	list := &krknv1alpha1.KrknScenarioRunList{}
	if err := h.client.List(ctx, list, client.InNamespace(h.namespace)); err != nil {
		return mcpserver.ListScenarioRunsOutput{}, errMCPInternal()
	}

	runs := make([]mcpserver.ScenarioRunOutput, 0, len(list.Items))
	for i := range list.Items {
		run := &list.Items[i]
		if in.Phase != "" && run.Status.Phase != in.Phase {
			continue
		}
		if in.GraphRunName != "" && run.Labels["krkn.dev/graph-run"] != in.GraphRunName {
			continue
		}
		if in.OwnedByMe && run.Spec.OwnerUserID != "robot:"+p.RobotName {
			continue
		}
		urls := b.scenarioRunURLsOrEmpty(ctx, run)
		out, ok := filterScenarioRun(run, perms, false, urls)
		if !ok {
			continue
		}
		runs = append(runs, out)
	}

	sort.Slice(runs, func(i, j int) bool { return runs[i].CreatedAt.After(runs[j].CreatedAt) })

	limit, offset := clampPage(in.Limit, in.Offset)
	total := len(runs)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	return mcpserver.ListScenarioRunsOutput{
		Runs:   runs[offset:end],
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// clampPage applies the documented pagination rules.
func clampPage(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// getVisibleScenarioRun loads a run and requires view on any target.
func (b *mcpBackend) getVisibleScenarioRun(ctx context.Context, p mcpserver.Principal, name string) (*krknv1alpha1.KrknScenarioRun, error) {
	run := &krknv1alpha1.KrknScenarioRun{}
	if err := b.h.client.Get(ctx, client.ObjectKey{Name: name, Namespace: b.h.namespace}, run); err != nil {
		return nil, errMCPNotFound("scenario run %q not found", name)
	}
	perms, err := b.h.robotClusterPerms(ctx, p.RobotName)
	if err != nil {
		return nil, errMCPInternal()
	}
	if _, ok := filterScenarioRun(run, perms, false, b.scenarioRunURLsOrEmpty(ctx, run)); !ok {
		return nil, errMCPNotFound("scenario run %q not found", name)
	}
	return run, nil
}

// scenarioRunURLs resolves a run's target cluster URLs via its target
// request. Each cluster name is resolved within its own provider's
// TargetData slice (see resolveTargetClusterURLs); a missing request or an
// ambiguous cluster name is an error so callers fail closed.
func (b *mcpBackend) scenarioRunURLs(ctx context.Context, run *krknv1alpha1.KrknScenarioRun) (map[string]string, error) {
	tr := &krknv1alpha1.KrknTargetRequest{}
	if err := b.h.client.Get(ctx, client.ObjectKey{Name: run.Spec.TargetRequestID, Namespace: b.h.namespace}, tr); err != nil {
		return nil, fmt.Errorf("target request %q not found", run.Spec.TargetRequestID)
	}
	return resolveTargetClusterURLs(run.Spec.TargetClusters, tr)
}

// scenarioRunURLsOrEmpty is scenarioRunURLs with a fail-closed empty result.
func (b *mcpBackend) scenarioRunURLsOrEmpty(ctx context.Context, run *krknv1alpha1.KrknScenarioRun) map[string]string {
	urls, err := b.scenarioRunURLs(ctx, run)
	if err != nil {
		return nil
	}
	return urls
}

// GetScenarioRun returns one visible run with jobs.
func (b *mcpBackend) GetScenarioRun(ctx context.Context, p mcpserver.Principal, in mcpserver.ScenarioRunRefInput) (mcpserver.ScenarioRunOutput, error) {
	run, err := b.getVisibleScenarioRun(ctx, p, in.ScenarioRunName)
	if err != nil {
		return mcpserver.ScenarioRunOutput{}, err
	}
	perms, err := b.h.robotClusterPerms(ctx, p.RobotName)
	if err != nil {
		return mcpserver.ScenarioRunOutput{}, errMCPInternal()
	}
	out, ok := filterScenarioRun(run, perms, true, b.scenarioRunURLsOrEmpty(ctx, run))
	if !ok {
		return mcpserver.ScenarioRunOutput{}, errMCPNotFound("scenario run %q not found", in.ScenarioRunName)
	}
	return out, nil
}

// GetJobLogs returns bounded logs of a visible job.
func (b *mcpBackend) GetJobLogs(ctx context.Context, p mcpserver.Principal, in mcpserver.JobLogsInput) (mcpserver.JobLogsOutput, error) {
	run, err := b.getVisibleScenarioRun(ctx, p, in.ScenarioRunName)
	if err != nil {
		return mcpserver.JobLogsOutput{}, err
	}
	perms, err := b.h.robotClusterPerms(ctx, p.RobotName)
	if err != nil {
		return mcpserver.JobLogsOutput{}, errMCPInternal()
	}

	var job *krknv1alpha1.ClusterJobStatus
	for i := range run.Status.ClusterJobs {
		if run.Status.ClusterJobs[i].JobID == in.JobID {
			job = &run.Status.ClusterJobs[i]
			break
		}
	}
	if job == nil || job.ClusterAPIURL == "" || !perms.can(job.ClusterAPIURL, "view") {
		return mcpserver.JobLogsOutput{}, errMCPNotFound("job %q not found", in.JobID)
	}
	if job.PodName == "" {
		return mcpserver.JobLogsOutput{}, &mcpserver.BackendError{Code: mcpserver.CodeUnavailable, Message: "logs are not available for this job"}
	}

	if in.TailLines < 0 {
		return mcpserver.JobLogsOutput{}, errMCPInvalidArg("tailLines must not be negative")
	}
	tail := in.TailLines
	if tail == 0 {
		tail = mcpLogDefaultTail
	}
	if tail > mcpLogMaxTail {
		tail = mcpLogMaxTail
	}
	limitBytes := int64(mcpLogLimitBytes)

	// Fetch WITHOUT TailLines and filter report blocks BEFORE applying the
	// tail, mirroring the REST log path (handlers.go): the scenario container
	// prints the base64 report payload between ===KRKN_REPORT_*=== markers as
	// its last output, and a Kubernetes-side tail window starting mid-block
	// would leak that encoded content.
	logReq := b.h.clientset.CoreV1().Pods(b.h.namespace).GetLogs(job.PodName, &corev1.PodLogOptions{
		Container:  "scenario",
		LimitBytes: &limitBytes,
	})
	raw, err := logReq.DoRaw(ctx)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return mcpserver.JobLogsOutput{}, &mcpserver.BackendError{Code: mcpserver.CodeUnavailable, Message: "logs are not available for this job"}
		}
		return mcpserver.JobLogsOutput{}, errMCPInternal()
	}

	logs := tailFilteredLogs(string(raw), tail)

	return mcpserver.JobLogsOutput{
		JobID:     job.JobID,
		PodName:   job.PodName,
		Truncated: len(raw) >= mcpLogLimitBytes,
		Logs:      logs,
	}, nil
}

// tailFilteredLogs strips KRKN report blocks and then keeps the last tail
// lines. The filtering order matches the REST handler's completed-stream path
// (handlers.go, filterReportLines).
func tailFilteredLogs(raw string, tail int) string {
	lines := strings.Split(raw, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	filtered := filterReportLines(lines)
	if len(filtered) > tail {
		filtered = filtered[len(filtered)-tail:]
	}
	if len(filtered) == 0 {
		return ""
	}
	return strings.Join(filtered, "\n") + "\n"
}

// GetScenarioRunConfig returns the redacted configuration of a visible run.
// Like the REST handler (run_config_handlers.go GetScenarioRunConfig), it
// requires run permission on ALL target clusters, not merely view on a
// visible one, because the config exposes replay-ready material.
func (b *mcpBackend) GetScenarioRunConfig(ctx context.Context, p mcpserver.Principal, in mcpserver.ScenarioRunRefInput) (mcpserver.ScenarioRunConfigOutput, error) {
	run, err := b.getVisibleScenarioRun(ctx, p, in.ScenarioRunName)
	if err != nil {
		return mcpserver.ScenarioRunConfigOutput{}, err
	}

	// Mirror REST: groupauth.ValidateScenarioRunAccess with the run's own
	// target request. Any denial (including an unresolvable target request or
	// a cluster-name collision) fails closed with a generic forbidden message
	// that does not leak cluster API URLs.
	tr := &krknv1alpha1.KrknTargetRequest{}
	if err := b.h.client.Get(ctx, client.ObjectKey{Name: run.Spec.TargetRequestID, Namespace: b.h.namespace}, tr); err != nil {
		return mcpserver.ScenarioRunConfigOutput{}, &mcpserver.BackendError{Code: mcpserver.CodeForbidden, Message: "permission denied"}
	}
	if err := groupauth.ValidateScenarioRunAccess(ctx, b.h.client, "robot:"+p.RobotName, b.h.namespace, run.Spec.TargetClusters, tr); err != nil {
		return mcpserver.ScenarioRunConfigOutput{}, &mcpserver.BackendError{Code: mcpserver.CodeForbidden, Message: "permission denied"}
	}

	perms, err := b.h.robotClusterPerms(ctx, p.RobotName)
	if err != nil {
		return mcpserver.ScenarioRunConfigOutput{}, errMCPInternal()
	}
	// The cloud credential reference is only disclosed when the robot can
	// actually use it, mirroring replayableCloudCredentialRef in REST replay
	// responses.
	cloudCredRef := b.h.replayableCloudCredentialRef(withRobotClaims(ctx, p.RobotName), run.Spec.CloudCredentialRef)
	return redactScenarioConfig(run, perms, b.scenarioRunURLsOrEmpty(ctx, run), cloudCredRef), nil
}

// GetReportStatus returns report generation status; Location is never included.
func (b *mcpBackend) GetReportStatus(ctx context.Context, p mcpserver.Principal, in mcpserver.ScenarioRunRefInput) (mcpserver.ReportStatusOutput, error) {
	run, err := b.getVisibleScenarioRun(ctx, p, in.ScenarioRunName)
	if err != nil {
		return mcpserver.ReportStatusOutput{}, err
	}

	out := mcpserver.ReportStatusOutput{
		ScenarioRunName: in.ScenarioRunName,
		Message:         "report not generated",
	}
	if run.Status.ReportStatus != nil {
		rs := run.Status.ReportStatus
		out.Generated = rs.Generated
		out.HTMLAvailable = rs.HTMLAvailable
		out.PDFAvailable = rs.PDFAvailable
		if rs.GeneratedAt != nil {
			t := rs.GeneratedAt.Time
			out.GeneratedAt = &t
		}
		if rs.Message != "" {
			out.Message = rs.Message
		}
	}
	return out, nil
}

// ListScenarios dispatches POST /api/v1/scenarios for the catalog.
func (b *mcpBackend) ListScenarios(ctx context.Context, p mcpserver.Principal, in mcpserver.ListScenariosInput) (mcpserver.ListScenariosOutput, error) {
	res, err := b.dispatchScenariosRequest(ctx, p.RobotName, ScenariosPath, in.RegistryName)
	if err != nil {
		return mcpserver.ListScenariosOutput{}, err
	}
	var resp ScenariosResponse
	if err := decodeJSON(res.Body, &resp); err != nil {
		return mcpserver.ListScenariosOutput{}, errMCPInternal()
	}
	out := mcpserver.ListScenariosOutput{Scenarios: make([]mcpserver.ScenarioSummaryOutput, 0, len(resp.Scenarios))}
	for _, tag := range resp.Scenarios {
		out.Scenarios = append(out.Scenarios, mcpserver.ScenarioSummaryOutput{
			Name:            tag.Name,
			SignatureStatus: tag.SignatureStatus,
			LastModified:    tag.LastModified,
		})
	}
	return out, nil
}

// GetScenarioDetail dispatches POST /api/v1/scenarios/detail/{name}.
func (b *mcpBackend) GetScenarioDetail(ctx context.Context, p mcpserver.Principal, in mcpserver.ScenarioDetailInput) (mcpserver.ScenarioDetailOutput, error) {
	res, err := b.dispatchScenarioDetail(ctx, p.RobotName, in.ScenarioName, in.RegistryName)
	if err != nil {
		return mcpserver.ScenarioDetailOutput{}, err
	}
	var resp ScenarioDetailResponse
	if err := decodeJSON(res.Body, &resp); err != nil {
		return mcpserver.ScenarioDetailOutput{}, errMCPInternal()
	}
	return scenarioDetailToOutput(resp), nil
}

// scenarioDetailToOutput maps the REST response to the MCP output type.
func scenarioDetailToOutput(resp ScenarioDetailResponse) mcpserver.ScenarioDetailOutput {
	out := mcpserver.ScenarioDetailOutput{
		Name:            resp.Name,
		Digest:          resp.Digest,
		Size:            resp.Size,
		SignatureStatus: resp.SignatureStatus,
		LastModified:    resp.LastModified,
		Title:           resp.Title,
		Description:     resp.Description,
		Fields:          make([]mcpserver.ScenarioInputFieldOutput, 0, len(resp.Fields)),
	}
	for _, f := range resp.Fields {
		out.Fields = append(out.Fields, mcpserver.ScenarioInputFieldOutput{
			Name: f.Name, ShortDescription: f.ShortDescription, Description: f.Description,
			Variable: f.Variable, Type: f.Type, Default: f.Default, Validator: f.Validator,
			ValidationMessage: f.ValidationMessage, Separator: f.Separator, AllowedValues: f.AllowedValues,
			Required: f.Required, MountPath: f.MountPath, Requires: f.Requires,
			MutuallyExcludes: f.MutuallyExcludes, Secret: f.Secret, Group: f.Group,
		})
	}
	return out
}

// dispatchScenariosRequest dispatches a scenarios catalog request as robot.
func (b *mcpBackend) dispatchScenariosRequest(ctx context.Context, robotName, path, registryName string) (dispatchResult, error) {
	req := ScenariosRequest{}
	if registryName != "" {
		name := registryName
		req.RegistryName = &name
	}
	res, err := b.h.dispatchAsRobot(ctx, robotName, "POST", path, nil, req, nil, b.h.PostScenarios)
	if err != nil {
		return dispatchResult{}, errMCPInternal()
	}
	if err := backendErrorFromDispatch(res); err != nil {
		return dispatchResult{}, err
	}
	return res, nil
}

// dispatchScenarioDetail dispatches the detail request as robot.
func (b *mcpBackend) dispatchScenarioDetail(ctx context.Context, robotName, scenarioName, registryName string) (dispatchResult, error) {
	req := ScenariosRequest{}
	if registryName != "" {
		name := registryName
		req.RegistryName = &name
	}
	path := ScenariosDetailPath + "/" + scenarioName
	res, err := b.h.dispatchAsRobot(ctx, robotName, "POST", path, nil, req, nil, b.h.PostScenarioDetail)
	if err != nil {
		return dispatchResult{}, errMCPInternal()
	}
	if err := backendErrorFromDispatch(res); err != nil {
		return dispatchResult{}, err
	}
	return res, nil
}

// ListAvailableResources lists registries, cloud credentials and files
// available to the robot, copying only non-secret fields.
func (b *mcpBackend) ListAvailableResources(ctx context.Context, p mcpserver.Principal, _ struct{}) (mcpserver.AvailableResourcesOutput, error) {
	registries, err := b.listAvailableRegistries(ctx, p.RobotName)
	if err != nil {
		return mcpserver.AvailableResourcesOutput{}, err
	}
	creds, err := b.listAvailableCloudCredentials(ctx, p.RobotName)
	if err != nil {
		return mcpserver.AvailableResourcesOutput{}, err
	}
	filesOut, err := b.listAvailableFiles(ctx, p.RobotName)
	if err != nil {
		return mcpserver.AvailableResourcesOutput{}, err
	}
	return mcpserver.AvailableResourcesOutput{
		Registries:       registries,
		CloudCredentials: creds,
		Files:            filesOut,
	}, nil
}

// listAvailableRegistries dispatches the available-registries handler.
func (b *mcpBackend) listAvailableRegistries(ctx context.Context, robotName string) ([]mcpserver.NamedResourceOutput, error) {
	res, err := b.h.dispatchAsRobot(ctx, robotName, "GET", RegistriesAvailablePath, nil, nil, nil, b.h.ListAvailableRegistries)
	if err != nil {
		return nil, errMCPInternal()
	}
	if err := backendErrorFromDispatch(res); err != nil {
		return nil, err
	}
	var resp registry.AvailableRegistriesResponse
	if err := decodeJSON(res.Body, &resp); err != nil {
		return nil, errMCPInternal()
	}
	out := make([]mcpserver.NamedResourceOutput, 0, len(resp.Registries))
	for _, r := range resp.Registries {
		out = append(out, mcpserver.NamedResourceOutput{Name: r.Name, Description: r.Description})
	}
	return out, nil
}

// listAvailableCloudCredentials dispatches the available-credentials handler.
func (b *mcpBackend) listAvailableCloudCredentials(ctx context.Context, robotName string) ([]mcpserver.NamedResourceOutput, error) {
	res, err := b.h.dispatchAsRobot(ctx, robotName, "GET", CloudCredentialsAvailablePath, nil, nil, nil, b.h.ListAvailableCloudCredentials)
	if err != nil {
		return nil, errMCPInternal()
	}
	if err := backendErrorFromDispatch(res); err != nil {
		return nil, err
	}
	var resp cloudcreds.ListCloudCredentialsResponse
	if err := decodeJSON(res.Body, &resp); err != nil {
		return nil, errMCPInternal()
	}
	out := make([]mcpserver.NamedResourceOutput, 0, len(resp.Credentials))
	for _, c := range resp.Credentials {
		out = append(out, mcpserver.NamedResourceOutput{Name: c.Name, Description: c.Description})
	}
	return out, nil
}

// listAvailableFiles dispatches the available-files handler.
func (b *mcpBackend) listAvailableFiles(ctx context.Context, robotName string) ([]mcpserver.FileResourceOutput, error) {
	res, err := b.h.dispatchAsRobot(ctx, robotName, "GET", FilesAvailablePath, nil, nil, nil, b.h.ListAvailableFiles)
	if err != nil {
		return nil, errMCPInternal()
	}
	if err := backendErrorFromDispatch(res); err != nil {
		return nil, err
	}
	var resp files.AvailableFilesResponse
	if err := decodeJSON(res.Body, &resp); err != nil {
		return nil, errMCPInternal()
	}
	out := make([]mcpserver.FileResourceOutput, 0, len(resp.Files))
	for _, f := range resp.Files {
		out = append(out, mcpserver.FileResourceOutput{FileID: f.FileID, Name: f.FileName, Purpose: f.FilePurpose})
	}
	return out, nil
}

// decodeJSON decodes a dispatch body.
func decodeJSON(body []byte, target any) error {
	if len(body) == 0 {
		return fmt.Errorf("empty body")
	}
	return json.Unmarshal(body, target)
}

// logMCP is a small helper for V(1) logging.
func logMCP(ctx context.Context, msg string, keysAndValues ...any) {
	log.FromContext(ctx).V(1).Info(msg, keysAndValues...)
}
