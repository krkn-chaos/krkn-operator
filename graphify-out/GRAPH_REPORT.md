# Graph Report - krkn-operator  (2026-11-11)

## Corpus Check
- 341 files · ~435,303 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 18 file(s) not represented in the graph (top: (none) 12, .jsonl 2, .toml 1)

## Summary
- 3678 nodes · 14982 edges · 161 communities (132 shown, 29 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 1393 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `5d50d354`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- main.go
- testing.T
- Handler
- .PostScenarioDetail
- GetClaimsFromContext
- context.Context
- NewTestHandler
- NewTestHandler
- utils.go
- rollback_handlers_test.go
- api/types.go
- Broadcaster
- migration.go
- addAdminContext
- Handler
- GetClaimsFromContext
- render-bundle.sh
- server.py
- category_run_configuration.go
- NewHub
- WSUnifiedJobItem
- user_handlers_test.go
- client_test.go
- KrknGraphRun
- elasticsearch/types.go
- installer.go
- NewTokenGenerator
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- KrknUserGroup
- client.go
- backup_restore_handlers_test.go
- handlers_test.go
- Handler
- ConnectionParams
- KrknOperatorTargetProviderConfig
- convertClusterJobs
- GetClaimsFromContext
- NewCachedUserStatusChecker
- watcher_test.go
- ComponentManifest
- bootstrap.go
- UnifiedJobItem
- wsorigin.go
- files/validation_test.go
- migration_test.go
- notify-operator-docs.test.js
- converter_test.go
- ssrf.go
- KrknScenarioRunSpec
- Hub
- InjectCredentials
- .PostScenarioRun
- test_helpers.go
- rate_limiter.go
- newReportTestHandler
- cloudcreds/types.go
- GetNodesRequest
- dataprovider_grpc.pb.go
- elasticsearch/validate.go
- parseResiliencyHistoryQuery
- .DeepCopyInto
- ScenarioRunRequest
- go_pkg_sync
- GET methods
- installer_test.go
- sync_test.go
- buildContainerImageFromReference
- github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2
- .ExecuteTerminal
- Publisher
- start_operator.sh
- setupAuthTestHandler
- PROGRESS-MVP.md
- Compute
- .Start
- KrknOperatorTargetProvider
- buildClusterAPIURLMap
- test-api.sh
- Frontend Authentication Implementation Tasks
- ExecuteKubectlRequest
- mockJob
- KrknScenarioRun
- captureLogSink
- category_parameter_profile.go
- Krkn-Operator Roadmap
- KrknUserGroup
- .flatten
- GraphRunResponse
- GraphClusterScore
- ResolveGraph
- buildUnifiedJobList
- SecretManager
- dataprovider_grpc.pb.go
- ExecuteKubectlResponse
- ProviderRegistration
- Provider Configuration API Specification
- NewServerWithComponentNamespace
- dataprovider.pb.go
- TestKrknGraphRun_ValidateGraph
- makeScenarioRun
- Authentication API Specification
- mockAuthzChecker
- StripCloudEnvVars
- setup-ocm-and-operator.sh
- File Management API Specification
- JobTracker
- commands.go
- Cluster Segregation Troubleshooting
- channel-head.sh
- Running gRPC Data Provider Locally for Development
- api_workflow_test.sh
- ClusterNameCollisionError
- select-compatible-release.sh
- test-open-kubernetes-operators-pr.sh
- CLAUDE.md - Project Instructions
- post-install.sh
- generate_proto_go.sh
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
- time.Duration
- API compatibility notes
- 🔧 TODO: Pod Recreation and Retry Logic
- Running Locally
- install-yq.sh
- test-release-workflow-gates.sh
- Deployment
- Available Make Targets
- Building Images
- Troubleshooting
- KrknOperatorTargetProviderConfigStatus
- Troubleshooting
- .GetScenarioRunLogs
- assets/README.md
- WorkflowResponse
- olm_bootstrap_test.go
- BuildUnifiedJobList
- krkngraphrun_controller_sanitization_test.go
- Pushing Images
- providerConfigTestScheme
- schema_validator.go
- GenerateFromToken

## God Nodes (most connected - your core abstractions)
1. `Handler` - 213 edges
2. `NewTestHandler()` - 167 edges
3. `writeJSONError()` - 133 edges
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
- `Overview` --references--> `KrknOperatorTargetProviderConfig`  [INFERRED]
  docs/api-provider-config-spec.md → api/v1alpha1/krknoperatortargetproviderconfig_types.go

## Import Cycles
- None detected.

## Communities (161 total, 29 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.10
Nodes (17): availableWorkspacePVC, CategoryEntityAssociationResponse, CreateCategoryRequest, ResiliencyHistoryQueryRequest, UpdateCategoryRequest, workspacePublishRequest, ContextKey, clusterTarget (+9 more)

### Community 1 - "testing.T"
Cohesion: 0.02
Nodes (171): TestKrknUserCreation(), TestKrknUserList(), TestKrknUserRoles(), TestKrknUserStatus(), TestKrknUserValidation(), TestScenarioReferenceValidate(), TestKrknCategoryDeepCopy(), TestListComponents() (+163 more)

### Community 2 - "Handler"
Cohesion: 0.07
Nodes (16): writeCategoryRunNotFoundOrInternalError(), buildCloudCredentialResponse(), updateCloudCredentialSecretData(), convertGraphClusterScores(), extractGroupNameAndUserIDFromPath(), extractGroupNameFromMembersPath(), validateClusterPermissions(), convertMetaTime() (+8 more)

### Community 3 - ".PostScenarioDetail"
Cohesion: 0.31
Nodes (3): convertInputFields(), createScenarioProvider(), getScenarioSignatureStatus()

### Community 4 - "GetClaimsFromContext"
Cohesion: 0.06
Nodes (47): AvailableFilesResponse, DeleteFileResponse, ListFilesResponse, UpdateFileResponse, buildFileInfo(), buildFileResponse(), deriveLogicalName(), extractLogicalName() (+39 more)

### Community 5 - "context.Context"
Cohesion: 0.04
Nodes (59): historyQueryCountingClient, mockUserStatusChecker, BackupConfig, resourceType, RestoreConfig, Code References, removeCategoryLabel(), validateCreateFileRequest() (+51 more)

### Community 6 - "NewTestHandler"
Cohesion: 0.06
Nodes (101): countingGetClient, esDoerFunc, TestGetActiveRunsOverview_AllUsersSeesAll(), TestGetActiveRunsOverview_MixedJobStates(), TestGetActiveRunsOverview_NoActiveRuns(), TestGetActiveRunsOverview_UserSeesAll(), boolPtr(), newEsAlertsTestSecretWithHost() (+93 more)

### Community 7 - "NewTestHandler"
Cohesion: 0.11
Nodes (39): TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(), TestCloudCredentialContract_GraphRunStripsNodeEnv(), newAWSTestSecret(), newCloudCredScheme(), newGCPTestSecret(), newGroupedAWSSecret(), TestCloudCredentialsRouter_MethodNotAllowed(), TestCloudCredentialsRouter_SubpathMethodNotAllowed() (+31 more)

### Community 8 - "utils.go"
Cohesion: 0.11
Nodes (21): tokenRequest, attemptDirectory(), Inspect(), Run(), TestInspectRequiresReadableDirectory(), TestInspectRetainedRollbackFiles(), TestE2E(), getMetricsOutput() (+13 more)

### Community 9 - "rollback_handlers_test.go"
Cohesion: 0.05
Nodes (91): failingWorkspaceGetClient, RollbackAvailability, newCategoryRequest(), newCategoryRequestAs(), newCategoryTestGroup(), newCategoryTestHandler(), newCategoryTestUser(), TestCategoriesRouterCRUD() (+83 more)

### Community 10 - "api/types.go"
Cohesion: 0.05
Nodes (45): ActiveRunsOverviewResponse, AddGroupMemberRequest, AddGroupMemberResponse, ChangePasswordRequest, ChangePasswordResponse, ClusterPermissionSet, ClusterResiliencyScoreResponse, CreateTargetResponse (+37 more)

### Community 11 - "Broadcaster"
Cohesion: 0.14
Nodes (5): Handler, hashBytes(), Broadcaster, graphRunEventHandler, scenarioRunEventHandler

### Community 12 - "migration.go"
Cohesion: 0.14
Nodes (35): clusterJobImage, clusterJobProvider, clusterJobRetries, graphScenarioReferences, migrationBlockedError, migrationPlan, resiliencyScoreStatus, addProvider() (+27 more)

### Community 13 - "addAdminContext"
Cohesion: 0.12
Nodes (41): addAdminContext(), addUserContext(), createTestAdminUser(), setupFilesTestHandler(), TestCanAccessFile(), TestCreateFile(), TestCreateFile_DefaultFilePurpose(), TestCreateFile_DuplicateName() (+33 more)

### Community 14 - "Handler"
Cohesion: 0.11
Nodes (8): cloudCredentialAccessContext, Handler, buildConnectionParams(), buildConnectionParamsForIndex(), buildElasticsearchConfigResponse(), canAccessElasticsearchConfig(), ExtractGroupsFromLabels(), SanitizeGroupName()

### Community 15 - "GetClaimsFromContext"
Cohesion: 0.11
Nodes (17): CategoryListResponse, CategoryResponse, ErrorResponse, buildCategoryResponse(), buildCategoryVisibilityLabels(), categoryErrorCode(), categoryGroups(), categoryIsPublic() (+9 more)

### Community 16 - "render-bundle.sh"
Cohesion: 0.07
Nodes (32): channel_has_entries(), ensure_catalog_channel(), catalog-channel.sh script, update_default_channel(), write_release_config(), catalog_submit_checkout(), catalog_submit_open_pr(), catalog-submit.sh script (+24 more)

### Community 17 - "server.py"
Cohesion: 0.06
Nodes (6): add_DataProviderServiceServicer_to_server(), DataProviderService, DataProviderServiceServicer, DataProviderServiceStub, DataProviderServicer, serve()

### Community 18 - "category_run_configuration.go"
Cohesion: 0.17
Nodes (26): scenarioFileBehavior, GraphScenarioNode, appendGraphSignaturePart(), compareCategoryRunConfigurations(), compareGraphConfigurations(), compareGraphConfigurationsWithStableNodeIDs(), compareGraphNodeConfigurations(), compareGraphRunConfigurations() (+18 more)

### Community 19 - "NewHub"
Cohesion: 0.10
Nodes (33): NewHandler(), NewBroadcaster(), TestBroadcastDashboardUpdate(), TestBroadcastGraphRunDeleted(), TestBroadcastGraphRunUpdate(), TestBroadcastOnlyToSubscribedClients(), TestBroadcastScenarioRunDeleted(), TestBroadcastScenarioRunUpdate() (+25 more)

### Community 20 - "WSUnifiedJobItem"
Cohesion: 0.11
Nodes (15): roleFromAdmin(), TestFilterJobsByCategoriesMatchesAnySelectedCategory(), TestPaginateJobItems(), computeWSJobStats(), filterJobCategoriesByVisibility(), filterJobsByCategories(), groupRollbackJobs(), paginateJobItems() (+7 more)

### Community 21 - "user_handlers_test.go"
Cohesion: 0.15
Nodes (35): createTestUser(), createUserContext(), kvValue(), setupUserTestHandler(), TestChangePassword_AdminChange_NoCurrentPassword(), TestChangePassword_AdminChangesOtherUser_EmitsAuditLog(), TestChangePassword_SamePassword_Rejected(), TestChangePassword_SelfChange_NoAdminAuditLog() (+27 more)

### Community 22 - "client_test.go"
Cohesion: 0.12
Nodes (23): doerFunc, roundTripperFunc, NewClient(), scenarioParametersSchema(), TestBaseURL(), TestClientTransportDefaultNotHTTPTransport(), TestClientTransportPooling(), TestConnectionParamsTLSConfig() (+15 more)

### Community 23 - "KrknGraphRun"
Cohesion: 0.16
Nodes (9): KrknGraphRunReconciler, TestGraphRunReconcileSkipsPendingMigration(), TestLoadFileConfigMapByID(), TestTranslateVolumesToFileMounts(), sanitizeNodeID(), TestCalculateResiliencyScore_StoresNodeScores(), overrideRef(), TestGraphRunInvalidImageSignatureFailsWithoutRetry() (+1 more)

### Community 24 - "elasticsearch/types.go"
Cohesion: 0.13
Nodes (21): AffectedPods, AlertDocument, CreateElasticsearchConfigResponse, DeleteElasticsearchConfigResponse, ListElasticsearchConfigsResponse, QueryAlertsRequest, QueryAlertsResponse, QueryTelemetryRequest (+13 more)

### Community 25 - "installer.go"
Cohesion: 0.09
Nodes (21): main(), resolveMigrationNamespace(), runCRDMigrationMode(), TestResolveMigrationNamespacePrefersHookNamespace(), TestRunCRDMigrationMode(), configMapClient, crdManifest, deploymentClient (+13 more)

### Community 26 - "NewTokenGenerator"
Cohesion: 0.07
Nodes (34): TestRequireAdminForMethods(), TestGroupNameValidation(), TestGroupsRouterEdgeCases(), TestCheckScenarioRunAccess(), TestFilterScenarioRunsByGroupPermission(), TestPostScenarioRun_ClusterNameCollision_Returns500(), TestPostScenarioRun_SameClusterSameURL_NoCollision(), TestPostScenarioRunSetsOwner() (+26 more)

### Community 27 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.10
Nodes (9): ClustersResponse, KrknTargetRequestReconciler, ClusterHealthStatus, ClusterTarget, KrknOperatorTarget, KrknOperatorTargetSpec, KrknTargetRequest, KrknTargetRequestSpec (+1 more)

### Community 28 - "KrknUserGroup"
Cohesion: 0.15
Nodes (12): categoryHistoryScore, containsString(), filterCategoryHistoryScoresByCluster(), filterGraphHistoryScoresByPermission(), filterScenarioHistoryScoresByPermission(), graphHistoryScores(), runHasAnyCategory(), scenarioHistoryScores() (+4 more)

### Community 29 - "client.go"
Cohesion: 0.12
Nodes (24): esAlertSearchResponse, facetField, filterAggregation, Option, StatusError, termsAggregation, buildAlertsSearchBody(), buildFacetAggs() (+16 more)

### Community 30 - "backup_restore_handlers_test.go"
Cohesion: 0.15
Nodes (22): NewJobTracker(), createMultipartBody(), createTestHandler(), TestGetRestoreStatusAdminOnly(), TestGetRestoreStatusTrackedJob(), TestGetRestoreStatusUnknownJob(), TestHandlerShutdownCancelsContext(), TestJobTrackerEviction() (+14 more)

### Community 31 - "handlers_test.go"
Cohesion: 0.06
Nodes (39): TestLogin_InactiveUser(), TestLogin_InvalidCredentials(), TestLogin_Success(), TestConvertInputFields(), TestDeleteTargetByUUID_AdminCanDeleteAnyResource(), TestDeleteTargetByUUID_InvalidUUID(), TestDeleteTargetByUUID_NonOwner_Forbidden(), TestDeleteTargetByUUID_NotFound() (+31 more)

### Community 32 - "Handler"
Cohesion: 0.22
Nodes (5): visibleCategoryNames(), AuthorizationChecker, categoryVisibilityChecker, ClientMessage, Handler

### Community 33 - "ConnectionParams"
Cohesion: 0.17
Nodes (15): Doer, esSearchResponse, FacetOption, QueryTelemetryResponse, TelemetryDocument, TelemetryStats, decodeTelemetry(), executeSearch() (+7 more)

### Community 34 - "KrknOperatorTargetProviderConfig"
Cohesion: 0.08
Nodes (23): KrknOperatorTargetProviderConfigReconciler, 1. List Providers, 2. Get Provider Config Status, 2. Update Provider Status, 3. Create Provider Config Request, 3. Update Provider Configuration, API Endpoints, Architecture Decision (+15 more)

### Community 35 - "convertClusterJobs"
Cohesion: 0.25
Nodes (6): convertClusterJobs(), TestConvertClusterJobs_EmptyInput(), TestConvertClusterJobs_NilInput(), TestConvertClusterJobs_OmitsClusterAPIURL(), ScenarioAttemptStatus, ClusterJobResponse

### Community 37 - "GetClaimsFromContext"
Cohesion: 0.17
Nodes (7): extractPathSuffix(), buildRegistryResponse(), buildUserResponse(), GetClaimsFromContext(), IsAdmin(), BuildRegistryLabels(), ExtractGroupsFromLabels()

### Community 38 - "NewCachedUserStatusChecker"
Cohesion: 0.17
Nodes (10): CachedUserStatusChecker, countingUserStatusChecker, userStatusCacheEntry, UserStatusChecker, NewCachedUserStatusChecker(), TestCachedUserStatusChecker(), TestCachedUserStatusChecker_EvictsExpiredEntries(), TestCachedUserStatusChecker_Invalidation() (+2 more)

### Community 39 - "watcher_test.go"
Cohesion: 0.19
Nodes (18): hasGraphRunStatusChanged(), hasScenarioRunStatusChanged(), TestHasGraphRunStatusChanged_NewNodeAdded(), TestHasGraphRunStatusChanged_NoChange(), TestHasGraphRunStatusChanged_NodeOrderChanged(), TestHasGraphRunStatusChanged_NodeStatusChange(), TestHasGraphRunStatusChanged_PhaseChange(), TestHasGraphRunStatusChanged_ResiliencyScoresChange() (+10 more)

### Community 40 - "ComponentManifest"
Cohesion: 0.15
Nodes (7): ListComponentsResponse, Load(), TestLoad(), TestLoadRejectsInvalidManifest(), ComponentDescription, ComponentManifest, KrknComponentStatus

### Community 41 - "bootstrap.go"
Cohesion: 0.20
Nodes (11): ensureJWTSecret(), EnsureResources(), ensureSCCBinding(), ensureScenarioRunnerRBAC(), ensureServiceAccount(), ensureServices(), intstrFromInt(), namespaceHash() (+3 more)

### Community 42 - "UnifiedJobItem"
Cohesion: 0.08
Nodes (17): ClusterJobStatusResponse, GraphRunListItem, GraphRunListResponse, GraphRunStatusResponse, NodeStatusResponse, PaginationMeta, RestoreResponse, RollbackRunSummary (+9 more)

### Community 43 - "wsorigin.go"
Cohesion: 0.22
Nodes (13): checkWebSocketOriginV2(), TestCheckWebSocketOriginV2(), checkWebSocketOrigin(), TestCheckWebSocketOrigin(), canonicalOrigin(), defaultPort(), IsAllowedOrigin(), portsMatch() (+5 more)

### Community 44 - "files/validation_test.go"
Cohesion: 0.27
Nodes (10): isValidConfigMapKey(), validateFileFields(), contains(), findSubstring(), TestValidateFileContent(), TestValidateFileGroups(), TestValidateUserFilePermissions(), ValidateFileContent() (+2 more)

### Community 45 - "migration_test.go"
Cohesion: 0.13
Nodes (34): migrationResult, migrationStats, resourceDefinition, Complete(), forEachResource(), loadGraphScenarioReferences(), logSummary(), patchMigrationResourceWithRetry() (+26 more)

### Community 46 - "notify-operator-docs.test.js"
Cohesion: 0.10
Nodes (11): assert, chartWorkflow, { execFileSync, spawnSync }, fs, notifier, notifyWorkflow, os, path (+3 more)

### Community 47 - "converter_test.go"
Cohesion: 0.13
Nodes (19): rollbackImageCache, rollbackImageCheck, Rollback artifact persistence, Rollback history in Jobs, BuildDockerConfigJSON(), ExtractRegistryV2FromSecret(), TestBuildDockerConfigJSON_MissingCredentials(), TestBuildDockerConfigJSON_OAuth() (+11 more)

### Community 48 - "ssrf.go"
Cohesion: 0.16
Nodes (9): responseWriter, guardedCheckRedirect(), guardedDialContext(), TestGuardedCheckRedirect(), TestValidateDestinationIP(), TestValidateDestinationURL(), validateDestinationIP(), validateDestinationURL() (+1 more)

### Community 49 - "KrknScenarioRunSpec"
Cohesion: 0.12
Nodes (12): KrknScenarioRunSpec, Configuration, Current Behavior, Implementation Plan (TODO), Option A: Cancellation Field in Status (Recommended), Option B: Finalizers for Cancellation Tracking, Option C: Separate CancellationRequest CR, Proposed Solution Options (+4 more)

### Community 50 - "Hub"
Cohesion: 0.27
Nodes (5): AuthorizationCheckFunc, BroadcastMessage, Client, Hub, PaginationClientState

### Community 51 - "InjectCredentials"
Cohesion: 0.07
Nodes (47): workspaceInfo, Publish and select a workspace, Replay and unavailable storage, Reuse from another operator, Scenario workspaces on a shared PVC, Storage validation, workspaceInfoFromCR(), TestAppendCloudCredentialInjection_EmptyRefNoop() (+39 more)

### Community 52 - ".PostScenarioRun"
Cohesion: 0.16
Nodes (11): k8sUserStatusChecker, writeScenarioRunAccessError(), parseBoolHeader(), TestParseBoolHeader(), sanitizeResourceName(), sanitizeRunNameLabel(), resolveMaxRetries(), sanitizeUsername() (+3 more)

### Community 53 - "test_helpers.go"
Cohesion: 0.15
Nodes (14): RemoveElasticsearchGrafanaURLAnnotations(), TestRemoveElasticsearchGrafanaURLAnnotations(), intPtr(), TestRawTelemetrySourceFlatten(), TestUpdateAnnotations(), UpdateAnnotations(), contains(), containsAt() (+6 more)

### Community 54 - "rate_limiter.go"
Cohesion: 0.18
Nodes (18): clientIPFromHeaders(), extractClientIP(), hostFromRemoteAddr(), isTrustedProxy(), newIPRateLimiter(), parseTrustedProxyCIDRs(), rateLimitMiddleware(), TestExtractClientIP() (+10 more)

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
Cohesion: 0.15
Nodes (5): DataProviderServiceServer, UnimplementedDataProviderServiceServer, _DataProviderService_ExecuteKubectl_Handler(), _DataProviderService_GetNodes_Handler(), RegisterDataProviderServiceServer()

### Community 59 - "elasticsearch/validate.go"
Cohesion: 0.27
Nodes (8): CreateElasticsearchConfigRequest, UpdateElasticsearchConfigRequest, TestValidateCreateRequest(), validateAccess(), ValidateCreateRequest(), ValidateMergedConnection(), validateTLSSettings(), ValidateUpdateRequest()

### Community 60 - "parseResiliencyHistoryQuery"
Cohesion: 0.18
Nodes (10): resiliencyHistoryQuerySelections, normalizeResiliencyHistoryProviders(), normalizeResiliencyHistorySelections(), parseResiliencyHistoryQuery(), createTarGz(), TestCreateAndExtractTarGz(), TestCreateTarGzFlushErrors(), TestExtractTarGzPathTraversal() (+2 more)

### Community 61 - ".DeepCopyInto"
Cohesion: 0.04
Nodes (24): filterUsers(), GraphRunSummary, GraphScenario, KrknCategoryList, KrknCategorySpec, KrknComponent, KrknComponentList, KrknGraphRunList (+16 more)

### Community 62 - "ScenarioRunRequest"
Cohesion: 0.15
Nodes (16): Phase 12: Scenario Job Management - /scenarios/run Endpoints ✅, Phase 9: POST /scenarios Endpoint ✅, FileMount, GlobalsRequest, JobsListResponse, JobStatusResponse, ScenarioRunRequest, ScenarioRunResponse (+8 more)

### Community 64 - "GET methods"
Cohesion: 0.07
Nodes (28): API Requirements, /clusters, DELETE methods, GET methods, GET methods, get nodes, get pods, Grpc python service requirement (+20 more)

### Community 65 - "installer_test.go"
Cohesion: 0.19
Nodes (12): crdClient, applyCRD(), applyCRDs(), TestApplyCRDsCreatesMissingAndUpdatesChangedSpecs(), TestApplyCRDsDoesNotPatchUnchangedSpecs(), TestApplyCRDsLabelsExistingCRDsWithoutChangingOtherLabels(), TestApplyCRDsRejectsNonCRDManifest(), testCRD() (+4 more)

### Community 66 - "sync_test.go"
Cohesion: 0.33
Nodes (6): SyncConfigMapToStore(), TestRoundTrip(), TestSyncConfigMapToStore(), TestSyncConfigMapToStore_NilStore(), TestWriteConfigMapData(), WriteConfigMapData()

### Community 67 - "buildContainerImageFromReference"
Cohesion: 0.17
Nodes (15): publicScenarioReference(), TestCreateScenarioRun_ResiliencyScore(), buildContainerImage(), buildContainerImageFromReference(), getTestConfig(), privateReference(), publicReference(), TestBuildContainerImage_PublicRegistryUsesKrknctl() (+7 more)

### Community 68 - "github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2"
Cohesion: 0.09
Nodes (23): POST methods, refactoring, /scenarios ✅ COMPLETED, /scenarios/detail/{scenario_name} ✅ COMPLETED, /scenarios/globals/{scenario_name} ✅ COMPLETED, /scenarios/run/{UUID}, /targets ✅ COMPLETED, mockScenarioProvider (+15 more)

### Community 69 - ".ExecuteTerminal"
Cohesion: 0.25
Nodes (3): TerminalRequest, TerminalResponse, NewDataProviderServiceClient()

### Community 70 - "Publisher"
Cohesion: 0.09
Nodes (10): Publisher, KrknWorkspaceReconciler, NewNamespaceFilter(), TestWorkspaceReconcilerMapsPVCWithoutOwningIt(), TestWorkspaceReconcilerTracksPVCState(), TestPublisherReconcile(), TestPublisherReconcileReturnsTransientErrors(), TestPublisherRecoversFromConcurrentCreateAndDelete() (+2 more)

### Community 71 - "start_operator.sh"
Cohesion: 0.41
Nodes (12): build_operator(), check_prerequisites(), cleanup(), create_service_account(), install_crds(), main(), print_error(), print_info() (+4 more)

### Community 72 - "setupAuthTestHandler"
Cohesion: 0.25
Nodes (7): setupAuthTestHandler(), TestIsRegistered_MethodNotAllowed(), TestIsRegistered_NoAdmins(), TestIsRegistered_WithAdmin(), TestRegister_FirstAdmin_Success(), TestRegister_FirstUser_MustBeAdmin(), TestRegister_Validation()

### Community 73 - "PROGRESS-MVP.md"
Cohesion: 0.04
Nodes (41): Architecture Decisions, Data Provider Architecture, Environment Variables, Future Work (Not in Current Scope), GET /clusters, GET /nodes, GET /targets/{UUID}, Implementation Files (+33 more)

### Community 74 - "Compute"
Cohesion: 0.22
Nodes (5): ComputeJobStats(), TestComputeJobStats(), Compute(), JobCounter, Summary

### Community 75 - ".Start"
Cohesion: 0.20
Nodes (8): ConfigStoreInitializer, NewConfigStoreInitializer(), TestConfigStoreInitializer_NeedLeaderElection(), TestConfigStoreInitializer_Start_ConfigMapNotFound(), TestConfigStoreInitializer_Start_Success(), BackfillLegacyProviderConfigLabel(), TestBackfillLegacyProviderConfigLabel(), TestBackfillLegacyProviderConfigLabelMissingConfigMap()

### Community 76 - "KrknOperatorTargetProvider"
Cohesion: 0.09
Nodes (20): Completed, Current Status, In Progress, Pending, Basic Usage (Default Configuration), CleanupOldResources, Config Struct, Configuration Options (+12 more)

### Community 77 - "buildClusterAPIURLMap"
Cohesion: 0.25
Nodes (8): buildClusterAPIURLMap(), contains(), containsInMiddle(), TestBuildClusterAPIURLMap(), TestBuildClusterAPIURLMap_Collision(), TestBuildClusterAPIURLMap_SameNameSameURL(), TestValidateScenarioRunAccess(), TestValidateScenarioRunAccess_NoGroups()

### Community 78 - "test-api.sh"
Cohesion: 0.39
Nodes (11): api_call(), cleanup(), extract_body(), extract_http_code(), log_error(), log_info(), log_step(), log_success() (+3 more)

### Community 79 - "Frontend Authentication Implementation Tasks"
Cohesion: 0.06
Nodes (30): 1. Read the API Documentation, 1. Start the Operator, 2. Check Available Work, 2. Test Registration (First Admin), 3. Start with Foundation Task, 3. Test Login, 401 on Protected Routes, 4. Claim a Task (+22 more)

### Community 82 - "KrknScenarioRun"
Cohesion: 0.05
Nodes (48): imageResolver, imageSignatureVerifier, KrknScenarioRunReconciler, preparedJobResources, migrationPending(), TestScenarioRunReconcileSkipsPendingMigration(), backfillLegacyMaxRetries(), countJobPhaseChanges() (+40 more)

### Community 85 - "Krkn-Operator Roadmap"
Cohesion: 0.29
Nodes (6): Krkn-Operator Roadmap, Ongoing (not release-blocked), Release 1 — 2026-08-18, Release 2 — 2026-10-18, Release 3 — 2026-12-22, Release 4 — 2027-02-23

### Community 86 - "KrknUserGroup"
Cohesion: 0.11
Nodes (35): CategoryConfigurationGroup, categoryHistoryRun, CategoryResiliencyDataPoint, CategoryResiliencyHistoryResponse, ResiliencyHistoryQueryResponse, categoryParameterProfileFingerprint(), categoryParameterProfileName(), TestCategoryParameterProfileFingerprintForEmptyEnvironment() (+27 more)

### Community 87 - ".flatten"
Cohesion: 0.43
Nodes (4): ClusterMetadata, NodeSummaryInfo, rawTelemetrySource, TestFlattenRepresentativeScenario()

### Community 88 - "GraphRunResponse"
Cohesion: 0.29
Nodes (3): buildGraphRunResponse(), convertGraphClusterScoresForSnapshot(), GraphRunResponse

### Community 89 - "GraphClusterScore"
Cohesion: 0.22
Nodes (4): convertNodeStatusesWithScores(), convertNodeStatusesWithScores(), GraphClusterScore, NodeStatus

### Community 90 - "ResolveGraph"
Cohesion: 0.15
Nodes (13): FromKrknctlScenarioSet(), publicScenarioReference(), publicScenario(), TestFromKrknctlScenarioSetCreatesPublicReferences(), TestToKrknctlScenarioSetUsesScenarioReference(), ToKrknctlScenarioSet(), MapScenarioNodeToScenarioRunSpec(), ResolveGraph() (+5 more)

### Community 91 - "buildUnifiedJobList"
Cohesion: 0.17
Nodes (12): TestBuildUnifiedJobList_SkipsGraphRunChildren(), averageResiliencyScore(), buildScenarioRunDetailResponse(), buildScenarioRunResponse(), buildUnifiedJobList(), convertClusterResiliencyScoresWS(), runCategoryNames(), sanitizedClusterJobs() (+4 more)

### Community 92 - "SecretManager"
Cohesion: 0.11
Nodes (10): Server, TestListComponentsRouteAndAuth(), NewHandler(), NewServer(), NewServerWithComponentNamespace(), generateRandomSecret(), getSecretKeys(), SecretManager (+2 more)

### Community 95 - "ProviderRegistration"
Cohesion: 0.15
Nodes (12): API Reference, CreateProviderConfigRequest, Interfaces, NewProviderRegistration, NewProviderRegistrationWithConfig, Provider Configuration Functions, Provider Registration Functions, UpdateProviderConfig (+4 more)

### Community 96 - "Provider Configuration API Specification"
Cohesion: 0.08
Nodes (24): 1. List Available Providers (Optional), 2. Create Config Request, 3. Poll for Completion, 4. Parse Schemas and Build UI, 5. Submit Configuration, Base URL, Change Log, Common Error Patterns (+16 more)

### Community 97 - "NewServerWithComponentNamespace"
Cohesion: 0.11
Nodes (13): Claims, Role, loggingMiddleware(), maxBodySizeMiddleware(), TestMaxBodySizeMiddleware(), mockTokenGen(), mockTokenGenInvalid(), TokenGenerator (+5 more)

### Community 98 - "dataprovider.pb.go"
Cohesion: 0.20
Nodes (3): file_proto_dataprovider_proto_init(), file_proto_dataprovider_proto_rawDescGZIP(), init()

### Community 99 - "TestKrknGraphRun_ValidateGraph"
Cohesion: 0.33
Nodes (5): SanitizeNodeIDForKubernetes(), stringPtr(), TestCollisionDetection(), TestKrknGraphRun_ValidateGraph(), TestSanitizeNodeIDForKubernetes()

### Community 100 - "makeScenarioRun"
Cohesion: 0.20
Nodes (15): GroupRollbackJobs(), makeGraphRun(), makeScenarioRun(), TestGroupRollbackJobs_UsesSourceUIDAndKeepsOrphans(), TestListJobs_GroupedRollbacksBeforePagination(), TestListJobs_IncludesStats(), TestListJobs_LimitOnlyReturnsAll(), TestListJobs_NoPagination() (+7 more)

### Community 101 - "Authentication API Specification"
Cohesion: 0.09
Nodes (22): 1. Check Admin Registration, 1. Initial Load, 2. First Admin Registration, 2. Register User, 3. Login, 3. Login, 4. Authenticated Requests, 5. Role-Based UI (+14 more)

### Community 103 - "StripCloudEnvVars"
Cohesion: 0.50
Nodes (4): IsCloudEnvVar(), StripCloudEnvVars(), TestIsCloudEnvVar(), TestStripCloudEnvVars()

### Community 104 - "setup-ocm-and-operator.sh"
Cohesion: 0.67
Nodes (5): log_error(), log_info(), log_warn(), setup-ocm-and-operator.sh script, wait_for_condition()

### Community 105 - "File Management API Specification"
Cohesion: 0.09
Nodes (22): Access Control, API Endpoints, ConfigMap Structure, Content Validation, Create a Public Configuration File, Create File, Delete File, Error Responses (+14 more)

### Community 108 - "commands.go"
Cohesion: 0.12
Nodes (21): AvailableCommandsResponse, TestExecuteTerminal_CommandParsing(), TestExecuteTerminal_ValidationLogic(), GetAllowedCommands(), GetBlockedFlags(), CommandSpec, IsCommandAllowed(), IsSubcommandAllowed() (+13 more)

### Community 109 - "Cluster Segregation Troubleshooting"
Cohesion: 0.09
Nodes (21): Access Rules, Cluster Segregation Troubleshooting, Common Issues, Debugging Steps, Example 1: Proper Segregation, Example 2: Multiple Groups, Example 3: Admin User, Expected Behavior Examples (+13 more)

### Community 110 - "channel-head.sh"
Cohesion: 0.50
Nodes (3): OLM_CHANNEL_NAME, OLM_PACKAGE_NAME, channel-head.sh script

### Community 111 - "Running gRPC Data Provider Locally for Development"
Cohesion: 0.12
Nodes (16): 1. Login and Get JWT Token, 2. Create KrknTargetRequest (or use existing UUID), 3. Test Terminal Command, Cleanup, Configure Operator to Use Local gRPC Server, Debug gRPC Server in PyCharm, Development Workflow, Notes (+8 more)

### Community 117 - "CLAUDE.md - Project Instructions"
Cohesion: 0.10
Nodes (19): Architecture Principles, Before Committing, Before Writing Code, CLAUDE.md - Project Instructions, Code Organization & Reusability, Code Quality Standards, Communication, Critical Rules (+11 more)

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
Nodes (14): Alerts Query API, Backup and Restore, Documentation, Ecosystem, krkn-operator, License, OLM / OperatorHub bundles, Operator release documentation snapshots (+6 more)

### Community 132 - "Deployment Guide"
Cohesion: 0.13
Nodes (13): Architecture, Build Installer Bundle, Deployment Guide, Environment Variables, Git Tag Support, Git Tag Workflow, Image Names (Auto-Generated), Makefile Variables (+5 more)

### Community 133 - "Quick Start - Terminal API Local Testing"
Cohesion: 0.12
Nodes (17): 1. Setup gRPC Server, 2. Run Operator (new terminal), 3. Test the API (new terminal), "401 Unauthorized", 🧪 Automated Testing, "Command not found: kubectl", 🔥 Common Issues, Debugging (+9 more)

### Community 134 - "Resiliency Score Feature - Frontend Implementation Guide"
Cohesion: 0.09
Nodes (22): GraphRunDetailResponse, GraphRunSpecResponse, 1.1 Resiliency weight per graph node, 1. Request - Create GraphRun (POST /api/v1/graphruns), 2. Response - List GraphRuns (GET /api/v1/graphruns), 3. Response - Get GraphRun Detail (GET /api/v1/graphruns/:name), API Changes, Backend Implementation Details (+14 more)

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

### Community 142 - "time.Duration"
Cohesion: 0.18
Nodes (4): ipRateLimiter, rateLimiterEntry, getTokenDuration(), TestGetTokenDuration()

### Community 143 - "API compatibility notes"
Cohesion: 0.50
Nodes (4): API compatibility notes, Graph node resiliency weights, Log streaming, Retry limits

### Community 144 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.17
Nodes (12): API and operator refactoring to support multiple clusters, API Endpoints Structure (Nested Approach), BREAKING CHANGES, Implementation Details, ✅ Implemented, KrknTargetRequest controller in operator, Moving from direct scenario creation to CRD approach, Overview (+4 more)

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

### Community 155 - "Troubleshooting"
Cohesion: 0.40
Nodes (5): Error: "Command not found: kubectl", Error: "failed to connect to gRPC server", Error: "Failed to get kubeconfig", Error: "no module named 'krkn_lib'", Troubleshooting

### Community 156 - ".GetScenarioRunLogs"
Cohesion: 0.19
Nodes (12): filterReportLines(), isReportMarkerEnd(), isReportMarkerStart(), isWebSocketDisconnectError(), maskToken(), preStartFailureReason(), sanitizeHeaders(), writeWSError() (+4 more)

### Community 162 - "WorkflowResponse"
Cohesion: 0.67
Nodes (3): workflowResponsesHaveCategories(), WorkflowResponse, ListWorkflowsResponse

### Community 163 - "olm_bootstrap_test.go"
Cohesion: 0.67
Nodes (3): bootstrapOLMResources(), TestBootstrapOLMResourcesPropagatesClientError(), TestBootstrapOLMResourcesWiring()

### Community 164 - "BuildUnifiedJobList"
Cohesion: 0.12
Nodes (13): convertClusterResiliencyScores(), averageResiliencyScore(), BuildUnifiedJobList(), getDefaultPageSize(), runCategoryNames(), scenarioRunImage(), TestBuildUnifiedJobList_IdentifiesRollbackFromSource(), TestBuildUnifiedJobList_IncludesResolvedScenarioImage() (+5 more)

### Community 166 - "krkngraphrun_controller_sanitization_test.go"
Cohesion: 0.83
Nodes (3): isAlphanumeric(), isValidLabelChar(), TestSanitizeNodeID()

### Community 167 - "Pushing Images"
Cohesion: 0.50
Nodes (4): Custom Push, Push Both Images, Push Individual Images, Pushing Images

### Community 168 - "providerConfigTestScheme"
Cohesion: 0.13
Nodes (18): addKnownTypes(), decodeErrorResponse(), providerConfigTestScheme(), TestGetProviderConfigByUUID_CompletedReturnsConfigData(), TestGetProviderConfigByUUID_InternalErrorReturnsSanitized500(), TestGetProviderConfigByUUID_NotFoundReturns404(), TestGetProviderConfigByUUID_PendingReturnsAccepted(), TestPostProviderConfig_FailureReturnsSanitizedError() (+10 more)

### Community 171 - "GenerateFromToken"
Cohesion: 0.18
Nodes (16): CreateTargetRequest, UpdateTargetRequest, generateKubeconfigFromCredentialsType(), generateKubeconfigFromKubeconfigType(), generateKubeconfigFromRequest(), generateKubeconfigFromTokenType(), TestCreateTarget_DuplicateClusterName(), TestCreateTarget_WithKubeconfig() (+8 more)

## Knowledge Gaps
- **17 isolated node(s):** `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME`, `ICON_BASE64`, `ICON_MEDIATYPE` (+12 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 731 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **29 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Handler` connect `Handler` to `main.go`, `Handler`, `.PostScenarioDetail`, `GetClaimsFromContext`, `context.Context`, `GetClaimsFromContext`, `KrknUserGroup`, `.GetScenarioRunLogs`, `ConnectionParams`, `BuildUnifiedJobList`, `GetClaimsFromContext`, `bootstrap.go`, `converter_test.go`, `.PostScenarioRun`, `github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2`, `.ExecuteTerminal`, `SecretManager`, `NewServerWithComponentNamespace`, `JobTracker`?**
  _High betweenness centrality (0.059) - this node is a cross-community bridge._
- **Are the 160 inferred relationships involving `NewTestHandler()` (e.g. with `setupAuthTestHandler()` and `TestLogin_InactiveUser()`) actually correct?**
  _`NewTestHandler()` has 160 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME` to the rest of the system?**
  _17 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.09797148903885317 - nodes in this community are weakly interconnected._
- **Why does `Running Locally` connect `Running Locally` to `krkn-operator`?**
  _High betweenness centrality (0.024) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.017402597402597402 - nodes in this community are weakly interconnected._
- **Why does `KrknWorkspace` connect `InjectCredentials` to `main.go`, `.DeepCopy`, `context.Context`, `rollback_handlers_test.go`, `Deployment`, `k8s.io/apimachinery/pkg/apis/meta/v1.Time`, `.DeepCopyInto`?**
  _High betweenness centrality (0.024) - this node is a cross-community bridge._