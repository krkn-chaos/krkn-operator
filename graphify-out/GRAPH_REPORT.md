# Graph Report - krkn-operator  (2026-11-13)

## Corpus Check
- cluster-only mode — file stats not available

## Summary
- 6126 nodes · 19809 edges · 270 communities (227 shown, 43 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 1888 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `fafc4445`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- main.go
- testing.T
- Handler
- NewTestHandler
- KrknScenarioRun
- sigs.k8s.io/controller-runtime/pkg/client.Client
- KrknAIRun
- InjectCredentials
- utils.go
- context.Context
- handlers_test.go
- api/types.go
- krknairun_cancellation_test.go
- newKrknAITestHandler
- k8s.io/api/core/v1.Secret
- with
- render-bundle.sh
- server.py
- addAdminContext
- .UpdateFile
- IsAdmin
- KrknGraphRun
- elasticsearch/client_test.go
- cloudcreds_handlers_test.go
- versions
- versions
- migration.go
- versions
- versions
- elasticsearch/client.go
- backup_restore_handlers_test.go
- k8s.io/apimachinery/pkg/apis/meta/v1.TypeMeta
- handler_test.go
- versions
- versions
- github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2
- versions
- versions
- versions
- versions
- versions
- ai.go
- versions
- versions
- versions
- versions
- notify-operator-docs.test.js
- versions
- installer.go
- versions
- versions
- GET methods
- versions
- versions
- versions
- versions
- user_handlers_test.go
- k8s.io/api/core/v1.ConfigMap
- dataprovider_grpc.pb.go
- rollbackTestHandler
- JobCounter
- spec
- ScenarioReference
- names
- Publisher
- names
- Frontend Authentication Implementation Tasks
- containers
- ssrf.go
- notify-website
- KrknUserGroup
- category_run_configuration.go
- render-and-validate
- .getKrknAIRunScenarioIndex
- scenario_catalog_cache_test.go
- Server
- Broadcaster
- scenarioRunConfigurationFixture
- BuildUnifiedJobList
- configstore_test.go
- steps
- PROGRESS-MVP.md
- KrknCategory
- values.yaml
- Provider Configuration API Specification
- krkn-operator Development Progress (Post-MVP)
- time.Time
- CreateBackup
- console
- steps
- GroupLabelKey
- KrknOperatorTargetProviderConfig
- ExtractRegistryV2FromSecret
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- KrknOperatorTarget
- Authentication API Specification
- File Management API Specification
- Cluster Segregation Troubleshooting
- dataprovider.pb.go
- steps
- GraphClusterScore
- allow-metrics-traffic.yaml
- endpoints
- Provider Package
- images
- values-olm-kubernetes.yaml
- labels
- time.Duration
- CLAUDE.md - Project Instructions
- ocm-setup
- linters
- limits
- values-olm-ocp.yaml
- ports
- Leader Election and API Server Separation
- elasticsearch/types.go
- newCategoryRequestAs
- krkn-operator
- KrknUser
- .GetScenarioRunLogs
- Hub
- steps
- steps
- .PostScenarioRun
- krkn_ai_types.go
- Deployment Guide
- Quick Start - Terminal API Local Testing
- Implementation Details
- GenerateFromToken
- ConnectionParams
- serviceMonitor
- cloudcreds/types.go
- leader_election_role_binding.yaml
- role_binding.yaml
- GetNodesRequest
- KrknOperatorTargetProvider
- Resiliency Score Feature - Frontend Implementation Guide
- .flatten
- stale.yml
- buildUnifiedJobList
- Running gRPC Data Provider Locally for Development
- AGENTS.md
- net/http.Response
- operator
- k8s.io/apimachinery/pkg/runtime.Scheme
- ResolveGraph
- acm
- default/kustomization.yaml
- workspace_role_binding.yaml
- ExecuteKubectlResponse
- needs-rebase.yml
- with
- WSUnifiedJobItem
- ScenarioRunRequest
- Chart.yaml
- Handler
- metrics_auth_role_binding.yaml
- makeScenarioRun
- 🔧 TODO: Pod Recreation and Retry Logic
- start_operator.sh
- commands.go
- ParseCommand
- aiService
- leader_election_role.yaml
- Option 1: Helm (Recommended)
- contents
- krkn-operator-data-provider
- test-api.sh
- .loadCategoryResiliencyHistories
- ComponentManifest
- target
- ExecuteKubectlRequest
- encoding/json.RawMessage
- jobs
- name
- slash-to-add-label.yml
- newReportTestHandler
- API Workflow Test
- KrknComponent
- captureLogSink
- .CreateWorkflow
- UnmarshalSecretData
- TokenGenerator
- manager.yaml
- labels
- workspace_role.yaml
- ValidateCreateRequest
- sortKrknAIScenarios
- gateway
- containers
- labels
- metrics_auth_role.yaml
- role.yaml
- Optional Krkn-AI with OLM on OpenShift
- publish-olm-bundles
- entries
- ClusterTarget
- manager/kustomization.yaml
- spec
- metrics_reader_role.yaml
- config.yaml
- submit-operatorhub-catalog
- buildClusterAPIURLMap
- CategoryResiliencyHistoryResponse
- .ExecuteKubectl
- ScenariosResponse
- volumes
- Store
- entries
- Running Locally
- Krkn-Operator Roadmap
- .ValidateGraph
- submit-community-catalog
- ComputeJobStats
- newProvidersHandler
- setup-ocm-and-operator.sh
- updateStrategy
- Available Make Targets
- Building Images
- Troubleshooting
- steps
- steps
- Setup
- Troubleshooting
- livenessProbe
- readinessProbe
- olm/kustomization.yaml
- Complete Workflow Examples
- .gosec.yaml
- channel-head.sh
- select-ai-release.sh
- krkngraphrun_controller_sanitization_test.go
- ConfigValidationError
- api_workflow_test.sh
- scenarioCatalogError
- resources
- securityContext
- configurationError
- select-compatible-release.sh
- test-open-kubernetes-operators-pr.sh
- test-select-ai-release.sh
- imageProbeError
- RollbackSource
- WorkspaceReference
- krknAIArchiveUpstreamError
- crd/kustomization.yaml
- manifests/kustomization.yaml
- network-policy/kustomization.yaml
- assets/README.md
- prometheus/kustomization.yaml
- rbac/kustomization.yaml
- post-install.sh
- generate_proto_go.sh
- ClusterNameCollisionError
- install-yq.sh
- test-channel-head.sh
- test-chart-exposure.sh
- test-release-workflow-gates.sh
- test-select-compatible-release.sh
- TestValidateValueAgainstSchema
- generate_proto.sh
- debug-cluster-permissions.sh
- notify-operator-docs.sh
- github.com/krkn-chaos/krkn-operator

## God Nodes (most connected - your core abstractions)
1. `Handler` - 256 edges
2. `NewTestHandler()` - 170 edges
3. `writeJSONError()` - 155 edges
4. `writeJSON()` - 114 edges
5. `createAdminContext()` - 92 edges
6. `IsAdmin()` - 92 edges
7. `KrknScenarioRun` - 74 edges
8. `GetClaimsFromContext()` - 67 edges
9. `newEsScheme()` - 57 edges
10. `createUserContext()` - 43 edges

## Surprising Connections (you probably didn't know these)
- `4. Verify a run` --references--> `KrknAIRun`  [INFERRED]
  docs/olm-openshift-krkn-ai.md → api/v1alpha1/krknairun_types.go
- `2. Run the operator` --references--> `KrknComponent`  [INFERRED]
  README.md → api/v1alpha1/krkncomponent_types.go
- `How It Works` --references--> `KrknOperatorTargetProvider`  [INFERRED]
  pkg/provider/README.md → api/v1alpha1/krknoperatortargetprovider_types.go
- `Requirements` --references--> `KrknOperatorTargetProvider`  [INFERRED]
  pkg/provider/README.md → api/v1alpha1/krknoperatortargetprovider_types.go
- `3. Create Provider Config Request` --references--> `KrknOperatorTargetProviderConfig`  [INFERRED]
  docs/api-provider-config-spec.md → api/v1alpha1/krknoperatortargetproviderconfig_types.go

## Import Cycles
- None detected.

## Communities (270 total, 43 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.08
Nodes (27): availableWorkspacePVC, CategoryEntityAssociationResponse, categoryEnvironmentParameter, CreateCategoryRequest, krknAIScenarioKey, ResiliencyHistoryQueryRequest, UpdateCategoryRequest, workspacePublishRequest (+19 more)

### Community 1 - "testing.T"
Cohesion: 0.02
Nodes (200): cancellationValidationObject(), TestRunCancellationValidation(), TestKrknUserCreation(), TestKrknUserList(), TestKrknUserRoles(), TestKrknUserStatus(), TestKrknUserValidation(), TestScenarioReferenceValidate() (+192 more)

### Community 2 - "Handler"
Cohesion: 0.06
Nodes (24): Handler, workspaceInfo, buildCloudCredentialResponse(), buildUserGroupResponse(), extractGroupNameAndUserIDFromPath(), extractGroupNameFromMembersPath(), validateClusterPermissions(), convertInputFields() (+16 more)

### Community 3 - "NewTestHandler"
Cohesion: 0.07
Nodes (102): esDoerFunc, TestGetActiveRunsOverview_AllUsersSeesAll(), TestGetActiveRunsOverview_MixedJobStates(), TestGetActiveRunsOverview_NoActiveRuns(), TestGetActiveRunsOverview_UserSeesAll(), boolPtr(), newEsAlertsTestSecretWithHost(), newEsScheme() (+94 more)

### Community 4 - "KrknScenarioRun"
Cohesion: 0.04
Nodes (55): KrknScenarioRunReconciler, preStartFailureReason(), TestPreStartFailureReason(), TestGroupRollbackJobsWebSocketKeepsFlatViewAndChecksUID(), TestScenarioRunReconcileSkipsPendingMigration(), containerStatusHasStarted(), podHasStartedContainer(), backfillLegacyMaxRetries() (+47 more)

### Community 5 - "sigs.k8s.io/controller-runtime/pkg/client.Client"
Cohesion: 0.04
Nodes (55): completeAIRunOnCancelClient, countingGetClient, failingWorkspaceGetClient, historyQueryCountingClient, k8sUserStatusChecker, krknAIRunReadHookClient, KrknAIServiceConfig, rollbackImageCache (+47 more)

### Community 6 - "KrknAIRun"
Cohesion: 0.05
Nodes (40): aiRunRegistryImages, krknAIPodOptions, KrknAIRunPodConfig, KrknAIRunReconciler, migrationPending(), aiOrchestratorPodName(), artifactsFinalizationPending(), buildOrchestratorPod() (+32 more)

### Community 7 - "InjectCredentials"
Cohesion: 0.05
Nodes (55): preparedJobResources, Custom Images, Custom Namespace, Deployment, Install CRDs Only, OpenShift Deployment, Quick Deploy, Publish and select a workspace (+47 more)

### Community 8 - "utils.go"
Cohesion: 0.12
Nodes (21): tokenRequest, attemptDirectory(), Inspect(), Run(), TestInspectRequiresReadableDirectory(), TestInspectRetainedRollbackFiles(), TestE2E(), getMetricsOutput() (+13 more)

### Community 9 - "context.Context"
Cohesion: 0.05
Nodes (25): countingUserStatusChecker, mockUserStatusChecker, KrknOperatorTargetProviderConfigReconciler, KrknTargetRequestReconciler, resolveRollbackDigest(), isConflictError(), checkProviderActive(), countActiveProviders() (+17 more)

### Community 10 - "handlers_test.go"
Cohesion: 0.04
Nodes (69): TestLogin_InactiveUser(), TestLogin_InvalidCredentials(), TestLogin_Success(), contains(), containsMiddle(), publicScenarioReference(), TestCreateGraphRun_WithFileReferences(), TestCreateGraphRun_ResiliencyScore() (+61 more)

### Community 11 - "api/types.go"
Cohesion: 0.05
Nodes (49): ActiveRunsOverviewResponse, AddGroupMemberRequest, AddGroupMemberResponse, ChangePasswordRequest, ChangePasswordResponse, ClusterPermissionSet, ClusterResiliencyScoreResponse, CreateTargetResponse (+41 more)

### Community 12 - "krknairun_cancellation_test.go"
Cohesion: 0.07
Nodes (49): cancellationScenario(), clientContainsFinalizer(), newCancellationScenarioReconciler(), setAIRunPodOwner(), setScenarioPodOwner(), TestCancellationCleanupReceiptParsesCoreDecision(), TestCancellationCleanupReceiptRecognizesForcedStop(), TestKrknAIRunCancellationBeforeProvisioningRetainsRun() (+41 more)

### Community 13 - "newKrknAITestHandler"
Cohesion: 0.08
Nodes (46): krknAIArchiveTestFile, krknAIScenarioIndexFixture, krknAIArchiveManifestJSON(), krknAIArchiveTestEntry(), newKrknAIArchiveTestHandler(), TestKrknAIRunArchiveDownloadContainsCommittedTextAndBinaryFiles(), TestKrknAIRunArchiveDownloadPropagatesMissingManifest(), TestKrknAIRunArchiveDownloadRejectsSnapshotMismatchWithoutZip() (+38 more)

### Community 14 - "k8s.io/api/core/v1.Secret"
Cohesion: 0.06
Nodes (30): buildConnectionParams(), buildConnectionParamsForIndex(), buildElasticsearchConfigResponse(), canAccessElasticsearchConfig(), buildRegistryInfo(), buildRegistryResponse(), ExtractGroupsFromLabels(), SanitizeGroupName() (+22 more)

### Community 15 - "with"
Cohesion: 0.04
Nodes (45): required, GH_TOKEN, INPUT_OPERATOR_TAG, OPERATOR_REPO_ROOT, PUSHED_TAG, SOURCE_RELEASE_CHART, WORKFLOW_RUN_HEAD_SHA, operator_tag (+37 more)

### Community 16 - "render-bundle.sh"
Cohesion: 0.06
Nodes (37): channel_has_entries(), ensure_catalog_channel(), catalog-channel.sh script, update_default_channel(), write_release_config(), catalog_submit_checkout(), catalog_submit_open_pr(), catalog-submit.sh script (+29 more)

### Community 17 - "server.py"
Cohesion: 0.06
Nodes (6): add_DataProviderServiceServicer_to_server(), DataProviderService, DataProviderServiceServicer, DataProviderServiceStub, DataProviderServicer, serve()

### Community 18 - "addAdminContext"
Cohesion: 0.11
Nodes (42): addAdminContext(), addUserContext(), createTestAdminUser(), setupFilesTestHandler(), TestCanAccessFile(), TestCreateFile(), TestCreateFile_DefaultFilePurpose(), TestCreateFile_DuplicateName() (+34 more)

### Community 19 - ".UpdateFile"
Cohesion: 0.07
Nodes (31): deriveLogicalName(), extractLogicalName(), isValidConfigMapKey(), validateCreateFileRequest(), validateFileFields(), validateUpdateFileRequest(), decodeKrknAIRequest(), TestKrknAIConfigOwnerCanListAndReadSavedConfig() (+23 more)

### Community 20 - "IsAdmin"
Cohesion: 0.09
Nodes (13): cloudCredentialAccessContext, validateCategoryName(), writeCategoryRunNotFoundOrInternalError(), updateCloudCredentialSecretData(), convertGraphClusterScores(), GetClaimsFromContext(), IsAdmin(), ExtractGroupsFromLabels() (+5 more)

### Community 21 - "KrknGraphRun"
Cohesion: 0.11
Nodes (14): imageResolver, imageSignatureVerifier, KrknGraphRunReconciler, graphHistoryRun(), TestGraphRunReconcileSkipsPendingMigration(), TestGraphRunCancelledChildDoesNotLeaveGraphRunning(), TestLoadFileConfigMapByID(), TestTranslateVolumesToFileMounts() (+6 more)

### Community 22 - "elasticsearch/client_test.go"
Cohesion: 0.09
Nodes (29): doerFunc, roundTripperFunc, NewClient(), intPtr(), scenarioParametersSchema(), TestBaseURL(), TestClientTransportDefaultNotHTTPTransport(), TestClientTransportPooling() (+21 more)

### Community 23 - "cloudcreds_handlers_test.go"
Cohesion: 0.11
Nodes (39): TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(), TestCloudCredentialContract_GraphRunStripsNodeEnv(), newAWSTestSecret(), newCloudCredScheme(), newGCPTestSecret(), newGroupedAWSSecret(), TestCloudCredentialsRouter_MethodNotAllowed(), TestCloudCredentialsRouter_SubpathMethodNotAllowed() (+31 more)

### Community 24 - "versions"
Cohesion: 0.05
Nodes (38): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+30 more)

### Community 25 - "versions"
Cohesion: 0.05
Nodes (38): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+30 more)

### Community 26 - "migration.go"
Cohesion: 0.08
Nodes (68): clusterJobImage, clusterJobProvider, clusterJobRetries, graphScenarioReferences, migrationBlockedError, migrationPlan, migrationResult, migrationStats (+60 more)

### Community 27 - "versions"
Cohesion: 0.05
Nodes (38): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+30 more)

### Community 28 - "versions"
Cohesion: 0.05
Nodes (38): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+30 more)

### Community 29 - "elasticsearch/client.go"
Cohesion: 0.14
Nodes (19): facetField, Option, StatusError, buildAlertsSearchBody(), buildFacetAggs(), buildFilterClauses(), buildSearchBody(), buildTimestampRange() (+11 more)

### Community 30 - "backup_restore_handlers_test.go"
Cohesion: 0.15
Nodes (22): NewJobTracker(), createMultipartBody(), createTestHandler(), TestGetRestoreStatusAdminOnly(), TestGetRestoreStatusTrackedJob(), TestGetRestoreStatusUnknownJob(), TestHandlerShutdownCancelsContext(), TestJobTrackerEviction() (+14 more)

### Community 31 - "k8s.io/apimachinery/pkg/apis/meta/v1.TypeMeta"
Cohesion: 0.09
Nodes (9): TestKrknCategoryDeepCopy(), KrknAIRunList, KrknCategoryList, KrknGraphRunList, KrknOperatorTargetProviderConfigList, KrknScenarioRunList, KrknTargetRequestList, KrknUserGroupList (+1 more)

### Community 32 - "handler_test.go"
Cohesion: 0.10
Nodes (37): NewHandler(), NewBroadcaster(), TestBroadcastDashboardUpdate(), TestBroadcastGraphRunDeleted(), TestBroadcastGraphRunUpdate(), TestBroadcastOnlyToSubscribedClients(), TestBroadcastScenarioRunDeleted(), TestBroadcastScenarioRunUpdate() (+29 more)

### Community 33 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 34 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 35 - "github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2"
Cohesion: 0.15
Nodes (13): mockScenarioProvider, registryImagesResult, scenarioDetailResult, ScenarioTag, fetchScenarioCatalog(), filterScenariosByIsAScenario(), filterScenariosByIsAScenarioComplete(), getRegistryImagesWithContext() (+5 more)

### Community 36 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 37 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 38 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 39 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 40 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 41 - "ai.go"
Cohesion: 0.05
Nodes (72): KrknAIRunPodConfigFromEnv(), TestKrknAIRunPodConfigPreservesBoundsWithPartialOverrides(), TestKrknAIRunPodConfigRejectsInvalidResources(), TestKrknAIRunPodConfigRejectsInvalidVolumeBounds(), AIConfigFromEnv(), aiContainerMatches(), aiCreateRace(), aiDeployment() (+64 more)

### Community 42 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 43 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 44 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 45 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 46 - "notify-operator-docs.test.js"
Cohesion: 0.10
Nodes (11): assert, chartWorkflow, { execFileSync, spawnSync }, fs, notifier, notifyWorkflow, os, path (+3 more)

### Community 47 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 48 - "installer.go"
Cohesion: 0.07
Nodes (39): bootstrapOLMResources(), main(), resolveKrknAIEnabled(), resolveMigrationNamespace(), runCRDMigrationMode(), TestResolveKrknAIEnabledPrecedence(), TestResolveMigrationNamespacePrefersHookNamespace(), TestRunCRDMigrationMode() (+31 more)

### Community 49 - "versions"
Cohesion: 0.05
Nodes (36): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, metadata, annotations (+28 more)

### Community 50 - "versions"
Cohesion: 0.05
Nodes (36): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, metadata, annotations (+28 more)

### Community 51 - "GET methods"
Cohesion: 0.06
Nodes (35): API Requirements, /clusters, DELETE methods, GET methods, GET methods, get nodes, get pods, Grpc python service requirement (+27 more)

### Community 52 - "versions"
Cohesion: 0.06
Nodes (35): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+27 more)

### Community 53 - "versions"
Cohesion: 0.06
Nodes (35): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+27 more)

### Community 54 - "versions"
Cohesion: 0.06
Nodes (35): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+27 more)

### Community 55 - "versions"
Cohesion: 0.06
Nodes (35): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+27 more)

### Community 56 - "user_handlers_test.go"
Cohesion: 0.14
Nodes (34): createTestUser(), kvValue(), setupUserTestHandler(), TestChangePassword_AdminChange_NoCurrentPassword(), TestChangePassword_AdminChangesOtherUser_EmitsAuditLog(), TestChangePassword_SamePassword_Rejected(), TestChangePassword_SelfChange_NoAdminAuditLog(), TestChangePassword_SelfChange_RequiresCurrentPassword() (+26 more)

### Community 57 - "k8s.io/api/core/v1.ConfigMap"
Cohesion: 0.09
Nodes (20): AvailableFilesResponse, ListFilesResponse, buildFileInfo(), buildFileResponse(), convertConfigMapToWorkflowInfo(), convertFileResponseToWorkflow(), filterVisibleWorkflowCategories(), parseWorkflowCategories() (+12 more)

### Community 58 - "dataprovider_grpc.pb.go"
Cohesion: 0.12
Nodes (6): DataProviderServiceServer, UnimplementedDataProviderServiceServer, UnsafeDataProviderServiceServer, _DataProviderService_ExecuteKubectl_Handler(), _DataProviderService_GetNodes_Handler(), RegisterDataProviderServiceServer()

### Community 59 - "rollbackTestHandler"
Cohesion: 0.16
Nodes (31): RollbackAvailability, RollbackState, rollbackProbeName(), rollbackRunAttemptName(), rollbackRunName(), decodeRollbackResponse(), rollbackAvailableProbe(), rollbackRegistrySecret() (+23 more)

### Community 61 - "spec"
Cohesion: 0.06
Nodes (34): capabilities, categories, containerImage, description, repository, support, apiVersion, supported (+26 more)

### Community 62 - "ScenarioReference"
Cohesion: 0.11
Nodes (22): KrknScenarioRunSpec, TestLegacyScenarioIdentityRemainsDecodable(), TestResolveScenarioReferenceMigratesOnlyLegacyIdentity(), scenarioReferenceForResponse(), publicScenarioReference(), TestCreateScenarioRun_ResiliencyScore(), buildContainerImage(), buildContainerImageFromReference() (+14 more)

### Community 63 - "names"
Cohesion: 0.06
Nodes (30): controller-gen.kubebuilder.io/version, apiVersion, kind, metadata, annotations, name, kind, listKind (+22 more)

### Community 64 - "Publisher"
Cohesion: 0.09
Nodes (10): Publisher, KrknWorkspaceReconciler, NewNamespaceFilter(), TestWorkspaceReconcilerMapsPVCWithoutOwningIt(), TestWorkspaceReconcilerTracksPVCState(), TestPublisherReconcile(), TestPublisherReconcileReturnsTransientErrors(), TestPublisherRecoversFromConcurrentCreateAndDelete() (+2 more)

### Community 65 - "names"
Cohesion: 0.06
Nodes (30): controller-gen.kubebuilder.io/version, apiVersion, kind, metadata, annotations, name, kind, listKind (+22 more)

### Community 66 - "Frontend Authentication Implementation Tasks"
Cohesion: 0.06
Nodes (30): 1. Read the API Documentation, 1. Start the Operator, 2. Check Available Work, 2. Test Registration (First Admin), 3. Start with Foundation Task, 3. Test Login, 401 on Protected Routes, 4. Claim a Task (+22 more)

### Community 67 - "containers"
Cohesion: 0.07
Nodes (29): apiVersion, image, imagePullPolicy, livenessProbe, name, ports, readinessProbe, resources (+21 more)

### Community 68 - "ssrf.go"
Cohesion: 0.06
Nodes (43): ipRateLimiter, rateLimiterEntry, responseWriter, clientIPFromHeaders(), extractClientIP(), hostFromRemoteAddr(), isTrustedProxy(), newIPRateLimiter() (+35 more)

### Community 69 - "notify-website"
Cohesion: 0.07
Nodes (29): env, CHART_REPO, GH_TOKEN, IMAGE_REGISTRY, REGISTRY, jobs, notify-website, release-chart (+21 more)

### Community 70 - "KrknUserGroup"
Cohesion: 0.10
Nodes (12): AggregateClusterPermissions(), CanPerformAction(), TestAggregateClusterPermissions(), TestCanPerformAction(), Action, FilterClustersByPermission(), hasAction(), TestFilterClustersByPermission_NoGroups() (+4 more)

### Community 71 - "category_run_configuration.go"
Cohesion: 0.15
Nodes (26): scenarioFileBehavior, GraphScenarioNode, writeCategoryProfileField(), appendGraphSignaturePart(), compareCategoryRunConfigurations(), compareGraphConfigurations(), compareGraphConfigurationsWithStableNodeIDs(), compareGraphNodeConfigurations() (+18 more)

### Community 72 - "render-and-validate"
Cohesion: 0.07
Nodes (28): MIN_KUBE_VERSION, VERSION, version, jobs, render-and-validate, profile, name, on (+20 more)

### Community 73 - ".getKrknAIRunScenarioIndex"
Cohesion: 0.10
Nodes (16): krknAIArchiveManifest, krknAIArchiveManifestFile, krknAIScenarioQuery, copyKrknAIArchiveServiceError(), safeKrknAIArchivePath(), writeKrknAIArchiveCorrupt(), copyKrknAIResponse(), hasKrknAIRunOwnerUID() (+8 more)

### Community 74 - "scenario_catalog_cache_test.go"
Cohesion: 0.12
Nodes (18): publicSignatureCache, publicSignatureCacheEntry, publicSignatureCachingProvider, InvalidImageSignatureError, isDefinitiveSignatureStatus(), digestPointer(), postScenariosForCacheTest(), TestFetchScenarioCatalogReturnsTimeoutInsteadOfPartialCatalog() (+10 more)

### Community 75 - "Server"
Cohesion: 0.09
Nodes (12): Server, Role, UserStatusChecker, loggingMiddleware(), maxBodySizeMiddleware(), TestMaxBodySizeMiddleware(), Handler, Middleware (+4 more)

### Community 76 - "Broadcaster"
Cohesion: 0.16
Nodes (7): hashBytes(), roleFromAdmin(), visibleCategoryNames(), AuthorizationCheckFunc, Broadcaster, graphRunEventHandler, scenarioRunEventHandler

### Community 77 - "scenarioRunConfigurationFixture"
Cohesion: 0.16
Nodes (26): categoryHistoryRun, categoryParameterProfileFingerprint(), categoryParameterProfileName(), TestCategoryParameterProfileFingerprintForEmptyEnvironment(), TestCategoryParameterProfileFingerprintGraphEnvironmentMultiset(), TestCategoryParameterProfileFingerprintIgnoresRunType(), TestCategoryParameterProfileFingerprintScenarioEnvironment(), TestCategoryParameterProfileNameIsDeterministicAndReadable() (+18 more)

### Community 78 - "BuildUnifiedJobList"
Cohesion: 0.10
Nodes (17): convertClusterResiliencyScores(), averageResiliencyScore(), convertMetaTime(), BuildUnifiedJobList(), getDefaultPageSize(), GroupRollbackJobs(), runCategoryNames(), scenarioRunImage() (+9 more)

### Community 79 - "configstore_test.go"
Cohesion: 0.12
Nodes (23): NewConfigStoreInitializer(), TestConfigStoreInitializer_NeedLeaderElection(), TestConfigStoreInitializer_Start_ConfigMapNotFound(), TestConfigStoreInitializer_Start_Success(), SyncConfigMapToStore(), TestRoundTrip(), TestSyncConfigMapToStore(), TestSyncConfigMapToStore_NilStore() (+15 more)

### Community 80 - "steps"
Cohesion: 0.08
Nodes (25): description, required, type, BUNDLE_IMAGE, OLM_NAMESPACE, UPGRADE_BUNDLE_IMAGE, bundle_image, upgrade_bundle_image (+17 more)

### Community 81 - "PROGRESS-MVP.md"
Cohesion: 0.08
Nodes (24): Architecture Decisions, Completed, Current Status, Data Provider Architecture, Environment Variables, Future Work (Not in Current Scope), GET /clusters, GET /nodes (+16 more)

### Community 82 - "KrknCategory"
Cohesion: 0.12
Nodes (17): CategoryListResponse, CategoryResponse, ErrorResponse, buildCategoryResponse(), buildCategoryVisibilityLabels(), categoryErrorCode(), categoryGroups(), categoryIsPublic() (+9 more)

### Community 83 - "values.yaml"
Cohesion: 0.08
Nodes (24): auth, jwtExpiryHours, jwtSecret, trustedProxyCIDRs, websocketAllowedOrigins, commonAnnotations, commonLabels, crds (+16 more)

### Community 84 - "Provider Configuration API Specification"
Cohesion: 0.08
Nodes (24): 1. List Available Providers (Optional), 2. Create Config Request, 3. Poll for Completion, 4. Parse Schemas and Build UI, 5. Submit Configuration, Base URL, Change Log, Common Error Patterns (+16 more)

### Community 85 - "krkn-operator Development Progress (Post-MVP)"
Cohesion: 0.08
Nodes (21): Agent Specifications - MVP Archive, Files, MVP Completion Date, MVP Features Delivered, Post-MVP Development, API Endpoints Completed, Completed, Core Infrastructure (+13 more)

### Community 86 - "time.Time"
Cohesion: 0.11
Nodes (16): GraphRunDetailResponse, GraphRunListItem, GraphRunListResponse, GraphRunSpecResponse, GraphRunStatusResponse, RestoreResponse, RollbackRunSummary, ScenarioRunListItem (+8 more)

### Community 87 - "CreateBackup"
Cohesion: 0.09
Nodes (20): resiliencyHistoryQuerySelections, BackupConfig, resourceType, normalizeResiliencyHistoryProviders(), normalizeResiliencyHistorySelections(), parseResiliencyHistoryQuery(), backupResourceType(), CreateBackup() (+12 more)

### Community 88 - "console"
Cohesion: 0.09
Nodes (24): console, affinity, enabled, extraEnv, ingress, nodeSelector, replicaCount, route (+16 more)

### Community 89 - "steps"
Cohesion: 0.08
Nodes (23): concurrency, cancel-in-progress, group, COLOR, jobs, test, name, on (+15 more)

### Community 90 - "GroupLabelKey"
Cohesion: 0.18
Nodes (23): newCategoryRequest(), newCategoryTestGroup(), newCategoryTestHandler(), newCategoryTestUser(), TestCategoriesRouterCRUD(), TestCategoriesRouterGroupCreationRequiresMembership(), TestCategoriesRouterGroupVisibility(), TestCategoriesRouterNameMaximumLength() (+15 more)

### Community 91 - "KrknOperatorTargetProviderConfig"
Cohesion: 0.12
Nodes (14): getProviderNames(), CreateProviderConfigRequest(), UpdateProviderConfig(), Configuration Functions, Configuration Workflow, CreateProviderConfigRequest, Provider Configuration, UpdateProviderConfig (+6 more)

### Community 92 - "ExtractRegistryV2FromSecret"
Cohesion: 0.10
Nodes (24): addAIRegistrySecret(), privateAIRunFixture(), TestAIChildCannotChangeParentRegistry(), TestPrivateAIRunUsesSelectedRuntimeImagesAndContext(), TestPrivateAIRunValidatesImagesBeforeCreatingWorkloads(), resolvePrivateRegistry(), TestInspectAIImagesUsesPrivateAuthAndReturnsDigestPinnedRuntimeImages(), BuildDockerConfigJSON() (+16 more)

### Community 93 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.10
Nodes (16): ClusterJobStatusResponse, ListProvidersResponse, NodeStatusResponse, ProviderResponse, ScenarioRunStatusResponse, convertClusterJobs(), convertMetaTime(), TestConvertClusterJobs_EmptyInput() (+8 more)

### Community 94 - "KrknOperatorTarget"
Cohesion: 0.12
Nodes (12): API and operator refactoring to support multiple clusters, BREAKING CHANGES, Implementation Details, KrknTargetRequest controller in operator, Overview, Overview, "ROAD TO PRODUCTION" REQUIREMENTS, Testing (+4 more)

### Community 95 - "Authentication API Specification"
Cohesion: 0.09
Nodes (22): 1. Check Admin Registration, 1. Initial Load, 2. First Admin Registration, 2. Register User, 3. Login, 3. Login, 4. Authenticated Requests, 5. Role-Based UI (+14 more)

### Community 96 - "File Management API Specification"
Cohesion: 0.09
Nodes (22): Access Control, API Endpoints, ConfigMap Structure, Content Validation, Create a Public Configuration File, Create File, Delete File, Error Responses (+14 more)

### Community 97 - "Cluster Segregation Troubleshooting"
Cohesion: 0.09
Nodes (22): Access Rules, Cluster Segregation Troubleshooting, Code References, Common Issues, Debugging Steps, Example 1: Proper Segregation, Example 2: Multiple Groups, Example 3: Admin User (+14 more)

### Community 98 - "dataprovider.pb.go"
Cohesion: 0.15
Nodes (3): file_proto_dataprovider_proto_init(), file_proto_dataprovider_proto_rawDescGZIP(), init()

### Community 99 - "steps"
Cohesion: 0.09
Nodes (22): name, runs-on, steps, timeout-minutes, concurrency, cancel-in-progress, group, jobs (+14 more)

### Community 100 - "GraphClusterScore"
Cohesion: 0.11
Nodes (9): convertNodeStatusesWithScores(), convertNodeStatusesWithScores(), buildGraphRunResponse(), convertGraphClusterScoresForSnapshot(), GraphClusterScore, GraphRunSummary, KrknGraphRunStatus, NodeStatus (+1 more)

### Community 101 - "allow-metrics-traffic.yaml"
Cohesion: 0.09
Nodes (21): apiVersion, namespaceSelector, from, ports, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, app.kubernetes.io/name (+13 more)

### Community 102 - "endpoints"
Cohesion: 0.09
Nodes (21): apiVersion, bearerTokenFile, path, port, scheme, tlsConfig, kind, app.kubernetes.io/managed-by (+13 more)

### Community 103 - "Provider Package"
Cohesion: 0.09
Nodes (21): API Reference, Basic Usage (Default Configuration), CleanupOldResources, Config Struct, Configuration Options, CreateProviderConfigRequest, Custom Configuration, Default Values (+13 more)

### Community 104 - "images"
Cohesion: 0.10
Nodes (21): image, pullPolicy, image, pullPolicy, image, pullPolicy, image, pullPolicy (+13 more)

### Community 105 - "values-olm-kubernetes.yaml"
Cohesion: 0.10
Nodes (20): acm, enabled, console, enabled, gateway, ingress, route, enabled (+12 more)

### Community 106 - "labels"
Cohesion: 0.10
Nodes (20): apiVersion, kind, app.kubernetes.io/component, app.kubernetes.io/managed-by, app.kubernetes.io/name, control-plane, metadata, labels (+12 more)

### Community 107 - "time.Duration"
Cohesion: 0.13
Nodes (11): Architecture Decision, Components Requiring Leader Election (Leader-Only), Components Running on All Replicas (Multi-Active), getTokenDuration(), TestGetTokenDuration(), discoveryResponseDeadlineMiddleware(), TestDiscoveryResponseDeadlineIsRouteSpecific(), NewProviderRegistration() (+3 more)

### Community 108 - "CLAUDE.md - Project Instructions"
Cohesion: 0.10
Nodes (19): Architecture Principles, Before Committing, Before Writing Code, CLAUDE.md - Project Instructions, Code Organization & Reusability, Code Quality Standards, Communication, Critical Rules (+11 more)

### Community 109 - "ocm-setup"
Cohesion: 0.10
Nodes (19): managed-serviceaccount-version, ocm-version, jobs, ocm-setup, include, name, name, runs-on (+11 more)

### Community 110 - "linters"
Cohesion: 0.11
Nodes (19): generated, paths, rules, formatters, enable, exclusions, linters, default (+11 more)

### Community 111 - "limits"
Cohesion: 0.14
Nodes (19): resources, resources, resources, krknAI, enabled, orchestrator, resultUploader, cpu (+11 more)

### Community 112 - "values-olm-ocp.yaml"
Cohesion: 0.11
Nodes (18): acm, enabled, console, enabled, gateway, ingress, route, enabled (+10 more)

### Community 113 - "ports"
Cohesion: 0.11
Nodes (18): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, control-plane, metadata, labels, name (+10 more)

### Community 115 - "Leader Election and API Server Separation"
Cohesion: 0.11
Nodes (18): After (Multi-Active API), API Server Concurrent Operations, API Server Startup Wait, Before (Single Leader), Concurrency Safety, Controller Safety with Leader Election, Deployment Considerations, Expected Startup Logs (+10 more)

### Community 116 - "elasticsearch/types.go"
Cohesion: 0.16
Nodes (18): AffectedPods, CreateElasticsearchConfigResponse, DeleteElasticsearchConfigResponse, ListElasticsearchConfigsResponse, QueryAlertsRequest, QueryTelemetryRequest, RawJSON, RecoveredPod (+10 more)

### Community 117 - "newCategoryRequestAs"
Cohesion: 0.26
Nodes (17): newCategoryRequestAs(), setupScenarioRunTestHandler(), TestPostScenarioRun_CategoriesArePersistedOnCreatedCR(), TestPostScenarioRun_RejectsInvalidCategoryNameBeforeCreatingCR(), readyWorkspace(), TestAvailableWorkspacesTrailingSlashStillFiltersUnavailable(), TestGraphRunPinsNodeWorkspace(), TestListAvailableWorkspacePVCs() (+9 more)

### Community 118 - "krkn-operator"
Cohesion: 0.11
Nodes (19): Alerts Query API, API compatibility notes, Backup and Restore, Documentation, Ecosystem, Graph node resiliency weights, Krkn-AI Operator API, krkn-operator (+11 more)

### Community 119 - "KrknUser"
Cohesion: 0.16
Nodes (6): filterUsers(), newKrknUser(), KrknUser, KrknUserList, KrknUserSpec, KrknUserStatus

### Community 120 - ".GetScenarioRunLogs"
Cohesion: 0.17
Nodes (13): filterReportLines(), isReportMarkerEnd(), isReportMarkerStart(), isWebSocketDisconnectError(), maskToken(), podLogOptions(), sanitizeHeaders(), startLogWebSocketPings() (+5 more)

### Community 121 - "Hub"
Cohesion: 0.18
Nodes (6): JobStatus, JobTracker, BroadcastMessage, Client, Hub, PaginationClientState

### Community 122 - "steps"
Cohesion: 0.11
Nodes (17): concurrency, cancel-in-progress, group, jobs, security-and-quality, name, on, pull_request (+9 more)

### Community 123 - "steps"
Cohesion: 0.11
Nodes (17): concurrency, cancel-in-progress, group, jobs, test-e2e, name, on, pull_request (+9 more)

### Community 124 - ".PostScenarioRun"
Cohesion: 0.15
Nodes (13): writeScenarioRunAccessError(), parseBoolHeader(), TestParseBoolHeader(), sanitizeRunNameLabel(), resolveMaxRetries(), IsCloudEnvVar(), StripCloudEnvVars(), TestIsCloudEnvVar() (+5 more)

### Community 125 - "krkn_ai_types.go"
Cohesion: 0.17
Nodes (16): KrknAIArtifactSummary, KrknAIConfigRequest, KrknAIConfigValidationError, KrknAIConfigValidationRequest, KrknAIConfigValidationResponse, KrknAIDiscoveryRequest, KrknAIFitnessProgression, KrknAIRunRequest (+8 more)

### Community 126 - "Deployment Guide"
Cohesion: 0.12
Nodes (17): Architecture, Build Installer Bundle, Custom Push, Deployment Guide, Environment Variables, Git Tag Support, Git Tag Workflow, Image Names (Auto-Generated) (+9 more)

### Community 127 - "Quick Start - Terminal API Local Testing"
Cohesion: 0.12
Nodes (17): 1. Setup gRPC Server, 2. Run Operator (new terminal), 3. Test the API (new terminal), "401 Unauthorized", 🧪 Automated Testing, "Command not found: kubectl", 🔥 Common Issues, Debugging (+9 more)

### Community 128 - "Implementation Details"
Cohesion: 0.14
Nodes (16): Code Refactoring: getKubeconfigFromTarget() Helper ✅, Implementation Details, Phase 10: POST /scenarios/detail/{scenario_name} Endpoint ✅, Phase 11: POST /scenarios/globals/{scenario_name} Endpoint ✅, Phase 1: REST API Setup ✅, Phase 2: GET /clusters Endpoint ✅, Phase 3: Integration ✅, Phase 4: Testing ✅ (+8 more)

### Community 129 - "GenerateFromToken"
Cohesion: 0.18
Nodes (16): CreateTargetRequest, UpdateTargetRequest, generateKubeconfigFromCredentialsType(), generateKubeconfigFromKubeconfigType(), generateKubeconfigFromRequest(), generateKubeconfigFromTokenType(), TestCreateTarget_DuplicateClusterName(), TestCreateTarget_WithKubeconfig() (+8 more)

### Community 130 - "ConnectionParams"
Cohesion: 0.23
Nodes (7): Doer, executeSearch(), Client, ConnectionParams, newSearchRequest(), TestTransportKey(), transportKey()

### Community 131 - "serviceMonitor"
Cohesion: 0.15
Nodes (16): service, service, monitoring, enabled, service, serviceMonitor, service, annotations (+8 more)

### Community 132 - "cloudcreds/types.go"
Cohesion: 0.14
Nodes (15): CreateCloudCredentialResponse, DeleteCloudCredentialResponse, ListCloudCredentialsResponse, UpdateCloudCredentialResponse, buildCloudCredentialSecretData(), CloudCredentialResponse, CreateCloudCredentialRequest, UpdateCloudCredentialRequest (+7 more)

### Community 133 - "leader_election_role_binding.yaml"
Cohesion: 0.12
Nodes (15): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, metadata, labels, name, roleRef (+7 more)

### Community 134 - "role_binding.yaml"
Cohesion: 0.12
Nodes (15): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, metadata, labels, name, roleRef (+7 more)

### Community 136 - "KrknOperatorTargetProvider"
Cohesion: 0.17
Nodes (9): 1. List Providers, 2. Get Provider Config Status, 2. Update Provider Status, 3. Create Provider Config Request, 3. Update Provider Configuration, API Endpoints, KrknOperatorTargetProvider, KrknOperatorTargetProviderSpec (+1 more)

### Community 137 - "Resiliency Score Feature - Frontend Implementation Guide"
Cohesion: 0.12
Nodes (15): Backend Implementation Details, Backend Status: ✅ Completato, Example: Resiliency Score Badge Component, Expected Behavior, Frontend Implementation Guidelines, Manual Testing Steps, Overview, Phase 1: Display Score in List View (+7 more)

### Community 138 - ".flatten"
Cohesion: 0.21
Nodes (12): ClusterMetadata, FacetOption, NodeSummaryInfo, QueryTelemetryResponse, rawTelemetrySource, TelemetryDocument, TelemetryStats, decodeTelemetry() (+4 more)

### Community 139 - "stale.yml"
Cohesion: 0.12
Nodes (15): concurrency, cancel-in-progress, group, jobs, stale, name, on, schedule (+7 more)

### Community 140 - "buildUnifiedJobList"
Cohesion: 0.17
Nodes (12): TestBuildUnifiedJobList_SkipsGraphRunChildren(), averageResiliencyScore(), buildScenarioRunDetailResponse(), buildScenarioRunResponse(), buildUnifiedJobList(), convertClusterResiliencyScoresWS(), runCategoryNames(), sanitizedClusterJobs() (+4 more)

### Community 141 - "Running gRPC Data Provider Locally for Development"
Cohesion: 0.12
Nodes (16): 1. Login and Get JWT Token, 2. Create KrknTargetRequest (or use existing UUID), 3. Test Terminal Command, Cleanup, Configure Operator to Use Local gRPC Server, Debug gRPC Server in PyCharm, Development Workflow, Notes (+8 more)

### Community 142 - "AGENTS.md"
Cohesion: 0.13
Nodes (13): Beads Task Tracking, Database and project identity, Exports and Git, Go Operators â€” Only for `project.type = "go-operator"`, graphify, Handoff, Issue lifecycle, Project Configuration (+5 more)

### Community 143 - "net/http.Response"
Cohesion: 0.19
Nodes (4): krknAIArchiveServiceError, ImageRequirements, Client, ReadBody()

### Community 144 - "operator"
Cohesion: 0.15
Nodes (15): securityContext, securityContext, operator, affinity, enabled, extraEnv, nodeSelector, olmBootstrap (+7 more)

### Community 145 - "k8s.io/apimachinery/pkg/runtime.Scheme"
Cohesion: 0.18
Nodes (13): addKnownTypes(), decodeErrorResponse(), providerConfigTestScheme(), TestGetProviderConfigByUUID_CompletedReturnsConfigData(), TestGetProviderConfigByUUID_InternalErrorReturnsSanitized500(), TestGetProviderConfigByUUID_NotFoundReturns404(), TestGetProviderConfigByUUID_PendingReturnsAccepted(), TestPostProviderConfig_FailureReturnsSanitizedError() (+5 more)

### Community 146 - "ResolveGraph"
Cohesion: 0.15
Nodes (13): FromKrknctlScenarioSet(), publicScenarioReference(), publicScenario(), TestFromKrknctlScenarioSetCreatesPublicReferences(), TestToKrknctlScenarioSetUsesScenarioReference(), ToKrknctlScenarioSet(), MapScenarioNodeToScenarioRunSpec(), ResolveGraph() (+5 more)

### Community 147 - "acm"
Cohesion: 0.16
Nodes (14): acm, affinity, config, enabled, extraEnv, logging, nodeSelector, replicaCount (+6 more)

### Community 148 - "default/kustomization.yaml"
Cohesion: 0.14
Nodes (13): apiVersion, images, name, newName, newTag, kind, namePrefix, namespace (+5 more)

### Community 149 - "workspace_role_binding.yaml"
Cohesion: 0.14
Nodes (13): apiVersion, kind, metadata, name, namespace, roleRef, apiGroup, kind (+5 more)

### Community 151 - "needs-rebase.yml"
Cohesion: 0.14
Nodes (13): jobs, rebase, name, on, pull_request_target, push, permissions, contents (+5 more)

### Community 152 - "with"
Cohesion: 0.14
Nodes (14): with, build-args, cache-from, cache-to, context, if-no-files-found, name, password (+6 more)

### Community 153 - "WSUnifiedJobItem"
Cohesion: 0.21
Nodes (13): TestFilterJobsByCategoriesMatchesAnySelectedCategory(), TestPaginateJobItems(), computeWSJobStats(), filterJobCategoriesByVisibility(), filterJobsByCategories(), groupRollbackJobs(), paginateJobItems(), ServerMessage (+5 more)

### Community 154 - "ScenarioRunRequest"
Cohesion: 0.18
Nodes (12): Phase 12: Scenario Job Management - /scenarios/run Endpoints ✅, Phase 9: POST /scenarios Endpoint ✅, FileMount, GlobalsRequest, JobsListResponse, JobStatusResponse, ScenarioRunRequest, ScenariosRequest (+4 more)

### Community 155 - "Chart.yaml"
Cohesion: 0.15
Nodes (12): apiVersion, appVersion, description, home, keywords, maintainers, name, url (+4 more)

### Community 157 - "metrics_auth_role_binding.yaml"
Cohesion: 0.15
Nodes (12): apiVersion, kind, metadata, name, roleRef, apiGroup, kind, name (+4 more)

### Community 158 - "makeScenarioRun"
Cohesion: 0.24
Nodes (13): makeGraphRun(), makeScenarioRun(), TestListJobs_GroupedRollbacksBeforePagination(), TestListJobs_IncludesStats(), TestListJobs_LimitOnlyReturnsAll(), TestListJobs_NoPagination(), TestListJobs_Paginated(), TestListJobs_SkipsGraphRunChildren() (+5 more)

### Community 159 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.15
Nodes (13): API Endpoints Structure (Nested Approach), Configuration, Current Behavior, Implementation Plan (TODO), ✅ Implemented, Moving from direct scenario creation to CRD approach, Option A: Cancellation Field in Status (Recommended), Option B: Finalizers for Cancellation Tracking (+5 more)

### Community 160 - "start_operator.sh"
Cohesion: 0.41
Nodes (12): build_operator(), check_prerequisites(), cleanup(), create_service_account(), install_crds(), main(), print_error(), print_info() (+4 more)

### Community 161 - "commands.go"
Cohesion: 0.23
Nodes (11): AvailableCommandsResponse, TestExecuteTerminal_ValidationLogic(), GetAllowedCommands(), GetBlockedFlags(), CommandSpec, IsCommandAllowed(), IsSubcommandAllowed(), TestValidateCommand() (+3 more)

### Community 162 - "ParseCommand"
Cohesion: 0.17
Nodes (10): TerminalRequest, TerminalResponse, TestExecuteTerminal_CommandParsing(), ParsedCommand, ParseCommand(), parseLongFlag(), TestParseCommand(), TestParseLongFlag() (+2 more)

### Community 163 - "aiService"
Cohesion: 0.17
Nodes (12): aiService, networkPolicy, resources, serviceToken, storage, networkPolicy, egress, enabled (+4 more)

### Community 164 - "leader_election_role.yaml"
Cohesion: 0.17
Nodes (11): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, metadata, labels, name, rules (+3 more)

### Community 165 - "Option 1: Helm (Recommended)"
Cohesion: 0.17
Nodes (12): Configuration Options, Custom namespace, Disable console (operator only), Install on Kubernetes with Ingress, Install on OpenShift with Route, Install with ACM integration (OpenShift), Install with defaults (console enabled, ACM disabled), Installation Methods (+4 more)

### Community 166 - "contents"
Cohesion: 0.17
Nodes (12): name, permissions, runs-on, name, permissions, runs-on, steps, build-data-provider-amd64 (+4 more)

### Community 167 - "krkn-operator-data-provider"
Cohesion: 0.17
Nodes (11): API, Dependencies, Development, GetNodes, Installation, krkn-operator-data-provider, Overview, Prerequisites (+3 more)

### Community 168 - "test-api.sh"
Cohesion: 0.39
Nodes (11): api_call(), cleanup(), extract_body(), extract_http_code(), log_error(), log_info(), log_step(), log_success() (+3 more)

### Community 169 - ".loadCategoryResiliencyHistories"
Cohesion: 0.27
Nodes (10): categoryHistoryScore, containsString(), filterCategoryHistoryScoresByCluster(), filterGraphHistoryScoresByPermission(), filterScenarioHistoryScoresByPermission(), graphHistoryScores(), runHasAnyCategory(), scenarioHistoryScores() (+2 more)

### Community 170 - "ComponentManifest"
Cohesion: 0.24
Nodes (6): ListComponentsResponse, Load(), TestLoad(), TestLoadRejectsInvalidManifest(), ComponentDescription, ComponentManifest

### Community 171 - "target"
Cohesion: 0.18
Nodes (10): apiVersion, kind, patches, path, target, resources, group, kind (+2 more)

### Community 173 - "encoding/json.RawMessage"
Cohesion: 0.24
Nodes (9): AlertDocument, esAlertSearchResponse, esSearchResponse, filterAggregation, QueryAlertsResponse, termsAggregation, decodeAlerts(), redactSensitiveParameters() (+1 more)

### Community 174 - "jobs"
Cohesion: 0.18
Nodes (10): name, needs, permissions, runs-on, jobs, create-manifests, name, on (+2 more)

### Community 175 - "name"
Cohesion: 0.33
Nodes (11): steps, steps, COMMUNITY_OPERATORS_FORK, GH_TOKEN, KUBERNETES_OPERATORS_FORK, steps, env, name (+3 more)

### Community 176 - "slash-to-add-label.yml"
Cohesion: 0.18
Nodes (10): permissions, uses, types, jobs, add-label, name, on, issue_comment (+2 more)

### Community 177 - "newReportTestHandler"
Cohesion: 0.29
Nodes (10): newReportTestHandler(), reportRequest(), TestDownloadReportDeniesUserWithoutReportClusterAccess(), TestDownloadReportHTMLThroughV2Router(), TestDownloadReportPDF(), TestDownloadReportRejectsMalformedPath(), TestDownloadReportUnavailable(), TestGetReportStatusPending() (+2 more)

### Community 178 - "API Workflow Test"
Cohesion: 0.18
Nodes (10): API Workflow Test, Configuration, Error Handling, krkn-operator Test Scripts, Output, Overview, Prerequisites, Testing Against Production (+2 more)

### Community 179 - "KrknComponent"
Cohesion: 0.29
Nodes (3): KrknComponent, KrknComponentList, KrknComponentStatus

### Community 181 - ".CreateWorkflow"
Cohesion: 0.29
Nodes (8): validateWorkflowCategories(), FromFileContent(), StudioLayoutToJSON(), stringPtr(), TestToFileContent(), TestValidateWorkflowGraph(), ToFileContent(), ValidateWorkflowGraph()

### Community 182 - "UnmarshalSecretData"
Cohesion: 0.24
Nodes (9): TestUpdateTarget(), TestBuildClusterTargetsRecordsLiveness(), testTargetSecretData(), MarshalSecretData(), TestMarshalUnmarshalSecretData(), TestUnmarshalSecretData_Invalid(), UnmarshalSecretData(), TestRefreshTargetStatusesMarksUnreachableTargetNotReady() (+1 more)

### Community 183 - "TokenGenerator"
Cohesion: 0.27
Nodes (4): Claims, mockTokenGen(), mockTokenGenInvalid(), TokenGenerator

### Community 184 - "manager.yaml"
Cohesion: 0.20
Nodes (9): apiVersion, kind, app.kubernetes.io/name, control-plane, matchLabels, spec, replicas, selector (+1 more)

### Community 185 - "labels"
Cohesion: 0.20
Nodes (10): kubectl.kubernetes.io/default-container, app.kubernetes.io/managed-by, app.kubernetes.io/name, control-plane, metadata, annotations, labels, name (+2 more)

### Community 186 - "workspace_role.yaml"
Cohesion: 0.20
Nodes (9): apiVersion, kind, metadata, name, namespace, rules, apiGroups, resources (+1 more)

### Community 187 - "ValidateCreateRequest"
Cohesion: 0.22
Nodes (10): CreateElasticsearchConfigRequest, UpdateElasticsearchConfigRequest, strPtr(), TestValidateCreateRequest(), TestValidateUpdateRequest(), validateAccess(), ValidateCreateRequest(), ValidateMergedConnection() (+2 more)

### Community 188 - "sortKrknAIScenarios"
Cohesion: 0.36
Nodes (9): KrknAIScenarioIndexItem, compareInt(), compareKrknAIScenarioIdentity(), compareKrknAIScenarioIDs(), compareKrknAIScenarioMeasurements(), compareKrknAIScenarioStrings(), isKrknAIScenarioNumericID(), krknAIScenarioStatus() (+1 more)

### Community 189 - "gateway"
Cohesion: 0.22
Nodes (9): gateway, annotations, enabled, gatewayName, gatewayNamespace, hostname, path, pathType (+1 more)

### Community 190 - "containers"
Cohesion: 0.22
Nodes (9): args, command, env, image, imagePullPolicy, name, ports, volumeMounts (+1 more)

### Community 191 - "labels"
Cohesion: 0.22
Nodes (8): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, control-plane, metadata, labels, name

### Community 192 - "metrics_auth_role.yaml"
Cohesion: 0.22
Nodes (8): apiVersion, kind, metadata, name, rules, apiGroups, resources, verbs

### Community 193 - "role.yaml"
Cohesion: 0.22
Nodes (8): apiVersion, kind, metadata, name, rules, apiGroups, resources, verbs

### Community 194 - "Optional Krkn-AI with OLM on OpenShift"
Cohesion: 0.22
Nodes (9): 1. Build a development bundle, 2. Install with AI disabled, 3. Enable AI, 4. Verify a run, Disable, upgrade, and remove, Optional Krkn-AI with OLM on OpenShift, Prerequisites, Reset a disposable test installation (+1 more)

### Community 195 - "publish-olm-bundles"
Cohesion: 0.22
Nodes (9): IMAGE_NAMESPACE, REGISTRY, USE_IMAGE_DIGESTS, publish-olm-bundles, env, name, needs, permissions (+1 more)

### Community 196 - "entries"
Cohesion: 0.25
Nodes (8): entries, defaultChannel, entries, name, package, replaces, schema, schema

### Community 197 - "ClusterTarget"
Cohesion: 0.29
Nodes (6): ClustersResponse, countClustersFromTargetData(), TestCountClustersFromTargetData(), TestFilterClustersByPermission(), ClusterHealthStatus, ClusterTarget

### Community 198 - "manager/kustomization.yaml"
Cohesion: 0.25
Nodes (7): apiVersion, images, name, newName, newTag, kind, resources

### Community 199 - "spec"
Cohesion: 0.25
Nodes (8): type, runAsNonRoot, seccompProfile, securityContext, serviceAccountName, terminationGracePeriodSeconds, volumes, spec

### Community 200 - "metrics_reader_role.yaml"
Cohesion: 0.25
Nodes (7): apiVersion, kind, metadata, name, rules, nonResourceURLs, verbs

### Community 201 - "config.yaml"
Cohesion: 0.25
Nodes (7): apiVersion, kind, metadata, name, stages, parallel, tests

### Community 202 - "submit-operatorhub-catalog"
Cohesion: 0.25
Nodes (8): RELEASE_TAG, submit-operatorhub-catalog, env, env, if, name, needs, runs-on

### Community 203 - "buildClusterAPIURLMap"
Cohesion: 0.25
Nodes (8): buildClusterAPIURLMap(), contains(), containsInMiddle(), TestBuildClusterAPIURLMap(), TestBuildClusterAPIURLMap_Collision(), TestBuildClusterAPIURLMap_SameNameSameURL(), TestValidateScenarioRunAccess(), TestValidateScenarioRunAccess_NoGroups()

### Community 204 - "CategoryResiliencyHistoryResponse"
Cohesion: 0.38
Nodes (7): CategoryConfigurationGroup, CategoryResiliencyDataPoint, CategoryResiliencyHistoryResponse, ResiliencyHistoryQueryResponse, datapointForRunID(), datapointRunIDs(), assembleResiliencyHistoryQueryResponse()

### Community 205 - ".ExecuteKubectl"
Cohesion: 0.33
Nodes (3): DataProviderServiceClient, dataProviderServiceClient, NewDataProviderServiceClient()

### Community 206 - "ScenariosResponse"
Cohesion: 0.48
Nodes (4): publicScenarioCache, publicScenarioCacheEntry, ScenariosResponse, cloneScenariosResponse()

### Community 207 - "volumes"
Cohesion: 0.29
Nodes (7): volumes, sizeLimit, sizeLimit, sizeLimit, output, tmp, uploadState

### Community 209 - "entries"
Cohesion: 0.33
Nodes (6): entries, entries, name, package, schema, schema

### Community 210 - "Running Locally"
Cohesion: 0.29
Nodes (7): 1. Start the gRPC data provider, 2. Run the operator, 3. Create an admin user and log in, 4. (Optional) Run the web console, Architecture overview, Prerequisites, Running Locally

### Community 211 - "Krkn-Operator Roadmap"
Cohesion: 0.29
Nodes (6): Krkn-Operator Roadmap, Ongoing (not release-blocked), Release 1 — 2026-08-18, Release 2 — 2026-10-18, Release 3 — 2026-12-22, Release 4 — 2027-02-23

### Community 212 - ".ValidateGraph"
Cohesion: 0.33
Nodes (5): SanitizeNodeIDForKubernetes(), stringPtr(), TestCollisionDetection(), TestKrknGraphRun_ValidateGraph(), TestSanitizeNodeIDForKubernetes()

### Community 214 - "submit-community-catalog"
Cohesion: 0.33
Nodes (6): submit-community-catalog, if, name, needs, runs-on, steps

### Community 215 - "ComputeJobStats"
Cohesion: 0.40
Nodes (5): ComputeJobStats(), TestComputeJobStats(), Compute(), TestCompute(), Summary

### Community 216 - "newProvidersHandler"
Cohesion: 0.40
Nodes (5): newProvider(), newProvidersHandler(), TestListProviders_Empty(), TestListProviders_ScopedToNamespace(), TestUpdateProviderStatus()

### Community 217 - "setup-ocm-and-operator.sh"
Cohesion: 0.67
Nodes (5): log_error(), log_info(), log_warn(), setup-ocm-and-operator.sh script, wait_for_condition()

### Community 218 - "updateStrategy"
Cohesion: 0.40
Nodes (5): maxSurge, maxUnavailable, updateStrategy, rollingUpdate, type

### Community 219 - "Available Make Targets"
Cohesion: 0.40
Nodes (5): Available Make Targets, Build Targets, Deployment Targets, Other Targets, Push Targets

### Community 220 - "Building Images"
Cohesion: 0.40
Nodes (5): Build Both Images, Build Individual Images, Building Images, Custom Registry/Images, Using Podman

### Community 221 - "Troubleshooting"
Cohesion: 0.40
Nodes (5): gRPC Connection Issues, Image Pull Errors, Namespace Issues, Pod Not Starting, Troubleshooting

### Community 222 - "steps"
Cohesion: 0.40
Nodes (5): name, permissions, runs-on, steps, build-data-provider-arm64

### Community 223 - "steps"
Cohesion: 0.40
Nodes (5): name, permissions, runs-on, steps, build-operator-arm64

### Community 224 - "Setup"
Cohesion: 0.40
Nodes (5): 1. Create Virtual Environment (Python 3.11), 2. Install Dependencies, 3. Verify Generated Stubs, 4. Start gRPC Server, Setup

### Community 225 - "Troubleshooting"
Cohesion: 0.40
Nodes (5): Error: "Command not found: kubectl", Error: "failed to connect to gRPC server", Error: "Failed to get kubeconfig", Error: "no module named 'krkn_lib'", Troubleshooting

### Community 226 - "livenessProbe"
Cohesion: 0.50
Nodes (4): livenessProbe, httpGet, initialDelaySeconds, periodSeconds

### Community 227 - "readinessProbe"
Cohesion: 0.50
Nodes (4): readinessProbe, httpGet, initialDelaySeconds, periodSeconds

### Community 228 - "olm/kustomization.yaml"
Cohesion: 0.50
Nodes (3): apiVersion, kind, resources

### Community 229 - "Complete Workflow Examples"
Cohesion: 0.50
Nodes (4): Complete Workflow Examples, Custom Registry, Development (Local), Production (With Git Tag)

### Community 230 - ".gosec.yaml"
Cohesion: 0.50
Nodes (3): exclude-dirs, exclude-generated, excludes

### Community 231 - "channel-head.sh"
Cohesion: 0.50
Nodes (3): OLM_CHANNEL_NAME, OLM_PACKAGE_NAME, channel-head.sh script

### Community 232 - "select-ai-release.sh"
Cohesion: 0.83
Nodes (3): inspect_image(), select-ai-release.sh script, usage()

### Community 233 - "krkngraphrun_controller_sanitization_test.go"
Cohesion: 0.83
Nodes (3): isAlphanumeric(), isValidLabelChar(), TestSanitizeNodeID()

### Community 237 - "resources"
Cohesion: 0.67
Nodes (3): resources, limits, requests

### Community 238 - "securityContext"
Cohesion: 0.67
Nodes (3): securityContext, allowPrivilegeEscalation, capabilities

## Knowledge Gaps
- **21 isolated node(s):** `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME`, `ICON_BASE64`, `ICON_MEDIATYPE` (+16 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 1971 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **43 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Handler` connect `Handler` to `main.go`, `ConnectionParams`, `KrknScenarioRun`, `sigs.k8s.io/controller-runtime/pkg/client.Client`, `KrknAIRun`, `InjectCredentials`, `context.Context`, `k8s.io/api/core/v1.Secret`, `net/http.Response`, `.UpdateFile`, `IsAdmin`, `ParseCommand`, `github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2`, `.loadCategoryResiliencyHistories`, `ai.go`, `.CreateWorkflow`, `UnmarshalSecretData`, `TokenGenerator`, `k8s.io/api/core/v1.ConfigMap`, `rollbackTestHandler`, `KrknUserGroup`, `category_run_configuration.go`, `.getKrknAIRunScenarioIndex`, `scenario_catalog_cache_test.go`, `Server`, `.ExecuteKubectl`, `BuildUnifiedJobList`, `ScenariosResponse`, `KrknCategory`, `time.Time`, `KrknOperatorTarget`, `KrknUser`, `.GetScenarioRunLogs`, `Hub`, `.PostScenarioRun`?**
  _High betweenness centrality (0.030) - this node is a cross-community bridge._
- **Are the 163 inferred relationships involving `NewTestHandler()` (e.g. with `setupAuthTestHandler()` and `TestLogin_InactiveUser()`) actually correct?**
  _`NewTestHandler()` has 163 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME` to the rest of the system?**
  _21 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.07568563531996315 - nodes in this community are weakly interconnected._
- **Why does `KrknAIRun` connect `KrknAIRun` to `main.go`, `Optional Krkn-AI with OLM on OpenShift`, `KrknUserGroup`, `.getKrknAIRunScenarioIndex`, `krknairun_cancellation_test.go`, `.UpdateFile`, `KrknUser`, `ExtractRegistryV2FromSecret`, `k8s.io/apimachinery/pkg/apis/meta/v1.TypeMeta`?**
  _High betweenness centrality (0.025) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.015160210425633668 - nodes in this community are weakly interconnected._
- **Why does `Optional Krkn-AI with OLM on OpenShift` connect `Optional Krkn-AI with OLM on OpenShift` to `README.md`, `KrknAIRun`?**
  _High betweenness centrality (0.024) - this node is a cross-community bridge._