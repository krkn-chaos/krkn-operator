# Graph Report - krkn-operator  (2026-10-28)

## Corpus Check
- 323 files · ~414,516 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 18 file(s) not represented in the graph (top: (none) 12, .jsonl 2, .toml 1)

## Summary
- 3549 nodes · 13695 edges · 162 communities (134 shown, 28 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 1291 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `98a6f9a0`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- main.go
- testing.T
- net/http.Request
- .DeepCopyInto
- files/types.go
- context.Context
- NewTestHandler
- k8s.io/apimachinery/pkg/apis/meta/v1/unstructured.Unstructured
- utils.go
- KrknScenarioRunReconciler
- api/types.go
- .GetScenarioReplay
- migration.go
- addAdminContext
- makeScenarioRun
- KrknCategory
- render-bundle.sh
- server.py
- ConfigStoreInitializer
- NewHub
- WSUnifiedJobItem
- user_handlers_test.go
- client_test.go
- KrknGraphRun
- elasticsearch/types.go
- InjectCredentials
- installer.go
- migration_test.go
- KrknTargetRequest
- client.go
- backup_restore_handlers_test.go
- setupTestHandler
- Broadcaster
- ConnectionParams
- KrknOperatorTargetProviderConfig
- wsorigin.go
- .DeepCopy
- Handler
- installer_test.go
- .Get
- ComponentManifest
- bootstrap.go
- UnifiedJobItem
- time.Duration
- Handler
- k8s.io/apimachinery/pkg/runtime.Scheme
- notify-operator-docs.test.js
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- watcher_test.go
- time.Time
- Hub
- GraphScenarioNode
- time.Duration
- ScenarioReference
- RestoreBackup
- github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2
- cloudcreds/types.go
- GetNodesRequest
- dataprovider_grpc.pb.go
- GroupLabelKey
- converter_test.go
- GetClaimsFromContext
- ScenarioRunRequest
- KrknScenarioRunSpec
- websocket/types.go
- ResolveGraph
- CreateBackup
- .flatten
- rate_limiter_test.go
- .GetScenarioRunLogs
- Publisher
- start_operator.sh
- KrknScenarioRun
- PROGRESS-MVP.md
- Compute
- buildScenarioRunResponse
- KrknOperatorTargetProvider
- registry/types.go
- test-api.sh
- Frontend Authentication Implementation Tasks
- ExecuteKubectlRequest
- mockJob
- ValidateCreateRequest
- captureLogSink
- KrknOperatorTargetProviderConfigStatus
- registry/labels_test.go
- TestJWTSecret
- GraphClusterScore
- ssrf.go
- GET methods
- .QueryTelemetryPage
- files/validation_test.go
- Terminal API - Frontend Integration Guide
- KrknScenarioRunSpec
- ExecuteKubectlResponse
- parseResiliencyHistoryQuery
- Provider Configuration API Specification
- Task Breakdown
- dataprovider.pb.go
- TestKrknGraphRun_ValidateGraph
- newProvidersHandler
- Authentication API Specification
- NewTokenGenerator
- API Reference
- setup-ocm-and-operator.sh
- File Management API Specification
- generateKubeconfigFromRequest
- commands.go
- Cluster Segregation Troubleshooting
- channel-head.sh
- Running gRPC Data Provider Locally for Development
- k8s.io/apimachinery/pkg/runtime.Scheme
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
- Piano Implementazione: GET /scenarios/run/replay/{jobId}
- Implementation Details
- Leader Election and API Server Separation
- krkn-operator
- Deployment Guide
- Quick Start - Terminal API Local Testing
- Resiliency Score Feature - Frontend Implementation Guide
- AGENTS.md
- Option 1: Helm (Recommended)
- krkn-operator-data-provider
- Moving from direct scenario creation to CRD approach
- API Workflow Test
- scenario_filter_test.go
- JobTracker
- Krkn-Operator Roadmap
- dataProviderServiceClient
- 🔧 TODO: Pod Recreation and Retry Logic
- DataProviderServiceServer
- Running Locally
- install-yq.sh
- test-release-workflow-gates.sh
- Deployment
- Available Make Targets
- Building Images
- Troubleshooting
- Troubleshooting
- Pushing Images
- API compatibility notes
- GetUserGroups
- assets/README.md
- .PostScenarioDetail
- .CreateWorkflow
- KrknUserGroupSpec
- .ExecuteTerminal

## God Nodes (most connected - your core abstractions)
1. `Handler` - 193 edges
2. `NewTestHandler()` - 166 edges
3. `writeJSONError()` - 125 edges
4. `writeJSON()` - 96 edges
5. `createAdminContext()` - 91 edges
6. `GetClaimsFromContext()` - 61 edges
7. `newEsScheme()` - 57 edges
8. `KrknScenarioRun` - 56 edges
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
- `3. Create Provider Config Request` --references--> `KrknOperatorTargetProviderConfig`  [INFERRED]
  docs/api-provider-config-spec.md → api/v1alpha1/krknoperatortargetproviderconfig_types.go

## Import Cycles
- None detected.

## Communities (162 total, 28 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.09
Nodes (15): CategoryEntityAssociationResponse, CreateCategoryRequest, ResiliencyHistoryQueryRequest, scenarioFileBehavior, UpdateCategoryRequest, ContextKey, clusterTarget, jobOutcome (+7 more)

### Community 1 - "testing.T"
Cohesion: 0.02
Nodes (176): TestKrknUserCreation(), TestKrknUserList(), TestKrknUserRoles(), TestKrknUserStatus(), TestKrknUserValidation(), TestScenarioReferenceValidate(), bootstrapOLMResources(), NewConfigStoreInitializer() (+168 more)

### Community 2 - "net/http.Request"
Cohesion: 0.09
Nodes (17): writeCategoryRunNotFoundOrInternalError(), convertGraphClusterScores(), buildUserGroupResponse(), extractGroupNameAndUserIDFromPath(), extractGroupNameFromMembersPath(), validateClusterPermissions(), sanitizeResourceName(), sanitizeRunNameLabel() (+9 more)

### Community 3 - ".DeepCopyInto"
Cohesion: 0.05
Nodes (17): filterUsers(), GraphScenario, KrknCategoryList, KrknCategorySpec, KrknComponentList, KrknGraphRunList, KrknOperatorTargetList, KrknOperatorTargetProviderConfigList (+9 more)

### Community 4 - "files/types.go"
Cohesion: 0.08
Nodes (27): AvailableFilesResponse, ListFilesResponse, buildFileInfo(), buildFileResponse(), deriveLogicalName(), extractLogicalName(), validateCreateFileRequest(), validateUpdateFileRequest() (+19 more)

### Community 5 - "context.Context"
Cohesion: 0.09
Nodes (13): historyQueryCountingClient, mockUserStatusChecker, removeCategoryLabel(), migrationPending(), IsAdmin(), CleanupOldResources(), deleteResourceSafely(), extractItemsFromList() (+5 more)

### Community 6 - "NewTestHandler"
Cohesion: 0.05
Nodes (136): esDoerFunc, TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(), TestCloudCredentialContract_GraphRunStripsNodeEnv(), newAWSTestSecret(), newCloudCredScheme(), newGCPTestSecret(), newGroupedAWSSecret(), TestCloudCredentialsRouter_MethodNotAllowed() (+128 more)

### Community 7 - "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured.Unstructured"
Cohesion: 0.17
Nodes (16): migrationResult, resourceDefinition, checkAnnotationSize(), forEachResource(), loadGraphScenarioReferences(), patchMigrationResourceWithRetry(), refreshedMigrationPlan(), sanitizeGraphNodeID() (+8 more)

### Community 8 - "utils.go"
Cohesion: 0.14
Nodes (16): tokenRequest, TestE2E(), getMetricsOutput(), GetNonEmptyLines(), GetProjectDir(), InstallCertManager(), InstallPrometheusOperator(), IsCertManagerCRDsInstalled() (+8 more)

### Community 9 - "KrknScenarioRunReconciler"
Cohesion: 0.06
Nodes (29): imageResolver, imageSignatureVerifier, KrknScenarioRunReconciler, preparedJobResources, TestScenarioRunReconcileSkipsPendingMigration(), backfillLegacyMaxRetries(), countJobPhaseChanges(), extractDelimitedContent() (+21 more)

### Community 10 - "api/types.go"
Cohesion: 0.05
Nodes (44): ActiveRunsOverviewResponse, AddGroupMemberRequest, AddGroupMemberResponse, ChangePasswordRequest, ChangePasswordResponse, ClusterPermissionSet, ClusterResiliencyScoreResponse, CreateTargetResponse (+36 more)

### Community 11 - ".GetScenarioReplay"
Cohesion: 0.15
Nodes (11): convertClusterResiliencyScores(), averageResiliencyScore(), convertMetaTime(), BuildUnifiedJobList(), getDefaultPageSize(), runCategoryNames(), scenarioRunImage(), PaginateSlice() (+3 more)

### Community 12 - "migration.go"
Cohesion: 0.15
Nodes (34): clusterJobImage, clusterJobProvider, clusterJobRetries, graphScenarioReferences, migrationBlockedError, migrationPlan, migrationStats, resiliencyScoreStatus (+26 more)

### Community 13 - "addAdminContext"
Cohesion: 0.11
Nodes (43): addAdminContext(), addUserContext(), createTestAdminUser(), setupFilesTestHandler(), TestCanAccessFile(), TestCreateFile(), TestCreateFile_DefaultFilePurpose(), TestCreateFile_DuplicateName() (+35 more)

### Community 14 - "makeScenarioRun"
Cohesion: 0.27
Nodes (12): makeGraphRun(), makeScenarioRun(), TestListJobs_IncludesStats(), TestListJobs_LimitOnlyReturnsAll(), TestListJobs_NoPagination(), TestListJobs_Paginated(), TestListJobs_SkipsGraphRunChildren(), TestListJobs_TypeDiscriminator() (+4 more)

### Community 15 - "KrknCategory"
Cohesion: 0.12
Nodes (16): CategoryListResponse, CategoryResponse, buildCategoryResponse(), buildCategoryVisibilityLabels(), categoryErrorCode(), categoryGroups(), categoryIsPublic(), categoryVisibleToUser() (+8 more)

### Community 16 - "render-bundle.sh"
Cohesion: 0.07
Nodes (32): channel_has_entries(), ensure_catalog_channel(), catalog-channel.sh script, update_default_channel(), write_release_config(), catalog_submit_checkout(), catalog_submit_open_pr(), catalog-submit.sh script (+24 more)

### Community 17 - "server.py"
Cohesion: 0.06
Nodes (6): add_DataProviderServiceServicer_to_server(), DataProviderService, DataProviderServiceServicer, DataProviderServiceStub, DataProviderServicer, serve()

### Community 18 - "ConfigStoreInitializer"
Cohesion: 0.29
Nodes (4): ConfigStoreInitializer, BackfillLegacyProviderConfigLabel(), TestBackfillLegacyProviderConfigLabel(), TestBackfillLegacyProviderConfigLabelMissingConfigMap()

### Community 19 - "NewHub"
Cohesion: 0.10
Nodes (36): NewHandler(), NewBroadcaster(), TestBroadcastDashboardUpdate(), TestBroadcastGraphRunDeleted(), TestBroadcastGraphRunUpdate(), TestBroadcastOnlyToSubscribedClients(), TestBroadcastScenarioRunDeleted(), TestBroadcastScenarioRunUpdate() (+28 more)

### Community 20 - "WSUnifiedJobItem"
Cohesion: 0.14
Nodes (9): TestFilterJobsByCategoriesMatchesAnySelectedCategory(), TestPaginateJobItems(), computeWSJobStats(), filterJobCategoriesByVisibility(), filterJobsByCategories(), paginateJobItems(), WSJobStatsSummary, WSUnifiedJobItem (+1 more)

### Community 21 - "user_handlers_test.go"
Cohesion: 0.12
Nodes (36): TestListComponentsRouteAndAuth(), sanitizeUsername(), createTestUser(), kvValue(), setupUserTestHandler(), TestChangePassword_AdminChange_NoCurrentPassword(), TestChangePassword_AdminChangesOtherUser_EmitsAuditLog(), TestChangePassword_SamePassword_Rejected() (+28 more)

### Community 22 - "client_test.go"
Cohesion: 0.10
Nodes (22): doerFunc, roundTripperFunc, NewClient(), scenarioParametersSchema(), TestBaseURL(), TestClientTransportDefaultNotHTTPTransport(), TestClientTransportPooling(), TestConnectionParamsTLSConfig() (+14 more)

### Community 23 - "KrknGraphRun"
Cohesion: 0.11
Nodes (16): KrknGraphRunReconciler, graphHistoryRun(), TestBuildUnifiedJobList_SkipsGraphRunChildren(), buildGraphRunResponse(), buildUnifiedJobList(), convertGraphClusterScoresForSnapshot(), runCategoryNames(), TestGraphRunReconcileSkipsPendingMigration() (+8 more)

### Community 24 - "elasticsearch/types.go"
Cohesion: 0.12
Nodes (24): AffectedPods, AlertDocument, ClusterMetadata, CreateElasticsearchConfigResponse, DeleteElasticsearchConfigResponse, ListElasticsearchConfigsResponse, NodeSummaryInfo, QueryAlertsRequest (+16 more)

### Community 25 - "InjectCredentials"
Cohesion: 0.18
Nodes (21): TestAppendCloudCredentialInjection_EmptyRefNoop(), TestAppendCloudCredentialInjection_UsesSecretKeyRef(), injectAWS(), injectAzure(), injectBaremetal(), InjectCredentials(), injectGCP(), injectIBMCloud() (+13 more)

### Community 26 - "installer.go"
Cohesion: 0.09
Nodes (23): main(), resolveMigrationNamespace(), runCRDMigrationMode(), TestResolveMigrationNamespacePrefersHookNamespace(), TestRunCRDMigrationMode(), configMapClient, crdClient, crdManifest (+15 more)

### Community 27 - "migration_test.go"
Cohesion: 0.20
Nodes (25): Complete(), Stage(), containsSubstring(), countPatchActions(), jsonMarshal(), migrationTestLogger(), newMigrationClient(), scenarioRunFixture() (+17 more)

### Community 28 - "KrknTargetRequest"
Cohesion: 0.10
Nodes (9): ClustersResponse, KrknTargetRequestReconciler, ClusterHealthStatus, ClusterTarget, KrknOperatorTarget, KrknOperatorTargetSpec, KrknTargetRequest, KrknTargetRequestSpec (+1 more)

### Community 29 - "client.go"
Cohesion: 0.10
Nodes (24): categoryEnvironmentParameter, esAlertSearchResponse, facetField, filterAggregation, Option, StatusError, termsAggregation, buildAlertsSearchBody() (+16 more)

### Community 30 - "backup_restore_handlers_test.go"
Cohesion: 0.15
Nodes (22): NewJobTracker(), createMultipartBody(), createTestHandler(), TestGetRestoreStatusAdminOnly(), TestGetRestoreStatusTrackedJob(), TestGetRestoreStatusUnknownJob(), TestHandlerShutdownCancelsContext(), TestJobTrackerEviction() (+14 more)

### Community 31 - "setupTestHandler"
Cohesion: 0.08
Nodes (27): setupTestHandler(), TestCreateTarget_DuplicateClusterName(), TestCreateTarget_MissingRequiredFields(), TestCreateTarget_WithCredentials(), TestCreateTarget_WithKubeconfig(), TestCreateTarget_WithToken(), TestDeleteQuietly_MissingObjectIsNoOp(), TestDeleteQuietly_RunsWithCanceledContext() (+19 more)

### Community 32 - "Broadcaster"
Cohesion: 0.12
Nodes (10): convertNodeStatusesWithScores(), hashBytes(), visibleCategoryNames(), SetupWatchers(), AuthorizationChecker, AuthorizationCheckFunc, Broadcaster, categoryVisibilityChecker (+2 more)

### Community 33 - "ConnectionParams"
Cohesion: 0.21
Nodes (8): Doer, TelemetryDocument, executeSearch(), Client, ConnectionParams, newSearchRequest(), TestTransportKey(), transportKey()

### Community 34 - "KrknOperatorTargetProviderConfig"
Cohesion: 0.09
Nodes (19): KrknOperatorTargetProviderConfigReconciler, Architecture Decision, Components Requiring Leader Election (Leader-Only), Components Running on All Replicas (Multi-Active), isConflictError(), checkProviderActive(), countActiveProviders(), CreateProviderConfigRequest() (+11 more)

### Community 35 - "wsorigin.go"
Cohesion: 0.21
Nodes (13): checkWebSocketOriginV2(), TestCheckWebSocketOriginV2(), checkWebSocketOrigin(), TestCheckWebSocketOrigin(), canonicalOrigin(), defaultPort(), IsAllowedOrigin(), portsMatch() (+5 more)

### Community 37 - "Handler"
Cohesion: 0.09
Nodes (16): cloudCredentialAccessContext, ErrorResponse, Handler, buildCloudCredentialResponse(), updateCloudCredentialSecretData(), buildConnectionParams(), buildConnectionParamsForIndex(), buildElasticsearchConfigResponse() (+8 more)

### Community 38 - "installer_test.go"
Cohesion: 0.23
Nodes (10): applyCRDs(), TestApplyCRDsCreatesMissingAndUpdatesChangedSpecs(), TestApplyCRDsDoesNotPatchUnchangedSpecs(), TestApplyCRDsLabelsExistingCRDsWithoutChangingOtherLabels(), TestApplyCRDsRejectsNonCRDManifest(), testCRD(), TestCreateMissingCRDsBeforeRolloutLeavesExistingSpecsUntouched(), TestWaitForEstablished() (+2 more)

### Community 39 - ".Get"
Cohesion: 0.13
Nodes (18): countingGetClient, TestConcurrentAccess(), TestConcurrentSnapshot(), TestDelete(), TestDelete_NonExistent(), TestEmptyValue(), TestExists(), TestGet_Singleton() (+10 more)

### Community 40 - "ComponentManifest"
Cohesion: 0.12
Nodes (8): ListComponentsResponse, Load(), TestLoad(), TestLoadRejectsInvalidManifest(), ComponentDescription, ComponentManifest, KrknComponent, KrknComponentStatus

### Community 41 - "bootstrap.go"
Cohesion: 0.17
Nodes (13): RemoveElasticsearchGrafanaURLAnnotations(), TestRemoveElasticsearchGrafanaURLAnnotations(), ensureJWTSecret(), EnsureResources(), ensureSCCBinding(), ensureScenarioRunnerRBAC(), ensureServiceAccount(), ensureServices() (+5 more)

### Community 42 - "UnifiedJobItem"
Cohesion: 0.12
Nodes (7): PaginationMeta, ScenarioRunListItem, ScenarioRunListResponse, UnifiedJobItem, UnifiedJobsResponse, ComputeJobStats(), TestComputeJobStats()

### Community 43 - "time.Duration"
Cohesion: 0.08
Nodes (22): k8sUserStatusChecker, Server, removeGroupFromAllMembers(), NewHandler(), NewServer(), NewServerWithComponentNamespace(), newK8sUserStatusChecker(), Handler (+14 more)

### Community 44 - "Handler"
Cohesion: 0.28
Nodes (3): roleFromAdmin(), Get(), Handler

### Community 45 - "k8s.io/apimachinery/pkg/runtime.Scheme"
Cohesion: 0.33
Nodes (8): addKnownTypes(), newTargetRequest(), newTestScheme(), TestGetClusterAPIURL_ClusterNotFound(), TestGetClusterAPIURL_Found(), TestGetClusterAPIURL_ProviderNotFound(), TestGetClusterAPIURL_TargetRequestNotFound(), TestReconcile_FailedJobStatus_IncludesClusterAPIURL()

### Community 46 - "notify-operator-docs.test.js"
Cohesion: 0.10
Nodes (11): assert, chartWorkflow, { execFileSync, spawnSync }, fs, notifier, notifyWorkflow, os, path (+3 more)

### Community 47 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.18
Nodes (10): GraphRunDetailResponse, GraphRunListItem, GraphRunListResponse, GraphRunSpecResponse, GraphRunStatusResponse, 1.1 Resiliency weight per graph node, 1. Request - Create GraphRun (POST /api/v1/graphruns), 2. Response - List GraphRuns (GET /api/v1/graphruns) (+2 more)

### Community 48 - "watcher_test.go"
Cohesion: 0.19
Nodes (18): hasGraphRunStatusChanged(), hasScenarioRunStatusChanged(), TestHasGraphRunStatusChanged_NewNodeAdded(), TestHasGraphRunStatusChanged_NoChange(), TestHasGraphRunStatusChanged_NodeOrderChanged(), TestHasGraphRunStatusChanged_NodeStatusChange(), TestHasGraphRunStatusChanged_PhaseChange(), TestHasGraphRunStatusChanged_ResiliencyScoresChange() (+10 more)

### Community 49 - "time.Time"
Cohesion: 0.14
Nodes (12): ClusterJobStatusResponse, NodeStatusResponse, RestoreResponse, ScenarioRunStatusResponse, CachedUserStatusChecker, userStatusCacheEntry, convertClusterJobs(), convertMetaTime() (+4 more)

### Community 50 - "Hub"
Cohesion: 0.18
Nodes (5): Store, BroadcastMessage, Client, Hub, PaginationClientState

### Community 51 - "GraphScenarioNode"
Cohesion: 0.13
Nodes (18): GraphRunCreateRequest, GraphScenarioNode, writeCategoryProfileField(), appendGraphSignaturePart(), compareGraphConfigurations(), compareGraphConfigurationsWithStableNodeIDs(), compareGraphRunConfigurations(), graphConfigurationSignature() (+10 more)

### Community 52 - "time.Duration"
Cohesion: 0.27
Nodes (6): getTokenDuration(), TestGetTokenDuration(), newResiliencyTestScheme(), newTestClientsetWithLogs(), TestCalculateResiliencyScores(), TestReconcile_ResiliencyScoreFailure_DoesNotBlockTerminalPhase()

### Community 53 - "ScenarioReference"
Cohesion: 0.20
Nodes (15): publicScenarioReference(), TestCreateScenarioRun_ResiliencyScore(), buildContainerImage(), buildContainerImageFromReference(), getTestConfig(), privateReference(), publicReference(), TestBuildContainerImage_PublicRegistryUsesKrknctl() (+7 more)

### Community 54 - "RestoreBackup"
Cohesion: 0.13
Nodes (14): resiliencyHistoryQuerySelections, RestoreConfig, normalizeResiliencyHistoryProviders(), normalizeResiliencyHistorySelections(), parseResiliencyHistoryQuery(), createTarGz(), TestCreateAndExtractTarGz(), TestCreateTarGzFlushErrors() (+6 more)

### Community 55 - "github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2"
Cohesion: 0.13
Nodes (12): POST methods, refactoring, /scenarios ✅ COMPLETED, /scenarios/detail/{scenario_name} ✅ COMPLETED, /scenarios/globals/{scenario_name} ✅ COMPLETED, /scenarios/run/{UUID}, /targets ✅ COMPLETED, mockScenarioProvider (+4 more)

### Community 56 - "cloudcreds/types.go"
Cohesion: 0.14
Nodes (15): CreateCloudCredentialResponse, DeleteCloudCredentialResponse, ListCloudCredentialsResponse, UpdateCloudCredentialResponse, buildCloudCredentialSecretData(), CloudCredentialResponse, CreateCloudCredentialRequest, UpdateCloudCredentialRequest (+7 more)

### Community 57 - "GetNodesRequest"
Cohesion: 0.15
Nodes (3): DataProviderServiceClient, GetNodesRequest, GetNodesResponse

### Community 58 - "dataprovider_grpc.pb.go"
Cohesion: 0.16
Nodes (5): UnimplementedDataProviderServiceServer, UnsafeDataProviderServiceServer, _DataProviderService_ExecuteKubectl_Handler(), _DataProviderService_GetNodes_Handler(), RegisterDataProviderServiceServer()

### Community 59 - "GroupLabelKey"
Cohesion: 0.09
Nodes (49): newCategoryRequest(), newCategoryRequestAs(), newCategoryTestGroup(), newCategoryTestHandler(), newCategoryTestUser(), TestCategoriesRouterCRUD(), TestCategoriesRouterGroupCreationRequiresMembership(), TestCategoriesRouterGroupVisibility() (+41 more)

### Community 60 - "converter_test.go"
Cohesion: 0.20
Nodes (15): BuildDockerConfigJSON(), ExtractRegistryV2FromSecret(), TestBuildDockerConfigJSON_MissingCredentials(), TestBuildDockerConfigJSON_OAuth(), TestBuildDockerConfigJSON_Password(), TestExtractRegistryV2FromSecret_InvalidType(), TestExtractRegistryV2FromSecret_MissingAnnotations(), TestExtractRegistryV2FromSecret_MissingComponent() (+7 more)

### Community 61 - "GetClaimsFromContext"
Cohesion: 0.14
Nodes (7): Claims, extractPathSuffix(), buildRegistryResponse(), buildUserResponse(), GetClaimsFromContext(), BuildRegistryLabels(), ExtractGroupsFromLabels()

### Community 62 - "ScenarioRunRequest"
Cohesion: 0.24
Nodes (10): Phase 12: Scenario Job Management - /scenarios/run Endpoints ✅, Phase 9: POST /scenarios Endpoint ✅, GlobalsRequest, JobsListResponse, JobStatusResponse, ScenarioRunResponse, ScenariosRequest, ScenariosResponse (+2 more)

### Community 63 - "KrknScenarioRunSpec"
Cohesion: 0.20
Nodes (10): KrknScenarioRunSpec, compareGraphNodeConfigurations(), compareScenarioRunConfigurations(), compareStringMaps(), optionalBoolValue(), sameGraphVolumeMounts(), sameScenarioFiles(), sameScenarioReference() (+2 more)

### Community 64 - "websocket/types.go"
Cohesion: 0.15
Nodes (10): ClientMessage, ClusterResiliencyScoreResponse, ErrorMessage, GraphClusterScoreResponse, GraphRunResponse, GraphRunSummaryResponse, NodeStatusResponse, ScenarioRunStatusResponse (+2 more)

### Community 65 - "ResolveGraph"
Cohesion: 0.15
Nodes (13): FromKrknctlScenarioSet(), publicScenarioReference(), publicScenario(), TestFromKrknctlScenarioSetCreatesPublicReferences(), TestToKrknctlScenarioSetUsesScenarioReference(), ToKrknctlScenarioSet(), MapScenarioNodeToScenarioRunSpec(), ResolveGraph() (+5 more)

### Community 66 - "CreateBackup"
Cohesion: 0.17
Nodes (10): BackupConfig, resourceType, backupResourceType(), CreateBackup(), stripMetadata(), TestCreateBackupExcludesProviderResources(), TestStripMetadata(), TestStripMetadataRemovesEmptyAnnotations() (+2 more)

### Community 67 - ".flatten"
Cohesion: 0.22
Nodes (10): rawTelemetrySource, intPtr(), TestFlattenRepresentativeScenario(), TestRawTelemetrySourceFlatten(), contains(), containsAt(), TestMetadataJSONMarshaling(), TestMetadataPresenceTracking() (+2 more)

### Community 68 - "rate_limiter_test.go"
Cohesion: 0.06
Nodes (37): ipRateLimiter, rateLimiterEntry, countingUserStatusChecker, Role, UserStatusChecker, clientIPFromHeaders(), extractClientIP(), hostFromRemoteAddr() (+29 more)

### Community 69 - ".GetScenarioRunLogs"
Cohesion: 0.14
Nodes (14): filterReportLines(), isReportMarkerEnd(), isReportMarkerStart(), isWebSocketDisconnectError(), maskToken(), preStartFailureReason(), sanitizeHeaders(), TestMaskToken() (+6 more)

### Community 70 - "Publisher"
Cohesion: 0.11
Nodes (7): Publisher, NewNamespaceFilter(), TestPublisherReconcile(), TestPublisherReconcileReturnsTransientErrors(), TestPublisherRecoversFromConcurrentCreateAndDelete(), TestPublisherStartRetriesAndReturnsCancellation(), ProviderConfigContributorReconciler

### Community 71 - "start_operator.sh"
Cohesion: 0.41
Nodes (12): build_operator(), check_prerequisites(), cleanup(), create_service_account(), install_crds(), main(), print_error(), print_info() (+4 more)

### Community 72 - "KrknScenarioRun"
Cohesion: 0.07
Nodes (39): CategoryConfigurationGroup, categoryHistoryRun, categoryHistoryScore, CategoryResiliencyDataPoint, CategoryResiliencyHistoryResponse, ResiliencyHistoryQueryResponse, buildCategoryResiliencyHistory(), categoryBehaviorConfigurationSignature() (+31 more)

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

### Community 77 - "registry/types.go"
Cohesion: 0.18
Nodes (13): validateCreateRegistryRequest(), validateUpdateRegistryRequest(), CreateRegistryRequest, RegistryInfo, RegistryResponse, UpdateRegistryRequest, AuthEntry, AvailableRegistriesResponse (+5 more)

### Community 78 - "test-api.sh"
Cohesion: 0.39
Nodes (11): api_call(), cleanup(), extract_body(), extract_http_code(), log_error(), log_info(), log_step(), log_success() (+3 more)

### Community 79 - "Frontend Authentication Implementation Tasks"
Cohesion: 0.06
Nodes (30): 1. Read the API Documentation, 1. Start the Operator, 2. Check Available Work, 2. Test Registration (First Admin), 3. Start with Foundation Task, 3. Test Login, 401 on Protected Routes, 4. Claim a Task (+22 more)

### Community 82 - "ValidateCreateRequest"
Cohesion: 0.22
Nodes (10): CreateElasticsearchConfigRequest, UpdateElasticsearchConfigRequest, strPtr(), TestValidateCreateRequest(), TestValidateUpdateRequest(), validateAccess(), ValidateCreateRequest(), ValidateMergedConnection() (+2 more)

### Community 84 - "KrknOperatorTargetProviderConfigStatus"
Cohesion: 0.29
Nodes (3): getProviderNames(), KrknOperatorTargetProviderConfigStatus, ProviderConfigData

### Community 85 - "registry/labels_test.go"
Cohesion: 0.29
Nodes (6): BuildRegistryAnnotations(), TestBuildRegistryAnnotations(), TestBuildRegistryLabels(), TestExtractGroupsFromLabels(), TestUpdateRegistryAnnotations(), UpdateRegistryAnnotations()

### Community 86 - "TestJWTSecret"
Cohesion: 0.06
Nodes (41): TestLogin_InactiveUser(), TestLogin_InvalidCredentials(), TestLogin_Success(), convertInputFields(), setupScenarioRunTestHandler(), TestConvertInputFields(), TestDeleteTargetByUUID_AdminCanDeleteAnyResource(), TestDeleteTargetByUUID_InvalidUUID() (+33 more)

### Community 87 - "GraphClusterScore"
Cohesion: 0.17
Nodes (4): convertNodeStatusesWithScores(), GraphRunSummary, KrknGraphRunStatus, NodeStatus

### Community 88 - "ssrf.go"
Cohesion: 0.16
Nodes (10): responseWriter, guardedCheckRedirect(), guardedDialContext(), TestGuardedCheckRedirect(), TestQueryTelemetryAllowsUnrestrictedLoopback(), TestValidateDestinationIP(), TestValidateDestinationURL(), validateDestinationIP() (+2 more)

### Community 89 - "GET methods"
Cohesion: 0.07
Nodes (28): API Requirements, /clusters, DELETE methods, GET methods, GET methods, get nodes, get pods, Grpc python service requirement (+20 more)

### Community 90 - ".QueryTelemetryPage"
Cohesion: 0.48
Nodes (7): esSearchResponse, FacetOption, QueryTelemetryResponse, TelemetryStats, decodeTelemetry(), facetsFromBuckets(), statsFromBuckets()

### Community 91 - "files/validation_test.go"
Cohesion: 0.27
Nodes (10): isValidConfigMapKey(), validateFileFields(), contains(), findSubstring(), TestValidateFileContent(), TestValidateFileGroups(), TestValidateUserFilePermissions(), ValidateFileContent() (+2 more)

### Community 92 - "Terminal API - Frontend Integration Guide"
Cohesion: 0.07
Nodes (28): Allowed Subcommands (Read-Only), Audit Logging, Authentication, Blocked Flags (No Streaming in v1), Command Parsing, Decoding Output, Endpoint, Example 1: Get Pods (+20 more)

### Community 93 - "KrknScenarioRunSpec"
Cohesion: 0.33
Nodes (6): Analisi CRD Reale dal Cluster, ✅ BUONE NOTIZIE: Volume Mounts Sono Persistiti!, Due Possibili Scenari, Scenario A: Files è Completo, Scenario B: Files è Incompleto, Verifica Necessaria

### Community 95 - "parseResiliencyHistoryQuery"
Cohesion: 0.36
Nodes (6): SyncConfigMapToStore(), TestRoundTrip(), TestSyncConfigMapToStore(), TestSyncConfigMapToStore_NilStore(), TestWriteConfigMapData(), WriteConfigMapData()

### Community 96 - "Provider Configuration API Specification"
Cohesion: 0.06
Nodes (30): 1. List Available Providers (Optional), 1. List Providers, 2. Create Config Request, 2. Get Provider Config Status, 2. Update Provider Status, 3. Create Provider Config Request, 3. Poll for Completion, 3. Update Provider Configuration (+22 more)

### Community 97 - "Task Breakdown"
Cohesion: 0.12
Nodes (17): FileMount, ScenarioRunRequest, normalizeLegacyScenarioRunRequest(), TestNormalizeLegacyScenarioRunRequestIgnoresImage(), FileReference, TASK 10: Documentazione, TASK 1: Investigazione Volume Mounts (PRIORITÀ ALTA), TASK 2a: Estensione CRD (SE NECESSARIO) (+9 more)

### Community 98 - "dataprovider.pb.go"
Cohesion: 0.20
Nodes (3): file_proto_dataprovider_proto_init(), file_proto_dataprovider_proto_rawDescGZIP(), init()

### Community 99 - "TestKrknGraphRun_ValidateGraph"
Cohesion: 0.33
Nodes (5): SanitizeNodeIDForKubernetes(), stringPtr(), TestCollisionDetection(), TestKrknGraphRun_ValidateGraph(), TestSanitizeNodeIDForKubernetes()

### Community 100 - "newProvidersHandler"
Cohesion: 0.40
Nodes (5): newProvider(), newProvidersHandler(), TestListProviders_Empty(), TestListProviders_ScopedToNamespace(), TestUpdateProviderStatus()

### Community 101 - "Authentication API Specification"
Cohesion: 0.09
Nodes (22): 1. Check Admin Registration, 1. Initial Load, 2. First Admin Registration, 2. Register User, 3. Login, 3. Login, 4. Authenticated Requests, 5. Role-Based UI (+14 more)

### Community 102 - "NewTokenGenerator"
Cohesion: 0.07
Nodes (36): TestGroupNameValidation(), TestGroupsRouterEdgeCases(), TestCheckScenarioRunAccess(), TestFilterScenarioRunsByGroupPermission(), TestPostScenarioRun_ClusterNameCollision_Returns500(), TestPostScenarioRun_SameClusterSameURL_NoCollision(), TestPostScenarioRunSetsOwner(), TestClusterSegregationLegacyRunsWithoutJobs() (+28 more)

### Community 103 - "API Reference"
Cohesion: 0.25
Nodes (8): API Reference, CreateProviderConfigRequest, Interfaces, NewProviderRegistration, NewProviderRegistrationWithConfig, Provider Configuration Functions, Provider Registration Functions, UpdateProviderConfig

### Community 104 - "setup-ocm-and-operator.sh"
Cohesion: 0.67
Nodes (5): log_error(), log_info(), log_warn(), setup-ocm-and-operator.sh script, wait_for_condition()

### Community 105 - "File Management API Specification"
Cohesion: 0.09
Nodes (22): Access Control, API Endpoints, ConfigMap Structure, Content Validation, Create a Public Configuration File, Create File, Delete File, Error Responses (+14 more)

### Community 107 - "generateKubeconfigFromRequest"
Cohesion: 0.53
Nodes (6): CreateTargetRequest, UpdateTargetRequest, generateKubeconfigFromCredentialsType(), generateKubeconfigFromKubeconfigType(), generateKubeconfigFromRequest(), generateKubeconfigFromTokenType()

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
Nodes (21): 1. Create Virtual Environment (Python 3.11), 1. Login and Get JWT Token, 2. Create KrknTargetRequest (or use existing UUID), 2. Install Dependencies, 3. Test Terminal Command, 3. Verify Generated Stubs, 4. Start gRPC Server, Cleanup (+13 more)

### Community 112 - "k8s.io/apimachinery/pkg/runtime.Scheme"
Cohesion: 0.83
Nodes (3): isAlphanumeric(), isValidLabelChar(), TestSanitizeNodeID()

### Community 117 - "CLAUDE.md - Project Instructions"
Cohesion: 0.10
Nodes (19): Architecture Principles, Before Committing, Before Writing Code, CLAUDE.md - Project Instructions, Code Organization & Reusability, Code Quality Standards, Communication, Critical Rules (+11 more)

### Community 128 - "Piano Implementazione: GET /scenarios/run/replay/{jobId}"
Cohesion: 0.12
Nodes (16): 1. Endpoint Specificato, 2. Flusso Logico, Acceptance Criteria, Analisi Requisiti, Dependency Graph, Next Steps, Obiettivo, Piano Implementazione: GET /scenarios/run/replay/{jobId} (+8 more)

### Community 129 - "Implementation Details"
Cohesion: 0.14
Nodes (16): Code Refactoring: getKubeconfigFromTarget() Helper ✅, Implementation Details, Phase 10: POST /scenarios/detail/{scenario_name} Endpoint ✅, Phase 11: POST /scenarios/globals/{scenario_name} Endpoint ✅, Phase 1: REST API Setup ✅, Phase 2: GET /clusters Endpoint ✅, Phase 3: Integration ✅, Phase 4: Testing ✅ (+8 more)

### Community 130 - "Leader Election and API Server Separation"
Cohesion: 0.11
Nodes (18): After (Multi-Active API), API Server Concurrent Operations, API Server Startup Wait, Before (Single Leader), Concurrency Safety, Controller Safety with Leader Election, Deployment Considerations, Expected Startup Logs (+10 more)

### Community 131 - "krkn-operator"
Cohesion: 0.11
Nodes (14): Alerts Query API, Backup and Restore, Documentation, Ecosystem, krkn-operator, License, OLM / OperatorHub bundles, Operator release documentation snapshots (+6 more)

### Community 132 - "Deployment Guide"
Cohesion: 0.12
Nodes (17): Architecture, Build Installer Bundle, Complete Workflow Examples, Custom Registry, Deployment Guide, Development (Local), Environment Variables, Git Tag Support (+9 more)

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

### Community 138 - "Moving from direct scenario creation to CRD approach"
Cohesion: 0.17
Nodes (12): API and operator refactoring to support multiple clusters, API Endpoints Structure (Nested Approach), BREAKING CHANGES, Implementation Details, ✅ Implemented, KrknTargetRequest controller in operator, Moving from direct scenario creation to CRD approach, Overview (+4 more)

### Community 139 - "API Workflow Test"
Cohesion: 0.18
Nodes (10): API Workflow Test, Configuration, Error Handling, krkn-operator Test Scripts, Output, Overview, Prerequisites, Testing Against Production (+2 more)

### Community 140 - "scenario_filter_test.go"
Cohesion: 0.47
Nodes (4): filterScenariosByIsAScenario(), TestFilterScenariosByIsAScenario(), TestFilterScenariosByIsAScenario_CancelledContext(), TestFilterScenariosByIsAScenario_PreservesTagMetadata()

### Community 142 - "Krkn-Operator Roadmap"
Cohesion: 0.33
Nodes (5): Krkn-Operator Roadmap, Ongoing (not release-blocked), Release 1 — 2026-08-18, Release 2 — 2026-10-18, Release 4 — 2027-02-23

### Community 144 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.25
Nodes (8): Configuration, Current Behavior, Option A: Cancellation Field in Status (Recommended), Option B: Finalizers for Cancellation Tracking, Option C: Separate CancellationRequest CR, Proposed Solution Options, Requirements, 🔧 TODO: Pod Recreation and Retry Logic

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

### Community 152 - "Troubleshooting"
Cohesion: 0.40
Nodes (5): gRPC Connection Issues, Image Pull Errors, Namespace Issues, Pod Not Starting, Troubleshooting

### Community 153 - "Troubleshooting"
Cohesion: 0.40
Nodes (5): Error: "Command not found: kubectl", Error: "failed to connect to gRPC server", Error: "Failed to get kubeconfig", Error: "no module named 'krkn_lib'", Troubleshooting

### Community 154 - "Pushing Images"
Cohesion: 0.50
Nodes (4): Custom Push, Push Both Images, Push Individual Images, Pushing Images

### Community 155 - "API compatibility notes"
Cohesion: 0.50
Nodes (4): API compatibility notes, Graph node resiliency weights, Log streaming, Retry limits

### Community 156 - "GetUserGroups"
Cohesion: 0.13
Nodes (11): writeScenarioRunAccessError(), parseBoolHeader(), TestParseBoolHeader(), resolveMaxRetries(), IsCloudEnvVar(), StripCloudEnvVars(), TestIsCloudEnvVar(), TestStripCloudEnvVars() (+3 more)

### Community 159 - ".CreateWorkflow"
Cohesion: 0.12
Nodes (20): convertConfigMapToWorkflowInfo(), convertFileResponseToWorkflow(), extractIDFromPath(), filterVisibleWorkflowCategories(), parseWorkflowCategories(), validateWorkflowCategories(), workflowInfosHaveCategories(), workflowResponsesHaveCategories() (+12 more)

### Community 164 - ".ExecuteTerminal"
Cohesion: 0.22
Nodes (4): TerminalRequest, TerminalResponse, HasClusterPermission(), NewDataProviderServiceClient()

## Knowledge Gaps
- **17 isolated node(s):** `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME`, `ICON_BASE64`, `ICON_MEDIATYPE` (+12 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 766 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **28 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `krkn-operator` connect `krkn-operator` to `Running Locally`, `API compatibility notes`?**
  _High betweenness centrality (0.046) - this node is a cross-community bridge._
- **Are the 159 inferred relationships involving `NewTestHandler()` (e.g. with `setupAuthTestHandler()` and `TestLogin_InactiveUser()`) actually correct?**
  _`NewTestHandler()` has 159 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME` to the rest of the system?**
  _17 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.09248518471819443 - nodes in this community are weakly interconnected._
- **Why does `Handler` connect `Handler` to `main.go`, `ConnectionParams`, `net/http.Request`, `.ExecuteTerminal`, `files/types.go`, `context.Context`, `.GetScenarioRunLogs`, `KrknScenarioRun`, `bootstrap.go`, `.GetScenarioReplay`, `time.Duration`, `JobTracker`, `KrknCategory`, `GetUserGroups`, `GetClaimsFromContext`, `.PostScenarioDetail`, `.CreateWorkflow`?**
  _High betweenness centrality (0.041) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.01712100054641491 - nodes in this community are weakly interconnected._
- **Why does `KrknScenarioRun` connect `KrknScenarioRun` to `main.go`, `Piano Implementazione: GET /scenarios/run/replay/{jobId}`, `net/http.Request`, `.DeepCopyInto`, `context.Context`, `KrknScenarioRunReconciler`, `Moving from direct scenario creation to CRD approach`, `.GetScenarioReplay`, `makeScenarioRun`, `KrknGraphRun`, `Broadcaster`, `.DeepCopy`, `Handler`, `ComponentManifest`, `k8s.io/apimachinery/pkg/runtime.Scheme`, `watcher_test.go`, `GroupLabelKey`, `KrknScenarioRunSpec`, `buildScenarioRunResponse`, `KrknScenarioRunSpec`, `Task Breakdown`?**
  _High betweenness centrality (0.033) - this node is a cross-community bridge._