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
	"errors"
	"sort"

	"sigs.k8s.io/controller-runtime/pkg/client"

	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"github.com/krkn-chaos/krkn-operator/internal/mcpserver"
	"github.com/krkn-chaos/krkn-operator/pkg/workflows"
)

// graphRunVisible resolves a graph run's target URLs and applies the
// mixed-cluster filter. A run whose target request cannot be resolved is not
// visible (fail closed) and only logged at V(1).
func (b *mcpBackend) graphRunVisible(ctx context.Context, p mcpserver.Principal, gr *krknv1alpha1.KrknGraphRun) (mcpserver.GraphRunOutput, bool) {
	urls, err := b.h.graphTargetURLs(ctx, gr)
	if err != nil {
		logMCP(ctx, "graph run target request unresolvable, hiding run", "graphRun", gr.Name)
		return mcpserver.GraphRunOutput{}, false
	}
	perms, err := b.h.robotClusterPerms(ctx, p.RobotName)
	if err != nil {
		return mcpserver.GraphRunOutput{}, false
	}
	return filterGraphRun(gr, urls, perms)
}

// ListGraphRuns lists visible graph runs with filters and pagination. List
// items omit Nodes, Definition and ResiliencyScores.
func (b *mcpBackend) ListGraphRuns(ctx context.Context, p mcpserver.Principal, in mcpserver.ListGraphRunsInput) (mcpserver.ListGraphRunsOutput, error) {
	list := &krknv1alpha1.KrknGraphRunList{}
	if err := b.h.client.List(ctx, list, client.InNamespace(b.h.namespace)); err != nil {
		return mcpserver.ListGraphRunsOutput{}, errMCPInternal()
	}

	runs := make([]mcpserver.GraphRunOutput, 0, len(list.Items))
	for i := range list.Items {
		gr := &list.Items[i]
		if in.Phase != "" && gr.Status.Phase != in.Phase {
			continue
		}
		if in.OwnedByMe && gr.Spec.OwnerUserID != "robot:"+p.RobotName {
			continue
		}
		out, ok := b.graphRunVisible(ctx, p, gr)
		if !ok {
			continue
		}
		// List items are lightweight: drop the heavy fields.
		out.Nodes = nil
		out.Definition = nil
		out.ResiliencyScores = nil
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

	return mcpserver.ListGraphRunsOutput{
		GraphRuns: runs[offset:end],
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}, nil
}

// GetGraphRun returns one visible graph run in full.
func (b *mcpBackend) GetGraphRun(ctx context.Context, p mcpserver.Principal, in mcpserver.GraphRunRefInput) (mcpserver.GraphRunOutput, error) {
	gr := &krknv1alpha1.KrknGraphRun{}
	if err := b.h.client.Get(ctx, client.ObjectKey{Name: in.GraphRunName, Namespace: b.h.namespace}, gr); err != nil {
		return mcpserver.GraphRunOutput{}, errMCPNotFound("graph run %q not found", in.GraphRunName)
	}
	out, ok := b.graphRunVisible(ctx, p, gr)
	if !ok {
		return mcpserver.GraphRunOutput{}, errMCPNotFound("graph run %q not found", in.GraphRunName)
	}
	out.Definition = redactGraphNodes(gr.Spec.Graph)
	return out, nil
}

// ListWorkflows dispatches GET /api/v1/workflows/available as the robot.
func (b *mcpBackend) ListWorkflows(ctx context.Context, p mcpserver.Principal, _ struct{}) (mcpserver.ListWorkflowsOutput, error) {
	res, err := b.h.dispatchAsRobot(ctx, p.RobotName, "GET", WorkflowsAvailablePath, nil, nil, nil, b.h.ListAvailableWorkflows)
	if err != nil {
		return mcpserver.ListWorkflowsOutput{}, errMCPInternal()
	}
	if err := backendErrorFromDispatch(res); err != nil {
		return mcpserver.ListWorkflowsOutput{}, err
	}
	var resp workflows.AvailableWorkflowsResponse
	if err := decodeJSON(res.Body, &resp); err != nil {
		return mcpserver.ListWorkflowsOutput{}, errMCPInternal()
	}
	out := mcpserver.ListWorkflowsOutput{Workflows: make([]mcpserver.WorkflowSummaryOutput, 0, len(resp.Workflows))}
	for _, wf := range resp.Workflows {
		out.Workflows = append(out.Workflows, mcpserver.WorkflowSummaryOutput{
			WorkflowID:   wf.WorkflowID,
			WorkflowName: wf.WorkflowName,
			Description:  wf.Description,
			NodeCount:    wf.NodeCount,
			Categories:   wf.Categories,
		})
	}
	return out, nil
}

// GetWorkflow dispatches GET /api/v1/workflows/{id} as the robot and redacts
// the graph. 404 and 403 from dispatch both map to not_found so workflow
// existence does not leak.
func (b *mcpBackend) GetWorkflow(ctx context.Context, p mcpserver.Principal, in mcpserver.WorkflowRefInput) (mcpserver.WorkflowOutput, error) {
	res, err := b.h.dispatchAsRobot(ctx, p.RobotName, "GET", WorkflowsPath+"/"+in.WorkflowID, nil, nil, nil, b.h.GetWorkflow)
	if err != nil {
		return mcpserver.WorkflowOutput{}, errMCPInternal()
	}
	dispatchErr := backendErrorFromDispatch(res)
	if dispatchErr != nil {
		var be *mcpserver.BackendError
		if errors.As(dispatchErr, &be) && (be.Code == mcpserver.CodeNotFound || be.Code == mcpserver.CodeForbidden) {
			return mcpserver.WorkflowOutput{}, errMCPNotFound("workflow %q not found", in.WorkflowID)
		}
		return mcpserver.WorkflowOutput{}, dispatchErr
	}

	var resp workflows.WorkflowResponse
	if err := decodeJSON(res.Body, &resp); err != nil {
		return mcpserver.WorkflowOutput{}, errMCPInternal()
	}
	return mcpserver.WorkflowOutput{
		WorkflowID:   resp.WorkflowID,
		WorkflowName: resp.WorkflowName,
		Description:  resp.Description,
		Categories:   resp.Categories,
		Nodes:        redactGraphNodes(resp.Graph),
	}, nil
}
