# Graph Report - krkn-operator  (2026-11-03)

## Corpus Check
- 439 files · ~459,204 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 18 file(s) not represented in the graph (top: (none) 12, .jsonl 2, .toml 1)

## Summary
- 5591 nodes · 17093 edges · 263 communities (224 shown, 39 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 1352 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `d6fc7ade`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- main.go
- testing.T
- Handler
- .DeepCopyInto
- .UpdateFile
- sigs.k8s.io/controller-runtime/pkg/client.Client
- NewTestHandler
- KrknAIRun
- utils.go
- KrknScenarioRun
- api/types.go
- .GetScenarioRunStatus
- migration.go
- addAdminContext
- makeScenarioRun
- GetClaimsFromContext
- render-bundle.sh
- server.py
- Store
- handler_test.go
- WSUnifiedJobItem
- user_handlers_test.go
- client_test.go
- KrknGraphRun
- elasticsearch/types.go
- InjectCredentials
- installer.go
- migration_test.go
- KrknTargetRequest
- elasticsearch/client.go
- backup_restore_handlers_test.go
- setupTestHandler
- Broadcaster
- .QueryTelemetryPage
- KrknOperatorTargetProviderConfig
- wsorigin.go
- .DeepCopy
- k8s.io/api/core/v1.Secret
- krkn_ai_types.go
- newKrknAITestHandler
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- k8s.io/client-go/kubernetes.Interface
- UnifiedJobItem
- TokenGenerator
- github.com/go-logr/logr.Logger
- k8s.io/apimachinery/pkg/runtime.Scheme
- notify-operator-docs.test.js
- API Changes
- createAdminContext
- time.Time
- with
- WorkflowResponse
- time.Duration
- ScenarioReference
- versions
- github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2
- cloudcreds/types.go
- GetNodesRequest
- dataprovider_grpc.pb.go
- GroupLabelKey
- converter_test.go
- versions
- ScenarioRunRequest
- category_run_configuration.go
- versions
- ResolveGraph
- versions
- time.Duration
- rate_limiter_test.go
- .GetScenarioRunLogs
- Publisher
- start_operator.sh
- KrknUserGroup
- PROGRESS-MVP.md
- Compute
- buildScenarioRunResponse
- KrknOperatorTargetProvider
- .UpdateRegistry
- test-api.sh
- Frontend Authentication Implementation Tasks
- ExecuteKubectlRequest
- mockJob
- ValidateCreateRequest
- captureLogSink
- versions
- versions
- HashPassword
- versions
- guardedDialContext
- GET methods
- .flatten
- contains
- versions
- versions
- ExecuteKubectlResponse
- versions
- Provider Configuration API Specification
- versions
- dataprovider.pb.go
- TestKrknGraphRun_ValidateGraph
- newProvidersHandler
- Authentication API Specification
- versions
- versions
- setup-ocm-and-operator.sh
- File Management API Specification
- versions
- commands.go
- Cluster Segregation Troubleshooting
- channel-head.sh
- Running gRPC Data Provider Locally for Development
- versions
- api_workflow_test.sh
- select-compatible-release.sh
- test-open-kubernetes-operators-pr.sh
- CLAUDE.md - Project Instructions
- post-install.sh
- generate_proto_go.sh
- ClusterNameCollisionError
- test-channel-head.sh
- test-chart-exposure.sh
- test-select-compatible-release.sh
- generate_proto.sh
- debug-cluster-permissions.sh
- notify-operator-docs.sh
- github.com/krkn-chaos/krkn-operator
- versions
- Implementation Details
- Leader Election and API Server Separation
- krkn-operator
- Deployment Guide
- Quick Start - Terminal API Local Testing
- Resiliency Score Feature - Frontend Implementation Guide
- AGENTS.md
- Option 1: Helm (Recommended)
- krkn-operator-data-provider
- 🔧 TODO: Pod Recreation and Retry Logic
- API Workflow Test
- scenario_filter_test.go
- JobTracker
- Krkn-Operator Roadmap
- dataProviderServiceClient
- versions
- DataProviderServiceServer
- Running Locally
- install-yq.sh
- test-release-workflow-gates.sh
- Deployment
- Available Make Targets
- Building Images
- versions
- versions
- Pushing Images
- versions
- context.Context
- assets/README.md
- Handler
- spec
- names
- names
- containers
- notify-website
- Server
- net/http.Request
- render-and-validate
- steps
- steps
- ai_test.go
- steps
- values.yaml
- allow-metrics-traffic.yaml
- endpoints
- ai.go
- images
- values-olm-kubernetes.yaml
- labels
- ocm-setup
- linters
- limits
- values-olm-ocp.yaml
- ports
- krknAIArchiveUpstreamError
- steps
- steps
- submit-operatorhub-catalog
- leader_election_role_binding.yaml
- role_binding.yaml
- stale.yml
- AIConfig
- aiDeploymentFieldsMatch
- acm
- default/kustomization.yaml
- .QueryAlerts
- needs-rebase.yml
- with
- Chart.yaml
- metrics_auth_role_binding.yaml
- console
- ingress
- leader_election_role.yaml
- AIResources
- target
- publish-olm-bundles
- slash-to-add-label.yml
- newReportTestHandler
- manager.yaml
- labels
- jobs
- KrknScenarioRunSpec
- gateway
- serviceMonitor
- operator
- main
- containers
- labels
- metrics_auth_role.yaml
- role.yaml
- entries
- manager/kustomization.yaml
- spec
- metrics_reader_role.yaml
- config.yaml
- contents
- submit-community-catalog
- buildClusterAPIURLMap
- CategoryResiliencyHistoryResponse
- service
- entries
- seccompProfile
- networkPolicy
- auth
- updateStrategy
- file_proto_dataprovider_proto_rawDescGZIP
- steps
- steps
- GraphClusterScore
- Setup
- CreateBackup
- livenessProbe
- readinessProbe
- olm/kustomization.yaml
- Complete Workflow Examples
- generateKubeconfigFromRequest
- .gosec.yaml
- resources
- securityContext
- crd/kustomization.yaml
- manifests/kustomization.yaml
- network-policy/kustomization.yaml
- prometheus/kustomization.yaml
- rbac/kustomization.yaml
- submit-operatorhub-catalog
- k8s.io/apimachinery/pkg/apis/meta/v1/unstructured.Unstructured
- sync_test.go
- elasticsearch/labels_test.go
- KrknUserGroupSpec
- extractTarGz
- krkngraphrun_controller_sanitization_test.go
- StripCloudEnvVars
- KrknGraphRunSpec

## God Nodes (most connected - your core abstractions)
1. `Handler` - 226 edges
2. `NewTestHandler()` - 168 edges
3. `writeJSONError()` - 142 edges
4. `writeJSON()` - 106 edges
5. `createAdminContext()` - 92 edges
6. `GetClaimsFromContext()` - 64 edges
7. `newEsScheme()` - 57 edges
8. `KrknScenarioRun` - 55 edges
9. `createUserContext()` - 43 edges
10. `KrknGraphRun` - 41 edges

## Surprising Connections (you probably didn't know these)
- `2. Run the operator` --references--> `KrknComponent`  [INFERRED]
  README.md → api/v1alpha1/krkncomponent_types.go
- `BREAKING CHANGES` --references--> `KrknOperatorTarget`  [INFERRED]
  spec.orig/REQUIREMENTS.md → api/v1alpha1/krknoperatortarget_types.go
- `Overview` --references--> `KrknOperatorTarget`  [INFERRED]
  spec.orig/REQUIREMENTS.md → api/v1alpha1/krknoperatortarget_types.go
- `1. List Providers` --references--> `KrknOperatorTargetProvider`  [INFERRED]
  docs/api-provider-config-spec.md → api/v1alpha1/krknoperatortargetprovider_types.go
- `Overview` --references--> `KrknOperatorTargetProviderConfig`  [INFERRED]
  docs/api-provider-config-spec.md → api/v1alpha1/krknoperatortargetproviderconfig_types.go

## Import Cycles
- None detected.

## Communities (263 total, 39 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.09
Nodes (15): CategoryEntityAssociationResponse, CreateCategoryRequest, krknAIScenarioKey, ResiliencyHistoryQueryRequest, UpdateCategoryRequest, ContextKey, clusterTarget, jobOutcome (+7 more)

### Community 1 - "testing.T"
Cohesion: 0.01
Nodes (254): TestKrknUserCreation(), TestKrknUserList(), TestKrknUserRoles(), TestKrknUserStatus(), TestKrknUserValidation(), TestScenarioReferenceValidate(), setupAuthTestHandler(), TestIsRegistered_MethodNotAllowed() (+246 more)

### Community 2 - "Handler"
Cohesion: 0.07
Nodes (19): writeScenarioRunAccessError(), writeCategoryRunNotFoundOrInternalError(), buildCloudCredentialResponse(), convertGraphClusterScores(), buildUserGroupResponse(), extractGroupNameAndUserIDFromPath(), extractGroupNameFromMembersPath(), extractPathSuffix() (+11 more)

### Community 3 - ".DeepCopyInto"
Cohesion: 0.04
Nodes (19): filterUsers(), GraphRunSummary, GraphScenario, KrknAIRunList, KrknAIRunStatus, KrknCategoryList, KrknCategorySpec, KrknComponentList (+11 more)

### Community 4 - ".UpdateFile"
Cohesion: 0.06
Nodes (40): AvailableFilesResponse, ListFilesResponse, buildFileInfo(), buildFileResponse(), deriveLogicalName(), extractLogicalName(), validateCreateFileRequest(), validateUpdateFileRequest() (+32 more)

### Community 5 - "sigs.k8s.io/controller-runtime/pkg/client.Client"
Cohesion: 0.07
Nodes (33): historyQueryCountingClient, k8sUserStatusChecker, KrknAIServiceConfig, RestoreConfig, removeGroupFromAllMembers(), NewHandler(), TestKrknAIOrchestratorLogsWebSocketAuthorizationAndContainers(), NewServer() (+25 more)

### Community 6 - "NewTestHandler"
Cohesion: 0.07
Nodes (103): countingGetClient, esDoerFunc, TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(), TestCloudCredentialContract_GraphRunStripsNodeEnv(), boolPtr(), newEsAlertsTestSecretWithHost(), newEsScheme(), newEsTestSecret() (+95 more)

### Community 7 - "KrknAIRun"
Cohesion: 0.05
Nodes (48): holdChildPodDeleteClient, KrknAIRunReconciler, 1. Build a bundle containing your local changes, 2. Install the development bundle with AI disabled, 3. Enable AI through the Subscription, 4. Exercise the API and an actual run, 5. Verify disable and upgrade retention, OLM configuration (+40 more)

### Community 8 - "utils.go"
Cohesion: 0.16
Nodes (16): tokenRequest, TestE2E(), getMetricsOutput(), GetNonEmptyLines(), GetProjectDir(), InstallCertManager(), InstallPrometheusOperator(), IsCertManagerCRDsInstalled() (+8 more)

### Community 9 - "KrknScenarioRun"
Cohesion: 0.06
Nodes (36): addKnownTypes(), KrknScenarioRunReconciler, preparedJobResources, TestAppendCloudCredentialInjection_EmptyRefNoop(), TestAppendCloudCredentialInjection_UsesSecretKeyRef(), TestScenarioRunReconcileSkipsPendingMigration(), countJobPhaseChanges(), extractDelimitedContent() (+28 more)

### Community 10 - "api/types.go"
Cohesion: 0.05
Nodes (43): ActiveRunsOverviewResponse, AddGroupMemberRequest, AddGroupMemberResponse, ChangePasswordRequest, ChangePasswordResponse, ClusterPermissionSet, ClusterResiliencyScoreResponse, CreateTargetResponse (+35 more)

### Community 11 - ".GetScenarioRunStatus"
Cohesion: 0.18
Nodes (10): convertClusterResiliencyScores(), averageResiliencyScore(), convertMetaTime(), scenarioReferenceForResponse(), BuildUnifiedJobList(), getDefaultPageSize(), runCategoryNames(), scenarioRunImage() (+2 more)

### Community 12 - "migration.go"
Cohesion: 0.15
Nodes (35): clusterJobImage, clusterJobProvider, clusterJobRetries, graphScenarioReferences, migrationBlockedError, migrationPlan, resiliencyScoreStatus, addProvider() (+27 more)

### Community 13 - "addAdminContext"
Cohesion: 0.11
Nodes (43): addAdminContext(), addUserContext(), createTestAdminUser(), setupFilesTestHandler(), TestCanAccessFile(), TestCreateFile(), TestCreateFile_DefaultFilePurpose(), TestCreateFile_DuplicateName() (+35 more)

### Community 14 - "makeScenarioRun"
Cohesion: 0.27
Nodes (12): makeGraphRun(), makeScenarioRun(), TestListJobs_IncludesStats(), TestListJobs_LimitOnlyReturnsAll(), TestListJobs_NoPagination(), TestListJobs_Paginated(), TestListJobs_SkipsGraphRunChildren(), TestListJobs_TypeDiscriminator() (+4 more)

### Community 15 - "GetClaimsFromContext"
Cohesion: 0.12
Nodes (15): CategoryListResponse, CategoryResponse, buildCategoryResponse(), buildCategoryVisibilityLabels(), categoryErrorCode(), categoryGroups(), categoryIsPublic(), categoryVisibleToUser() (+7 more)

### Community 16 - "render-bundle.sh"
Cohesion: 0.06
Nodes (37): channel_has_entries(), ensure_catalog_channel(), catalog-channel.sh script, update_default_channel(), write_release_config(), catalog_submit_checkout(), catalog_submit_open_pr(), catalog-submit.sh script (+29 more)

### Community 17 - "server.py"
Cohesion: 0.06
Nodes (6): add_DataProviderServiceServicer_to_server(), DataProviderService, DataProviderServiceServicer, DataProviderServiceStub, DataProviderServicer, serve()

### Community 18 - "Store"
Cohesion: 0.13
Nodes (10): ConfigStoreInitializer, NewConfigStoreInitializer(), TestConfigStoreInitializer_NeedLeaderElection(), TestConfigStoreInitializer_Start_ConfigMapNotFound(), TestConfigStoreInitializer_Start_Success(), Store, Get(), BackfillLegacyProviderConfigLabel() (+2 more)

### Community 19 - "handler_test.go"
Cohesion: 0.10
Nodes (33): NewHandler(), NewBroadcaster(), TestBroadcastDashboardUpdate(), TestBroadcastGraphRunDeleted(), TestBroadcastGraphRunUpdate(), TestBroadcastOnlyToSubscribedClients(), TestBroadcastScenarioRunDeleted(), TestBroadcastScenarioRunUpdate() (+25 more)

### Community 20 - "WSUnifiedJobItem"
Cohesion: 0.06
Nodes (24): TestBuildUnifiedJobList_SkipsGraphRunChildren(), TestFilterJobsByCategoriesMatchesAnySelectedCategory(), TestPaginateJobItems(), buildGraphRunResponse(), buildUnifiedJobList(), computeWSJobStats(), convertGraphClusterScoresForSnapshot(), filterJobCategoriesByVisibility() (+16 more)

### Community 21 - "user_handlers_test.go"
Cohesion: 0.14
Nodes (34): createTestUser(), kvValue(), setupUserTestHandler(), TestChangePassword_AdminChange_NoCurrentPassword(), TestChangePassword_AdminChangesOtherUser_EmitsAuditLog(), TestChangePassword_SamePassword_Rejected(), TestChangePassword_SelfChange_NoAdminAuditLog(), TestChangePassword_SelfChange_RequiresCurrentPassword() (+26 more)

### Community 22 - "client_test.go"
Cohesion: 0.09
Nodes (30): doerFunc, roundTripperFunc, NewClient(), intPtr(), scenarioParametersSchema(), TestBaseURL(), TestClientTransportDefaultNotHTTPTransport(), TestClientTransportPooling() (+22 more)

### Community 23 - "KrknGraphRun"
Cohesion: 0.11
Nodes (14): imageResolver, imageSignatureVerifier, KrknGraphRunReconciler, migrationPending(), TestGraphRunReconcileSkipsPendingMigration(), TestLoadFileConfigMapByID(), TestTranslateVolumesToFileMounts(), sanitizeNodeID() (+6 more)

### Community 24 - "elasticsearch/types.go"
Cohesion: 0.15
Nodes (19): AffectedPods, CreateElasticsearchConfigResponse, DeleteElasticsearchConfigResponse, ListElasticsearchConfigsResponse, QueryAlertsRequest, QueryTelemetryRequest, RawJSON, RecoveredPod (+11 more)

### Community 25 - "InjectCredentials"
Cohesion: 0.23
Nodes (19): injectAWS(), injectAzure(), injectBaremetal(), InjectCredentials(), injectGCP(), injectIBMCloud(), injectOpenStack(), injectVMware() (+11 more)

### Community 26 - "installer.go"
Cohesion: 0.09
Nodes (28): configMapClient, crdClient, crdManifest, deploymentClient, applyCRD(), applyCRDs(), createMissingCRDs(), ensureMigrationGuard() (+20 more)

### Community 27 - "migration_test.go"
Cohesion: 0.14
Nodes (33): migrationResult, migrationStats, resourceDefinition, checkAnnotationSize(), Complete(), forEachResource(), logSummary(), patchMigrationResourceWithRetry() (+25 more)

### Community 28 - "KrknTargetRequest"
Cohesion: 0.08
Nodes (17): ClustersResponse, KrknOperatorTargetProviderConfigReconciler, KrknTargetRequestReconciler, Architecture Decision, Components Requiring Leader Election (Leader-Only), Components Running on All Replicas (Multi-Active), isConflictError(), checkProviderActive() (+9 more)

### Community 29 - "elasticsearch/client.go"
Cohesion: 0.14
Nodes (19): esAlertSearchResponse, esSearchResponse, facetField, filterAggregation, Option, StatusError, termsAggregation, buildAlertsSearchBody() (+11 more)

### Community 30 - "backup_restore_handlers_test.go"
Cohesion: 0.15
Nodes (22): NewJobTracker(), createMultipartBody(), createTestHandler(), TestGetRestoreStatusAdminOnly(), TestGetRestoreStatusTrackedJob(), TestGetRestoreStatusUnknownJob(), TestHandlerShutdownCancelsContext(), TestJobTrackerEviction() (+14 more)

### Community 31 - "setupTestHandler"
Cohesion: 0.09
Nodes (25): setupTestHandler(), TestCreateTarget_DuplicateClusterName(), TestCreateTarget_MissingRequiredFields(), TestCreateTarget_WithCredentials(), TestCreateTarget_WithKubeconfig(), TestCreateTarget_WithToken(), TestDeleteQuietly_MissingObjectIsNoOp(), TestDeleteQuietly_RunsWithCanceledContext() (+17 more)

### Community 32 - "Broadcaster"
Cohesion: 0.12
Nodes (10): hashBytes(), roleFromAdmin(), AuthorizationCheckFunc, Broadcaster, BroadcastMessage, Client, graphRunEventHandler, Hub (+2 more)

### Community 33 - ".QueryTelemetryPage"
Cohesion: 0.22
Nodes (8): Doer, executeSearch(), Client, ConnectionParams, newSearchRequest(), TestTransportKey(), transportKey(), validateInlineDestination()

### Community 34 - "KrknOperatorTargetProviderConfig"
Cohesion: 0.11
Nodes (17): 1. List Providers, 2. Get Provider Config Status, 2. Update Provider Status, 3. Create Provider Config Request, 3. Update Provider Configuration, API Endpoints, CreateProviderConfigRequest(), UpdateProviderConfig() (+9 more)

### Community 35 - "wsorigin.go"
Cohesion: 0.14
Nodes (17): checkWebSocketOriginV2(), TestCheckWebSocketOriginV2(), checkWebSocketOrigin(), TestCheckWebSocketOrigin(), guardedCheckRedirect(), TestGuardedCheckRedirect(), TestValidateDestinationURL(), validateDestinationURL() (+9 more)

### Community 37 - "k8s.io/api/core/v1.Secret"
Cohesion: 0.15
Nodes (11): updateCloudCredentialSecretData(), buildConnectionParams(), buildConnectionParamsForIndex(), buildElasticsearchConfigResponse(), canAccessElasticsearchConfig(), UpdateCloudCredentialRequest, TestValidateUpdateRequest(), ValidateUpdateRequest() (+3 more)

### Community 38 - "krkn_ai_types.go"
Cohesion: 0.12
Nodes (25): KrknAIArtifactSummary, KrknAIConfigRequest, KrknAIConfigValidationError, KrknAIConfigValidationRequest, KrknAIConfigValidationResponse, KrknAIDiscoveryRequest, KrknAIFitnessProgression, KrknAIRunRequest (+17 more)

### Community 39 - "newKrknAITestHandler"
Cohesion: 0.09
Nodes (32): krknAIArchiveTestFile, krknAIRunReadHookClient, krknAIScenarioIndexFixture, krknAIArchiveManifestJSON(), krknAIArchiveTestEntry(), newKrknAIArchiveTestHandler(), TestKrknAIRunArchiveDownloadContainsCommittedTextAndBinaryFiles(), TestKrknAIRunArchiveDownloadPropagatesMissingManifest() (+24 more)

### Community 40 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.09
Nodes (10): GraphRunListItem, GraphRunListResponse, GraphRunStatusResponse, getProviderNames(), timeEqual(), KrknOperatorTargetProviderConfigStatus, KrknOperatorTargetProviderStatus, KrknOperatorTargetStatus (+2 more)

### Community 41 - "k8s.io/client-go/kubernetes.Interface"
Cohesion: 0.13
Nodes (13): categoryEnvironmentParameter, aiOrchestratorSCCBindingName(), ensureJWTSecret(), EnsureResources(), ensureSCCBinding(), ensureScenarioRunnerRBAC(), ensureServiceAccount(), ensureServices() (+5 more)

### Community 42 - "UnifiedJobItem"
Cohesion: 0.14
Nodes (8): JobStatsSummary, PaginationMeta, ScenarioRunListItem, ScenarioRunListResponse, UnifiedJobItem, UnifiedJobsResponse, ComputeJobStats(), TestComputeJobStats()

### Community 43 - "TokenGenerator"
Cohesion: 0.11
Nodes (13): Claims, Role, loggingMiddleware(), maxBodySizeMiddleware(), TestMaxBodySizeMiddleware(), mockTokenGen(), mockTokenGenInvalid(), TokenGenerator (+5 more)

### Community 44 - "github.com/go-logr/logr.Logger"
Cohesion: 0.24
Nodes (4): visibleCategoryNames(), AuthorizationChecker, categoryVisibilityChecker, Handler

### Community 45 - "k8s.io/apimachinery/pkg/runtime.Scheme"
Cohesion: 0.32
Nodes (8): decodeErrorResponse(), providerConfigTestScheme(), TestGetProviderConfigByUUID_CompletedReturnsConfigData(), TestGetProviderConfigByUUID_InternalErrorReturnsSanitized500(), TestGetProviderConfigByUUID_NotFoundReturns404(), TestGetProviderConfigByUUID_PendingReturnsAccepted(), TestPostProviderConfig_FailureReturnsSanitizedError(), TestPostProviderConfig_ReturnsAcceptedWithUUID()

### Community 46 - "notify-operator-docs.test.js"
Cohesion: 0.10
Nodes (11): assert, chartWorkflow, { execFileSync, spawnSync }, fs, notifier, notifyWorkflow, os, path (+3 more)

### Community 47 - "API Changes"
Cohesion: 0.29
Nodes (7): GraphRunDetailResponse, GraphRunSpecResponse, 1.1 Resiliency weight per graph node, 1. Request - Create GraphRun (POST /api/v1/graphruns), 2. Response - List GraphRuns (GET /api/v1/graphruns), 3. Response - Get GraphRun Detail (GET /api/v1/graphruns/:name), API Changes

### Community 48 - "createAdminContext"
Cohesion: 0.11
Nodes (39): newAWSTestSecret(), newCloudCredScheme(), newGCPTestSecret(), newGroupedAWSSecret(), TestCloudCredentialsRouter_MethodNotAllowed(), TestCloudCredentialsRouter_SubpathMethodNotAllowed(), TestCreateCloudCredential_AWS_Success(), TestCreateCloudCredential_DuplicateName() (+31 more)

### Community 49 - "time.Time"
Cohesion: 0.15
Nodes (12): ClusterJobStatusResponse, ListTargetsResponse, NodeStatusResponse, RestoreResponse, ScenarioRunStatusResponse, TargetResponse, convertClusterJobs(), convertMetaTime() (+4 more)

### Community 50 - "with"
Cohesion: 0.04
Nodes (45): required, GH_TOKEN, INPUT_OPERATOR_TAG, OPERATOR_REPO_ROOT, PUSHED_TAG, SOURCE_RELEASE_CHART, WORKFLOW_RUN_HEAD_SHA, operator_tag (+37 more)

### Community 51 - "WorkflowResponse"
Cohesion: 0.67
Nodes (3): workflowResponsesHaveCategories(), WorkflowResponse, ListWorkflowsResponse

### Community 52 - "time.Duration"
Cohesion: 0.18
Nodes (10): CachedUserStatusChecker, countingUserStatusChecker, userStatusCacheEntry, UserStatusChecker, NewCachedUserStatusChecker(), TestCachedUserStatusChecker(), TestCachedUserStatusChecker_EvictsExpiredEntries(), TestCachedUserStatusChecker_Invalidation() (+2 more)

### Community 53 - "ScenarioReference"
Cohesion: 0.29
Nodes (10): buildContainerImage(), buildContainerImageFromReference(), getTestConfig(), privateReference(), publicReference(), TestBuildContainerImage_PublicRegistryUsesKrknctl(), TestBuildContainerImage_RejectsPrivateWithoutResolvedRegistry(), TestBuildContainerImageFromReference_PrivateRegistry() (+2 more)

### Community 54 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 55 - "github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2"
Cohesion: 0.19
Nodes (7): mockScenarioProvider, InvalidImageSignatureError, resolveImageSignature(), resolveImmutableImage(), resolvePrivateImageDigest(), scenarioTagForImage(), TestResolvePrivateImageDigestUsesManifestDigest()

### Community 56 - "cloudcreds/types.go"
Cohesion: 0.17
Nodes (12): CreateCloudCredentialResponse, DeleteCloudCredentialResponse, ListCloudCredentialsResponse, UpdateCloudCredentialResponse, buildCloudCredentialSecretData(), CloudCredentialResponse, CreateCloudCredentialRequest, TestValidateCreateRequest() (+4 more)

### Community 57 - "GetNodesRequest"
Cohesion: 0.15
Nodes (3): DataProviderServiceClient, GetNodesRequest, GetNodesResponse

### Community 58 - "dataprovider_grpc.pb.go"
Cohesion: 0.13
Nodes (5): UnimplementedDataProviderServiceServer, UnsafeDataProviderServiceServer, _DataProviderService_ExecuteKubectl_Handler(), _DataProviderService_GetNodes_Handler(), RegisterDataProviderServiceServer()

### Community 59 - "GroupLabelKey"
Cohesion: 0.06
Nodes (63): newCategoryRequest(), newCategoryRequestAs(), newCategoryTestGroup(), newCategoryTestHandler(), newCategoryTestUser(), TestCategoriesRouterCRUD(), TestCategoriesRouterGroupCreationRequiresMembership(), TestCategoriesRouterGroupVisibility() (+55 more)

### Community 60 - "converter_test.go"
Cohesion: 0.20
Nodes (15): BuildDockerConfigJSON(), ExtractRegistryV2FromSecret(), TestBuildDockerConfigJSON_MissingCredentials(), TestBuildDockerConfigJSON_OAuth(), TestBuildDockerConfigJSON_Password(), TestExtractRegistryV2FromSecret_InvalidType(), TestExtractRegistryV2FromSecret_MissingAnnotations(), TestExtractRegistryV2FromSecret_MissingComponent() (+7 more)

### Community 61 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 62 - "ScenarioRunRequest"
Cohesion: 0.16
Nodes (15): Phase 12: Scenario Job Management - /scenarios/run Endpoints ✅, Phase 9: POST /scenarios Endpoint ✅, FileMount, GlobalsRequest, JobsListResponse, JobStatusResponse, ScenarioRunRequest, ScenarioRunResponse (+7 more)

### Community 63 - "category_run_configuration.go"
Cohesion: 0.15
Nodes (26): scenarioFileBehavior, GraphScenarioNode, writeCategoryProfileField(), appendGraphSignaturePart(), compareCategoryRunConfigurations(), compareGraphConfigurations(), compareGraphConfigurationsWithStableNodeIDs(), compareGraphNodeConfigurations() (+18 more)

### Community 64 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 65 - "ResolveGraph"
Cohesion: 0.15
Nodes (13): FromKrknctlScenarioSet(), publicScenarioReference(), publicScenario(), TestFromKrknctlScenarioSetCreatesPublicReferences(), TestToKrknctlScenarioSetUsesScenarioReference(), ToKrknctlScenarioSet(), MapScenarioNodeToScenarioRunSpec(), ResolveGraph() (+5 more)

### Community 66 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 67 - "time.Duration"
Cohesion: 0.11
Nodes (14): getTokenDuration(), TestGetTokenDuration(), API Reference, CreateProviderConfigRequest, Interfaces, NewProviderRegistration, NewProviderRegistrationWithConfig, Provider Configuration Functions (+6 more)

### Community 68 - "rate_limiter_test.go"
Cohesion: 0.13
Nodes (20): ipRateLimiter, rateLimiterEntry, clientIPFromHeaders(), extractClientIP(), hostFromRemoteAddr(), isTrustedProxy(), newIPRateLimiter(), parseTrustedProxyCIDRs() (+12 more)

### Community 69 - ".GetScenarioRunLogs"
Cohesion: 0.09
Nodes (22): resiliencyHistoryQuerySelections, filterReportLines(), isReportMarkerEnd(), isReportMarkerStart(), isWebSocketDisconnectError(), maskToken(), podLogOptions(), preStartFailureReason() (+14 more)

### Community 71 - "start_operator.sh"
Cohesion: 0.41
Nodes (12): build_operator(), check_prerequisites(), cleanup(), create_service_account(), install_crds(), main(), print_error(), print_info() (+4 more)

### Community 72 - "KrknUserGroup"
Cohesion: 0.22
Nodes (15): categoryHistoryRun, categoryHistoryScore, buildCategoryResiliencyHistory(), categoryBehaviorConfigurationSignature(), categoryRunScenarioNames(), containsString(), filterCategoryHistoryScoresByCluster(), filterGraphHistoryScoresByPermission() (+7 more)

### Community 73 - "PROGRESS-MVP.md"
Cohesion: 0.04
Nodes (41): Architecture Decisions, Data Provider Architecture, Environment Variables, Future Work (Not in Current Scope), GET /clusters, GET /nodes, GET /targets/{UUID}, Implementation Files (+33 more)

### Community 74 - "Compute"
Cohesion: 0.30
Nodes (3): Compute(), JobCounter, Summary

### Community 75 - "buildScenarioRunResponse"
Cohesion: 0.23
Nodes (8): averageResiliencyScore(), buildScenarioRunDetailResponse(), buildScenarioRunResponse(), convertClusterResiliencyScoresWS(), sanitizedClusterJobs(), TestBuildScenarioRunResponse_IncludesClusterJobs(), TestBuildScenarioRunResponse_NoClusterJobs(), ClusterResiliencyScore

### Community 76 - "KrknOperatorTargetProvider"
Cohesion: 0.08
Nodes (21): Completed, Current Status, In Progress, Pending, CheckClusterLiveness(), Basic Usage (Default Configuration), CleanupOldResources, Config Struct (+13 more)

### Community 77 - ".UpdateRegistry"
Cohesion: 0.10
Nodes (18): buildRegistryInfo(), buildRegistryResponse(), validateCreateRegistryRequest(), validateUpdateRegistryRequest(), BuildRegistryLabels(), ExtractGroupsFromLabels(), TestExtractGroupsFromLabels(), CreateRegistryRequest (+10 more)

### Community 78 - "test-api.sh"
Cohesion: 0.39
Nodes (11): api_call(), cleanup(), extract_body(), extract_http_code(), log_error(), log_info(), log_step(), log_success() (+3 more)

### Community 79 - "Frontend Authentication Implementation Tasks"
Cohesion: 0.06
Nodes (30): 1. Read the API Documentation, 1. Start the Operator, 2. Check Available Work, 2. Test Registration (First Admin), 3. Start with Foundation Task, 3. Test Login, 401 on Protected Routes, 4. Claim a Task (+22 more)

### Community 82 - "ValidateCreateRequest"
Cohesion: 0.22
Nodes (10): CreateElasticsearchConfigRequest, UpdateElasticsearchConfigRequest, strPtr(), TestValidateCreateRequest(), TestValidateUpdateRequest(), validateAccess(), ValidateCreateRequest(), ValidateMergedConnection() (+2 more)

### Community 84 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 85 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 86 - "HashPassword"
Cohesion: 0.24
Nodes (10): TestLogin_InactiveUser(), TestLogin_InvalidCredentials(), TestLogin_Success(), HashPassword(), TestHashAndVerify_Integration(), TestHashPassword(), TestHashPassword_Uniqueness(), TestVerifyPassword() (+2 more)

### Community 87 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 88 - "guardedDialContext"
Cohesion: 0.25
Nodes (4): responseWriter, guardedDialContext(), TestValidateDestinationIP(), validateDestinationIP()

### Community 89 - "GET methods"
Cohesion: 0.06
Nodes (35): API Requirements, /clusters, DELETE methods, GET methods, GET methods, get nodes, get pods, Grpc python service requirement (+27 more)

### Community 90 - ".flatten"
Cohesion: 0.22
Nodes (12): ClusterMetadata, FacetOption, NodeSummaryInfo, QueryTelemetryResponse, rawTelemetrySource, TelemetryDocument, TelemetryStats, decodeTelemetry() (+4 more)

### Community 91 - "contains"
Cohesion: 0.22
Nodes (10): isValidConfigMapKey(), validateFileFields(), contains(), findSubstring(), TestValidateFileContent(), TestValidateFileGroups(), TestValidateUserFilePermissions(), ValidateFileContent() (+2 more)

### Community 92 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 93 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 95 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 96 - "Provider Configuration API Specification"
Cohesion: 0.05
Nodes (35): ListComponentsResponse, Publisher, 1. List Available Providers (Optional), 2. Create Config Request, 3. Poll for Completion, 4. Parse Schemas and Build UI, 5. Submit Configuration, Base URL (+27 more)

### Community 97 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 99 - "TestKrknGraphRun_ValidateGraph"
Cohesion: 0.33
Nodes (5): SanitizeNodeIDForKubernetes(), stringPtr(), TestCollisionDetection(), TestKrknGraphRun_ValidateGraph(), TestSanitizeNodeIDForKubernetes()

### Community 100 - "newProvidersHandler"
Cohesion: 0.40
Nodes (5): newProvider(), newProvidersHandler(), TestListProviders_Empty(), TestListProviders_ScopedToNamespace(), TestUpdateProviderStatus()

### Community 101 - "Authentication API Specification"
Cohesion: 0.09
Nodes (22): 1. Check Admin Registration, 1. Initial Load, 2. First Admin Registration, 2. Register User, 3. Login, 3. Login, 4. Authenticated Requests, 5. Role-Based UI (+14 more)

### Community 102 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 103 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 104 - "setup-ocm-and-operator.sh"
Cohesion: 0.67
Nodes (5): log_error(), log_info(), log_warn(), setup-ocm-and-operator.sh script, wait_for_condition()

### Community 105 - "File Management API Specification"
Cohesion: 0.09
Nodes (22): Access Control, API Endpoints, ConfigMap Structure, Content Validation, Create a Public Configuration File, Create File, Delete File, Error Responses (+14 more)

### Community 107 - "versions"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 108 - "commands.go"
Cohesion: 0.13
Nodes (19): AvailableCommandsResponse, TestExecuteTerminal_CommandParsing(), TestExecuteTerminal_ValidationLogic(), GetAllowedCommands(), GetBlockedFlags(), CommandSpec, IsCommandAllowed(), IsSubcommandAllowed() (+11 more)

### Community 109 - "Cluster Segregation Troubleshooting"
Cohesion: 0.09
Nodes (22): Access Rules, Cluster Segregation Troubleshooting, Code References, Common Issues, Debugging Steps, Example 1: Proper Segregation, Example 2: Multiple Groups, Example 3: Admin User (+14 more)

### Community 110 - "channel-head.sh"
Cohesion: 0.50
Nodes (3): OLM_CHANNEL_NAME, OLM_PACKAGE_NAME, channel-head.sh script

### Community 111 - "Running gRPC Data Provider Locally for Development"
Cohesion: 0.10
Nodes (21): 1. Login and Get JWT Token, 2. Create KrknTargetRequest (or use existing UUID), 3. Test Terminal Command, Cleanup, Configure Operator to Use Local gRPC Server, Debug gRPC Server in PyCharm, Development Workflow, Error: "Command not found: kubectl" (+13 more)

### Community 112 - "versions"
Cohesion: 0.05
Nodes (36): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, metadata, annotations (+28 more)

### Community 117 - "CLAUDE.md - Project Instructions"
Cohesion: 0.10
Nodes (19): Architecture Principles, Before Committing, Before Writing Code, CLAUDE.md - Project Instructions, Code Organization & Reusability, Code Quality Standards, Communication, Critical Rules (+11 more)

### Community 128 - "versions"
Cohesion: 0.05
Nodes (36): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, metadata, annotations (+28 more)

### Community 129 - "Implementation Details"
Cohesion: 0.12
Nodes (18): Code Refactoring: getKubeconfigFromTarget() Helper ✅, Implementation Details, Phase 10: POST /scenarios/detail/{scenario_name} Endpoint ✅, Phase 11: POST /scenarios/globals/{scenario_name} Endpoint ✅, Phase 1: REST API Setup ✅, Phase 2: GET /clusters Endpoint ✅, Phase 3: Integration ✅, Phase 4: Testing ✅ (+10 more)

### Community 130 - "Leader Election and API Server Separation"
Cohesion: 0.11
Nodes (18): After (Multi-Active API), API Server Concurrent Operations, API Server Startup Wait, Before (Single Leader), Concurrency Safety, Controller Safety with Leader Election, Deployment Considerations, Expected Startup Logs (+10 more)

### Community 131 - "krkn-operator"
Cohesion: 0.11
Nodes (19): Alerts Query API, API compatibility notes, Backup and Restore, Documentation, Ecosystem, Graph node resiliency weights, Krkn-AI Operator API, krkn-operator (+11 more)

### Community 132 - "Deployment Guide"
Cohesion: 0.11
Nodes (18): Architecture, Build Installer Bundle, Deployment Guide, Environment Variables, Git Tag Support, Git Tag Workflow, gRPC Connection Issues, Image Names (Auto-Generated) (+10 more)

### Community 133 - "Quick Start - Terminal API Local Testing"
Cohesion: 0.12
Nodes (17): 1. Setup gRPC Server, 2. Run Operator (new terminal), 3. Test the API (new terminal), "401 Unauthorized", 🧪 Automated Testing, "Command not found: kubectl", 🔥 Common Issues, Debugging (+9 more)

### Community 134 - "Resiliency Score Feature - Frontend Implementation Guide"
Cohesion: 0.12
Nodes (15): Backend Implementation Details, Backend Status: ✅ Completato, Example: Resiliency Score Badge Component, Expected Behavior, Frontend Implementation Guidelines, Manual Testing Steps, Overview, Phase 1: Display Score in List View (+7 more)

### Community 135 - "AGENTS.md"
Cohesion: 0.13
Nodes (13): Beads Task Tracking, Database and project identity, Exports and Git, Go Operators â€” Only for `project.type = "go-operator"`, graphify, Handoff, Issue lifecycle, Project Configuration (+5 more)

### Community 136 - "Option 1: Helm (Recommended)"
Cohesion: 0.17
Nodes (12): Configuration Options, Custom namespace, Disable console (operator only), Install on Kubernetes with Ingress, Install on OpenShift with Route, Install with ACM integration (OpenShift), Install with defaults (console enabled, ACM disabled), Installation Methods (+4 more)

### Community 137 - "krkn-operator-data-provider"
Cohesion: 0.17
Nodes (11): API, Dependencies, Development, GetNodes, Installation, krkn-operator-data-provider, Overview, Prerequisites (+3 more)

### Community 138 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.10
Nodes (21): API and operator refactoring to support multiple clusters, API Endpoints Structure (Nested Approach), BREAKING CHANGES, Configuration, Current Behavior, Implementation Details, Implementation Plan (TODO), ✅ Implemented (+13 more)

### Community 139 - "API Workflow Test"
Cohesion: 0.18
Nodes (10): API Workflow Test, Configuration, Error Handling, krkn-operator Test Scripts, Output, Overview, Prerequisites, Testing Against Production (+2 more)

### Community 140 - "scenario_filter_test.go"
Cohesion: 0.28
Nodes (6): filterScenariosByIsAScenario(), TestFilterScenariosByIsAScenario(), TestFilterScenariosByIsAScenario_CancelledContext(), TestFilterScenariosByIsAScenario_PreservesTagMetadata(), TestValidateValueAgainstSchema(), ValidateValueAgainstSchema()

### Community 142 - "Krkn-Operator Roadmap"
Cohesion: 0.29
Nodes (6): Krkn-Operator Roadmap, Ongoing (not release-blocked), Release 1 — 2026-08-18, Release 2 — 2026-10-18, Release 3 — 2026-12-22, Release 4 — 2027-02-23

### Community 144 - "versions"
Cohesion: 0.06
Nodes (35): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+27 more)

### Community 146 - "Running Locally"
Cohesion: 0.29
Nodes (7): 1. Start the gRPC data provider, 2. Run the operator, 3. Create an admin user and log in, 4. (Optional) Run the web console, Architecture overview, Prerequisites, Running Locally

### Community 149 - "Deployment"
Cohesion: 0.33
Nodes (6): Custom Images, Custom Namespace, Deployment, Install CRDs Only, OpenShift Deployment, Quick Deploy

### Community 150 - "Available Make Targets"
Cohesion: 0.40
Nodes (5): Available Make Targets, Build Targets, Deployment Targets, Other Targets, Push Targets

### Community 151 - "Building Images"
Cohesion: 0.40
Nodes (5): Build Both Images, Build Individual Images, Building Images, Custom Registry/Images, Using Podman

### Community 152 - "versions"
Cohesion: 0.06
Nodes (35): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+27 more)

### Community 153 - "versions"
Cohesion: 0.06
Nodes (35): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+27 more)

### Community 154 - "Pushing Images"
Cohesion: 0.50
Nodes (4): Custom Push, Push Both Images, Push Individual Images, Pushing Images

### Community 155 - "versions"
Cohesion: 0.06
Nodes (35): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+27 more)

### Community 156 - "context.Context"
Cohesion: 0.06
Nodes (29): cloudCredentialAccessContext, ErrorResponse, TerminalRequest, TerminalResponse, mockUserStatusChecker, removeCategoryLabel(), validateClusterPermissions(), resolveManagedClusterKubeconfig() (+21 more)

### Community 158 - "Handler"
Cohesion: 0.13
Nodes (6): Handler, createScenarioProvider(), getScenarioSignatureStatus(), intPtr(), Client, New()

### Community 159 - "spec"
Cohesion: 0.06
Nodes (34): capabilities, categories, containerImage, description, repository, support, apiVersion, supported (+26 more)

### Community 160 - "names"
Cohesion: 0.06
Nodes (30): controller-gen.kubebuilder.io/version, apiVersion, kind, metadata, annotations, name, kind, listKind (+22 more)

### Community 161 - "names"
Cohesion: 0.06
Nodes (30): controller-gen.kubebuilder.io/version, apiVersion, kind, metadata, annotations, name, kind, listKind (+22 more)

### Community 162 - "containers"
Cohesion: 0.07
Nodes (29): apiVersion, image, imagePullPolicy, livenessProbe, name, ports, readinessProbe, resources (+21 more)

### Community 163 - "notify-website"
Cohesion: 0.07
Nodes (29): env, CHART_REPO, GH_TOKEN, IMAGE_REGISTRY, REGISTRY, jobs, notify-website, release-chart (+21 more)

### Community 165 - "net/http.Request"
Cohesion: 0.08
Nodes (19): krknAIArchiveManifest, krknAIArchiveManifestFile, krknAIArchiveServiceError, krknAIScenarioQuery, copyKrknAIArchiveServiceError(), safeKrknAIArchivePath(), copyKrknAIResponse(), decodeKrknAIRequest() (+11 more)

### Community 166 - "render-and-validate"
Cohesion: 0.07
Nodes (28): MIN_KUBE_VERSION, VERSION, version, jobs, render-and-validate, profile, name, on (+20 more)

### Community 167 - "steps"
Cohesion: 0.08
Nodes (25): description, required, type, BUNDLE_IMAGE, OLM_NAMESPACE, UPGRADE_BUNDLE_IMAGE, bundle_image, upgrade_bundle_image (+17 more)

### Community 168 - "steps"
Cohesion: 0.08
Nodes (23): concurrency, cancel-in-progress, group, COLOR, jobs, test, name, on (+15 more)

### Community 169 - "ai_test.go"
Cohesion: 0.23
Nodes (9): newAITokenSecret(), testAIConfig(), TestAIResourcesLeavesUserPVCUnchanged(), TestAIResourcesPreservesDefaultedDeploymentFields(), TestAIResourcesReconcilesNetworkPolicyDrift(), TestAIResourcesRefusesForeignObjectsAndPropagatesAPIErrors(), TestAIResourcesRejectsManagedClaimConflictsWithoutMutation(), TestAIResourcesStableTokenAndReconcilesDeploymentDrift() (+1 more)

### Community 170 - "steps"
Cohesion: 0.09
Nodes (22): name, runs-on, steps, timeout-minutes, concurrency, cancel-in-progress, group, jobs (+14 more)

### Community 171 - "values.yaml"
Cohesion: 0.09
Nodes (21): commonAnnotations, commonLabels, crds, keep, fullnameOverride, global, namespaceOverride, krknAI (+13 more)

### Community 172 - "allow-metrics-traffic.yaml"
Cohesion: 0.09
Nodes (21): apiVersion, namespaceSelector, from, ports, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, app.kubernetes.io/name (+13 more)

### Community 173 - "endpoints"
Cohesion: 0.09
Nodes (21): apiVersion, bearerTokenFile, path, port, scheme, tlsConfig, kind, app.kubernetes.io/managed-by (+13 more)

### Community 174 - "ai.go"
Cohesion: 0.37
Nodes (16): aiCreateRace(), aiDeployment(), aiLabels(), aiPodLabels(), applyLabels(), ensureAIDeployment(), ensureAINetworkPolicy(), ensureAIOrchestratorRole() (+8 more)

### Community 175 - "images"
Cohesion: 0.10
Nodes (21): image, pullPolicy, image, pullPolicy, image, pullPolicy, image, pullPolicy (+13 more)

### Community 176 - "values-olm-kubernetes.yaml"
Cohesion: 0.10
Nodes (20): acm, enabled, console, enabled, gateway, ingress, route, enabled (+12 more)

### Community 177 - "labels"
Cohesion: 0.10
Nodes (20): apiVersion, kind, app.kubernetes.io/component, app.kubernetes.io/managed-by, app.kubernetes.io/name, control-plane, metadata, labels (+12 more)

### Community 178 - "ocm-setup"
Cohesion: 0.10
Nodes (19): managed-serviceaccount-version, ocm-version, jobs, ocm-setup, include, name, name, runs-on (+11 more)

### Community 179 - "linters"
Cohesion: 0.11
Nodes (19): generated, paths, rules, formatters, enable, exclusions, linters, default (+11 more)

### Community 180 - "limits"
Cohesion: 0.13
Nodes (19): resources, aiService, replicaCount, resources, serviceToken, storage, resources, resources (+11 more)

### Community 181 - "values-olm-ocp.yaml"
Cohesion: 0.11
Nodes (18): acm, enabled, console, enabled, gateway, ingress, route, enabled (+10 more)

### Community 182 - "ports"
Cohesion: 0.11
Nodes (18): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, control-plane, metadata, labels, name (+10 more)

### Community 184 - "steps"
Cohesion: 0.11
Nodes (17): concurrency, cancel-in-progress, group, jobs, security-and-quality, name, on, pull_request (+9 more)

### Community 185 - "steps"
Cohesion: 0.11
Nodes (17): concurrency, cancel-in-progress, group, jobs, test-e2e, name, on, pull_request (+9 more)

### Community 186 - "submit-operatorhub-catalog"
Cohesion: 0.33
Nodes (11): steps, steps, COMMUNITY_OPERATORS_FORK, GH_TOKEN, KUBERNETES_OPERATORS_FORK, steps, env, name (+3 more)

### Community 187 - "leader_election_role_binding.yaml"
Cohesion: 0.12
Nodes (15): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, metadata, labels, name, roleRef (+7 more)

### Community 188 - "role_binding.yaml"
Cohesion: 0.12
Nodes (15): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, metadata, labels, name, roleRef (+7 more)

### Community 189 - "stale.yml"
Cohesion: 0.12
Nodes (15): concurrency, cancel-in-progress, group, jobs, stale, name, on, schedule (+7 more)

### Community 190 - "AIConfig"
Cohesion: 0.15
Nodes (15): krknAIPodOptions, defaultPullPolicy(), AIConfigFromEnv(), decodeStrictJSON(), defaultAIServiceResources(), localObjectReferencesEqual(), parseImagePullSecrets(), parsePullPolicy() (+7 more)

### Community 191 - "aiDeploymentFieldsMatch"
Cohesion: 0.17
Nodes (14): aiContainerMatches(), aiDeploymentFieldsMatch(), hasExactlyOneAIContainer(), hasExactlyOneAIVolume(), hasExactlyOneNamedValue(), podSecurityContextMatches(), reconcileAIContainer(), reconcileAIDeployment() (+6 more)

### Community 192 - "acm"
Cohesion: 0.16
Nodes (14): acm, affinity, config, enabled, extraEnv, logging, nodeSelector, replicaCount (+6 more)

### Community 193 - "default/kustomization.yaml"
Cohesion: 0.14
Nodes (13): apiVersion, images, name, newName, newTag, kind, namePrefix, namespace (+5 more)

### Community 194 - ".QueryAlerts"
Cohesion: 0.25
Nodes (8): AlertDocument, QueryAlertsResponse, decodeAlerts(), isSensitiveKey(), redactSensitiveParameters(), redactValueWithTracking(), TestIsSensitiveKey(), TestRedactSensitiveParameters()

### Community 195 - "needs-rebase.yml"
Cohesion: 0.14
Nodes (13): jobs, rebase, name, on, pull_request_target, push, permissions, contents (+5 more)

### Community 196 - "with"
Cohesion: 0.14
Nodes (14): with, build-args, cache-from, cache-to, context, if-no-files-found, name, password (+6 more)

### Community 197 - "Chart.yaml"
Cohesion: 0.15
Nodes (12): apiVersion, appVersion, description, home, keywords, maintainers, name, url (+4 more)

### Community 198 - "metrics_auth_role_binding.yaml"
Cohesion: 0.15
Nodes (12): apiVersion, kind, metadata, name, roleRef, apiGroup, kind, name (+4 more)

### Community 199 - "console"
Cohesion: 0.17
Nodes (12): console, affinity, enabled, extraEnv, nodeSelector, replicaCount, route, tolerations (+4 more)

### Community 200 - "ingress"
Cohesion: 0.17
Nodes (12): ingress, annotations, className, enabled, hostname, hosts, tls, tls (+4 more)

### Community 201 - "leader_election_role.yaml"
Cohesion: 0.17
Nodes (11): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, metadata, labels, name, rules (+3 more)

### Community 202 - "AIResources"
Cohesion: 0.22
Nodes (4): containsAccessMode(), requireOwnedLabels(), validateAIClaim(), AIResources

### Community 203 - "target"
Cohesion: 0.18
Nodes (10): apiVersion, kind, patches, path, target, resources, group, kind (+2 more)

### Community 204 - "publish-olm-bundles"
Cohesion: 0.22
Nodes (9): IMAGE_NAMESPACE, REGISTRY, USE_IMAGE_DIGESTS, publish-olm-bundles, env, name, needs, permissions (+1 more)

### Community 205 - "slash-to-add-label.yml"
Cohesion: 0.18
Nodes (10): permissions, uses, types, jobs, add-label, name, on, issue_comment (+2 more)

### Community 206 - "newReportTestHandler"
Cohesion: 0.29
Nodes (10): newReportTestHandler(), reportRequest(), TestDownloadReportDeniesUserWithoutReportClusterAccess(), TestDownloadReportHTMLThroughV2Router(), TestDownloadReportPDF(), TestDownloadReportRejectsMalformedPath(), TestDownloadReportUnavailable(), TestGetReportStatusPending() (+2 more)

### Community 207 - "manager.yaml"
Cohesion: 0.20
Nodes (9): apiVersion, kind, app.kubernetes.io/name, control-plane, matchLabels, spec, replicas, selector (+1 more)

### Community 208 - "labels"
Cohesion: 0.20
Nodes (10): kubectl.kubernetes.io/default-container, app.kubernetes.io/managed-by, app.kubernetes.io/name, control-plane, metadata, annotations, labels, name (+2 more)

### Community 209 - "jobs"
Cohesion: 0.18
Nodes (10): name, needs, permissions, runs-on, jobs, create-manifests, name, on (+2 more)

### Community 210 - "KrknScenarioRunSpec"
Cohesion: 0.22
Nodes (7): KrknScenarioRunSpec, TestLegacyScenarioIdentityRemainsDecodable(), TestResolveScenarioReferenceMigratesOnlyLegacyIdentity(), publicScenarioReference(), TestCreateScenarioRun_ResiliencyScore(), FileMount, ScenarioReference

### Community 211 - "gateway"
Cohesion: 0.22
Nodes (9): gateway, annotations, enabled, gatewayName, gatewayNamespace, hostname, path, pathType (+1 more)

### Community 212 - "serviceMonitor"
Cohesion: 0.22
Nodes (9): monitoring, enabled, service, serviceMonitor, annotations, enabled, interval, labels (+1 more)

### Community 213 - "operator"
Cohesion: 0.22
Nodes (9): operator, affinity, enabled, extraEnv, nodeSelector, olmBootstrap, olmOpenShift, replicaCount (+1 more)

### Community 214 - "main"
Cohesion: 0.18
Nodes (11): bootstrapOLMResources(), main(), resolveKrknAIEnabled(), resolveMigrationNamespace(), runCRDMigrationMode(), TestResolveKrknAIEnabledPrecedence(), TestResolveMigrationNamespacePrefersHookNamespace(), TestRunCRDMigrationMode() (+3 more)

### Community 215 - "containers"
Cohesion: 0.22
Nodes (9): args, command, env, image, imagePullPolicy, name, ports, volumeMounts (+1 more)

### Community 216 - "labels"
Cohesion: 0.22
Nodes (8): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, control-plane, metadata, labels, name

### Community 217 - "metrics_auth_role.yaml"
Cohesion: 0.22
Nodes (8): apiVersion, kind, metadata, name, rules, apiGroups, resources, verbs

### Community 218 - "role.yaml"
Cohesion: 0.22
Nodes (8): apiVersion, kind, metadata, name, rules, apiGroups, resources, verbs

### Community 219 - "entries"
Cohesion: 0.25
Nodes (8): entries, defaultChannel, entries, name, package, replaces, schema, schema

### Community 220 - "manager/kustomization.yaml"
Cohesion: 0.25
Nodes (7): apiVersion, images, name, newName, newTag, kind, resources

### Community 221 - "spec"
Cohesion: 0.25
Nodes (8): type, runAsNonRoot, seccompProfile, securityContext, serviceAccountName, terminationGracePeriodSeconds, volumes, spec

### Community 222 - "metrics_reader_role.yaml"
Cohesion: 0.25
Nodes (7): apiVersion, kind, metadata, name, rules, nonResourceURLs, verbs

### Community 223 - "config.yaml"
Cohesion: 0.25
Nodes (7): apiVersion, kind, metadata, name, stages, parallel, tests

### Community 224 - "contents"
Cohesion: 0.17
Nodes (12): name, permissions, runs-on, name, permissions, runs-on, steps, build-data-provider-amd64 (+4 more)

### Community 225 - "submit-community-catalog"
Cohesion: 0.33
Nodes (6): submit-community-catalog, if, name, needs, runs-on, steps

### Community 226 - "buildClusterAPIURLMap"
Cohesion: 0.25
Nodes (8): buildClusterAPIURLMap(), contains(), containsInMiddle(), TestBuildClusterAPIURLMap(), TestBuildClusterAPIURLMap_Collision(), TestBuildClusterAPIURLMap_SameNameSameURL(), TestValidateScenarioRunAccess(), TestValidateScenarioRunAccess_NoGroups()

### Community 227 - "CategoryResiliencyHistoryResponse"
Cohesion: 0.38
Nodes (7): CategoryConfigurationGroup, CategoryResiliencyDataPoint, CategoryResiliencyHistoryResponse, ResiliencyHistoryQueryResponse, datapointForRunID(), datapointRunIDs(), assembleResiliencyHistoryQueryResponse()

### Community 228 - "service"
Cohesion: 0.43
Nodes (7): service, service, service, grpcPort, nodePort, port, type

### Community 229 - "entries"
Cohesion: 0.33
Nodes (6): entries, entries, name, package, schema, schema

### Community 230 - "seccompProfile"
Cohesion: 0.47
Nodes (6): securityContext, securityContext, securityContext, type, runAsNonRoot, seccompProfile

### Community 231 - "networkPolicy"
Cohesion: 0.40
Nodes (5): networkPolicy, networkPolicy, egress, enabled, ingress

### Community 232 - "auth"
Cohesion: 0.40
Nodes (5): auth, jwtExpiryHours, jwtSecret, trustedProxyCIDRs, websocketAllowedOrigins

### Community 233 - "updateStrategy"
Cohesion: 0.40
Nodes (5): maxSurge, maxUnavailable, updateStrategy, rollingUpdate, type

### Community 236 - "steps"
Cohesion: 0.40
Nodes (5): name, permissions, runs-on, steps, build-data-provider-arm64

### Community 237 - "steps"
Cohesion: 0.40
Nodes (5): name, permissions, runs-on, steps, build-operator-arm64

### Community 238 - "GraphClusterScore"
Cohesion: 0.22
Nodes (4): convertNodeStatusesWithScores(), convertNodeStatusesWithScores(), GraphClusterScore, NodeStatus

### Community 239 - "Setup"
Cohesion: 0.40
Nodes (5): 1. Create Virtual Environment (Python 3.11), 2. Install Dependencies, 3. Verify Generated Stubs, 4. Start gRPC Server, Setup

### Community 240 - "CreateBackup"
Cohesion: 0.22
Nodes (7): BackupConfig, resourceType, backupResourceType(), CreateBackup(), TestCreateBackupExcludesProviderResources(), TestValidateBackupName(), validateBackupName()

### Community 241 - "livenessProbe"
Cohesion: 0.50
Nodes (4): livenessProbe, httpGet, initialDelaySeconds, periodSeconds

### Community 242 - "readinessProbe"
Cohesion: 0.50
Nodes (4): readinessProbe, httpGet, initialDelaySeconds, periodSeconds

### Community 243 - "olm/kustomization.yaml"
Cohesion: 0.50
Nodes (3): apiVersion, kind, resources

### Community 244 - "Complete Workflow Examples"
Cohesion: 0.50
Nodes (4): Complete Workflow Examples, Custom Registry, Development (Local), Production (With Git Tag)

### Community 245 - "generateKubeconfigFromRequest"
Cohesion: 0.36
Nodes (8): CreateTargetRequest, UpdateTargetRequest, generateKubeconfigFromCredentialsType(), generateKubeconfigFromKubeconfigType(), generateKubeconfigFromRequest(), generateKubeconfigFromTokenType(), ExtractAPIURL(), GenerateFromCredentials()

### Community 246 - ".gosec.yaml"
Cohesion: 0.50
Nodes (3): exclude-dirs, exclude-generated, excludes

### Community 247 - "resources"
Cohesion: 0.67
Nodes (3): resources, limits, requests

### Community 248 - "securityContext"
Cohesion: 0.67
Nodes (3): securityContext, allowPrivilegeEscalation, capabilities

### Community 254 - "submit-operatorhub-catalog"
Cohesion: 0.25
Nodes (8): RELEASE_TAG, submit-operatorhub-catalog, env, env, if, name, needs, runs-on

### Community 255 - "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured.Unstructured"
Cohesion: 0.36
Nodes (7): TestApplyResourceRestoresArchivedStatus(), applyResource(), applyResourcesFromFile(), isKrknCRD(), isNotFound(), isOperatorManagedSecret(), restoreArchivedStatus()

### Community 256 - "sync_test.go"
Cohesion: 0.36
Nodes (6): SyncConfigMapToStore(), TestRoundTrip(), TestSyncConfigMapToStore(), TestSyncConfigMapToStore_NilStore(), TestWriteConfigMapData(), WriteConfigMapData()

### Community 257 - "elasticsearch/labels_test.go"
Cohesion: 0.33
Nodes (5): TestBuildAnnotations(), TestBuildLabels(), TestBuildLabelsAccessControl(), TestUpdateAnnotations(), UpdateAnnotations()

### Community 259 - "extractTarGz"
Cohesion: 0.40
Nodes (5): createTarGz(), TestCreateAndExtractTarGz(), TestCreateTarGzFlushErrors(), TestExtractTarGzPathTraversal(), extractTarGz()

### Community 260 - "krkngraphrun_controller_sanitization_test.go"
Cohesion: 0.83
Nodes (3): isAlphanumeric(), isValidLabelChar(), TestSanitizeNodeID()

### Community 261 - "StripCloudEnvVars"
Cohesion: 0.50
Nodes (4): IsCloudEnvVar(), StripCloudEnvVars(), TestIsCloudEnvVar(), TestStripCloudEnvVars()

## Knowledge Gaps
- **21 isolated node(s):** `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME`, `ICON_BASE64`, `ICON_MEDIATYPE` (+16 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 1902 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **39 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `KrknScenarioRun` connect `KrknScenarioRun` to `main.go`, `testing.T`, `krkn-operator`, `.DeepCopyInto`, `KrknAIRun`, `🔧 TODO: Pod Recreation and Retry Logic`, `.GetScenarioRunStatus`, `makeScenarioRun`, `WSUnifiedJobItem`, `KrknGraphRun`, `KrknTargetRequest`, `context.Context`, `Handler`, `Broadcaster`, `.DeepCopy`, `github.com/go-logr/logr.Logger`, `GroupLabelKey`, `KrknUserGroup`, `buildScenarioRunResponse`, `KrknScenarioRunSpec`?**
  _High betweenness centrality (0.026) - this node is a cross-community bridge._
- **Are the 161 inferred relationships involving `NewTestHandler()` (e.g. with `setupAuthTestHandler()` and `TestLogin_InactiveUser()`) actually correct?**
  _`NewTestHandler()` has 161 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME` to the rest of the system?**
  _21 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.08763153355183716 - nodes in this community are weakly interconnected._
- **Why does `Handler` connect `Handler` to `main.go`, `.QueryTelemetryPage`, `Handler`, `.UpdateFile`, `k8s.io/api/core/v1.Secret`, `net/http.Request`, `.GetScenarioRunLogs`, `KrknUserGroup`, `Server`, `sigs.k8s.io/controller-runtime/pkg/client.Client`, `.GetScenarioRunStatus`, `.UpdateRegistry`, `JobTracker`, `GetClaimsFromContext`, `ai.go`, `context.Context`?**
  _High betweenness centrality (0.018) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.011759504862953138 - nodes in this community are weakly interconnected._
- **Why does `krkn-operator` connect `krkn-operator` to `Running Locally`, `README.md`?**
  _High betweenness centrality (0.017) - this node is a cross-community bridge._