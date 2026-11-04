# Graph Report - krkn-operator  (2026-11-04)

## Corpus Check
- 335 files · ~428,150 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 19 file(s) not represented in the graph (top: (none) 13, .jsonl 2, .toml 1)

## Summary
- 3623 nodes · 14593 edges · 164 communities (135 shown, 29 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 1355 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `efcd055c`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- main.go
- testing.T
- Handler
- GroupLabelKey
- GetClaimsFromContext
- context.Context
- NewTestHandler
- createAdminContext
- backup.go
- NewTokenGenerator
- api/types.go
- Broadcaster
- migration.go
- addAdminContext
- handlers_test.go
- KrknCategory
- render-bundle.sh
- server.py
- NewConfigStoreInitializer
- NewHub
- WSUnifiedJobItem
- user_handlers_test.go
- client_test.go
- KrknGraphRun
- elasticsearch/types.go
- InjectCredentials
- installer.go
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- ssrf.go
- client.go
- backup_restore_handlers_test.go
- time.Duration
- Handler
- ConnectionParams
- KrknOperatorTargetProviderConfig
- wsorigin.go
- k8s.io/api/core/v1.Secret
- Middleware
- time.Time
- ComponentManifest
- bootstrap.go
- UnifiedJobItem
- installer_test.go
- parseResiliencyHistoryQuery
- migration_test.go
- notify-operator-docs.test.js
- .Get
- scenarioRunConfigurationFixture
- .listWorkspaces
- Hub
- buildScenarioRunResponse
- k8s.io/apimachinery/pkg/runtime.Scheme
- ScenarioReference
- k8s.io/apimachinery/pkg/apis/meta/v1/unstructured.Unstructured
- github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2
- cloudcreds/types.go
- GetNodesRequest
- dataprovider_grpc.pb.go
- GetClaimsFromContext
- converter_test.go
- .DeepCopyInto
- ScenarioRunRequest
- category_run_configuration.go
- CategoryResiliencyHistoryResponse
- Middleware
- 🔧 TODO: Pod Recreation and Retry Logic
- metadata_presence_test.go
- rate_limiter_test.go
- NewServerWithComponentNamespace
- k8s.io/apimachinery/pkg/runtime.Scheme
- start_operator.sh
- .getClusterAPIURL
- Agent Specifications - MVP Archive
- Compute
- KrknScenarioRunSpec
- KrknOperatorTargetProvider
- registry/types.go
- test-api.sh
- Frontend Authentication Implementation Tasks
- ExecuteKubectlRequest
- mockJob
- ValidateCreateRequest
- captureLogSink
- GraphClusterScore
- Krkn-Operator Roadmap
- newCategoryRequestAs
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- newReportTestHandler
- GET methods
- ResolveGraph
- KrknUserGroupSpec
- validateInlineDestination
- watcher_test.go
- ExecuteKubectlResponse
- ProviderRegistration
- Provider Configuration API Specification
- newProvidersHandler
- dataprovider.pb.go
- TestKrknGraphRun_ValidateGraph
- makeScenarioRun
- Authentication API Specification
- PROGRESS-MVP.md
- scenario_filter_test.go
- setup-ocm-and-operator.sh
- File Management API Specification
- krkn-operator Development Progress (Post-MVP)
- commands.go
- Cluster Segregation Troubleshooting
- channel-head.sh
- Running gRPC Data Provider Locally for Development
- REST API Endpoints
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
- API Changes
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
- .checkRollbackImage
- JobTracker
- KrknScenarioRun
- elasticsearch/labels_test.go
- 🔧 TODO: Pod Recreation and Retry Logic
- Technical Notes
- Running Locally
- install-yq.sh
- test-release-workflow-gates.sh
- Deployment
- Available Make Targets
- Building Images
- Troubleshooting
- Setup
- .loadCategoryResiliencyHistories
- Post-MVP Development
- .GetScenarioRunLogs
- assets/README.md
- KrknCategorySpec
- dataProviderServiceClient
- Scenario workspaces on a shared PVC
- files/validation_test.go
- DataProviderServiceServer
- generateKubeconfigFromRequest

## God Nodes (most connected - your core abstractions)
1. `Handler` - 210 edges
2. `NewTestHandler()` - 166 edges
3. `writeJSONError()` - 133 edges
4. `writeJSON()` - 100 edges
5. `createAdminContext()` - 91 edges
6. `GetClaimsFromContext()` - 64 edges
7. `KrknScenarioRun` - 59 edges
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

## Communities (164 total, 29 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.10
Nodes (15): availableWorkspacePVC, CategoryEntityAssociationResponse, CreateCategoryRequest, ResiliencyHistoryQueryRequest, scenarioFileBehavior, UpdateCategoryRequest, workspacePublishRequest, ContextKey (+7 more)

### Community 1 - "testing.T"
Cohesion: 0.01
Nodes (227): TestKrknUserCreation(), TestKrknUserList(), TestKrknUserRoles(), TestKrknUserStatus(), TestKrknUserValidation(), TestScenarioReferenceValidate(), bootstrapOLMResources(), NewConfigStoreInitializer() (+219 more)

### Community 2 - "Handler"
Cohesion: 0.06
Nodes (31): ErrorResponse, Handler, writeCategoryRunNotFoundOrInternalError(), buildCloudCredentialResponse(), updateCloudCredentialSecretData(), convertClusterResiliencyScores(), convertGraphClusterScores(), averageResiliencyScore() (+23 more)

### Community 3 - "GroupLabelKey"
Cohesion: 0.11
Nodes (13): buildUserGroupResponse(), extractGroupNameAndUserIDFromPath(), extractGroupNameFromMembersPath(), removeGroupFromAllMembers(), validateClusterPermissions(), sanitizeResourceName(), sanitizeUsername(), ValidateEmail() (+5 more)

### Community 4 - "GetClaimsFromContext"
Cohesion: 0.05
Nodes (44): AvailableFilesResponse, DeleteFileResponse, ListFilesResponse, UpdateFileResponse, buildFileInfo(), buildFileResponse(), deriveLogicalName(), extractLogicalName() (+36 more)

### Community 5 - "context.Context"
Cohesion: 0.06
Nodes (36): historyQueryCountingClient, k8sUserStatusChecker, mockUserStatusChecker, BackupConfig, resourceType, RestoreConfig, KrknWorkspaceReconciler, removeCategoryLabel() (+28 more)

### Community 6 - "NewTestHandler"
Cohesion: 0.08
Nodes (89): esDoerFunc, TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(), TestCloudCredentialContract_GraphRunStripsNodeEnv(), boolPtr(), newEsAlertsTestSecretWithHost(), newEsScheme(), newEsTestSecret(), newEsTestSecretWithHost() (+81 more)

### Community 7 - "createAdminContext"
Cohesion: 0.12
Nodes (36): newAWSTestSecret(), newCloudCredScheme(), newGCPTestSecret(), newGroupedAWSSecret(), TestCloudCredentialsRouter_MethodNotAllowed(), TestCloudCredentialsRouter_SubpathMethodNotAllowed(), TestCreateCloudCredential_AWS_Success(), TestCreateCloudCredential_DuplicateName() (+28 more)

### Community 8 - "backup.go"
Cohesion: 0.13
Nodes (16): tokenRequest, TestE2E(), getMetricsOutput(), GetNonEmptyLines(), GetProjectDir(), InstallCertManager(), InstallPrometheusOperator(), IsCertManagerCRDsInstalled() (+8 more)

### Community 9 - "NewTokenGenerator"
Cohesion: 0.06
Nodes (37): Claims, TestGroupNameValidation(), TestGroupsRouterEdgeCases(), TestCheckScenarioRunAccess(), TestFilterScenarioRunsByGroupPermission(), TestPostScenarioRun_ClusterNameCollision_Returns500(), TestPostScenarioRun_SameClusterSameURL_NoCollision(), TestPostScenarioRunSetsOwner() (+29 more)

### Community 10 - "api/types.go"
Cohesion: 0.05
Nodes (44): ActiveRunsOverviewResponse, AddGroupMemberRequest, AddGroupMemberResponse, ChangePasswordRequest, ChangePasswordResponse, ClusterPermissionSet, ClusterResiliencyScoreResponse, CreateTargetResponse (+36 more)

### Community 11 - "Broadcaster"
Cohesion: 0.19
Nodes (4): hashBytes(), Broadcaster, graphRunEventHandler, scenarioRunEventHandler

### Community 12 - "migration.go"
Cohesion: 0.15
Nodes (35): clusterJobImage, clusterJobProvider, clusterJobRetries, graphScenarioReferences, migrationBlockedError, migrationPlan, resiliencyScoreStatus, addProvider() (+27 more)

### Community 13 - "addAdminContext"
Cohesion: 0.11
Nodes (43): addAdminContext(), addUserContext(), createTestAdminUser(), setupFilesTestHandler(), TestCanAccessFile(), TestCreateFile(), TestCreateFile_DefaultFilePurpose(), TestCreateFile_DuplicateName() (+35 more)

### Community 14 - "handlers_test.go"
Cohesion: 0.04
Nodes (68): TestLogin_InactiveUser(), TestLogin_InvalidCredentials(), TestLogin_Success(), contains(), containsMiddle(), publicScenarioReference(), TestCreateGraphRun_WithFileReferences(), TestCreateGraphRun_ResiliencyScore() (+60 more)

### Community 15 - "KrknCategory"
Cohesion: 0.13
Nodes (16): CategoryListResponse, CategoryResponse, buildCategoryResponse(), buildCategoryVisibilityLabels(), categoryErrorCode(), categoryGroups(), categoryIsPublic(), categoryVisibleToUser() (+8 more)

### Community 16 - "render-bundle.sh"
Cohesion: 0.07
Nodes (32): channel_has_entries(), ensure_catalog_channel(), catalog-channel.sh script, update_default_channel(), write_release_config(), catalog_submit_checkout(), catalog_submit_open_pr(), catalog-submit.sh script (+24 more)

### Community 17 - "server.py"
Cohesion: 0.06
Nodes (6): add_DataProviderServiceServicer_to_server(), DataProviderService, DataProviderServiceServicer, DataProviderServiceStub, DataProviderServicer, serve()

### Community 18 - "NewConfigStoreInitializer"
Cohesion: 0.16
Nodes (10): ConfigStoreInitializer, SyncConfigMapToStore(), TestRoundTrip(), TestSyncConfigMapToStore(), TestSyncConfigMapToStore_NilStore(), TestWriteConfigMapData(), WriteConfigMapData(), BackfillLegacyProviderConfigLabel() (+2 more)

### Community 19 - "NewHub"
Cohesion: 0.10
Nodes (33): NewHandler(), NewBroadcaster(), TestBroadcastDashboardUpdate(), TestBroadcastGraphRunDeleted(), TestBroadcastGraphRunUpdate(), TestBroadcastOnlyToSubscribedClients(), TestBroadcastScenarioRunDeleted(), TestBroadcastScenarioRunUpdate() (+25 more)

### Community 20 - "WSUnifiedJobItem"
Cohesion: 0.08
Nodes (18): TestFilterJobsByCategoriesMatchesAnySelectedCategory(), TestPaginateJobItems(), computeWSJobStats(), filterJobsByCategories(), paginateJobItems(), ClientMessage, ClusterResiliencyScoreResponse, ErrorMessage (+10 more)

### Community 21 - "user_handlers_test.go"
Cohesion: 0.14
Nodes (34): createTestUser(), kvValue(), setupUserTestHandler(), TestChangePassword_AdminChange_NoCurrentPassword(), TestChangePassword_AdminChangesOtherUser_EmitsAuditLog(), TestChangePassword_SamePassword_Rejected(), TestChangePassword_SelfChange_NoAdminAuditLog(), TestChangePassword_SelfChange_RequiresCurrentPassword() (+26 more)

### Community 22 - "client_test.go"
Cohesion: 0.11
Nodes (24): doerFunc, Option, roundTripperFunc, NewClient(), scenarioParametersSchema(), TestBaseURL(), TestClientTransportDefaultNotHTTPTransport(), TestClientTransportPooling() (+16 more)

### Community 23 - "KrknGraphRun"
Cohesion: 0.13
Nodes (12): imageResolver, imageSignatureVerifier, KrknGraphRunReconciler, TestGraphRunReconcileSkipsPendingMigration(), isAlphanumeric(), isValidLabelChar(), TestSanitizeNodeID(), sanitizeNodeID() (+4 more)

### Community 24 - "elasticsearch/types.go"
Cohesion: 0.13
Nodes (26): AffectedPods, AlertDocument, ClusterMetadata, CreateElasticsearchConfigResponse, DeleteElasticsearchConfigResponse, FacetOption, ListElasticsearchConfigsResponse, NodeSummaryInfo (+18 more)

### Community 25 - "InjectCredentials"
Cohesion: 0.17
Nodes (22): preparedJobResources, TestAppendCloudCredentialInjection_EmptyRefNoop(), TestAppendCloudCredentialInjection_UsesSecretKeyRef(), injectAWS(), injectAzure(), injectBaremetal(), InjectCredentials(), injectGCP() (+14 more)

### Community 26 - "installer.go"
Cohesion: 0.08
Nodes (23): main(), resolveMigrationNamespace(), runCRDMigrationMode(), TestResolveMigrationNamespacePrefersHookNamespace(), TestRunCRDMigrationMode(), configMapClient, crdClient, crdManifest (+15 more)

### Community 27 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.10
Nodes (9): ClustersResponse, KrknTargetRequestReconciler, ClusterHealthStatus, ClusterTarget, KrknOperatorTarget, KrknOperatorTargetSpec, KrknTargetRequest, KrknTargetRequestSpec (+1 more)

### Community 28 - "ssrf.go"
Cohesion: 0.14
Nodes (14): RemoveElasticsearchGrafanaURLAnnotations(), TestRemoveElasticsearchGrafanaURLAnnotations(), TestFlattenRedactsParameters(), TestIsSensitiveKey(), TestRedactSensitiveParameters(), guardedCheckRedirect(), guardedDialContext(), TestGuardedCheckRedirect() (+6 more)

### Community 29 - "client.go"
Cohesion: 0.12
Nodes (25): esAlertSearchResponse, esSearchResponse, facetField, filterAggregation, StatusError, termsAggregation, buildAlertsSearchBody(), buildFacetAggs() (+17 more)

### Community 30 - "backup_restore_handlers_test.go"
Cohesion: 0.15
Nodes (22): NewJobTracker(), createMultipartBody(), createTestHandler(), TestGetRestoreStatusAdminOnly(), TestGetRestoreStatusTrackedJob(), TestGetRestoreStatusUnknownJob(), TestHandlerShutdownCancelsContext(), TestJobTrackerEviction() (+14 more)

### Community 31 - "time.Duration"
Cohesion: 0.16
Nodes (8): ipRateLimiter, rateLimiterEntry, getTokenDuration(), TestGetTokenDuration(), newResiliencyTestScheme(), newTestClientsetWithLogs(), TestCalculateResiliencyScores(), TestReconcile_ResiliencyScoreFailure_DoesNotBlockTerminalPhase()

### Community 32 - "Handler"
Cohesion: 0.17
Nodes (8): roleFromAdmin(), buildGraphRunResponse(), convertGraphClusterScoresForSnapshot(), filterJobCategoriesByVisibility(), visibleCategoryNames(), AuthorizationChecker, categoryVisibilityChecker, Handler

### Community 33 - "ConnectionParams"
Cohesion: 0.23
Nodes (7): Doer, executeSearch(), Client, ConnectionParams, newSearchRequest(), TestTransportKey(), transportKey()

### Community 34 - "KrknOperatorTargetProviderConfig"
Cohesion: 0.08
Nodes (21): KrknOperatorTargetProviderConfigReconciler, 1. List Providers, 2. Get Provider Config Status, 2. Update Provider Status, 3. Create Provider Config Request, 3. Update Provider Configuration, API Endpoints, Architecture Decision (+13 more)

### Community 35 - "wsorigin.go"
Cohesion: 0.22
Nodes (13): checkWebSocketOriginV2(), TestCheckWebSocketOriginV2(), checkWebSocketOrigin(), TestCheckWebSocketOrigin(), canonicalOrigin(), defaultPort(), IsAllowedOrigin(), portsMatch() (+5 more)

### Community 37 - "k8s.io/api/core/v1.Secret"
Cohesion: 0.13
Nodes (9): cloudCredentialAccessContext, buildConnectionParams(), buildConnectionParamsForIndex(), buildElasticsearchConfigResponse(), canAccessElasticsearchConfig(), buildRegistryInfo(), ExtractGroupsFromLabels(), SanitizeGroupName() (+1 more)

### Community 38 - "Middleware"
Cohesion: 0.10
Nodes (12): CachedUserStatusChecker, countingUserStatusChecker, userStatusCacheEntry, UserStatusChecker, Store, NewCachedUserStatusChecker(), TestCachedUserStatusChecker(), TestCachedUserStatusChecker_EvictsExpiredEntries() (+4 more)

### Community 39 - "time.Time"
Cohesion: 0.14
Nodes (12): ClusterJobStatusResponse, NodeStatusResponse, RestoreResponse, ScenarioRunListItem, ScenarioRunStatusResponse, convertClusterJobs(), convertMetaTime(), TestConvertClusterJobs_EmptyInput() (+4 more)

### Community 40 - "ComponentManifest"
Cohesion: 0.15
Nodes (7): ListComponentsResponse, Load(), TestLoad(), TestLoadRejectsInvalidManifest(), ComponentDescription, ComponentManifest, KrknComponentStatus

### Community 41 - "bootstrap.go"
Cohesion: 0.20
Nodes (11): ensureJWTSecret(), EnsureResources(), ensureSCCBinding(), ensureScenarioRunnerRBAC(), ensureServiceAccount(), ensureServices(), intstrFromInt(), namespaceHash() (+3 more)

### Community 42 - "UnifiedJobItem"
Cohesion: 0.12
Nodes (6): PaginationMeta, ScenarioRunListResponse, UnifiedJobItem, UnifiedJobsResponse, ComputeJobStats(), TestComputeJobStats()

### Community 43 - "installer_test.go"
Cohesion: 0.32
Nodes (10): applyCRDs(), TestApplyCRDsCreatesMissingAndUpdatesChangedSpecs(), TestApplyCRDsDoesNotPatchUnchangedSpecs(), TestApplyCRDsLabelsExistingCRDsWithoutChangingOtherLabels(), TestApplyCRDsRejectsNonCRDManifest(), testCRD(), TestCreateMissingCRDsBeforeRolloutLeavesExistingSpecsUntouched(), TestWaitForEstablished() (+2 more)

### Community 44 - "parseResiliencyHistoryQuery"
Cohesion: 0.33
Nodes (5): resiliencyHistoryQuerySelections, normalizeResiliencyHistoryProviders(), normalizeResiliencyHistorySelections(), parseResiliencyHistoryQuery(), extractFile()

### Community 45 - "migration_test.go"
Cohesion: 0.20
Nodes (25): Complete(), Stage(), containsSubstring(), countPatchActions(), jsonMarshal(), migrationTestLogger(), newMigrationClient(), scenarioRunFixture() (+17 more)

### Community 46 - "notify-operator-docs.test.js"
Cohesion: 0.10
Nodes (11): assert, chartWorkflow, { execFileSync, spawnSync }, fs, notifier, notifyWorkflow, os, path (+3 more)

### Community 47 - ".Get"
Cohesion: 0.22
Nodes (13): countingGetClient, TestConcurrentAccess(), TestConcurrentSnapshot(), TestDelete(), TestDelete_NonExistent(), TestEmptyValue(), TestExists(), TestGet_Singleton() (+5 more)

### Community 48 - "scenarioRunConfigurationFixture"
Cohesion: 0.13
Nodes (23): categoryEnvironmentParameter, categoryParameterProfileFingerprint(), categoryParameterProfileName(), TestCategoryParameterProfileFingerprintForEmptyEnvironment(), TestCategoryParameterProfileFingerprintGraphEnvironmentMultiset(), TestCategoryParameterProfileFingerprintIgnoresRunType(), TestCategoryParameterProfileFingerprintScenarioEnvironment(), TestCategoryParameterProfileNameIsDeterministicAndReadable() (+15 more)

### Community 49 - ".listWorkspaces"
Cohesion: 0.38
Nodes (4): workspaceInfo, workspaceAvailable(), workspaceInfoFromCR(), WorkspaceReference

### Community 50 - "Hub"
Cohesion: 0.27
Nodes (5): AuthorizationCheckFunc, BroadcastMessage, Client, Hub, PaginationClientState

### Community 51 - "buildScenarioRunResponse"
Cohesion: 0.17
Nodes (11): TestBuildUnifiedJobList_SkipsGraphRunChildren(), averageResiliencyScore(), buildScenarioRunDetailResponse(), buildScenarioRunResponse(), buildUnifiedJobList(), convertClusterResiliencyScoresWS(), runCategoryNames(), sanitizedClusterJobs() (+3 more)

### Community 52 - "k8s.io/apimachinery/pkg/runtime.Scheme"
Cohesion: 0.32
Nodes (8): decodeErrorResponse(), providerConfigTestScheme(), TestGetProviderConfigByUUID_CompletedReturnsConfigData(), TestGetProviderConfigByUUID_InternalErrorReturnsSanitized500(), TestGetProviderConfigByUUID_NotFoundReturns404(), TestGetProviderConfigByUUID_PendingReturnsAccepted(), TestPostProviderConfig_FailureReturnsSanitizedError(), TestPostProviderConfig_ReturnsAcceptedWithUUID()

### Community 53 - "ScenarioReference"
Cohesion: 0.17
Nodes (15): publicScenarioReference(), TestCreateScenarioRun_ResiliencyScore(), buildContainerImage(), buildContainerImageFromReference(), getTestConfig(), privateReference(), publicReference(), TestBuildContainerImage_PublicRegistryUsesKrknctl() (+7 more)

### Community 54 - "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured.Unstructured"
Cohesion: 0.17
Nodes (13): migrationResult, migrationStats, resourceDefinition, checkAnnotationSize(), forEachResource(), logSummary(), patchMigrationResourceWithRetry(), writeMigrationAnnotation() (+5 more)

### Community 55 - "github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2"
Cohesion: 0.10
Nodes (19): POST methods, refactoring, /scenarios ✅ COMPLETED, /scenarios/detail/{scenario_name} ✅ COMPLETED, /scenarios/globals/{scenario_name} ✅ COMPLETED, /scenarios/run/{UUID}, /targets ✅ COMPLETED, mockScenarioProvider (+11 more)

### Community 56 - "cloudcreds/types.go"
Cohesion: 0.14
Nodes (15): CreateCloudCredentialResponse, DeleteCloudCredentialResponse, ListCloudCredentialsResponse, UpdateCloudCredentialResponse, buildCloudCredentialSecretData(), CloudCredentialResponse, CreateCloudCredentialRequest, UpdateCloudCredentialRequest (+7 more)

### Community 57 - "GetNodesRequest"
Cohesion: 0.15
Nodes (3): DataProviderServiceClient, GetNodesRequest, GetNodesResponse

### Community 58 - "dataprovider_grpc.pb.go"
Cohesion: 0.16
Nodes (5): UnimplementedDataProviderServiceServer, UnsafeDataProviderServiceServer, _DataProviderService_ExecuteKubectl_Handler(), _DataProviderService_GetNodes_Handler(), RegisterDataProviderServiceServer()

### Community 59 - "GetClaimsFromContext"
Cohesion: 0.09
Nodes (21): TerminalRequest, TerminalResponse, Code References, writeScenarioRunAccessError(), filterScenarioHistoryScoresByPermission(), sanitizeRunNameLabel(), resolveMaxRetries(), GetClaimsFromContext() (+13 more)

### Community 60 - "converter_test.go"
Cohesion: 0.20
Nodes (15): BuildDockerConfigJSON(), ExtractRegistryV2FromSecret(), TestBuildDockerConfigJSON_MissingCredentials(), TestBuildDockerConfigJSON_OAuth(), TestBuildDockerConfigJSON_Password(), TestExtractRegistryV2FromSecret_InvalidType(), TestExtractRegistryV2FromSecret_MissingAnnotations(), TestExtractRegistryV2FromSecret_MissingComponent() (+7 more)

### Community 61 - ".DeepCopyInto"
Cohesion: 0.05
Nodes (20): filterUsers(), CanAccess(), TestCanAccess(), GraphScenario, KrknCategoryList, KrknComponent, KrknComponentList, KrknGraphRunList (+12 more)

### Community 62 - "ScenarioRunRequest"
Cohesion: 0.16
Nodes (15): Phase 12: Scenario Job Management - /scenarios/run Endpoints ✅, Phase 9: POST /scenarios Endpoint ✅, FileMount, GlobalsRequest, JobsListResponse, JobStatusResponse, ScenarioRunRequest, ScenarioRunResponse (+7 more)

### Community 63 - "category_run_configuration.go"
Cohesion: 0.12
Nodes (22): GraphRunCreateRequest, GraphScenarioNode, appendGraphSignaturePart(), compareGraphConfigurations(), compareGraphConfigurationsWithStableNodeIDs(), compareGraphNodeConfigurations(), compareGraphRunConfigurations(), graphConfigurationSignature() (+14 more)

### Community 64 - "CategoryResiliencyHistoryResponse"
Cohesion: 0.38
Nodes (7): CategoryConfigurationGroup, CategoryResiliencyDataPoint, CategoryResiliencyHistoryResponse, ResiliencyHistoryQueryResponse, datapointForRunID(), datapointRunIDs(), assembleResiliencyHistoryQueryResponse()

### Community 65 - "Middleware"
Cohesion: 0.18
Nodes (9): Role, loggingMiddleware(), maxBodySizeMiddleware(), TestMaxBodySizeMiddleware(), Middleware, NewLazyMiddleware(), handlerOrStatus(), TestCheckClusterLiveness() (+1 more)

### Community 66 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.22
Nodes (9): Configuration, Current Behavior, Implementation Plan (TODO), Option A: Cancellation Field in Status (Recommended), Option B: Finalizers for Cancellation Tracking, Option C: Separate CancellationRequest CR, Proposed Solution Options, Requirements (+1 more)

### Community 67 - "metadata_presence_test.go"
Cohesion: 0.26
Nodes (9): rawTelemetrySource, intPtr(), TestFlattenRepresentativeScenario(), TestRawTelemetrySourceFlatten(), contains(), containsAt(), TestMetadataJSONMarshaling(), TestMetadataPresenceTracking() (+1 more)

### Community 68 - "rate_limiter_test.go"
Cohesion: 0.17
Nodes (18): clientIPFromHeaders(), extractClientIP(), hostFromRemoteAddr(), isTrustedProxy(), newIPRateLimiter(), parseTrustedProxyCIDRs(), rateLimitMiddleware(), TestExtractClientIP() (+10 more)

### Community 69 - "NewServerWithComponentNamespace"
Cohesion: 0.11
Nodes (10): Server, NewHandler(), NewServer(), NewServerWithComponentNamespace(), Handler, generateRandomSecret(), getSecretKeys(), SecretManager (+2 more)

### Community 70 - "k8s.io/apimachinery/pkg/runtime.Scheme"
Cohesion: 0.10
Nodes (8): addKnownTypes(), Publisher, NewNamespaceFilter(), TestPublisherReconcile(), TestPublisherReconcileReturnsTransientErrors(), TestPublisherRecoversFromConcurrentCreateAndDelete(), TestPublisherStartRetriesAndReturnsCancellation(), ProviderConfigContributorReconciler

### Community 71 - "start_operator.sh"
Cohesion: 0.41
Nodes (12): build_operator(), check_prerequisites(), cleanup(), create_service_account(), install_crds(), main(), print_error(), print_info() (+4 more)

### Community 72 - ".getClusterAPIURL"
Cohesion: 0.46
Nodes (7): newTargetRequest(), newTestScheme(), TestGetClusterAPIURL_ClusterNotFound(), TestGetClusterAPIURL_Found(), TestGetClusterAPIURL_ProviderNotFound(), TestGetClusterAPIURL_TargetRequestNotFound(), TestReconcile_FailedJobStatus_IncludesClusterAPIURL()

### Community 73 - "Agent Specifications - MVP Archive"
Cohesion: 0.20
Nodes (7): Agent Specifications - MVP Archive, Files, MVP Completion Date, MVP Features Delivered, Post-MVP Development, create configmap for operator based on KrknOperatorTargetProviderConfig, KrknOperatorTargetProviderConfig

### Community 74 - "Compute"
Cohesion: 0.30
Nodes (3): Compute(), JobCounter, Summary

### Community 75 - "KrknScenarioRunSpec"
Cohesion: 0.18
Nodes (8): KrknScenarioRunSpec, compareScenarioRunConfigurations(), compareStringMaps(), optionalBoolValue(), sameScenarioFiles(), sameScenarioReference(), FileMount, RollbackSource

### Community 76 - "KrknOperatorTargetProvider"
Cohesion: 0.09
Nodes (20): Completed, Current Status, In Progress, Pending, Basic Usage (Default Configuration), CleanupOldResources, Config Struct, Configuration Options (+12 more)

### Community 77 - "registry/types.go"
Cohesion: 0.21
Nodes (11): CreateRegistryRequest, RegistryInfo, RegistryResponse, UpdateRegistryRequest, AuthEntry, AvailableRegistriesResponse, CreateRegistryResponse, DeleteRegistryResponse (+3 more)

### Community 78 - "test-api.sh"
Cohesion: 0.39
Nodes (11): api_call(), cleanup(), extract_body(), extract_http_code(), log_error(), log_info(), log_step(), log_success() (+3 more)

### Community 79 - "Frontend Authentication Implementation Tasks"
Cohesion: 0.06
Nodes (30): 1. Read the API Documentation, 1. Start the Operator, 2. Check Available Work, 2. Test Registration (First Admin), 3. Start with Foundation Task, 3. Test Login, 401 on Protected Routes, 4. Claim a Task (+22 more)

### Community 82 - "ValidateCreateRequest"
Cohesion: 0.22
Nodes (10): CreateElasticsearchConfigRequest, UpdateElasticsearchConfigRequest, strPtr(), TestValidateCreateRequest(), TestValidateUpdateRequest(), validateAccess(), ValidateCreateRequest(), ValidateMergedConnection() (+2 more)

### Community 84 - "GraphClusterScore"
Cohesion: 0.10
Nodes (8): graphHistoryRun(), convertNodeStatusesWithScores(), convertNodeStatusesWithScores(), GraphClusterScore, GraphRunSummary, KrknGraphRunStatus, KrknWorkspaceStatus, NodeStatus

### Community 85 - "Krkn-Operator Roadmap"
Cohesion: 0.29
Nodes (6): Krkn-Operator Roadmap, Ongoing (not release-blocked), Release 1 — 2026-08-18, Release 2 — 2026-10-18, Release 3 — 2026-12-22, Release 4 — 2027-02-23

### Community 86 - "newCategoryRequestAs"
Cohesion: 0.08
Nodes (54): RollbackAvailability, newCategoryRequest(), newCategoryRequestAs(), newCategoryTestGroup(), newCategoryTestHandler(), newCategoryTestUser(), TestCategoriesRouterCRUD(), TestCategoriesRouterGroupCreationRequiresMembership() (+46 more)

### Community 87 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.09
Nodes (10): GraphRunListItem, GraphRunListResponse, GraphRunStatusResponse, getProviderNames(), timeEqual(), KrknOperatorTargetProviderConfigStatus, KrknOperatorTargetProviderStatus, KrknOperatorTargetStatus (+2 more)

### Community 88 - "newReportTestHandler"
Cohesion: 0.29
Nodes (10): newReportTestHandler(), reportRequest(), TestDownloadReportDeniesUserWithoutReportClusterAccess(), TestDownloadReportHTMLThroughV2Router(), TestDownloadReportPDF(), TestDownloadReportRejectsMalformedPath(), TestDownloadReportUnavailable(), TestGetReportStatusPending() (+2 more)

### Community 89 - "GET methods"
Cohesion: 0.07
Nodes (28): API Requirements, /clusters, DELETE methods, GET methods, GET methods, get nodes, get pods, Grpc python service requirement (+20 more)

### Community 90 - "ResolveGraph"
Cohesion: 0.15
Nodes (13): FromKrknctlScenarioSet(), publicScenarioReference(), publicScenario(), TestFromKrknctlScenarioSetCreatesPublicReferences(), TestToKrknctlScenarioSetUsesScenarioReference(), ToKrknctlScenarioSet(), MapScenarioNodeToScenarioRunSpec(), ResolveGraph() (+5 more)

### Community 93 - "watcher_test.go"
Cohesion: 0.19
Nodes (18): hasGraphRunStatusChanged(), hasScenarioRunStatusChanged(), TestHasGraphRunStatusChanged_NewNodeAdded(), TestHasGraphRunStatusChanged_NoChange(), TestHasGraphRunStatusChanged_NodeOrderChanged(), TestHasGraphRunStatusChanged_NodeStatusChange(), TestHasGraphRunStatusChanged_PhaseChange(), TestHasGraphRunStatusChanged_ResiliencyScoresChange() (+10 more)

### Community 95 - "ProviderRegistration"
Cohesion: 0.15
Nodes (12): API Reference, CreateProviderConfigRequest, Interfaces, NewProviderRegistration, NewProviderRegistrationWithConfig, Provider Configuration Functions, Provider Registration Functions, UpdateProviderConfig (+4 more)

### Community 96 - "Provider Configuration API Specification"
Cohesion: 0.08
Nodes (24): 1. List Available Providers (Optional), 2. Create Config Request, 3. Poll for Completion, 4. Parse Schemas and Build UI, 5. Submit Configuration, Base URL, Change Log, Common Error Patterns (+16 more)

### Community 97 - "newProvidersHandler"
Cohesion: 0.40
Nodes (5): newProvider(), newProvidersHandler(), TestListProviders_Empty(), TestListProviders_ScopedToNamespace(), TestUpdateProviderStatus()

### Community 98 - "dataprovider.pb.go"
Cohesion: 0.20
Nodes (3): file_proto_dataprovider_proto_init(), file_proto_dataprovider_proto_rawDescGZIP(), init()

### Community 99 - "TestKrknGraphRun_ValidateGraph"
Cohesion: 0.33
Nodes (5): SanitizeNodeIDForKubernetes(), stringPtr(), TestCollisionDetection(), TestKrknGraphRun_ValidateGraph(), TestSanitizeNodeIDForKubernetes()

### Community 100 - "makeScenarioRun"
Cohesion: 0.27
Nodes (12): makeGraphRun(), makeScenarioRun(), TestListJobs_IncludesStats(), TestListJobs_LimitOnlyReturnsAll(), TestListJobs_NoPagination(), TestListJobs_Paginated(), TestListJobs_SkipsGraphRunChildren(), TestListJobs_TypeDiscriminator() (+4 more)

### Community 101 - "Authentication API Specification"
Cohesion: 0.09
Nodes (22): 1. Check Admin Registration, 1. Initial Load, 2. First Admin Registration, 2. Register User, 3. Login, 3. Login, 4. Authenticated Requests, 5. Role-Based UI (+14 more)

### Community 102 - "PROGRESS-MVP.md"
Cohesion: 0.22
Nodes (8): Architecture Decisions, Data Provider Architecture, Future Work (Not in Current Scope), Integration Points, Namespace Configuration, Next Steps, Requirements Overview, REST API Framework

### Community 103 - "scenario_filter_test.go"
Cohesion: 0.28
Nodes (6): filterScenariosByIsAScenario(), TestFilterScenariosByIsAScenario(), TestFilterScenariosByIsAScenario_CancelledContext(), TestFilterScenariosByIsAScenario_PreservesTagMetadata(), TestValidateValueAgainstSchema(), ValidateValueAgainstSchema()

### Community 104 - "setup-ocm-and-operator.sh"
Cohesion: 0.67
Nodes (5): log_error(), log_info(), log_warn(), setup-ocm-and-operator.sh script, wait_for_condition()

### Community 105 - "File Management API Specification"
Cohesion: 0.09
Nodes (22): Access Control, API Endpoints, ConfigMap Structure, Content Validation, Create a Public Configuration File, Create File, Delete File, Error Responses (+14 more)

### Community 107 - "krkn-operator Development Progress (Post-MVP)"
Cohesion: 0.22
Nodes (9): API Endpoints Completed, Core Infrastructure, Current Status, Development Phases Template, Future Considerations, Key Technical Achievements, krkn-operator Development Progress (Post-MVP), MVP Completion Summary (+1 more)

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
Cohesion: 0.10
Nodes (21): 1. Login and Get JWT Token, 2. Create KrknTargetRequest (or use existing UUID), 3. Test Terminal Command, Cleanup, Configure Operator to Use Local gRPC Server, Debug gRPC Server in PyCharm, Development Workflow, Error: "Command not found: kubectl" (+13 more)

### Community 112 - "REST API Endpoints"
Cohesion: 0.29
Nodes (7): GET /clusters, GET /nodes, GET /targets/{UUID}, POST /scenarios, POST /scenarios/detail/{scenario_name}, POST /targets, REST API Endpoints

### Community 117 - "CLAUDE.md - Project Instructions"
Cohesion: 0.10
Nodes (19): Architecture Principles, Before Committing, Before Writing Code, CLAUDE.md - Project Instructions, Code Organization & Reusability, Code Quality Standards, Communication, Critical Rules (+11 more)

### Community 128 - "API Changes"
Cohesion: 0.29
Nodes (7): GraphRunDetailResponse, GraphRunSpecResponse, 1.1 Resiliency weight per graph node, 1. Request - Create GraphRun (POST /api/v1/graphruns), 2. Response - List GraphRuns (GET /api/v1/graphruns), 3. Response - Get GraphRun Detail (GET /api/v1/graphruns/:name), API Changes

### Community 129 - "Implementation Details"
Cohesion: 0.14
Nodes (16): Code Refactoring: getKubeconfigFromTarget() Helper ✅, Implementation Details, Phase 10: POST /scenarios/detail/{scenario_name} Endpoint ✅, Phase 11: POST /scenarios/globals/{scenario_name} Endpoint ✅, Phase 1: REST API Setup ✅, Phase 2: GET /clusters Endpoint ✅, Phase 3: Integration ✅, Phase 4: Testing ✅ (+8 more)

### Community 130 - "ProviderRegistration"
Cohesion: 0.11
Nodes (18): After (Multi-Active API), API Server Concurrent Operations, API Server Startup Wait, Before (Single Leader), Concurrency Safety, Controller Safety with Leader Election, Deployment Considerations, Expected Startup Logs (+10 more)

### Community 131 - "krkn-operator"
Cohesion: 0.11
Nodes (18): Alerts Query API, API compatibility notes, Backup and Restore, Documentation, Ecosystem, Graph node resiliency weights, krkn-operator, License (+10 more)

### Community 132 - "Deployment Guide"
Cohesion: 0.12
Nodes (17): Architecture, Build Installer Bundle, Custom Push, Deployment Guide, Environment Variables, Git Tag Support, Git Tag Workflow, Image Names (Auto-Generated) (+9 more)

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

### Community 140 - ".checkRollbackImage"
Cohesion: 0.48
Nodes (3): rollbackImageCache, rollbackImageCheck, Rollback artifact persistence

### Community 142 - "KrknScenarioRun"
Cohesion: 0.06
Nodes (27): KrknScenarioRunReconciler, migrationPending(), TestScenarioRunReconcileSkipsPendingMigration(), backfillLegacyMaxRetries(), countJobPhaseChanges(), getOwnerLabel(), TestExtractAndStoreReportsRejectsMultiClusterRun(), TestBackfillLegacyMaxRetries() (+19 more)

### Community 143 - "elasticsearch/labels_test.go"
Cohesion: 0.33
Nodes (5): TestBuildAnnotations(), TestBuildLabels(), TestBuildLabelsAccessControl(), TestUpdateAnnotations(), UpdateAnnotations()

### Community 144 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.17
Nodes (12): API and operator refactoring to support multiple clusters, API Endpoints Structure (Nested Approach), BREAKING CHANGES, Implementation Details, ✅ Implemented, KrknTargetRequest controller in operator, Moving from direct scenario creation to CRD approach, Overview (+4 more)

### Community 145 - "Technical Notes"
Cohesion: 0.40
Nodes (5): Environment Variables, Implementation Files, KrknTargetRequest Structure, Server Configuration, Technical Notes

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

### Community 153 - "Setup"
Cohesion: 0.40
Nodes (5): 1. Create Virtual Environment (Python 3.11), 2. Install Dependencies, 3. Verify Generated Stubs, 4. Start gRPC Server, Setup

### Community 154 - ".loadCategoryResiliencyHistories"
Cohesion: 0.19
Nodes (16): categoryHistoryRun, categoryHistoryScore, buildCategoryResiliencyHistory(), categoryBehaviorConfigurationSignature(), categoryRunScenarioNames(), containsString(), filterCategoryHistoryScoresByCluster(), filterGraphHistoryScoresByPermission() (+8 more)

### Community 155 - "Post-MVP Development"
Cohesion: 0.40
Nodes (5): Completed, In Progress, Pending, Phase 1: KrknOperatorTarget CRD and CRUD API, Post-MVP Development

### Community 156 - ".GetScenarioRunLogs"
Cohesion: 0.15
Nodes (14): filterReportLines(), isReportMarkerEnd(), isReportMarkerStart(), isWebSocketDisconnectError(), maskToken(), preStartFailureReason(), sanitizeHeaders(), TestMaskToken() (+6 more)

### Community 162 - "Scenario workspaces on a shared PVC"
Cohesion: 0.22
Nodes (4): Publish and select a workspace, Replay and unavailable storage, Scenario workspaces on a shared PVC, Storage validation

### Community 163 - "files/validation_test.go"
Cohesion: 0.31
Nodes (8): contains(), findSubstring(), TestValidateFileContent(), TestValidateFileGroups(), TestValidateUserFilePermissions(), ValidateFileContent(), ValidateFileGroups(), ValidateUserFilePermissions()

### Community 171 - "generateKubeconfigFromRequest"
Cohesion: 0.53
Nodes (6): CreateTargetRequest, UpdateTargetRequest, generateKubeconfigFromCredentialsType(), generateKubeconfigFromKubeconfigType(), generateKubeconfigFromRequest(), generateKubeconfigFromTokenType()

## Knowledge Gaps
- **17 isolated node(s):** `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME`, `ICON_BASE64`, `ICON_MEDIATYPE` (+12 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 734 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **29 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Handler` connect `Handler` to `main.go`, `ConnectionParams`, `GroupLabelKey`, `GetClaimsFromContext`, `k8s.io/api/core/v1.Secret`, `context.Context`, `NewServerWithComponentNamespace`, `bootstrap.go`, `.checkRollbackImage`, `JobTracker`, `KrknScenarioRun`, `KrknCategory`, `.listWorkspaces`, `github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2`, `.loadCategoryResiliencyHistories`, `GetClaimsFromContext`, `.GetScenarioRunLogs`?**
  _High betweenness centrality (0.053) - this node is a cross-community bridge._
- **Are the 159 inferred relationships involving `NewTestHandler()` (e.g. with `setupAuthTestHandler()` and `TestLogin_InactiveUser()`) actually correct?**
  _`NewTestHandler()` has 159 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME` to the rest of the system?**
  _17 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.09646251215205444 - nodes in this community are weakly interconnected._
- **Why does `krkn-operator` connect `krkn-operator` to `Scenario workspaces on a shared PVC`, `Running Locally`?**
  _High betweenness centrality (0.032) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.011580583099008435 - nodes in this community are weakly interconnected._
- **Why does `KrknUserGroup` connect `GetClaimsFromContext` to `main.go`, `GroupLabelKey`, `.DeepCopy`, `File Management API Specification`, `newCategoryRequestAs`, `.loadCategoryResiliencyHistories`, `KrknUserGroupSpec`, `.DeepCopyInto`?**
  _High betweenness centrality (0.025) - this node is a cross-community bridge._