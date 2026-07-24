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

Assisted-by: Claude Sonnet 4.5 (claude-sonnet-4-5@20250929)
*/

package websocket

import (
	"context"
	"reflect"

	"github.com/go-logr/logr"
	krknv1alpha1 "github.com/krkn-chaos/krkn-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SetupWatchers configures Kubernetes informers to broadcast WebSocket updates
// This is called from main.go after the manager is created
func SetupWatchers(ctx context.Context, informerCache cache.Cache, broadcaster *Broadcaster) error {
	logger := log.FromContext(ctx).WithName("websocket-watcher")

	// Setup ScenarioRun watcher
	scenarioRunInformer, err := informerCache.GetInformer(ctx, &krknv1alpha1.KrknScenarioRun{})
	if err != nil {
		return err
	}

	_, err = scenarioRunInformer.AddEventHandler(&scenarioRunEventHandler{
		broadcaster: broadcaster,
		logger:      logger.WithValues("resource", "scenariorun"),
	})
	if err != nil {
		return err
	}

	// Setup GraphRun watcher
	graphRunInformer, err := informerCache.GetInformer(ctx, &krknv1alpha1.KrknGraphRun{})
	if err != nil {
		return err
	}

	_, err = graphRunInformer.AddEventHandler(&graphRunEventHandler{
		broadcaster: broadcaster,
		logger:      logger.WithValues("resource", "graphrun"),
	})
	if err != nil {
		return err
	}

	logger.Info("WebSocket watchers configured successfully")
	return nil
}

// scenarioRunEventHandler handles ScenarioRun events and broadcasts to WebSocket clients
type scenarioRunEventHandler struct {
	broadcaster *Broadcaster
	logger      logr.Logger
}

func (h *scenarioRunEventHandler) OnAdd(obj interface{}, _ bool) {
	run := obj.(*krknv1alpha1.KrknScenarioRun)
	h.logger.V(1).Info("ScenarioRun added", "name", run.Name)
	h.broadcaster.BroadcastScenarioRunUpdate(run)
}

func (h *scenarioRunEventHandler) OnUpdate(oldObj, newObj interface{}) {
	oldRun := oldObj.(*krknv1alpha1.KrknScenarioRun)
	newRun := newObj.(*krknv1alpha1.KrknScenarioRun)

	// Filter: only broadcast if status actually changed
	if !hasScenarioRunStatusChanged(oldRun, newRun) {
		h.logger.V(2).Info("ScenarioRun status unchanged, skipping broadcast",
			"name", newRun.Name,
			"phase", newRun.Status.Phase)
		return
	}

	h.logger.V(1).Info("ScenarioRun status changed, broadcasting",
		"name", newRun.Name,
		"oldPhase", oldRun.Status.Phase,
		"newPhase", newRun.Status.Phase,
		"runningJobs", newRun.Status.RunningJobs)

	h.broadcaster.BroadcastScenarioRunUpdate(newRun)
}

func (h *scenarioRunEventHandler) OnDelete(obj interface{}) {
	run := obj.(*krknv1alpha1.KrknScenarioRun)
	h.logger.V(1).Info("ScenarioRun deleted", "name", run.Name)
	h.broadcaster.BroadcastScenarioRunDeleted(run.Name)
}

// graphRunEventHandler handles GraphRun events and broadcasts to WebSocket clients
type graphRunEventHandler struct {
	broadcaster *Broadcaster
	logger      logr.Logger
}

func (h *graphRunEventHandler) OnAdd(obj interface{}, _ bool) {
	run := obj.(*krknv1alpha1.KrknGraphRun)
	h.logger.V(1).Info("GraphRun added", "name", run.Name)
	h.broadcaster.BroadcastGraphRunUpdate(run)
}

func (h *graphRunEventHandler) OnUpdate(oldObj, newObj interface{}) {
	oldRun := oldObj.(*krknv1alpha1.KrknGraphRun)
	newRun := newObj.(*krknv1alpha1.KrknGraphRun)

	// Filter: only broadcast if status actually changed
	if !hasGraphRunStatusChanged(oldRun, newRun) {
		h.logger.V(2).Info("GraphRun status unchanged, skipping broadcast",
			"name", newRun.Name,
			"phase", newRun.Status.Phase)
		return
	}

	h.logger.V(1).Info("GraphRun status changed, broadcasting",
		"name", newRun.Name,
		"oldPhase", oldRun.Status.Phase,
		"newPhase", newRun.Status.Phase,
		"completedNodes", newRun.Status.Summary.CompletedNodes)

	h.broadcaster.BroadcastGraphRunUpdate(newRun)
}

func (h *graphRunEventHandler) OnDelete(obj interface{}) {
	run := obj.(*krknv1alpha1.KrknGraphRun)
	h.logger.V(1).Info("GraphRun deleted", "name", run.Name)
	h.broadcaster.BroadcastGraphRunDeleted(run.Name)
}

// hasScenarioRunStatusChanged checks if ScenarioRun status has meaningful changes
// Returns true if any user-visible status field changed
func hasScenarioRunStatusChanged(old, new *krknv1alpha1.KrknScenarioRun) bool {
	// Phase change is most important
	if old.Status.Phase != new.Status.Phase {
		return true
	}

	// Job counts changed
	if old.Status.TotalTargets != new.Status.TotalTargets ||
		old.Status.RunningJobs != new.Status.RunningJobs ||
		old.Status.SuccessfulJobs != new.Status.SuccessfulJobs ||
		old.Status.FailedJobs != new.Status.FailedJobs {
		return true
	}

	// ClusterJobs array changed (length or individual job status)
	if len(old.Status.ClusterJobs) != len(new.Status.ClusterJobs) {
		return true
	}

	for i := range old.Status.ClusterJobs {
		oldJob := &old.Status.ClusterJobs[i]
		newJob := &new.Status.ClusterJobs[i]

		if oldJob.Phase != newJob.Phase ||
			oldJob.RetryCount != newJob.RetryCount ||
			oldJob.CancelRequested != newJob.CancelRequested {
			return true
		}
	}

	return false
}

// hasGraphRunStatusChanged checks if GraphRun status has meaningful changes
// Returns true if any user-visible status field changed
func hasGraphRunStatusChanged(old, new *krknv1alpha1.KrknGraphRun) bool {
	// Phase change is most important
	if old.Status.Phase != new.Status.Phase {
		return true
	}

	// Summary counters changed
	if old.Status.Summary.TotalNodes != new.Status.Summary.TotalNodes ||
		old.Status.Summary.CompletedNodes != new.Status.Summary.CompletedNodes ||
		old.Status.Summary.RunningNodes != new.Status.Summary.RunningNodes ||
		old.Status.Summary.FailedNodes != new.Status.Summary.FailedNodes ||
		old.Status.Summary.PendingNodes != new.Status.Summary.PendingNodes {
		return true
	}

	// NodeStatuses array changed (length or individual node status)
	if len(old.Status.NodeStatuses) != len(new.Status.NodeStatuses) {
		return true
	}

	for i := range old.Status.NodeStatuses {
		oldNode := &old.Status.NodeStatuses[i]
		newNode := &new.Status.NodeStatuses[i]

		if oldNode.Phase != newNode.Phase ||
			oldNode.ScenarioRunRef != newNode.ScenarioRunRef {
			return true
		}
	}

	// ResiliencyScore changed (from nil to calculated, or status changed)
	if !reflect.DeepEqual(old.Status.ResiliencyScore, new.Status.ResiliencyScore) {
		return true
	}

	return false
}


