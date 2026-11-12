# Graph Report - krkn-operator  (2026-11-12)

## Corpus Check
- 343 files · ~438,736 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 20 file(s) not represented in the graph (top: (none) 14, .jsonl 2, .toml 1)

## Summary
- 3727 nodes · 15163 edges · 171 communities (140 shown, 31 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 1409 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `4b365b57`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- main.go
- testing.T
- Handler
- InjectCredentials
- GetClaimsFromContext
- context.Context
- NewTestHandler
- cloudcreds_handlers_test.go
- utils.go
- rollback_handlers_test.go
- api/types.go
- Broadcaster
- migration.go
- addAdminContext
- k8s.io/api/core/v1.Secret
- GetClaimsFromContext
- render-bundle.sh
- server.py
- category_run_configuration.go
- NewHub
- Handler
- user_handlers_test.go
- client_test.go
- KrknGraphRun
- elasticsearch/types.go
- installer.go
- Middleware
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- GraphClusterScore
- client.go
- backup_restore_handlers_test.go
- handlers_test.go
- Server
- .QueryTelemetryPage
- KrknOperatorTargetProviderConfig
- UnifiedJobItem
- k8s.io/apimachinery/pkg/apis/meta/v1.TypeMeta
- TokenGenerator
- watcher_test.go
- PROGRESS-MVP.md
- bootstrap.go
- github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2
- wsorigin.go
- WSUnifiedJobItem
- migration_test.go
- notify-operator-docs.test.js
- converter_test.go
- ssrf.go
- KrknScenarioRunSpec
- Hub
- KrknWorkspace
- POST methods
- captureLogSink
- providerConfigTestScheme
- newReportTestHandler
- cloudcreds/types.go
- GetNodesRequest
- dataprovider_grpc.pb.go
- .UpdateRegistry
- parseResiliencyHistoryQuery
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- ScenarioRunRequest
- scenario_catalog_cache_test.go
- GET methods
- installer_test.go
- buildClusterAPIURLMap
- buildContainerImageFromReference
- elasticsearch/labels_test.go
- Handler
- Publisher
- start_operator.sh
- files/validation_test.go
- github.com/krkn-chaos/krknctl/pkg/verify.SignatureStatus
- Compute
- scenarioRunConfigurationFixture
- KrknOperatorTargetProvider
- .PostScenarioRun
- test-api.sh
- Frontend Authentication Implementation Tasks
- ExecuteKubectlRequest
- mockJob
- KrknScenarioRunReconciler
- .extractAndStoreReports
- category_parameter_profile.go
- Krkn-Operator Roadmap
- KrknUserGroup
- .flatten
- ScenariosResponse
- metadata_presence_test.go
- ResolveGraph
- KrknCategory
- WorkspaceReference
- makeScenarioRun
- ExecuteKubectlResponse
- ProviderRegistration
- Provider Configuration API Specification
- .PostScenarioDetail
- dataprovider.pb.go
- TestKrknGraphRun_ValidateGraph
- CategoryResiliencyHistoryResponse
- Authentication API Specification
- .ListAvailableCloudCredentials
- Publisher
- setup-ocm-and-operator.sh
- File Management API Specification
- JobTracker
- commands.go
- Cluster Segregation Troubleshooting
- channel-head.sh
- Running gRPC Data Provider Locally for Development
- api_workflow_test.sh
- scenarioCatalogError
- select-compatible-release.sh
- test-open-kubernetes-operators-pr.sh
- CLAUDE.md - Project Instructions
- post-install.sh
- generate_proto_go.sh
- ResolvePrivateTagDigest
- test-channel-head.sh
- test-chart-exposure.sh
- test-select-compatible-release.sh
- generate_proto.sh
- debug-cluster-permissions.sh
- notify-operator-docs.sh
- github.com/krkn-chaos/krkn-operator
- Setup
- Implementation Details
- ProviderRegistration
- krkn-operator
- Deployment Guide
- Quick Start - Terminal API Local Testing
- Resiliency Score Feature - Frontend Implementation Guide
- AGENTS.md
- Option 1: Helm (Recommended)
- krkn-operator-data-provider
- Complete Workflow Examples
- API Workflow Test
- ClusterNameCollisionError
- sync_test.go
- rate_limiter_test.go
- .getClusterAPIURL
- 🔧 TODO: Pod Recreation and Retry Logic
- API Changes
- Running Locally
- install-yq.sh
- test-release-workflow-gates.sh
- Deployment
- Available Make Targets
- Building Images
- Troubleshooting
- KrknOperatorTargetProviderConfigStatus
- newProvidersHandler
- Troubleshooting
- .GetScenarioRunLogs
- assets/README.md
- GenerateFromToken
- scenario_filter_test.go
- NewSecretManager
- KrknUserGroupSpec
- mockAuthzChecker
- krkngraphrun_controller_sanitization_test.go
- BuildUnifiedJobList
- WorkflowResponse
- migrationBlockedError
- Pushing Images
- TestValidateValueAgainstSchema
- dataProviderServiceClient
- DataProviderServiceServer

## God Nodes (most connected - your core abstractions)
1. `Handler` - 218 edges
2. `NewTestHandler()` - 167 edges
3. `writeJSONError()` - 134 edges
4. `writeJSON()` - 100 edges
5. `createAdminContext()` - 91 edges
6. `GetClaimsFromContext()` - 64 edges
7. `KrknScenarioRun` - 63 edges
8. `newEsScheme()` - 57 edges
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

## Communities (171 total, 31 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.08
Nodes (20): availableWorkspacePVC, CategoryEntityAssociationResponse, categoryEnvironmentParameter, CreateCategoryRequest, ResiliencyHistoryQueryRequest, UpdateCategoryRequest, workspacePublishRequest, ContextKey (+12 more)

### Community 1 - "testing.T"
Cohesion: 0.02
Nodes (183): TestKrknUserCreation(), TestKrknUserList(), TestKrknUserRoles(), TestKrknUserStatus(), TestKrknUserValidation(), TestScenarioReferenceValidate(), NewConfigStoreInitializer(), TestConfigStoreInitializer_NeedLeaderElection() (+175 more)

### Community 2 - "Handler"
Cohesion: 0.08
Nodes (23): workspaceInfo, writeCategoryRunNotFoundOrInternalError(), buildCloudCredentialResponse(), updateCloudCredentialSecretData(), convertGraphClusterScores(), buildUserGroupResponse(), extractGroupNameAndUserIDFromPath(), extractGroupNameFromMembersPath() (+15 more)

### Community 3 - "InjectCredentials"
Cohesion: 0.09
Nodes (39): Publish and select a workspace, Replay and unavailable storage, Reuse from another operator, Scenario workspaces on a shared PVC, Storage validation, TestAppendCloudCredentialInjection_EmptyRefNoop(), injectAWS(), injectAzure() (+31 more)

### Community 4 - "GetClaimsFromContext"
Cohesion: 0.05
Nodes (49): AvailableFilesResponse, DeleteFileResponse, ListFilesResponse, UpdateFileResponse, buildFileInfo(), buildFileResponse(), deriveLogicalName(), extractLogicalName() (+41 more)

### Community 5 - "context.Context"
Cohesion: 0.04
Nodes (52): historyQueryCountingClient, mockUserStatusChecker, BackupConfig, resourceType, RestoreConfig, KrknWorkspaceReconciler, removeCategoryLabel(), isValidConfigMapKey() (+44 more)

### Community 6 - "NewTestHandler"
Cohesion: 0.11
Nodes (68): countingGetClient, esDoerFunc, boolPtr(), newEsAlertsTestSecretWithHost(), newEsScheme(), newEsTestSecret(), newEsTestSecretWithHost(), TestCreateElasticsearchConfig_DuplicateName() (+60 more)

### Community 7 - "cloudcreds_handlers_test.go"
Cohesion: 0.08
Nodes (56): TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(), TestCloudCredentialContract_GraphRunStripsNodeEnv(), newAWSTestSecret(), newCloudCredScheme(), newGCPTestSecret(), newGroupedAWSSecret(), TestCloudCredentialsRouter_MethodNotAllowed(), TestCloudCredentialsRouter_SubpathMethodNotAllowed() (+48 more)

### Community 8 - "utils.go"
Cohesion: 0.13
Nodes (19): attemptDirectory(), Inspect(), Run(), TestInspectRequiresReadableDirectory(), TestInspectRetainedRollbackFiles(), getMetricsOutput(), GetNonEmptyLines(), GetProjectDir() (+11 more)

### Community 9 - "rollback_handlers_test.go"
Cohesion: 0.05
Nodes (93): failingWorkspaceGetClient, RollbackAvailability, RollbackState, newCategoryRequest(), newCategoryRequestAs(), newCategoryTestGroup(), newCategoryTestHandler(), newCategoryTestUser() (+85 more)

### Community 10 - "api/types.go"
Cohesion: 0.05
Nodes (45): ActiveRunsOverviewResponse, AddGroupMemberRequest, AddGroupMemberResponse, ChangePasswordRequest, ChangePasswordResponse, ClusterPermissionSet, ClusterResiliencyScoreResponse, CreateTargetResponse (+37 more)

### Community 11 - "Broadcaster"
Cohesion: 0.11
Nodes (14): hashBytes(), TestFilterJobsByCategoriesMatchesAnySelectedCategory(), TestPaginateJobItems(), computeWSJobStats(), filterJobCategoriesByVisibility(), filterJobsByCategories(), groupRollbackJobs(), paginateJobItems() (+6 more)

### Community 12 - "migration.go"
Cohesion: 0.12
Nodes (43): clusterJobImage, clusterJobProvider, clusterJobRetries, graphScenarioReferences, migrationPlan, migrationResult, migrationStats, resiliencyScoreStatus (+35 more)

### Community 13 - "addAdminContext"
Cohesion: 0.12
Nodes (41): addAdminContext(), addUserContext(), createTestAdminUser(), setupFilesTestHandler(), TestCanAccessFile(), TestCreateFile(), TestCreateFile_DefaultFilePurpose(), TestCreateFile_DuplicateName() (+33 more)

### Community 14 - "k8s.io/api/core/v1.Secret"
Cohesion: 0.14
Nodes (7): buildConnectionParams(), buildConnectionParamsForIndex(), buildElasticsearchConfigResponse(), canAccessElasticsearchConfig(), buildRegistryInfo(), ExtractGroupsFromLabels(), SanitizeGroupName()

### Community 15 - "GetClaimsFromContext"
Cohesion: 0.16
Nodes (11): CategoryListResponse, CategoryResponse, buildCategoryResponse(), buildCategoryVisibilityLabels(), categoryErrorCode(), replaceCategoryVisibilityLabels(), validateCategoryColor(), validateCategoryName() (+3 more)

### Community 16 - "render-bundle.sh"
Cohesion: 0.07
Nodes (32): channel_has_entries(), ensure_catalog_channel(), catalog-channel.sh script, update_default_channel(), write_release_config(), catalog_submit_checkout(), catalog_submit_open_pr(), catalog-submit.sh script (+24 more)

### Community 17 - "server.py"
Cohesion: 0.06
Nodes (6): add_DataProviderServiceServicer_to_server(), DataProviderService, DataProviderServiceServicer, DataProviderServiceStub, DataProviderServicer, serve()

### Community 18 - "category_run_configuration.go"
Cohesion: 0.15
Nodes (25): scenarioFileBehavior, GraphScenarioNode, appendGraphSignaturePart(), compareGraphConfigurations(), compareGraphConfigurationsWithStableNodeIDs(), compareGraphNodeConfigurations(), compareGraphRunConfigurations(), compareScenarioRunConfigurations() (+17 more)

### Community 19 - "NewHub"
Cohesion: 0.10
Nodes (37): NewHandler(), NewBroadcaster(), TestBroadcastDashboardUpdate(), TestBroadcastGraphRunDeleted(), TestBroadcastGraphRunUpdate(), TestBroadcastOnlyToSubscribedClients(), TestBroadcastScenarioRunDeleted(), TestBroadcastScenarioRunUpdate() (+29 more)

### Community 20 - "Handler"
Cohesion: 0.10
Nodes (18): roleFromAdmin(), TestBuildUnifiedJobList_SkipsGraphRunChildren(), averageResiliencyScore(), buildGraphRunResponse(), buildScenarioRunDetailResponse(), buildScenarioRunResponse(), buildUnifiedJobList(), convertClusterResiliencyScoresWS() (+10 more)

### Community 21 - "user_handlers_test.go"
Cohesion: 0.15
Nodes (33): createTestUser(), kvValue(), setupUserTestHandler(), TestChangePassword_AdminChange_NoCurrentPassword(), TestChangePassword_AdminChangesOtherUser_EmitsAuditLog(), TestChangePassword_SamePassword_Rejected(), TestChangePassword_SelfChange_NoAdminAuditLog(), TestChangePassword_SelfChange_RequiresCurrentPassword() (+25 more)

### Community 22 - "client_test.go"
Cohesion: 0.10
Nodes (22): doerFunc, roundTripperFunc, NewClient(), scenarioParametersSchema(), TestBaseURL(), TestClientTransportDefaultNotHTTPTransport(), TestClientTransportPooling(), TestConnectionParamsTLSConfig() (+14 more)

### Community 23 - "KrknGraphRun"
Cohesion: 0.14
Nodes (10): KrknGraphRunReconciler, migrationPending(), TestGraphRunReconcileSkipsPendingMigration(), TestLoadFileConfigMapByID(), TestTranslateVolumesToFileMounts(), sanitizeNodeID(), TestCalculateResiliencyScore_StoresNodeScores(), overrideRef() (+2 more)

### Community 24 - "elasticsearch/types.go"
Cohesion: 0.10
Nodes (28): AffectedPods, CreateElasticsearchConfigRequest, CreateElasticsearchConfigResponse, DeleteElasticsearchConfigResponse, ListElasticsearchConfigsResponse, QueryAlertsRequest, QueryTelemetryRequest, RawJSON (+20 more)

### Community 25 - "installer.go"
Cohesion: 0.09
Nodes (24): bootstrapOLMResources(), main(), resolveMigrationNamespace(), runCRDMigrationMode(), TestResolveMigrationNamespacePrefersHookNamespace(), TestRunCRDMigrationMode(), TestBootstrapOLMResourcesPropagatesClientError(), TestBootstrapOLMResourcesWiring() (+16 more)

### Community 26 - "Middleware"
Cohesion: 0.17
Nodes (10): CachedUserStatusChecker, countingUserStatusChecker, userStatusCacheEntry, UserStatusChecker, NewCachedUserStatusChecker(), TestCachedUserStatusChecker(), TestCachedUserStatusChecker_EvictsExpiredEntries(), TestCachedUserStatusChecker_Invalidation() (+2 more)

### Community 27 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.13
Nodes (7): KrknTargetRequestReconciler, TestUnmarshalSecretData_Invalid(), UnmarshalSecretData(), KrknOperatorTarget, KrknOperatorTargetSpec, KrknTargetRequest, KrknTargetRequestSpec

### Community 28 - "GraphClusterScore"
Cohesion: 0.11
Nodes (7): convertNodeStatusesWithScores(), convertNodeStatusesWithScores(), GraphClusterScore, GraphRunSummary, KrknGraphRunStatus, KrknWorkspaceStatus, NodeStatus

### Community 29 - "client.go"
Cohesion: 0.11
Nodes (27): AlertDocument, esAlertSearchResponse, esSearchResponse, facetField, filterAggregation, Option, QueryAlertsResponse, StatusError (+19 more)

### Community 30 - "backup_restore_handlers_test.go"
Cohesion: 0.15
Nodes (22): NewJobTracker(), createMultipartBody(), createTestHandler(), TestGetRestoreStatusAdminOnly(), TestGetRestoreStatusTrackedJob(), TestGetRestoreStatusUnknownJob(), TestHandlerShutdownCancelsContext(), TestJobTrackerEviction() (+14 more)

### Community 31 - "handlers_test.go"
Cohesion: 0.04
Nodes (61): TestLogin_InactiveUser(), TestLogin_InvalidCredentials(), TestLogin_Success(), contains(), containsMiddle(), publicScenarioReference(), TestCreateGraphRun_WithFileReferences(), TestCreateGraphRun_ResiliencyScore() (+53 more)

### Community 32 - "Server"
Cohesion: 0.15
Nodes (8): Server, NewHandler(), maxBodySizeMiddleware(), NewServer(), NewServerWithComponentNamespace(), TestMaxBodySizeMiddleware(), Handler, SecretManager

### Community 33 - ".QueryTelemetryPage"
Cohesion: 0.22
Nodes (8): Doer, executeSearch(), Client, ConnectionParams, newSearchRequest(), TestTransportKey(), transportKey(), validateInlineDestination()

### Community 34 - "KrknOperatorTargetProviderConfig"
Cohesion: 0.10
Nodes (17): KrknOperatorTargetProviderConfigReconciler, Architecture Decision, Components Requiring Leader Election (Leader-Only), Components Running on All Replicas (Multi-Active), isConflictError(), countActiveProviders(), UpdateProviderConfig(), Configuration Functions (+9 more)

### Community 35 - "UnifiedJobItem"
Cohesion: 0.08
Nodes (17): ClusterJobStatusResponse, NodeStatusResponse, PaginationMeta, RestoreResponse, RollbackRunSummary, ScenarioRunListItem, ScenarioRunListResponse, ScenarioRunStatusResponse (+9 more)

### Community 37 - "k8s.io/apimachinery/pkg/apis/meta/v1.TypeMeta"
Cohesion: 0.05
Nodes (16): filterUsers(), GraphScenario, KrknCategoryList, KrknComponent, KrknComponentList, KrknGraphRunList, KrknOperatorTargetList, KrknOperatorTargetProviderConfigList (+8 more)

### Community 38 - "TokenGenerator"
Cohesion: 0.13
Nodes (11): Claims, Role, loggingMiddleware(), mockTokenGen(), mockTokenGenInvalid(), TokenGenerator, Middleware, NewLazyMiddleware() (+3 more)

### Community 39 - "watcher_test.go"
Cohesion: 0.19
Nodes (18): hasGraphRunStatusChanged(), hasScenarioRunStatusChanged(), TestHasGraphRunStatusChanged_NewNodeAdded(), TestHasGraphRunStatusChanged_NoChange(), TestHasGraphRunStatusChanged_NodeOrderChanged(), TestHasGraphRunStatusChanged_NodeStatusChange(), TestHasGraphRunStatusChanged_PhaseChange(), TestHasGraphRunStatusChanged_ResiliencyScoresChange() (+10 more)

### Community 40 - "PROGRESS-MVP.md"
Cohesion: 0.05
Nodes (39): Architecture Decisions, Data Provider Architecture, Environment Variables, Future Work (Not in Current Scope), GET /clusters, GET /nodes, GET /targets/{UUID}, Implementation Files (+31 more)

### Community 41 - "bootstrap.go"
Cohesion: 0.17
Nodes (13): RemoveElasticsearchGrafanaURLAnnotations(), TestRemoveElasticsearchGrafanaURLAnnotations(), ensureJWTSecret(), EnsureResources(), ensureSCCBinding(), ensureScenarioRunnerRBAC(), ensureServiceAccount(), ensureServices() (+5 more)

### Community 42 - "github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2"
Cohesion: 0.18
Nodes (12): /scenarios/detail/{scenario_name} ✅ COMPLETED, /scenarios/globals/{scenario_name} ✅ COMPLETED, mockScenarioProvider, registryImagesResult, scenarioDetailResult, ScenarioTag, fetchScenarioCatalog(), filterScenariosByIsAScenario() (+4 more)

### Community 43 - "wsorigin.go"
Cohesion: 0.24
Nodes (13): checkWebSocketOriginV2(), TestCheckWebSocketOriginV2(), checkWebSocketOrigin(), TestCheckWebSocketOrigin(), canonicalOrigin(), defaultPort(), IsAllowedOrigin(), portsMatch() (+5 more)

### Community 44 - "WSUnifiedJobItem"
Cohesion: 0.11
Nodes (7): GraphRunResponse, ServerMessage, WSJobStatsSummary, WSPaginationMeta, WSRollbackRunSummary, WSUnifiedJobItem, WSUnifiedJobsSnapshot

### Community 45 - "migration_test.go"
Cohesion: 0.20
Nodes (25): Complete(), Stage(), containsSubstring(), countPatchActions(), jsonMarshal(), migrationTestLogger(), newMigrationClient(), scenarioRunFixture() (+17 more)

### Community 46 - "notify-operator-docs.test.js"
Cohesion: 0.10
Nodes (11): assert, chartWorkflow, { execFileSync, spawnSync }, fs, notifier, notifyWorkflow, os, path (+3 more)

### Community 47 - "converter_test.go"
Cohesion: 0.13
Nodes (19): rollbackImageCache, rollbackImageCheck, Rollback artifact persistence, Rollback history in Jobs, BuildDockerConfigJSON(), ExtractRegistryV2FromSecret(), TestBuildDockerConfigJSON_MissingCredentials(), TestBuildDockerConfigJSON_OAuth() (+11 more)

### Community 48 - "ssrf.go"
Cohesion: 0.12
Nodes (9): responseWriter, guardedCheckRedirect(), guardedDialContext(), TestGuardedCheckRedirect(), TestQueryTelemetryAllowsUnrestrictedLoopback(), TestValidateDestinationIP(), TestValidateDestinationURL(), validateDestinationIP() (+1 more)

### Community 49 - "KrknScenarioRunSpec"
Cohesion: 0.15
Nodes (13): API Endpoints Structure (Nested Approach), Configuration, Current Behavior, Implementation Plan (TODO), ✅ Implemented, Moving from direct scenario creation to CRD approach, Option A: Cancellation Field in Status (Recommended), Option B: Finalizers for Cancellation Tracking (+5 more)

### Community 50 - "Hub"
Cohesion: 0.24
Nodes (6): isWebSocketDisconnectError(), writeWSError(), BroadcastMessage, Client, Hub, PaginationClientState

### Community 51 - "KrknWorkspace"
Cohesion: 0.25
Nodes (4): CanAccess(), TestCanAccess(), KrknWorkspace, KrknWorkspaceSpec

### Community 52 - "POST methods"
Cohesion: 0.12
Nodes (15): API Requirements, /clusters, GET methods, get nodes, get pods, Grpc python service requirement, /nodes, Overview (+7 more)

### Community 54 - "providerConfigTestScheme"
Cohesion: 0.18
Nodes (13): addKnownTypes(), decodeErrorResponse(), providerConfigTestScheme(), TestGetProviderConfigByUUID_CompletedReturnsConfigData(), TestGetProviderConfigByUUID_InternalErrorReturnsSanitized500(), TestGetProviderConfigByUUID_NotFoundReturns404(), TestGetProviderConfigByUUID_PendingReturnsAccepted(), TestPostProviderConfig_FailureReturnsSanitizedError() (+5 more)

### Community 55 - "newReportTestHandler"
Cohesion: 0.29
Nodes (10): newReportTestHandler(), reportRequest(), TestDownloadReportDeniesUserWithoutReportClusterAccess(), TestDownloadReportHTMLThroughV2Router(), TestDownloadReportPDF(), TestDownloadReportRejectsMalformedPath(), TestDownloadReportUnavailable(), TestGetReportStatusPending() (+2 more)

### Community 56 - "cloudcreds/types.go"
Cohesion: 0.14
Nodes (15): CreateCloudCredentialResponse, DeleteCloudCredentialResponse, ListCloudCredentialsResponse, UpdateCloudCredentialResponse, buildCloudCredentialSecretData(), CloudCredentialResponse, CreateCloudCredentialRequest, UpdateCloudCredentialRequest (+7 more)

### Community 57 - "GetNodesRequest"
Cohesion: 0.15
Nodes (3): DataProviderServiceClient, GetNodesRequest, GetNodesResponse

### Community 58 - "dataprovider_grpc.pb.go"
Cohesion: 0.16
Nodes (5): UnimplementedDataProviderServiceServer, UnsafeDataProviderServiceServer, _DataProviderService_ExecuteKubectl_Handler(), _DataProviderService_GetNodes_Handler(), RegisterDataProviderServiceServer()

### Community 59 - ".UpdateRegistry"
Cohesion: 0.17
Nodes (7): buildRegistryResponse(), validateCreateRegistryRequest(), validateUpdateRegistryRequest(), BuildRegistryLabels(), ExtractGroupsFromLabels(), TestUpdateRegistryAnnotations(), UpdateRegistryAnnotations()

### Community 60 - "parseResiliencyHistoryQuery"
Cohesion: 0.18
Nodes (10): resiliencyHistoryQuerySelections, normalizeResiliencyHistoryProviders(), normalizeResiliencyHistorySelections(), parseResiliencyHistoryQuery(), createTarGz(), TestCreateAndExtractTarGz(), TestCreateTarGzFlushErrors(), TestExtractTarGzPathTraversal() (+2 more)

### Community 61 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.07
Nodes (16): ClustersResponse, GraphRunListItem, GraphRunListResponse, GraphRunStatusResponse, timeEqual(), countClustersFromTargetData(), TestCountClustersFromTargetData(), TestFilterClustersByPermission() (+8 more)

### Community 62 - "ScenarioRunRequest"
Cohesion: 0.18
Nodes (13): Phase 12: Scenario Job Management - /scenarios/run Endpoints ✅, Phase 9: POST /scenarios Endpoint ✅, FileMount, GlobalsRequest, JobsListResponse, JobStatusResponse, ScenarioRunRequest, ScenarioRunResponse (+5 more)

### Community 63 - "scenario_catalog_cache_test.go"
Cohesion: 0.20
Nodes (12): digestPointer(), postScenariosForCacheTest(), TestFetchScenarioCatalogReturnsTimeoutInsteadOfPartialCatalog(), TestGetRegistryImagesWithContextReturnsWhenProviderStalls(), TestPostScenariosCachesCatalogWhenTagHasNoScenarioLabels(), TestPostScenariosCachesPublicCatalog(), TestPostScenariosDoesNotCacheNilDetail(), TestPostScenariosRetriesUnknownSignatureBeforeCachingCatalog() (+4 more)

### Community 64 - "GET methods"
Cohesion: 0.11
Nodes (18): DELETE methods, GET methods, Implementation details, POST methods, Request example:, Request example:, Response:, Response example: (+10 more)

### Community 65 - "installer_test.go"
Cohesion: 0.18
Nodes (12): crdClient, applyCRD(), applyCRDs(), TestApplyCRDsCreatesMissingAndUpdatesChangedSpecs(), TestApplyCRDsDoesNotPatchUnchangedSpecs(), TestApplyCRDsLabelsExistingCRDsWithoutChangingOtherLabels(), TestApplyCRDsRejectsNonCRDManifest(), testCRD() (+4 more)

### Community 66 - "buildClusterAPIURLMap"
Cohesion: 0.25
Nodes (8): buildClusterAPIURLMap(), contains(), containsInMiddle(), TestBuildClusterAPIURLMap(), TestBuildClusterAPIURLMap_Collision(), TestBuildClusterAPIURLMap_SameNameSameURL(), TestValidateScenarioRunAccess(), TestValidateScenarioRunAccess_NoGroups()

### Community 67 - "buildContainerImageFromReference"
Cohesion: 0.22
Nodes (13): buildContainerImage(), buildContainerImageFromReference(), getTestConfig(), privateReference(), publicReference(), TestBuildContainerImage_PublicRegistryUsesKrknctl(), TestBuildContainerImage_RejectsPrivateWithoutResolvedRegistry(), TestBuildContainerImageFromReference_PrivateRegistry() (+5 more)

### Community 68 - "elasticsearch/labels_test.go"
Cohesion: 0.33
Nodes (5): TestBuildAnnotations(), TestBuildLabels(), TestBuildLabelsAccessControl(), TestUpdateAnnotations(), UpdateAnnotations()

### Community 69 - "Handler"
Cohesion: 0.09
Nodes (9): ErrorResponse, Handler, TerminalRequest, TerminalResponse, scenarioRunImage(), HasClusterPermission(), Action, IsValidAction() (+1 more)

### Community 70 - "Publisher"
Cohesion: 0.11
Nodes (7): Publisher, NewNamespaceFilter(), TestPublisherReconcile(), TestPublisherReconcileReturnsTransientErrors(), TestPublisherRecoversFromConcurrentCreateAndDelete(), TestPublisherStartRetriesAndReturnsCancellation(), ProviderConfigContributorReconciler

### Community 71 - "start_operator.sh"
Cohesion: 0.41
Nodes (12): build_operator(), check_prerequisites(), cleanup(), create_service_account(), install_crds(), main(), print_error(), print_info() (+4 more)

### Community 72 - "files/validation_test.go"
Cohesion: 0.31
Nodes (8): contains(), findSubstring(), TestValidateFileContent(), TestValidateFileGroups(), TestValidateUserFilePermissions(), ValidateFileContent(), ValidateFileGroups(), ValidateUserFilePermissions()

### Community 73 - "github.com/krkn-chaos/krknctl/pkg/verify.SignatureStatus"
Cohesion: 0.22
Nodes (6): publicSignatureCache, publicSignatureCacheEntry, publicSignatureCachingProvider, InvalidImageSignatureError, isDefinitiveSignatureStatus(), TestPublicSignatureCacheLoadOutlivesFirstWaiter()

### Community 74 - "Compute"
Cohesion: 0.22
Nodes (5): ComputeJobStats(), TestComputeJobStats(), Compute(), JobCounter, Summary

### Community 75 - "scenarioRunConfigurationFixture"
Cohesion: 0.21
Nodes (18): datapointForRunID(), datapointRunIDs(), TestBuildCategoryResiliencyHistoryGroupsTypedConfigurationsAndSortsPerCluster(), TestGraphHistoryAuthorizationRequiresAUsableClusterURL(), TestGraphHistoryScoresOnlyKeepsFinalScoresIncludingZero(), TestScenarioHistoryScoresOnlyKeepsCalculatedClusterScores(), compareCategoryRunConfigurations(), assertConfigurationDifferences() (+10 more)

### Community 76 - "KrknOperatorTargetProvider"
Cohesion: 0.09
Nodes (20): Completed, Current Status, In Progress, Pending, Basic Usage (Default Configuration), CleanupOldResources, Config Struct, Configuration Options (+12 more)

### Community 77 - ".PostScenarioRun"
Cohesion: 0.15
Nodes (14): Code References, writeScenarioRunAccessError(), parseBoolHeader(), TestParseBoolHeader(), sanitizeRunNameLabel(), resolveMaxRetries(), IsCloudEnvVar(), StripCloudEnvVars() (+6 more)

### Community 78 - "test-api.sh"
Cohesion: 0.39
Nodes (11): api_call(), cleanup(), extract_body(), extract_http_code(), log_error(), log_info(), log_step(), log_success() (+3 more)

### Community 79 - "Frontend Authentication Implementation Tasks"
Cohesion: 0.06
Nodes (30): 1. Read the API Documentation, 1. Start the Operator, 2. Check Available Work, 2. Test Registration (First Admin), 3. Start with Foundation Task, 3. Test Login, 401 on Protected Routes, 4. Claim a Task (+22 more)

### Community 82 - "KrknScenarioRunReconciler"
Cohesion: 0.07
Nodes (32): imageResolver, imageSignatureVerifier, KrknScenarioRunReconciler, preparedJobResources, preStartFailureReason(), TestPreStartFailureReason(), TestScenarioRunReconcileSkipsPendingMigration(), backfillLegacyMaxRetries() (+24 more)

### Community 83 - ".extractAndStoreReports"
Cohesion: 0.17
Nodes (10): decodeReportPayloads(), extractDelimitedContent(), TestDecodeReportPayloadsPreservesValidReport(), TestDecodeReportPayloadsReportsInvalidHTML(), TestExtractAndStoreReportsRejectsMultiClusterRun(), TestExtractDelimitedContent(), TestTargetClusterCount(), TestValidateReportPayloadSize() (+2 more)

### Community 84 - "category_parameter_profile.go"
Cohesion: 0.19
Nodes (14): categoryHistoryRun, categoryParameterProfileFingerprint(), categoryParameterProfileName(), TestCategoryParameterProfileFingerprintForEmptyEnvironment(), TestCategoryParameterProfileFingerprintGraphEnvironmentMultiset(), TestCategoryParameterProfileFingerprintIgnoresRunType(), TestCategoryParameterProfileFingerprintScenarioEnvironment(), TestCategoryParameterProfileNameIsDeterministicAndReadable() (+6 more)

### Community 85 - "Krkn-Operator Roadmap"
Cohesion: 0.29
Nodes (6): Krkn-Operator Roadmap, Ongoing (not release-blocked), Release 1 — 2026-08-18, Release 2 — 2026-10-18, Release 3 — 2026-12-22, Release 4 — 2027-02-23

### Community 86 - "KrknUserGroup"
Cohesion: 0.22
Nodes (14): categoryHistoryScore, containsString(), filterCategoryHistoryScoresByCluster(), filterGraphHistoryScoresByPermission(), filterScenarioHistoryScoresByPermission(), graphHistoryScores(), runHasAnyCategory(), scenarioHistoryScores() (+6 more)

### Community 87 - ".flatten"
Cohesion: 0.22
Nodes (12): ClusterMetadata, FacetOption, NodeSummaryInfo, QueryTelemetryResponse, rawTelemetrySource, TelemetryDocument, TelemetryStats, decodeTelemetry() (+4 more)

### Community 88 - "ScenariosResponse"
Cohesion: 0.60
Nodes (4): publicScenarioCache, publicScenarioCacheEntry, ScenariosResponse, cloneScenariosResponse()

### Community 89 - "metadata_presence_test.go"
Cohesion: 0.27
Nodes (9): intPtr(), TestRawTelemetrySourceFlatten(), contains(), containsAt(), TestMetadataJSONMarshaling(), TestMetadataPresenceTracking(), boolPtr(), TestValidateCreateRequest() (+1 more)

### Community 90 - "ResolveGraph"
Cohesion: 0.12
Nodes (15): FromKrknctlScenarioSet(), publicScenarioReference(), publicScenario(), TestFromKrknctlScenarioSetCreatesPublicReferences(), TestToKrknctlScenarioSetUsesScenarioReference(), ToKrknctlScenarioSet(), publicScenarioReference(), TestCreateScenarioRun_ResiliencyScore() (+7 more)

### Community 91 - "KrknCategory"
Cohesion: 0.21
Nodes (8): categoryGroups(), categoryIsPublic(), categoryVisibleToUser(), publicHistoryCategory(), ExtractGroupNamesFromLabels(), TestExtractGroupNamesFromLabels(), KrknCategory, KrknCategorySpec

### Community 92 - "WorkspaceReference"
Cohesion: 0.18
Nodes (6): KrknScenarioRunSpec, WorkspaceSelection, MapScenarioNodeToScenarioRunSpec(), FileMount, RollbackSource, WorkspaceReference

### Community 93 - "makeScenarioRun"
Cohesion: 0.24
Nodes (13): makeGraphRun(), makeScenarioRun(), TestListJobs_GroupedRollbacksBeforePagination(), TestListJobs_IncludesStats(), TestListJobs_LimitOnlyReturnsAll(), TestListJobs_NoPagination(), TestListJobs_Paginated(), TestListJobs_SkipsGraphRunChildren() (+5 more)

### Community 95 - "ProviderRegistration"
Cohesion: 0.15
Nodes (12): API Reference, CreateProviderConfigRequest, Interfaces, NewProviderRegistration, NewProviderRegistrationWithConfig, Provider Configuration Functions, Provider Registration Functions, UpdateProviderConfig (+4 more)

### Community 96 - "Provider Configuration API Specification"
Cohesion: 0.06
Nodes (30): 1. List Available Providers (Optional), 1. List Providers, 2. Create Config Request, 2. Get Provider Config Status, 2. Update Provider Status, 3. Create Provider Config Request, 3. Poll for Completion, 3. Update Provider Configuration (+22 more)

### Community 97 - ".PostScenarioDetail"
Cohesion: 0.29
Nodes (3): convertInputFields(), createScenarioProvider(), writeScenarioCatalogError()

### Community 98 - "dataprovider.pb.go"
Cohesion: 0.20
Nodes (3): file_proto_dataprovider_proto_init(), file_proto_dataprovider_proto_rawDescGZIP(), init()

### Community 99 - "TestKrknGraphRun_ValidateGraph"
Cohesion: 0.33
Nodes (5): SanitizeNodeIDForKubernetes(), stringPtr(), TestCollisionDetection(), TestKrknGraphRun_ValidateGraph(), TestSanitizeNodeIDForKubernetes()

### Community 100 - "CategoryResiliencyHistoryResponse"
Cohesion: 0.60
Nodes (5): CategoryConfigurationGroup, CategoryResiliencyDataPoint, CategoryResiliencyHistoryResponse, ResiliencyHistoryQueryResponse, assembleResiliencyHistoryQueryResponse()

### Community 101 - "Authentication API Specification"
Cohesion: 0.09
Nodes (22): 1. Check Admin Registration, 1. Initial Load, 2. First Admin Registration, 2. Register User, 3. Login, 3. Login, 4. Authenticated Requests, 5. Role-Based UI (+14 more)

### Community 102 - ".ListAvailableCloudCredentials"
Cohesion: 0.31
Nodes (4): cloudCredentialAccessContext, ExtractGroupsFromLabels(), ResolveAccessControlForUpdate(), TestExtractGroupsFromLabels()

### Community 103 - "Publisher"
Cohesion: 0.15
Nodes (7): ListComponentsResponse, Load(), TestLoad(), TestLoadRejectsInvalidManifest(), ComponentDescription, ComponentManifest, KrknComponentStatus

### Community 104 - "setup-ocm-and-operator.sh"
Cohesion: 0.67
Nodes (5): log_error(), log_info(), log_warn(), setup-ocm-and-operator.sh script, wait_for_condition()

### Community 105 - "File Management API Specification"
Cohesion: 0.09
Nodes (22): Access Control, API Endpoints, ConfigMap Structure, Content Validation, Create a Public Configuration File, Create File, Delete File, Error Responses (+14 more)

### Community 108 - "commands.go"
Cohesion: 0.13
Nodes (19): AvailableCommandsResponse, TestExecuteTerminal_CommandParsing(), TestExecuteTerminal_ValidationLogic(), GetAllowedCommands(), GetBlockedFlags(), CommandSpec, IsCommandAllowed(), IsSubcommandAllowed() (+11 more)

### Community 109 - "Cluster Segregation Troubleshooting"
Cohesion: 0.09
Nodes (21): Access Rules, Cluster Segregation Troubleshooting, Common Issues, Debugging Steps, Example 1: Proper Segregation, Example 2: Multiple Groups, Example 3: Admin User, Expected Behavior Examples (+13 more)

### Community 110 - "channel-head.sh"
Cohesion: 0.50
Nodes (3): OLM_CHANNEL_NAME, OLM_PACKAGE_NAME, channel-head.sh script

### Community 111 - "Running gRPC Data Provider Locally for Development"
Cohesion: 0.11
Nodes (16): 1. Login and Get JWT Token, 2. Create KrknTargetRequest (or use existing UUID), 3. Test Terminal Command, Cleanup, Configure Operator to Use Local gRPC Server, Debug gRPC Server in PyCharm, Development Workflow, Notes (+8 more)

### Community 117 - "CLAUDE.md - Project Instructions"
Cohesion: 0.10
Nodes (19): Architecture Principles, Before Committing, Before Writing Code, CLAUDE.md - Project Instructions, Code Organization & Reusability, Code Quality Standards, Communication, Critical Rules (+11 more)

### Community 120 - "ResolvePrivateTagDigest"
Cohesion: 0.22
Nodes (9): resolveRollbackDigest(), resolvePrivateImageDigest(), TestResolvePrivateImageDigestUsesManifestDigest(), ResolvePrivateTagDigest(), ResolvePublicTagDigest(), TestResolvePrivateTagDigestMissing(), TestResolvePrivateTagDigestWithAuthentication(), TestResolvePublicTagDigestMissingAndCancelled() (+1 more)

### Community 128 - "Setup"
Cohesion: 0.40
Nodes (5): 1. Create Virtual Environment (Python 3.11), 2. Install Dependencies, 3. Verify Generated Stubs, 4. Start gRPC Server, Setup

### Community 129 - "Implementation Details"
Cohesion: 0.14
Nodes (16): Code Refactoring: getKubeconfigFromTarget() Helper ✅, Implementation Details, Phase 10: POST /scenarios/detail/{scenario_name} Endpoint ✅, Phase 11: POST /scenarios/globals/{scenario_name} Endpoint ✅, Phase 1: REST API Setup ✅, Phase 2: GET /clusters Endpoint ✅, Phase 3: Integration ✅, Phase 4: Testing ✅ (+8 more)

### Community 130 - "ProviderRegistration"
Cohesion: 0.11
Nodes (18): After (Multi-Active API), API Server Concurrent Operations, API Server Startup Wait, Before (Single Leader), Concurrency Safety, Controller Safety with Leader Election, Deployment Considerations, Expected Startup Logs (+10 more)

### Community 131 - "krkn-operator"
Cohesion: 0.12
Nodes (17): Alerts Query API, API compatibility notes, Backup and Restore, Documentation, Ecosystem, Graph node resiliency weights, krkn-operator, License (+9 more)

### Community 132 - "Deployment Guide"
Cohesion: 0.13
Nodes (13): Architecture, Build Installer Bundle, Deployment Guide, Environment Variables, Git Tag Support, Git Tag Workflow, Image Names (Auto-Generated), Makefile Variables (+5 more)

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

### Community 138 - "Complete Workflow Examples"
Cohesion: 0.50
Nodes (4): Complete Workflow Examples, Custom Registry, Development (Local), Production (With Git Tag)

### Community 139 - "API Workflow Test"
Cohesion: 0.18
Nodes (10): API Workflow Test, Configuration, Error Handling, krkn-operator Test Scripts, Output, Overview, Prerequisites, Testing Against Production (+2 more)

### Community 141 - "sync_test.go"
Cohesion: 0.11
Nodes (12): ConfigStoreInitializer, Store, SyncConfigMapToStore(), TestRoundTrip(), TestSyncConfigMapToStore(), TestSyncConfigMapToStore_NilStore(), TestWriteConfigMapData(), WriteConfigMapData() (+4 more)

### Community 142 - "rate_limiter_test.go"
Cohesion: 0.11
Nodes (22): ipRateLimiter, rateLimiterEntry, getTokenDuration(), TestGetTokenDuration(), clientIPFromHeaders(), extractClientIP(), hostFromRemoteAddr(), isTrustedProxy() (+14 more)

### Community 143 - ".getClusterAPIURL"
Cohesion: 0.46
Nodes (7): newTargetRequest(), newTestScheme(), TestGetClusterAPIURL_ClusterNotFound(), TestGetClusterAPIURL_Found(), TestGetClusterAPIURL_ProviderNotFound(), TestGetClusterAPIURL_TargetRequestNotFound(), TestReconcile_FailedJobStatus_IncludesClusterAPIURL()

### Community 144 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.18
Nodes (10): API and operator refactoring to support multiple clusters, BREAKING CHANGES, create configmap for operator based on KrknOperatorTargetProviderConfig, Implementation Details, KrknOperatorTargetProviderConfig, KrknTargetRequest controller in operator, Overview, Overview (+2 more)

### Community 145 - "API Changes"
Cohesion: 0.29
Nodes (7): GraphRunDetailResponse, GraphRunSpecResponse, 1.1 Resiliency weight per graph node, 1. Request - Create GraphRun (POST /api/v1/graphruns), 2. Response - List GraphRuns (GET /api/v1/graphruns), 3. Response - Get GraphRun Detail (GET /api/v1/graphruns/:name), API Changes

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

### Community 153 - "KrknOperatorTargetProviderConfigStatus"
Cohesion: 0.29
Nodes (3): getProviderNames(), KrknOperatorTargetProviderConfigStatus, ProviderConfigData

### Community 154 - "newProvidersHandler"
Cohesion: 0.40
Nodes (5): newProvider(), newProvidersHandler(), TestListProviders_Empty(), TestListProviders_ScopedToNamespace(), TestUpdateProviderStatus()

### Community 155 - "Troubleshooting"
Cohesion: 0.40
Nodes (5): Error: "Command not found: kubectl", Error: "failed to connect to gRPC server", Error: "Failed to get kubeconfig", Error: "no module named 'krkn_lib'", Troubleshooting

### Community 156 - ".GetScenarioRunLogs"
Cohesion: 0.24
Nodes (8): filterReportLines(), isReportMarkerEnd(), isReportMarkerStart(), maskToken(), sanitizeHeaders(), TestFilterReportLines(), TestIsReportMarkerEnd(), TestIsReportMarkerStart()

### Community 158 - "GenerateFromToken"
Cohesion: 0.18
Nodes (16): CreateTargetRequest, UpdateTargetRequest, generateKubeconfigFromCredentialsType(), generateKubeconfigFromKubeconfigType(), generateKubeconfigFromRequest(), generateKubeconfigFromTokenType(), TestCreateTarget_DuplicateClusterName(), TestCreateTarget_WithKubeconfig() (+8 more)

### Community 159 - "scenario_filter_test.go"
Cohesion: 0.33
Nodes (3): TestFilterScenariosByIsAScenario(), TestFilterScenariosByIsAScenario_CancelledContext(), TestFilterScenariosByIsAScenario_PreservesTagMetadata()

### Community 160 - "NewSecretManager"
Cohesion: 0.11
Nodes (20): k8sUserStatusChecker, TestListComponentsRouteAndAuth(), TestQueryElasticsearchTelemetry_RouteAndAuth(), sanitizeUsername(), TestSanitizeUsername_ProducesValidK8sNames(), newK8sUserStatusChecker(), newKrknUser(), newUserStatusScheme() (+12 more)

### Community 163 - "krkngraphrun_controller_sanitization_test.go"
Cohesion: 0.83
Nodes (3): isAlphanumeric(), isValidLabelChar(), TestSanitizeNodeID()

### Community 164 - "BuildUnifiedJobList"
Cohesion: 0.11
Nodes (16): convertClusterResiliencyScores(), averageResiliencyScore(), BuildUnifiedJobList(), getDefaultPageSize(), GroupRollbackJobs(), runCategoryNames(), TestBuildUnifiedJobList_IdentifiesRollbackFromSource(), TestBuildUnifiedJobList_IncludesResolvedScenarioImage() (+8 more)

### Community 165 - "WorkflowResponse"
Cohesion: 0.67
Nodes (3): workflowResponsesHaveCategories(), WorkflowResponse, ListWorkflowsResponse

### Community 167 - "Pushing Images"
Cohesion: 0.50
Nodes (4): Custom Push, Push Both Images, Push Individual Images, Pushing Images

## Knowledge Gaps
- **17 isolated node(s):** `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME`, `ICON_BASE64`, `ICON_MEDIATYPE` (+12 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 735 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **31 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Handler` connect `Handler` to `main.go`, `Handler`, `GetClaimsFromContext`, `context.Context`, `k8s.io/api/core/v1.Secret`, `GetClaimsFromContext`, `.GetScenarioRunLogs`, `Server`, `.QueryTelemetryPage`, `BuildUnifiedJobList`, `bootstrap.go`, `github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2`, `converter_test.go`, `.UpdateRegistry`, `github.com/krkn-chaos/krknctl/pkg/verify.SignatureStatus`, `.PostScenarioRun`, `KrknUserGroup`, `ScenariosResponse`, `.PostScenarioDetail`, `.ListAvailableCloudCredentials`, `JobTracker`?**
  _High betweenness centrality (0.070) - this node is a cross-community bridge._
- **Are the 160 inferred relationships involving `NewTestHandler()` (e.g. with `setupAuthTestHandler()` and `TestLogin_InactiveUser()`) actually correct?**
  _`NewTestHandler()` has 160 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME` to the rest of the system?**
  _17 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.08378879724402928 - nodes in this community are weakly interconnected._
- **Why does `KrknWorkspace` connect `KrknWorkspace` to `main.go`, `Handler`, `InjectCredentials`, `.DeepCopy`, `Handler`, `k8s.io/apimachinery/pkg/apis/meta/v1.TypeMeta`, `rollback_handlers_test.go`, `Deployment`, `k8s.io/apimachinery/pkg/apis/meta/v1.Time`, `GraphClusterScore`?**
  _High betweenness centrality (0.041) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.01633028578000115 - nodes in this community are weakly interconnected._
- **Why does `Deployment Guide` connect `Deployment Guide` to `Pushing Images`, `Option 1: Helm (Recommended)`, `Complete Workflow Examples`, `Deployment`, `Available Make Targets`, `Building Images`, `Troubleshooting`?**
  _High betweenness centrality (0.039) - this node is a cross-community bridge._