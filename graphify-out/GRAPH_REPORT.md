# Graph Report - krkn-operator  (2026-11-11)

## Corpus Check
- 342 files · ~436,175 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 18 file(s) not represented in the graph (top: (none) 12, .jsonl 2, .toml 1)

## Summary
- 3684 nodes · 15004 edges · 179 communities (147 shown, 32 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 1399 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `b9c9f29e`
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
- newCategoryRequestAs
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- KrknUserGroupSpec
- client.go
- backup_restore_handlers_test.go
- handlers_test.go
- Handler
- ConnectionParams
- KrknOperatorTargetProviderConfig
- time.Time
- .DeepCopy
- .UpdateRegistry
- NewCachedUserStatusChecker
- run_config_handlers_test.go
- ComponentManifest
- bootstrap.go
- UnifiedJobItem
- wsorigin.go
- GroupLabelKey
- migration_test.go
- notify-operator-docs.test.js
- converter_test.go
- validateInlineDestination
- KrknScenarioRunSpec
- Hub
- InjectCredentials
- .prepareJobResources
- .updateClusterJobStatuses
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
- Handler
- Publisher
- start_operator.sh
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- Agent Specifications - MVP Archive
- Compute
- scenarioRunConfigurationFixture
- KrknOperatorTargetProvider
- buildClusterAPIURLMap
- test-api.sh
- Frontend Authentication Implementation Tasks
- ExecuteKubectlRequest
- mockJob
- KrknScenarioRunReconciler
- registry/types.go
- category_parameter_profile.go
- Krkn-Operator Roadmap
- .loadCategoryResiliencyHistories
- .flatten
- GraphRunResponse
- GraphClusterScore
- ResolveGraph
- buildUnifiedJobList
- SecretManager
- REQUIREMENTS-MVP.md
- ExecuteKubectlResponse
- ProviderRegistration
- Provider Configuration API Specification
- NewServerWithComponentNamespace
- dataprovider.pb.go
- TestKrknGraphRun_ValidateGraph
- Server
- Authentication API Specification
- k8s.io/apimachinery/pkg/runtime.Scheme
- encoding/json.RawMessage
- setup-ocm-and-operator.sh
- File Management API Specification
- JobTracker
- commands.go
- Cluster Segregation Troubleshooting
- channel-head.sh
- Running gRPC Data Provider Locally for Development
- api_workflow_test.sh
- .QueryTelemetryPage
- select-compatible-release.sh
- test-open-kubernetes-operators-pr.sh
- CLAUDE.md - Project Instructions
- post-install.sh
- generate_proto_go.sh
- PROGRESS-MVP.md
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
- .extractAndStoreReports
- krkn-operator Development Progress (Post-MVP)
- rate_limiter.go
- API compatibility notes
- 🔧 TODO: Pod Recreation and Retry Logic
- .checkRollbackImage
- Running Locally
- install-yq.sh
- test-release-workflow-gates.sh
- Deployment
- Available Make Targets
- Building Images
- Troubleshooting
- KrknOperatorTargetProviderConfigStatus
- REST API Endpoints
- Troubleshooting
- .GetScenarioRunLogs
- assets/README.md
- generateKubeconfigFromRequest
- newProvidersHandler
- SanitizeUserIDForLabel
- registry/labels_test.go
- WorkflowResponse
- olm_bootstrap_test.go
- BuildUnifiedJobList
- Technical Notes
- krkngraphrun_controller_sanitization_test.go
- Pushing Images
- providerConfigTestScheme
- file_proto_dataprovider_proto_rawDescGZIP
- extractTarGz
- setupTestHandler
- Post-MVP Development
- dataProviderServiceClient
- DataProviderServiceServer
- manifest_test.go
- visibleCategoryNames
- RollbackAvailability
- migrationBlockedError

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
- `3. Create Provider Config Request` --references--> `KrknOperatorTargetProviderConfig`  [INFERRED]
  docs/api-provider-config-spec.md → api/v1alpha1/krknoperatortargetproviderconfig_types.go
- `Overview` --references--> `KrknOperatorTargetProviderConfig`  [INFERRED]
  docs/api-provider-config-spec.md → api/v1alpha1/krknoperatortargetproviderconfig_types.go

## Import Cycles
- None detected.

## Communities (179 total, 32 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.08
Nodes (18): availableWorkspacePVC, CategoryEntityAssociationResponse, CreateCategoryRequest, ResiliencyHistoryQueryRequest, UpdateCategoryRequest, workspacePublishRequest, ContextKey, clusterTarget (+10 more)

### Community 1 - "testing.T"
Cohesion: 0.01
Nodes (236): TestKrknUserCreation(), TestKrknUserList(), TestKrknUserRoles(), TestKrknUserStatus(), TestKrknUserValidation(), TestScenarioReferenceValidate(), NewConfigStoreInitializer(), TestConfigStoreInitializer_NeedLeaderElection() (+228 more)

### Community 2 - "Handler"
Cohesion: 0.06
Nodes (26): workspaceInfo, writeCategoryRunNotFoundOrInternalError(), updateCloudCredentialSecretData(), convertGraphClusterScores(), buildUserGroupResponse(), extractGroupNameAndUserIDFromPath(), extractGroupNameFromMembersPath(), removeGroupFromAllMembers() (+18 more)

### Community 3 - "InjectCredentials"
Cohesion: 0.18
Nodes (21): TestAppendCloudCredentialInjection_EmptyRefNoop(), TestAppendCloudCredentialInjection_UsesSecretKeyRef(), injectAWS(), injectAzure(), injectBaremetal(), InjectCredentials(), injectGCP(), injectIBMCloud() (+13 more)

### Community 4 - "GetClaimsFromContext"
Cohesion: 0.06
Nodes (44): AvailableFilesResponse, DeleteFileResponse, ListFilesResponse, UpdateFileResponse, buildFileInfo(), buildFileResponse(), deriveLogicalName(), extractLogicalName() (+36 more)

### Community 5 - "context.Context"
Cohesion: 0.05
Nodes (39): historyQueryCountingClient, k8sUserStatusChecker, mockUserStatusChecker, BackupConfig, resourceType, RestoreConfig, removeCategoryLabel(), newK8sUserStatusChecker() (+31 more)

### Community 6 - "NewTestHandler"
Cohesion: 0.07
Nodes (99): countingGetClient, esDoerFunc, TestGetActiveRunsOverview_AllUsersSeesAll(), TestGetActiveRunsOverview_MixedJobStates(), TestGetActiveRunsOverview_NoActiveRuns(), TestGetActiveRunsOverview_UserSeesAll(), boolPtr(), newEsAlertsTestSecretWithHost() (+91 more)

### Community 7 - "NewTestHandler"
Cohesion: 0.08
Nodes (51): TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(), TestCloudCredentialContract_GraphRunStripsNodeEnv(), newAWSTestSecret(), newCloudCredScheme(), newGCPTestSecret(), newGroupedAWSSecret(), TestCloudCredentialsRouter_MethodNotAllowed(), TestCloudCredentialsRouter_SubpathMethodNotAllowed() (+43 more)

### Community 8 - "utils.go"
Cohesion: 0.11
Nodes (21): tokenRequest, attemptDirectory(), Inspect(), Run(), TestInspectRequiresReadableDirectory(), TestInspectRetainedRollbackFiles(), TestE2E(), getMetricsOutput() (+13 more)

### Community 9 - "rollback_handlers_test.go"
Cohesion: 0.21
Nodes (30): rollbackProbeName(), rollbackRunAttemptName(), rollbackRunName(), decodeRollbackResponse(), rollbackAvailableProbe(), rollbackRegistrySecret(), rollbackRequest(), rollbackTestHandler() (+22 more)

### Community 10 - "api/types.go"
Cohesion: 0.05
Nodes (45): ActiveRunsOverviewResponse, AddGroupMemberRequest, AddGroupMemberResponse, ChangePasswordRequest, ChangePasswordResponse, ClusterPermissionSet, ClusterResiliencyScoreResponse, CreateTargetResponse (+37 more)

### Community 11 - "Broadcaster"
Cohesion: 0.18
Nodes (5): hashBytes(), roleFromAdmin(), Broadcaster, graphRunEventHandler, scenarioRunEventHandler

### Community 12 - "migration.go"
Cohesion: 0.13
Nodes (42): clusterJobImage, clusterJobProvider, clusterJobRetries, graphScenarioReferences, migrationPlan, migrationResult, migrationStats, resiliencyScoreStatus (+34 more)

### Community 13 - "addAdminContext"
Cohesion: 0.09
Nodes (52): addAdminContext(), addUserContext(), createTestAdminUser(), setupFilesTestHandler(), TestCanAccessFile(), TestCreateFile(), TestCreateFile_DefaultFilePurpose(), TestCreateFile_DuplicateName() (+44 more)

### Community 14 - "Handler"
Cohesion: 0.14
Nodes (9): cloudCredentialAccessContext, buildCloudCredentialResponse(), buildConnectionParams(), buildConnectionParamsForIndex(), buildElasticsearchConfigResponse(), canAccessElasticsearchConfig(), ExtractGroupsFromLabels(), ExtractGroupsFromLabels() (+1 more)

### Community 15 - "GetClaimsFromContext"
Cohesion: 0.11
Nodes (19): CategoryListResponse, CategoryResponse, ErrorResponse, buildCategoryResponse(), buildCategoryVisibilityLabels(), categoryErrorCode(), categoryGroups(), categoryIsPublic() (+11 more)

### Community 16 - "render-bundle.sh"
Cohesion: 0.07
Nodes (32): channel_has_entries(), ensure_catalog_channel(), catalog-channel.sh script, update_default_channel(), write_release_config(), catalog_submit_checkout(), catalog_submit_open_pr(), catalog-submit.sh script (+24 more)

### Community 17 - "server.py"
Cohesion: 0.06
Nodes (6): add_DataProviderServiceServicer_to_server(), DataProviderService, DataProviderServiceServicer, DataProviderServiceStub, DataProviderServicer, serve()

### Community 18 - "category_run_configuration.go"
Cohesion: 0.17
Nodes (25): scenarioFileBehavior, GraphScenarioNode, appendGraphSignaturePart(), compareCategoryRunConfigurations(), compareGraphConfigurations(), compareGraphConfigurationsWithStableNodeIDs(), compareGraphNodeConfigurations(), compareGraphRunConfigurations() (+17 more)

### Community 19 - "NewHub"
Cohesion: 0.15
Nodes (26): NewHandler(), NewBroadcaster(), TestBroadcastDashboardUpdate(), TestBroadcastGraphRunDeleted(), TestBroadcastGraphRunUpdate(), TestBroadcastOnlyToSubscribedClients(), TestBroadcastScenarioRunDeleted(), TestBroadcastScenarioRunUpdate() (+18 more)

### Community 20 - "WSUnifiedJobItem"
Cohesion: 0.10
Nodes (14): TestFilterJobsByCategoriesMatchesAnySelectedCategory(), TestPaginateJobItems(), computeWSJobStats(), filterJobCategoriesByVisibility(), filterJobsByCategories(), groupRollbackJobs(), paginateJobItems(), TestGroupRollbackJobsWebSocketKeepsFlatViewAndChecksUID() (+6 more)

### Community 21 - "user_handlers_test.go"
Cohesion: 0.10
Nodes (36): capturedLogEntry, captureLogSink, createTestUser(), kvValue(), setupUserTestHandler(), TestChangePassword_AdminChange_NoCurrentPassword(), TestChangePassword_AdminChangesOtherUser_EmitsAuditLog(), TestChangePassword_SamePassword_Rejected() (+28 more)

### Community 22 - "client_test.go"
Cohesion: 0.08
Nodes (34): doerFunc, roundTripperFunc, NewClient(), intPtr(), scenarioParametersSchema(), TestBaseURL(), TestClientTransportDefaultNotHTTPTransport(), TestClientTransportPooling() (+26 more)

### Community 23 - "KrknGraphRun"
Cohesion: 0.16
Nodes (9): KrknGraphRunReconciler, TestGraphRunReconcileSkipsPendingMigration(), TestLoadFileConfigMapByID(), TestTranslateVolumesToFileMounts(), sanitizeNodeID(), TestCalculateResiliencyScore_StoresNodeScores(), overrideRef(), TestGraphRunInvalidImageSignatureFailsWithoutRetry() (+1 more)

### Community 24 - "elasticsearch/types.go"
Cohesion: 0.16
Nodes (18): AffectedPods, CreateElasticsearchConfigResponse, DeleteElasticsearchConfigResponse, ListElasticsearchConfigsResponse, QueryAlertsRequest, QueryTelemetryRequest, RawJSON, RecoveredPod (+10 more)

### Community 25 - "installer.go"
Cohesion: 0.10
Nodes (21): main(), resolveMigrationNamespace(), runCRDMigrationMode(), TestResolveMigrationNamespacePrefersHookNamespace(), TestRunCRDMigrationMode(), configMapClient, crdManifest, deploymentClient (+13 more)

### Community 26 - "newCategoryRequestAs"
Cohesion: 0.17
Nodes (20): failingWorkspaceGetClient, newCategoryRequestAs(), setupScenarioRunTestHandler(), graphRunCategoryTestHandler(), TestCreateGraphRun_CategoriesArePersistedOnCreatedCR(), TestCreateGraphRun_RejectsMissingCategoryBeforeCreatingCR(), TestPostScenarioRun_RejectsInvalidCategoryNameBeforeCreatingCR(), readyWorkspace() (+12 more)

### Community 27 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.10
Nodes (9): ClustersResponse, KrknTargetRequestReconciler, ClusterHealthStatus, ClusterTarget, KrknOperatorTarget, KrknOperatorTargetSpec, KrknTargetRequest, KrknTargetRequestSpec (+1 more)

### Community 29 - "client.go"
Cohesion: 0.16
Nodes (17): facetField, filterAggregation, Option, StatusError, termsAggregation, buildAlertsSearchBody(), buildFacetAggs(), buildFilterClauses() (+9 more)

### Community 30 - "backup_restore_handlers_test.go"
Cohesion: 0.15
Nodes (22): NewJobTracker(), createMultipartBody(), createTestHandler(), TestGetRestoreStatusAdminOnly(), TestGetRestoreStatusTrackedJob(), TestGetRestoreStatusUnknownJob(), TestHandlerShutdownCancelsContext(), TestJobTrackerEviction() (+14 more)

### Community 31 - "handlers_test.go"
Cohesion: 0.05
Nodes (47): TestLogin_InactiveUser(), TestLogin_InvalidCredentials(), TestLogin_Success(), convertInputFields(), maskToken(), sanitizeHeaders(), TestConvertInputFields(), TestDeleteTargetByUUID_AdminCanDeleteAnyResource() (+39 more)

### Community 33 - "ConnectionParams"
Cohesion: 0.24
Nodes (6): Doer, executeSearch(), Client, ConnectionParams, newSearchRequest(), transportKey()

### Community 34 - "KrknOperatorTargetProviderConfig"
Cohesion: 0.10
Nodes (15): KrknOperatorTargetProviderConfigReconciler, Architecture Decision, Components Requiring Leader Election (Leader-Only), Components Running on All Replicas (Multi-Active), isConflictError(), countActiveProviders(), Configuration Functions, CreateProviderConfigRequest (+7 more)

### Community 35 - "time.Time"
Cohesion: 0.14
Nodes (12): ClusterJobStatusResponse, NodeStatusResponse, RestoreResponse, RollbackRunSummary, ScenarioRunStatusResponse, convertClusterJobs(), convertMetaTime(), TestConvertClusterJobs_EmptyInput() (+4 more)

### Community 37 - ".UpdateRegistry"
Cohesion: 0.21
Nodes (5): buildRegistryResponse(), validateCreateRegistryRequest(), validateUpdateRegistryRequest(), BuildRegistryLabels(), ExtractGroupsFromLabels()

### Community 38 - "NewCachedUserStatusChecker"
Cohesion: 0.17
Nodes (10): CachedUserStatusChecker, countingUserStatusChecker, userStatusCacheEntry, UserStatusChecker, NewCachedUserStatusChecker(), TestCachedUserStatusChecker(), TestCachedUserStatusChecker_EvictsExpiredEntries(), TestCachedUserStatusChecker_Invalidation() (+2 more)

### Community 39 - "run_config_handlers_test.go"
Cohesion: 0.12
Nodes (24): contains(), containsMiddle(), publicScenarioReference(), TestCreateGraphRun_WithFileReferences(), TestCreateGraphRun_ResiliencyScore(), runCategoryFixture(), TestPostScenarioRun_CategoriesArePersistedOnCreatedCR(), TestGetGraphRunConfig_NotFound() (+16 more)

### Community 40 - "ComponentManifest"
Cohesion: 0.20
Nodes (4): ListComponentsResponse, ComponentDescription, ComponentManifest, KrknComponentStatus

### Community 41 - "bootstrap.go"
Cohesion: 0.17
Nodes (13): RemoveElasticsearchGrafanaURLAnnotations(), TestRemoveElasticsearchGrafanaURLAnnotations(), ensureJWTSecret(), EnsureResources(), ensureSCCBinding(), ensureScenarioRunnerRBAC(), ensureServiceAccount(), ensureServices() (+5 more)

### Community 42 - "UnifiedJobItem"
Cohesion: 0.12
Nodes (7): PaginationMeta, ScenarioRunListItem, ScenarioRunListResponse, UnifiedJobItem, UnifiedJobsResponse, ComputeJobStats(), TestComputeJobStats()

### Community 43 - "wsorigin.go"
Cohesion: 0.16
Nodes (17): checkWebSocketOriginV2(), TestCheckWebSocketOriginV2(), checkWebSocketOrigin(), TestCheckWebSocketOrigin(), guardedCheckRedirect(), TestGuardedCheckRedirect(), TestValidateDestinationURL(), validateDestinationURL() (+9 more)

### Community 44 - "GroupLabelKey"
Cohesion: 0.19
Nodes (22): newCategoryRequest(), newCategoryTestGroup(), newCategoryTestHandler(), newCategoryTestUser(), TestCategoriesRouterCRUD(), TestCategoriesRouterGroupCreationRequiresMembership(), TestCategoriesRouterGroupVisibility(), TestCategoriesRouterNameMaximumLength() (+14 more)

### Community 45 - "migration_test.go"
Cohesion: 0.19
Nodes (25): Complete(), Stage(), containsSubstring(), countPatchActions(), jsonMarshal(), migrationTestLogger(), newMigrationClient(), scenarioRunFixture() (+17 more)

### Community 46 - "notify-operator-docs.test.js"
Cohesion: 0.10
Nodes (11): assert, chartWorkflow, { execFileSync, spawnSync }, fs, notifier, notifyWorkflow, os, path (+3 more)

### Community 47 - "converter_test.go"
Cohesion: 0.20
Nodes (15): BuildDockerConfigJSON(), ExtractRegistryV2FromSecret(), TestBuildDockerConfigJSON_MissingCredentials(), TestBuildDockerConfigJSON_OAuth(), TestBuildDockerConfigJSON_Password(), TestExtractRegistryV2FromSecret_InvalidType(), TestExtractRegistryV2FromSecret_MissingAnnotations(), TestExtractRegistryV2FromSecret_MissingComponent() (+7 more)

### Community 48 - "validateInlineDestination"
Cohesion: 0.22
Nodes (5): responseWriter, guardedDialContext(), TestValidateDestinationIP(), validateDestinationIP(), validateInlineDestination()

### Community 49 - "KrknScenarioRunSpec"
Cohesion: 0.22
Nodes (9): Configuration, Current Behavior, Implementation Plan (TODO), Option A: Cancellation Field in Status (Recommended), Option B: Finalizers for Cancellation Tracking, Option C: Separate CancellationRequest CR, Proposed Solution Options, Requirements (+1 more)

### Community 50 - "Hub"
Cohesion: 0.23
Nodes (6): isWebSocketDisconnectError(), writeWSError(), BroadcastMessage, Client, Hub, PaginationClientState

### Community 51 - "InjectCredentials"
Cohesion: 0.10
Nodes (25): Publish and select a workspace, Replay and unavailable storage, Reuse from another operator, Scenario workspaces on a shared PVC, Storage validation, Available(), Mount(), Resolve() (+17 more)

### Community 52 - ".prepareJobResources"
Cohesion: 0.14
Nodes (12): preparedJobResources, effectiveScenarioMaxRetries(), getOwnerLabel(), rollbackConfigTemplateCheck(), TestPrepareJobResourcesRejectsUnavailableWorkspaceBeforeCreatingPod(), TestPrepareJobResourcesRequiresWorkspaceForRollback(), TestRollbackConfigTemplateCheck(), TestScenarioPodMountsOnlyAssignedWorkspaceDirectory() (+4 more)

### Community 53 - ".updateClusterJobStatuses"
Cohesion: 0.12
Nodes (12): preStartFailureReason(), TestPreStartFailureReason(), backfillLegacyMaxRetries(), TestBackfillLegacyMaxRetries(), TestFailedAttemptRetainsRollbackIdentityBeforeRetry(), TestShouldRetryJobHonorsConfiguredRetryLimit(), TestUpdateClusterJobStatuses_ClusterNameEmpty(), TestUpdateClusterJobStatuses_PreservesExplicitZero() (+4 more)

### Community 54 - "rate_limiter.go"
Cohesion: 0.28
Nodes (11): newIPRateLimiter(), rateLimitMiddleware(), TestIPRateLimiter_AllowsUnderLimit(), TestIPRateLimiter_BlocksOverLimit(), TestIPRateLimiter_Cleanup(), TestIPRateLimiter_CleanupPreservesFresh(), TestIPRateLimiter_EvictsOldestWhenAtCapacity(), TestIPRateLimiter_SeparateLimitersPerIP() (+3 more)

### Community 55 - "newReportTestHandler"
Cohesion: 0.29
Nodes (10): newReportTestHandler(), reportRequest(), TestDownloadReportDeniesUserWithoutReportClusterAccess(), TestDownloadReportHTMLThroughV2Router(), TestDownloadReportPDF(), TestDownloadReportRejectsMalformedPath(), TestDownloadReportUnavailable(), TestGetReportStatusPending() (+2 more)

### Community 56 - "cloudcreds/types.go"
Cohesion: 0.16
Nodes (15): CreateCloudCredentialResponse, DeleteCloudCredentialResponse, ListCloudCredentialsResponse, UpdateCloudCredentialResponse, buildCloudCredentialSecretData(), CloudCredentialResponse, CreateCloudCredentialRequest, UpdateCloudCredentialRequest (+7 more)

### Community 57 - "GetNodesRequest"
Cohesion: 0.15
Nodes (3): DataProviderServiceClient, GetNodesRequest, GetNodesResponse

### Community 58 - "dataprovider_grpc.pb.go"
Cohesion: 0.17
Nodes (5): UnimplementedDataProviderServiceServer, UnsafeDataProviderServiceServer, _DataProviderService_ExecuteKubectl_Handler(), _DataProviderService_GetNodes_Handler(), RegisterDataProviderServiceServer()

### Community 59 - "elasticsearch/validate.go"
Cohesion: 0.25
Nodes (9): CreateElasticsearchConfigRequest, UpdateElasticsearchConfigRequest, strPtr(), TestValidateUpdateRequest(), validateAccess(), ValidateCreateRequest(), ValidateMergedConnection(), validateTLSSettings() (+1 more)

### Community 60 - "parseResiliencyHistoryQuery"
Cohesion: 0.33
Nodes (5): resiliencyHistoryQuerySelections, normalizeResiliencyHistoryProviders(), normalizeResiliencyHistorySelections(), parseResiliencyHistoryQuery(), extractFile()

### Community 61 - ".DeepCopyInto"
Cohesion: 0.04
Nodes (24): filterUsers(), GraphRunSummary, GraphScenario, KrknCategoryList, KrknCategorySpec, KrknComponent, KrknComponentList, KrknGraphRunList (+16 more)

### Community 62 - "ScenarioRunRequest"
Cohesion: 0.14
Nodes (16): Phase 12: Scenario Job Management - /scenarios/run Endpoints ✅, Phase 9: POST /scenarios Endpoint ✅, FileMount, GlobalsRequest, JobsListResponse, JobStatusResponse, ScenarioRunRequest, ScenarioRunResponse (+8 more)

### Community 63 - "go_pkg_sync"
Cohesion: 0.22
Nodes (3): ConfigStoreInitializer, Store, Get()

### Community 64 - "GET methods"
Cohesion: 0.11
Nodes (18): DELETE methods, GET methods, Implementation details, POST methods, Request example:, Request example:, Response:, Response example: (+10 more)

### Community 65 - "installer_test.go"
Cohesion: 0.18
Nodes (12): crdClient, applyCRD(), applyCRDs(), TestApplyCRDsCreatesMissingAndUpdatesChangedSpecs(), TestApplyCRDsDoesNotPatchUnchangedSpecs(), TestApplyCRDsLabelsExistingCRDsWithoutChangingOtherLabels(), TestApplyCRDsRejectsNonCRDManifest(), testCRD() (+4 more)

### Community 66 - "sync_test.go"
Cohesion: 0.29
Nodes (6): SyncConfigMapToStore(), TestRoundTrip(), TestSyncConfigMapToStore(), TestSyncConfigMapToStore_NilStore(), TestWriteConfigMapData(), WriteConfigMapData()

### Community 67 - "buildContainerImageFromReference"
Cohesion: 0.17
Nodes (15): publicScenarioReference(), TestCreateScenarioRun_ResiliencyScore(), buildContainerImage(), buildContainerImageFromReference(), getTestConfig(), privateReference(), publicReference(), TestBuildContainerImage_PublicRegistryUsesKrknctl() (+7 more)

### Community 68 - "github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2"
Cohesion: 0.09
Nodes (23): POST methods, refactoring, /scenarios ✅ COMPLETED, /scenarios/detail/{scenario_name} ✅ COMPLETED, /scenarios/globals/{scenario_name} ✅ COMPLETED, /scenarios/run/{UUID}, /targets ✅ COMPLETED, mockScenarioProvider (+15 more)

### Community 69 - "Handler"
Cohesion: 0.07
Nodes (24): Handler, TerminalRequest, TerminalResponse, Code References, writeScenarioRunAccessError(), filterScenarioHistoryScoresByPermission(), sanitizeRunNameLabel(), scenarioRunImage() (+16 more)

### Community 70 - "Publisher"
Cohesion: 0.09
Nodes (10): Publisher, KrknWorkspaceReconciler, NewNamespaceFilter(), TestWorkspaceReconcilerMapsPVCWithoutOwningIt(), TestWorkspaceReconcilerTracksPVCState(), TestPublisherReconcile(), TestPublisherReconcileReturnsTransientErrors(), TestPublisherRecoversFromConcurrentCreateAndDelete() (+2 more)

### Community 71 - "start_operator.sh"
Cohesion: 0.41
Nodes (12): build_operator(), check_prerequisites(), cleanup(), create_service_account(), install_crds(), main(), print_error(), print_info() (+4 more)

### Community 72 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.14
Nodes (12): GraphRunDetailResponse, GraphRunListItem, GraphRunListResponse, GraphRunSpecResponse, GraphRunStatusResponse, 1.1 Resiliency weight per graph node, 1. Request - Create GraphRun (POST /api/v1/graphruns), 2. Response - List GraphRuns (GET /api/v1/graphruns) (+4 more)

### Community 73 - "Agent Specifications - MVP Archive"
Cohesion: 0.20
Nodes (7): Agent Specifications - MVP Archive, Files, MVP Completion Date, MVP Features Delivered, Post-MVP Development, create configmap for operator based on KrknOperatorTargetProviderConfig, KrknOperatorTargetProviderConfig

### Community 74 - "Compute"
Cohesion: 0.27
Nodes (4): Compute(), TestCompute(), JobCounter, Summary

### Community 75 - "scenarioRunConfigurationFixture"
Cohesion: 0.30
Nodes (14): TestBuildCategoryResiliencyHistoryGroupsTypedConfigurationsAndSortsPerCluster(), TestGraphHistoryScoresOnlyKeepsFinalScoresIncludingZero(), assertConfigurationDifferences(), floatPointer(), graphRunConfigurationFixture(), scenarioReference(), scenarioRunConfigurationFixture(), stringPointer() (+6 more)

### Community 76 - "KrknOperatorTargetProvider"
Cohesion: 0.07
Nodes (27): Completed, Current Status, In Progress, Pending, 1. List Providers, 2. Get Provider Config Status, 2. Update Provider Status, 3. Create Provider Config Request (+19 more)

### Community 77 - "buildClusterAPIURLMap"
Cohesion: 0.25
Nodes (8): buildClusterAPIURLMap(), contains(), containsInMiddle(), TestBuildClusterAPIURLMap(), TestBuildClusterAPIURLMap_Collision(), TestBuildClusterAPIURLMap_SameNameSameURL(), TestValidateScenarioRunAccess(), TestValidateScenarioRunAccess_NoGroups()

### Community 78 - "test-api.sh"
Cohesion: 0.39
Nodes (11): api_call(), cleanup(), extract_body(), extract_http_code(), log_error(), log_info(), log_step(), log_success() (+3 more)

### Community 79 - "Frontend Authentication Implementation Tasks"
Cohesion: 0.06
Nodes (30): 1. Read the API Documentation, 1. Start the Operator, 2. Check Available Work, 2. Test Registration (First Admin), 3. Start with Foundation Task, 3. Test Login, 401 on Protected Routes, 4. Claim a Task (+22 more)

### Community 82 - "KrknScenarioRunReconciler"
Cohesion: 0.14
Nodes (11): imageResolver, imageSignatureVerifier, KrknScenarioRunReconciler, migrationPending(), TestScenarioRunReconcileSkipsPendingMigration(), countJobPhaseChanges(), timeEqual(), backfillLegacyGraphResiliencyScoreStatuses() (+3 more)

### Community 83 - "registry/types.go"
Cohesion: 0.21
Nodes (11): CreateRegistryRequest, RegistryInfo, RegistryResponse, UpdateRegistryRequest, AuthEntry, AvailableRegistriesResponse, CreateRegistryResponse, DeleteRegistryResponse (+3 more)

### Community 84 - "category_parameter_profile.go"
Cohesion: 0.18
Nodes (9): categoryEnvironmentParameter, categoryParameterProfileFingerprint(), categoryParameterProfileName(), TestCategoryParameterProfileFingerprintForEmptyEnvironment(), TestCategoryParameterProfileFingerprintGraphEnvironmentMultiset(), TestCategoryParameterProfileFingerprintIgnoresRunType(), TestCategoryParameterProfileFingerprintScenarioEnvironment(), TestCategoryParameterProfileNameIsDeterministicAndReadable() (+1 more)

### Community 85 - "Krkn-Operator Roadmap"
Cohesion: 0.33
Nodes (5): Krkn-Operator Roadmap, Ongoing (not release-blocked), Release 1 — 2026-08-18, Release 2 — 2026-10-18, Release 4 — 2027-02-23

### Community 86 - ".loadCategoryResiliencyHistories"
Cohesion: 0.13
Nodes (22): CategoryConfigurationGroup, categoryHistoryRun, categoryHistoryScore, CategoryResiliencyDataPoint, CategoryResiliencyHistoryResponse, ResiliencyHistoryQueryResponse, buildCategoryResiliencyHistory(), categoryBehaviorConfigurationSignature() (+14 more)

### Community 87 - ".flatten"
Cohesion: 0.43
Nodes (4): ClusterMetadata, NodeSummaryInfo, rawTelemetrySource, TestFlattenRedactsParameters()

### Community 88 - "GraphRunResponse"
Cohesion: 0.29
Nodes (3): buildGraphRunResponse(), convertGraphClusterScoresForSnapshot(), GraphRunResponse

### Community 89 - "GraphClusterScore"
Cohesion: 0.22
Nodes (4): convertNodeStatusesWithScores(), convertNodeStatusesWithScores(), GraphClusterScore, NodeStatus

### Community 90 - "ResolveGraph"
Cohesion: 0.10
Nodes (16): FromKrknctlScenarioSet(), publicScenarioReference(), publicScenario(), TestFromKrknctlScenarioSetCreatesPublicReferences(), TestToKrknctlScenarioSetUsesScenarioReference(), ToKrknctlScenarioSet(), KrknScenarioRunSpec, MapScenarioNodeToScenarioRunSpec() (+8 more)

### Community 91 - "buildUnifiedJobList"
Cohesion: 0.17
Nodes (12): TestBuildUnifiedJobList_SkipsGraphRunChildren(), averageResiliencyScore(), buildScenarioRunDetailResponse(), buildScenarioRunResponse(), buildUnifiedJobList(), convertClusterResiliencyScoresWS(), runCategoryNames(), sanitizedClusterJobs() (+4 more)

### Community 92 - "SecretManager"
Cohesion: 0.16
Nodes (10): NewHandler(), maxBodySizeMiddleware(), NewServer(), NewServerWithComponentNamespace(), TestMaxBodySizeMiddleware(), generateRandomSecret(), getSecretKeys(), SecretManager (+2 more)

### Community 93 - "REQUIREMENTS-MVP.md"
Cohesion: 0.18
Nodes (10): API Requirements, /clusters, GET methods, get nodes, get pods, Grpc python service requirement, /nodes, Overview (+2 more)

### Community 95 - "ProviderRegistration"
Cohesion: 0.10
Nodes (18): getTokenDuration(), TestGetTokenDuration(), newResiliencyTestScheme(), newTestClientsetWithLogs(), TestCalculateResiliencyScores(), TestReconcile_ResiliencyScoreFailure_DoesNotBlockTerminalPhase(), API Reference, CreateProviderConfigRequest (+10 more)

### Community 96 - "Provider Configuration API Specification"
Cohesion: 0.08
Nodes (24): 1. List Available Providers (Optional), 2. Create Config Request, 3. Poll for Completion, 4. Parse Schemas and Build UI, 5. Submit Configuration, Base URL, Change Log, Common Error Patterns (+16 more)

### Community 97 - "NewServerWithComponentNamespace"
Cohesion: 0.15
Nodes (8): Claims, Role, loggingMiddleware(), mockTokenGen(), mockTokenGenInvalid(), TokenGenerator, Middleware, NewLazyMiddleware()

### Community 99 - "TestKrknGraphRun_ValidateGraph"
Cohesion: 0.33
Nodes (5): SanitizeNodeIDForKubernetes(), stringPtr(), TestCollisionDetection(), TestKrknGraphRun_ValidateGraph(), TestSanitizeNodeIDForKubernetes()

### Community 101 - "Authentication API Specification"
Cohesion: 0.09
Nodes (22): 1. Check Admin Registration, 1. Initial Load, 2. First Admin Registration, 2. Register User, 3. Login, 3. Login, 4. Authenticated Requests, 5. Role-Based UI (+14 more)

### Community 102 - "k8s.io/apimachinery/pkg/runtime.Scheme"
Cohesion: 0.33
Nodes (8): addKnownTypes(), newTargetRequest(), newTestScheme(), TestGetClusterAPIURL_ClusterNotFound(), TestGetClusterAPIURL_Found(), TestGetClusterAPIURL_ProviderNotFound(), TestGetClusterAPIURL_TargetRequestNotFound(), TestReconcile_FailedJobStatus_IncludesClusterAPIURL()

### Community 103 - "encoding/json.RawMessage"
Cohesion: 0.22
Nodes (9): AlertDocument, esAlertSearchResponse, QueryAlertsResponse, decodeAlerts(), isSensitiveKey(), redactSensitiveParameters(), redactValueWithTracking(), TestIsSensitiveKey() (+1 more)

### Community 104 - "setup-ocm-and-operator.sh"
Cohesion: 0.67
Nodes (5): log_error(), log_info(), log_warn(), setup-ocm-and-operator.sh script, wait_for_condition()

### Community 105 - "File Management API Specification"
Cohesion: 0.09
Nodes (22): Access Control, API Endpoints, ConfigMap Structure, Content Validation, Create a Public Configuration File, Create File, Delete File, Error Responses (+14 more)

### Community 108 - "commands.go"
Cohesion: 0.12
Nodes (19): AvailableCommandsResponse, TestExecuteTerminal_CommandParsing(), TestExecuteTerminal_ValidationLogic(), GetAllowedCommands(), GetBlockedFlags(), CommandSpec, IsCommandAllowed(), IsSubcommandAllowed() (+11 more)

### Community 109 - "Cluster Segregation Troubleshooting"
Cohesion: 0.09
Nodes (21): Access Rules, Cluster Segregation Troubleshooting, Common Issues, Debugging Steps, Example 1: Proper Segregation, Example 2: Multiple Groups, Example 3: Admin User, Expected Behavior Examples (+13 more)

### Community 110 - "channel-head.sh"
Cohesion: 0.50
Nodes (3): OLM_CHANNEL_NAME, OLM_PACKAGE_NAME, channel-head.sh script

### Community 111 - "Running gRPC Data Provider Locally for Development"
Cohesion: 0.12
Nodes (16): 1. Login and Get JWT Token, 2. Create KrknTargetRequest (or use existing UUID), 3. Test Terminal Command, Cleanup, Configure Operator to Use Local gRPC Server, Debug gRPC Server in PyCharm, Development Workflow, Notes (+8 more)

### Community 114 - ".QueryTelemetryPage"
Cohesion: 0.42
Nodes (8): esSearchResponse, FacetOption, QueryTelemetryResponse, TelemetryDocument, TelemetryStats, decodeTelemetry(), facetsFromBuckets(), statsFromBuckets()

### Community 117 - "CLAUDE.md - Project Instructions"
Cohesion: 0.10
Nodes (19): Architecture Principles, Before Committing, Before Writing Code, CLAUDE.md - Project Instructions, Code Organization & Reusability, Code Quality Standards, Communication, Critical Rules (+11 more)

### Community 120 - "PROGRESS-MVP.md"
Cohesion: 0.22
Nodes (8): Architecture Decisions, Data Provider Architecture, Future Work (Not in Current Scope), Integration Points, Namespace Configuration, Next Steps, Requirements Overview, REST API Framework

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

### Community 140 - ".extractAndStoreReports"
Cohesion: 0.22
Nodes (7): extractDelimitedContent(), TestExtractAndStoreReportsRejectsMultiClusterRun(), TestExtractDelimitedContent(), TestTargetClusterCount(), TestValidateReportPayloadSize(), targetClusterCount(), validateReportPayloadSize()

### Community 141 - "krkn-operator Development Progress (Post-MVP)"
Cohesion: 0.22
Nodes (9): API Endpoints Completed, Core Infrastructure, Current Status, Development Phases Template, Future Considerations, Key Technical Achievements, krkn-operator Development Progress (Post-MVP), MVP Completion Summary (+1 more)

### Community 142 - "rate_limiter.go"
Cohesion: 0.20
Nodes (9): ipRateLimiter, rateLimiterEntry, clientIPFromHeaders(), extractClientIP(), hostFromRemoteAddr(), isTrustedProxy(), parseTrustedProxyCIDRs(), TestExtractClientIP() (+1 more)

### Community 143 - "API compatibility notes"
Cohesion: 0.50
Nodes (4): API compatibility notes, Graph node resiliency weights, Log streaming, Retry limits

### Community 144 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.17
Nodes (12): API and operator refactoring to support multiple clusters, API Endpoints Structure (Nested Approach), BREAKING CHANGES, Implementation Details, ✅ Implemented, KrknTargetRequest controller in operator, Moving from direct scenario creation to CRD approach, Overview (+4 more)

### Community 145 - ".checkRollbackImage"
Cohesion: 0.39
Nodes (4): rollbackImageCache, rollbackImageCheck, Rollback artifact persistence, Rollback history in Jobs

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

### Community 154 - "REST API Endpoints"
Cohesion: 0.29
Nodes (7): GET /clusters, GET /nodes, GET /targets/{UUID}, POST /scenarios, POST /scenarios/detail/{scenario_name}, POST /targets, REST API Endpoints

### Community 155 - "Troubleshooting"
Cohesion: 0.40
Nodes (5): Error: "Command not found: kubectl", Error: "failed to connect to gRPC server", Error: "Failed to get kubeconfig", Error: "no module named 'krkn_lib'", Troubleshooting

### Community 156 - ".GetScenarioRunLogs"
Cohesion: 0.32
Nodes (6): filterReportLines(), isReportMarkerEnd(), isReportMarkerStart(), TestFilterReportLines(), TestIsReportMarkerEnd(), TestIsReportMarkerStart()

### Community 158 - "generateKubeconfigFromRequest"
Cohesion: 0.43
Nodes (7): CreateTargetRequest, UpdateTargetRequest, generateKubeconfigFromCredentialsType(), generateKubeconfigFromKubeconfigType(), generateKubeconfigFromRequest(), generateKubeconfigFromTokenType(), GenerateFromCredentials()

### Community 159 - "newProvidersHandler"
Cohesion: 0.40
Nodes (5): newProvider(), newProvidersHandler(), TestListProviders_Empty(), TestListProviders_ScopedToNamespace(), TestUpdateProviderStatus()

### Community 160 - "SanitizeUserIDForLabel"
Cohesion: 0.40
Nodes (6): TestSanitizeUserIDForLabel(), SanitizeUserIDForLabel(), SanitizeUserIDForResourceName(), TestSanitizeUserIDConsistency(), TestSanitizeUserIDForLabel(), TestSanitizeUserIDForResourceName()

### Community 161 - "registry/labels_test.go"
Cohesion: 0.33
Nodes (5): TestBuildRegistryAnnotations(), TestBuildRegistryLabels(), TestExtractGroupsFromLabels(), TestUpdateRegistryAnnotations(), UpdateRegistryAnnotations()

### Community 162 - "WorkflowResponse"
Cohesion: 0.67
Nodes (3): workflowResponsesHaveCategories(), WorkflowResponse, ListWorkflowsResponse

### Community 163 - "olm_bootstrap_test.go"
Cohesion: 0.67
Nodes (3): bootstrapOLMResources(), TestBootstrapOLMResourcesPropagatesClientError(), TestBootstrapOLMResourcesWiring()

### Community 164 - "BuildUnifiedJobList"
Cohesion: 0.18
Nodes (9): convertClusterResiliencyScores(), averageResiliencyScore(), BuildUnifiedJobList(), getDefaultPageSize(), GroupRollbackJobs(), runCategoryNames(), TestGroupRollbackJobs_UsesSourceUIDAndKeepsOrphans(), PaginateSlice() (+1 more)

### Community 165 - "Technical Notes"
Cohesion: 0.40
Nodes (5): Environment Variables, Implementation Files, KrknTargetRequest Structure, Server Configuration, Technical Notes

### Community 166 - "krkngraphrun_controller_sanitization_test.go"
Cohesion: 0.83
Nodes (3): isAlphanumeric(), isValidLabelChar(), TestSanitizeNodeID()

### Community 167 - "Pushing Images"
Cohesion: 0.50
Nodes (4): Custom Push, Push Both Images, Push Individual Images, Pushing Images

### Community 168 - "providerConfigTestScheme"
Cohesion: 0.32
Nodes (8): decodeErrorResponse(), providerConfigTestScheme(), TestGetProviderConfigByUUID_CompletedReturnsConfigData(), TestGetProviderConfigByUUID_InternalErrorReturnsSanitized500(), TestGetProviderConfigByUUID_NotFoundReturns404(), TestGetProviderConfigByUUID_PendingReturnsAccepted(), TestPostProviderConfig_FailureReturnsSanitizedError(), TestPostProviderConfig_ReturnsAcceptedWithUUID()

### Community 170 - "extractTarGz"
Cohesion: 0.40
Nodes (5): createTarGz(), TestCreateAndExtractTarGz(), TestCreateTarGzFlushErrors(), TestExtractTarGzPathTraversal(), extractTarGz()

### Community 171 - "setupTestHandler"
Cohesion: 0.09
Nodes (26): setupTestHandler(), TestCreateTarget_DuplicateClusterName(), TestCreateTarget_MissingRequiredFields(), TestCreateTarget_WithCredentials(), TestCreateTarget_WithKubeconfig(), TestCreateTarget_WithToken(), TestDeleteQuietly_MissingObjectIsNoOp(), TestDeleteQuietly_RunsWithCanceledContext() (+18 more)

### Community 172 - "Post-MVP Development"
Cohesion: 0.40
Nodes (5): Completed, In Progress, Pending, Phase 1: KrknOperatorTarget CRD and CRUD API, Post-MVP Development

### Community 175 - "manifest_test.go"
Cohesion: 0.67
Nodes (3): Load(), TestLoad(), TestLoadRejectsInvalidManifest()

## Knowledge Gaps
- **17 isolated node(s):** `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME`, `ICON_BASE64`, `ICON_MEDIATYPE` (+12 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 731 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **32 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Handler` connect `Handler` to `main.go`, `ConnectionParams`, `Handler`, `GetClaimsFromContext`, `context.Context`, `.UpdateRegistry`, `BuildUnifiedJobList`, `Server`, `github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2`, `bootstrap.go`, `JobTracker`, `Handler`, `GetClaimsFromContext`, `SecretManager`, `.checkRollbackImage`, `.loadCategoryResiliencyHistories`, `.GetScenarioRunLogs`?**
  _High betweenness centrality (0.064) - this node is a cross-community bridge._
- **Are the 160 inferred relationships involving `NewTestHandler()` (e.g. with `setupAuthTestHandler()` and `TestLogin_InactiveUser()`) actually correct?**
  _`NewTestHandler()` has 160 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME` to the rest of the system?**
  _17 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.08412537917087967 - nodes in this community are weakly interconnected._
- **Why does `POST methods` connect `github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2` to `REQUIREMENTS-MVP.md`?**
  _High betweenness centrality (0.026) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.012481719552426622 - nodes in this community are weakly interconnected._
- **Why does `KrknWorkspace` connect `InjectCredentials` to `main.go`, `Handler`, `.DeepCopy`, `context.Context`, `Deployment`, `newCategoryRequestAs`, `k8s.io/apimachinery/pkg/apis/meta/v1.Time`, `.DeepCopyInto`?**
  _High betweenness centrality (0.025) - this node is a cross-community bridge._