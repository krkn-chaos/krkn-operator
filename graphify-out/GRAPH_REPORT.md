# Graph Report - krkn-operator  (2026-11-10)

## Corpus Check
- 337 files · ~432,475 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 18 file(s) not represented in the graph (top: (none) 12, .jsonl 2, .toml 1)

## Summary
- 3656 nodes · 14821 edges · 169 communities (139 shown, 30 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 1392 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `c4a25581`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- main.go
- testing.T
- Handler
- Handler
- GetClaimsFromContext
- context.Context
- NewTestHandler
- cloudcreds_handlers_test.go
- backup.go
- GroupLabelKey
- api/types.go
- Broadcaster
- migration.go
- addAdminContext
- k8s.io/api/core/v1.Secret
- GetClaimsFromContext
- render-bundle.sh
- server.py
- NewSecretManager
- NewHub
- WSUnifiedJobItem
- user_handlers_test.go
- client_test.go
- KrknGraphRun
- elasticsearch/types.go
- installer.go
- .DeepCopyInto
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- KrknUserGroupSpec
- client.go
- backup_restore_handlers_test.go
- KrknWorkspace
- Handler
- ConnectionParams
- KrknOperatorTargetProviderConfig
- injection.go
- .UpdateRegistry
- NewCachedUserStatusChecker
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- ComponentManifest
- categoryHistoryRun
- UnifiedJobItem
- wsorigin.go
- TestJWTSecret
- migration_test.go
- notify-operator-docs.test.js
- converter_test.go
- guardedDialContext
- buildUnifiedJobList
- Hub
- KrknScenarioRunSpec
- .getClusterAPIURL
- bootstrap.go
- time.Duration
- scenarioRunConfigurationFixture
- cloudcreds/types.go
- GetNodesRequest
- dataprovider_grpc.pb.go
- GetEnabled
- newReportTestHandler
- .DeepCopyInto
- ScenarioRunRequest
- github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2
- GET methods
- installer_test.go
- 🔧 TODO: Pod Recreation and Retry Logic
- ScenarioReference
- rate_limiter_test.go
- SecretManager
- sigs.k8s.io/controller-runtime.Manager
- start_operator.sh
- .Get
- PROGRESS-MVP.md
- Compute
- sync_test.go
- KrknOperatorTargetProvider
- registry/types.go
- test-api.sh
- Frontend Authentication Implementation Tasks
- ExecuteKubectlRequest
- mockJob
- .updateClusterJobStatuses
- captureLogSink
- GraphClusterScore
- Krkn-Operator Roadmap
- .loadCategoryResiliencyHistories
- KrknOperatorTargetProviderConfigStatus
- Publisher
- ClusterTarget
- category_run_configuration.go
- convertClusterJobs
- KrknOperatorTargetProviderList
- watcher_test.go
- ExecuteKubectlResponse
- ProviderRegistration
- Provider Configuration API Specification
- .extractAndStoreReports
- dataprovider.pb.go
- TestKrknGraphRun_ValidateGraph
- makeScenarioRun
- Authentication API Specification
- net/http.Handler
- buildClusterAPIURLMap
- setup-ocm-and-operator.sh
- File Management API Specification
- JobTracker
- commands.go
- Cluster Segregation Troubleshooting
- channel-head.sh
- Running gRPC Data Provider Locally for Development
- TokenGenerator
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
- .translateVolumesToFileMounts
- newProvidersHandler
- KrknScenarioRunReconciler
- dataProviderServiceClient
- 🔧 TODO: Pod Recreation and Retry Logic
- .flatten
- Running Locally
- install-yq.sh
- test-release-workflow-gates.sh
- Deployment
- Available Make Targets
- Building Images
- Troubleshooting
- file_proto_dataprovider_proto_rawDescGZIP
- DataProviderServiceServer
- Troubleshooting
- .GetScenarioRunLogs
- assets/README.md
- .QueryTelemetryPage
- bootstrapOLMResources
- files/validation_test.go
- BuildUnifiedJobList
- krkngraphrun_controller_sanitization_test.go
- Pushing Images
- providerConfigTestScheme
- elasticsearch/labels_test.go
- generator_test.go
- sanitizeHeaders
- WorkflowResponse

## God Nodes (most connected - your core abstractions)
1. `Handler` - 212 edges
2. `NewTestHandler()` - 167 edges
3. `writeJSONError()` - 133 edges
4. `writeJSON()` - 100 edges
5. `createAdminContext()` - 91 edges
6. `GetClaimsFromContext()` - 64 edges
7. `KrknScenarioRun` - 62 edges
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

## Communities (169 total, 30 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.09
Nodes (18): availableWorkspacePVC, CategoryEntityAssociationResponse, CreateCategoryRequest, ResiliencyHistoryQueryRequest, UpdateCategoryRequest, workspacePublishRequest, ContextKey, clusterTarget (+10 more)

### Community 1 - "testing.T"
Cohesion: 0.01
Nodes (198): TestKrknUserCreation(), TestKrknUserList(), TestKrknUserRoles(), TestKrknUserStatus(), TestKrknUserValidation(), TestScenarioReferenceValidate(), setupAuthTestHandler(), TestIsRegistered_MethodNotAllowed() (+190 more)

### Community 2 - "Handler"
Cohesion: 0.07
Nodes (21): workspaceInfo, writeCategoryRunNotFoundOrInternalError(), buildConnectionParams(), buildConnectionParamsForIndex(), convertGraphClusterScores(), buildUserGroupResponse(), extractGroupNameAndUserIDFromPath(), extractGroupNameFromMembersPath() (+13 more)

### Community 3 - "Handler"
Cohesion: 0.07
Nodes (24): Handler, TerminalRequest, TerminalResponse, Code References, writeScenarioRunAccessError(), filterScenarioHistoryScoresByPermission(), sanitizeRunNameLabel(), intPtr() (+16 more)

### Community 4 - "GetClaimsFromContext"
Cohesion: 0.05
Nodes (46): AvailableFilesResponse, DeleteFileResponse, ListFilesResponse, UpdateFileResponse, buildFileInfo(), buildFileResponse(), deriveLogicalName(), extractLogicalName() (+38 more)

### Community 5 - "context.Context"
Cohesion: 0.04
Nodes (57): historyQueryCountingClient, mockUserStatusChecker, RestoreConfig, KrknWorkspaceReconciler, Reuse from another operator, removeCategoryLabel(), isValidConfigMapKey(), validateFileFields() (+49 more)

### Community 6 - "NewTestHandler"
Cohesion: 0.06
Nodes (105): esDoerFunc, TestGetActiveRunsOverview_AllUsersSeesAll(), TestGetActiveRunsOverview_MixedJobStates(), TestGetActiveRunsOverview_NoActiveRuns(), TestGetActiveRunsOverview_UserSeesAll(), boolPtr(), newEsAlertsTestSecretWithHost(), newEsScheme() (+97 more)

### Community 7 - "cloudcreds_handlers_test.go"
Cohesion: 0.11
Nodes (39): TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(), TestCloudCredentialContract_GraphRunStripsNodeEnv(), newAWSTestSecret(), newCloudCredScheme(), newGCPTestSecret(), newGroupedAWSSecret(), TestCloudCredentialsRouter_MethodNotAllowed(), TestCloudCredentialsRouter_SubpathMethodNotAllowed() (+31 more)

### Community 8 - "backup.go"
Cohesion: 0.06
Nodes (36): resiliencyHistoryQuerySelections, BackupConfig, resourceType, tokenRequest, normalizeResiliencyHistoryProviders(), normalizeResiliencyHistorySelections(), parseResiliencyHistoryQuery(), backupResourceType() (+28 more)

### Community 9 - "GroupLabelKey"
Cohesion: 0.05
Nodes (82): RollbackAvailability, newCategoryRequest(), newCategoryRequestAs(), newCategoryTestGroup(), newCategoryTestHandler(), newCategoryTestUser(), TestCategoriesRouterCRUD(), TestCategoriesRouterGroupCreationRequiresMembership() (+74 more)

### Community 10 - "api/types.go"
Cohesion: 0.05
Nodes (45): ActiveRunsOverviewResponse, AddGroupMemberRequest, AddGroupMemberResponse, ChangePasswordRequest, ChangePasswordResponse, ClusterPermissionSet, ClusterResiliencyScoreResponse, CreateTargetResponse (+37 more)

### Community 11 - "Broadcaster"
Cohesion: 0.17
Nodes (5): hashBytes(), roleFromAdmin(), Broadcaster, graphRunEventHandler, scenarioRunEventHandler

### Community 12 - "migration.go"
Cohesion: 0.12
Nodes (44): clusterJobImage, clusterJobProvider, clusterJobRetries, graphScenarioReferences, migrationBlockedError, migrationPlan, migrationResult, migrationStats (+36 more)

### Community 13 - "addAdminContext"
Cohesion: 0.11
Nodes (42): addAdminContext(), addUserContext(), createTestAdminUser(), setupFilesTestHandler(), TestCanAccessFile(), TestCreateFile(), TestCreateFile_DefaultFilePurpose(), TestCreateFile_DuplicateName() (+34 more)

### Community 14 - "k8s.io/api/core/v1.Secret"
Cohesion: 0.15
Nodes (9): cloudCredentialAccessContext, buildCloudCredentialResponse(), updateCloudCredentialSecretData(), buildElasticsearchConfigResponse(), canAccessElasticsearchConfig(), ExtractGroupsFromLabels(), ResolveAccessControlForUpdate(), ExtractGroupsFromLabels() (+1 more)

### Community 15 - "GetClaimsFromContext"
Cohesion: 0.12
Nodes (19): CategoryConfigurationGroup, CategoryListResponse, CategoryResiliencyHistoryResponse, CategoryResponse, ErrorResponse, ResiliencyHistoryQueryResponse, buildCategoryResponse(), buildCategoryVisibilityLabels() (+11 more)

### Community 16 - "render-bundle.sh"
Cohesion: 0.07
Nodes (32): channel_has_entries(), ensure_catalog_channel(), catalog-channel.sh script, update_default_channel(), write_release_config(), catalog_submit_checkout(), catalog_submit_open_pr(), catalog-submit.sh script (+24 more)

### Community 17 - "server.py"
Cohesion: 0.06
Nodes (6): add_DataProviderServiceServicer_to_server(), DataProviderService, DataProviderServiceServicer, DataProviderServiceStub, DataProviderServicer, serve()

### Community 18 - "NewSecretManager"
Cohesion: 0.11
Nodes (20): k8sUserStatusChecker, TestListComponentsRouteAndAuth(), TestQueryElasticsearchTelemetry_RouteAndAuth(), sanitizeUsername(), TestSanitizeUsername_ProducesValidK8sNames(), newK8sUserStatusChecker(), newKrknUser(), newUserStatusScheme() (+12 more)

### Community 19 - "NewHub"
Cohesion: 0.10
Nodes (36): NewHandler(), NewBroadcaster(), TestBroadcastDashboardUpdate(), TestBroadcastGraphRunDeleted(), TestBroadcastGraphRunUpdate(), TestBroadcastOnlyToSubscribedClients(), TestBroadcastScenarioRunDeleted(), TestBroadcastScenarioRunUpdate() (+28 more)

### Community 20 - "WSUnifiedJobItem"
Cohesion: 0.10
Nodes (14): TestFilterJobsByCategoriesMatchesAnySelectedCategory(), TestPaginateJobItems(), computeWSJobStats(), filterJobCategoriesByVisibility(), filterJobsByCategories(), groupRollbackJobs(), paginateJobItems(), TestGroupRollbackJobsWebSocketKeepsFlatViewAndChecksUID() (+6 more)

### Community 21 - "user_handlers_test.go"
Cohesion: 0.15
Nodes (33): createTestUser(), kvValue(), setupUserTestHandler(), TestChangePassword_AdminChange_NoCurrentPassword(), TestChangePassword_AdminChangesOtherUser_EmitsAuditLog(), TestChangePassword_SamePassword_Rejected(), TestChangePassword_SelfChange_NoAdminAuditLog(), TestChangePassword_SelfChange_RequiresCurrentPassword() (+25 more)

### Community 22 - "client_test.go"
Cohesion: 0.10
Nodes (23): doerFunc, roundTripperFunc, NewClient(), scenarioParametersSchema(), TestBaseURL(), TestClientTransportDefaultNotHTTPTransport(), TestClientTransportPooling(), TestConnectionParamsTLSConfig() (+15 more)

### Community 23 - "KrknGraphRun"
Cohesion: 0.19
Nodes (7): KrknGraphRunReconciler, TestGraphRunReconcileSkipsPendingMigration(), sanitizeNodeID(), TestCalculateResiliencyScore_StoresNodeScores(), overrideRef(), TestGraphRunInvalidImageSignatureFailsWithoutRetry(), KrknGraphRun

### Community 24 - "elasticsearch/types.go"
Cohesion: 0.10
Nodes (28): AffectedPods, ClusterMetadata, CreateElasticsearchConfigRequest, CreateElasticsearchConfigResponse, DeleteElasticsearchConfigResponse, ListElasticsearchConfigsResponse, NodeSummaryInfo, QueryAlertsRequest (+20 more)

### Community 25 - "installer.go"
Cohesion: 0.09
Nodes (21): main(), resolveMigrationNamespace(), runCRDMigrationMode(), TestResolveMigrationNamespacePrefersHookNamespace(), TestRunCRDMigrationMode(), configMapClient, crdManifest, deploymentClient (+13 more)

### Community 26 - ".DeepCopyInto"
Cohesion: 0.04
Nodes (18): ListProvidersResponse, ProviderResponse, filterUsers(), GraphRunSummary, GraphScenario, KrknCategorySpec, KrknGraphRunStatus, KrknOperatorTargetProviderStatus (+10 more)

### Community 27 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.14
Nodes (5): KrknTargetRequestReconciler, KrknOperatorTarget, KrknOperatorTargetSpec, KrknTargetRequest, KrknTargetRequestSpec

### Community 29 - "client.go"
Cohesion: 0.11
Nodes (26): esAlertSearchResponse, esSearchResponse, facetField, filterAggregation, Option, StatusError, termsAggregation, buildAlertsSearchBody() (+18 more)

### Community 30 - "backup_restore_handlers_test.go"
Cohesion: 0.15
Nodes (22): NewJobTracker(), createMultipartBody(), createTestHandler(), TestGetRestoreStatusAdminOnly(), TestGetRestoreStatusTrackedJob(), TestGetRestoreStatusUnknownJob(), TestHandlerShutdownCancelsContext(), TestJobTrackerEviction() (+14 more)

### Community 31 - "KrknWorkspace"
Cohesion: 0.25
Nodes (4): CanAccess(), TestCanAccess(), KrknWorkspace, KrknWorkspaceSpec

### Community 32 - "Handler"
Cohesion: 0.22
Nodes (5): visibleCategoryNames(), AuthorizationChecker, categoryVisibilityChecker, ClientMessage, Handler

### Community 33 - "ConnectionParams"
Cohesion: 0.18
Nodes (10): AlertDocument, Doer, QueryAlertsResponse, decodeAlerts(), executeSearch(), Client, ConnectionParams, newSearchRequest() (+2 more)

### Community 34 - "KrknOperatorTargetProviderConfig"
Cohesion: 0.11
Nodes (16): 1. List Providers, 2. Get Provider Config Status, 2. Update Provider Status, 3. Create Provider Config Request, 3. Update Provider Configuration, API Endpoints, UpdateProviderConfig(), Configuration Functions (+8 more)

### Community 35 - "injection.go"
Cohesion: 0.17
Nodes (21): preparedJobResources, TestAppendCloudCredentialInjection_EmptyRefNoop(), injectAWS(), injectAzure(), injectBaremetal(), InjectCredentials(), injectGCP(), injectIBMCloud() (+13 more)

### Community 37 - ".UpdateRegistry"
Cohesion: 0.16
Nodes (6): buildRegistryInfo(), buildRegistryResponse(), validateCreateRegistryRequest(), validateUpdateRegistryRequest(), BuildRegistryLabels(), ExtractGroupsFromLabels()

### Community 38 - "NewCachedUserStatusChecker"
Cohesion: 0.15
Nodes (12): CachedUserStatusChecker, countingUserStatusChecker, userStatusCacheEntry, UserStatusChecker, Middleware, NewCachedUserStatusChecker(), NewLazyMiddleware(), TestCachedUserStatusChecker() (+4 more)

### Community 39 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.18
Nodes (10): GraphRunDetailResponse, GraphRunListItem, GraphRunListResponse, GraphRunSpecResponse, GraphRunStatusResponse, 1.1 Resiliency weight per graph node, 1. Request - Create GraphRun (POST /api/v1/graphruns), 2. Response - List GraphRuns (GET /api/v1/graphruns) (+2 more)

### Community 40 - "ComponentManifest"
Cohesion: 0.15
Nodes (7): ListComponentsResponse, Load(), TestLoad(), TestLoadRejectsInvalidManifest(), ComponentDescription, ComponentManifest, KrknComponentStatus

### Community 41 - "categoryHistoryRun"
Cohesion: 0.14
Nodes (15): categoryEnvironmentParameter, categoryHistoryRun, categoryParameterProfileFingerprint(), categoryParameterProfileName(), TestCategoryParameterProfileFingerprintForEmptyEnvironment(), TestCategoryParameterProfileFingerprintGraphEnvironmentMultiset(), TestCategoryParameterProfileFingerprintIgnoresRunType(), TestCategoryParameterProfileFingerprintScenarioEnvironment() (+7 more)

### Community 42 - "UnifiedJobItem"
Cohesion: 0.13
Nodes (6): NodeStatusResponse, PaginationMeta, ScenarioRunListItem, ScenarioRunListResponse, UnifiedJobItem, UnifiedJobsResponse

### Community 43 - "wsorigin.go"
Cohesion: 0.16
Nodes (17): checkWebSocketOriginV2(), TestCheckWebSocketOriginV2(), checkWebSocketOrigin(), TestCheckWebSocketOrigin(), guardedCheckRedirect(), TestGuardedCheckRedirect(), TestValidateDestinationURL(), validateDestinationURL() (+9 more)

### Community 44 - "TestJWTSecret"
Cohesion: 0.11
Nodes (21): TestLogin_InactiveUser(), TestLogin_InvalidCredentials(), TestLogin_Success(), TestDeleteTargetByUUID_AdminCanDeleteAnyResource(), TestDeleteTargetByUUID_InvalidUUID(), TestDeleteTargetByUUID_NonOwner_Forbidden(), TestDeleteTargetByUUID_NotFound(), TestDeleteTargetByUUID_OwnerCanDelete() (+13 more)

### Community 45 - "migration_test.go"
Cohesion: 0.20
Nodes (25): Complete(), Stage(), containsSubstring(), countPatchActions(), jsonMarshal(), migrationTestLogger(), newMigrationClient(), scenarioRunFixture() (+17 more)

### Community 46 - "notify-operator-docs.test.js"
Cohesion: 0.10
Nodes (11): assert, chartWorkflow, { execFileSync, spawnSync }, fs, notifier, notifyWorkflow, os, path (+3 more)

### Community 47 - "converter_test.go"
Cohesion: 0.20
Nodes (15): BuildDockerConfigJSON(), ExtractRegistryV2FromSecret(), TestBuildDockerConfigJSON_MissingCredentials(), TestBuildDockerConfigJSON_OAuth(), TestBuildDockerConfigJSON_Password(), TestExtractRegistryV2FromSecret_InvalidType(), TestExtractRegistryV2FromSecret_MissingAnnotations(), TestExtractRegistryV2FromSecret_MissingComponent() (+7 more)

### Community 48 - "guardedDialContext"
Cohesion: 0.22
Nodes (5): responseWriter, guardedDialContext(), TestValidateDestinationIP(), validateDestinationIP(), validateInlineDestination()

### Community 49 - "buildUnifiedJobList"
Cohesion: 0.17
Nodes (12): TestBuildUnifiedJobList_SkipsGraphRunChildren(), averageResiliencyScore(), buildScenarioRunDetailResponse(), buildScenarioRunResponse(), buildUnifiedJobList(), convertClusterResiliencyScoresWS(), runCategoryNames(), sanitizedClusterJobs() (+4 more)

### Community 50 - "Hub"
Cohesion: 0.22
Nodes (7): isWebSocketDisconnectError(), writeWSError(), AuthorizationCheckFunc, BroadcastMessage, Client, Hub, PaginationClientState

### Community 51 - "KrknScenarioRunSpec"
Cohesion: 0.15
Nodes (13): FromKrknctlScenarioSet(), publicScenarioReference(), publicScenario(), TestFromKrknctlScenarioSetCreatesPublicReferences(), TestToKrknctlScenarioSetUsesScenarioReference(), ToKrknctlScenarioSet(), MapScenarioNodeToScenarioRunSpec(), ResolveGraph() (+5 more)

### Community 52 - ".getClusterAPIURL"
Cohesion: 0.46
Nodes (7): newTargetRequest(), newTestScheme(), TestGetClusterAPIURL_ClusterNotFound(), TestGetClusterAPIURL_Found(), TestGetClusterAPIURL_ProviderNotFound(), TestGetClusterAPIURL_TargetRequestNotFound(), TestReconcile_FailedJobStatus_IncludesClusterAPIURL()

### Community 53 - "bootstrap.go"
Cohesion: 0.17
Nodes (13): RemoveElasticsearchGrafanaURLAnnotations(), TestRemoveElasticsearchGrafanaURLAnnotations(), ensureJWTSecret(), EnsureResources(), ensureSCCBinding(), ensureScenarioRunnerRBAC(), ensureServiceAccount(), ensureServices() (+5 more)

### Community 54 - "time.Duration"
Cohesion: 0.19
Nodes (4): ipRateLimiter, rateLimiterEntry, getTokenDuration(), TestGetTokenDuration()

### Community 55 - "scenarioRunConfigurationFixture"
Cohesion: 0.23
Nodes (17): CategoryResiliencyDataPoint, datapointForRunID(), datapointRunIDs(), TestBuildCategoryResiliencyHistoryGroupsTypedConfigurationsAndSortsPerCluster(), TestGraphHistoryScoresOnlyKeepsFinalScoresIncludingZero(), assertConfigurationDifferences(), floatPointer(), graphRunConfigurationFixture() (+9 more)

### Community 56 - "cloudcreds/types.go"
Cohesion: 0.16
Nodes (15): CreateCloudCredentialResponse, DeleteCloudCredentialResponse, ListCloudCredentialsResponse, UpdateCloudCredentialResponse, buildCloudCredentialSecretData(), CloudCredentialResponse, CreateCloudCredentialRequest, UpdateCloudCredentialRequest (+7 more)

### Community 57 - "GetNodesRequest"
Cohesion: 0.15
Nodes (3): DataProviderServiceClient, GetNodesRequest, GetNodesResponse

### Community 58 - "dataprovider_grpc.pb.go"
Cohesion: 0.16
Nodes (5): UnimplementedDataProviderServiceServer, UnsafeDataProviderServiceServer, _DataProviderService_ExecuteKubectl_Handler(), _DataProviderService_GetNodes_Handler(), RegisterDataProviderServiceServer()

### Community 59 - "GetEnabled"
Cohesion: 0.23
Nodes (8): rollbackImageCache, rollbackImageCheck, Publish and select a workspace, Replay and unavailable storage, Rollback artifact persistence, Rollback history in Jobs, Scenario workspaces on a shared PVC, Storage validation

### Community 60 - "newReportTestHandler"
Cohesion: 0.29
Nodes (10): newReportTestHandler(), reportRequest(), TestDownloadReportDeniesUserWithoutReportClusterAccess(), TestDownloadReportHTMLThroughV2Router(), TestDownloadReportPDF(), TestDownloadReportRejectsMalformedPath(), TestDownloadReportUnavailable(), TestGetReportStatusPending() (+2 more)

### Community 61 - ".DeepCopyInto"
Cohesion: 0.07
Nodes (11): KrknCategoryList, KrknComponent, KrknComponentList, KrknGraphRunList, KrknOperatorTargetList, KrknOperatorTargetProviderConfigList, KrknScenarioRunList, KrknTargetRequestList (+3 more)

### Community 62 - "ScenarioRunRequest"
Cohesion: 0.19
Nodes (13): Phase 12: Scenario Job Management - /scenarios/run Endpoints ✅, Phase 9: POST /scenarios Endpoint ✅, FileMount, GlobalsRequest, JobsListResponse, JobStatusResponse, ScenarioRunRequest, ScenariosRequest (+5 more)

### Community 63 - "github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2"
Cohesion: 0.11
Nodes (15): mockScenarioProvider, InvalidImageSignatureError, createScenarioProvider(), filterScenariosByIsAScenario(), getScenarioSignatureStatus(), resolveRollbackDigest(), TestFilterScenariosByIsAScenario(), TestFilterScenariosByIsAScenario_CancelledContext() (+7 more)

### Community 64 - "GET methods"
Cohesion: 0.06
Nodes (35): API Requirements, /clusters, DELETE methods, GET methods, GET methods, get nodes, get pods, Grpc python service requirement (+27 more)

### Community 65 - "installer_test.go"
Cohesion: 0.18
Nodes (12): crdClient, applyCRD(), applyCRDs(), TestApplyCRDsCreatesMissingAndUpdatesChangedSpecs(), TestApplyCRDsDoesNotPatchUnchangedSpecs(), TestApplyCRDsLabelsExistingCRDsWithoutChangingOtherLabels(), TestApplyCRDsRejectsNonCRDManifest(), testCRD() (+4 more)

### Community 66 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.25
Nodes (8): Configuration, Current Behavior, Option A: Cancellation Field in Status (Recommended), Option B: Finalizers for Cancellation Tracking, Option C: Separate CancellationRequest CR, Proposed Solution Options, Requirements, 🔧 TODO: Pod Recreation and Retry Logic

### Community 67 - "ScenarioReference"
Cohesion: 0.13
Nodes (16): KrknScenarioRunSpec, publicScenarioReference(), TestCreateScenarioRun_ResiliencyScore(), buildContainerImage(), buildContainerImageFromReference(), getTestConfig(), privateReference(), publicReference() (+8 more)

### Community 68 - "rate_limiter_test.go"
Cohesion: 0.18
Nodes (18): clientIPFromHeaders(), extractClientIP(), hostFromRemoteAddr(), isTrustedProxy(), newIPRateLimiter(), parseTrustedProxyCIDRs(), rateLimitMiddleware(), TestExtractClientIP() (+10 more)

### Community 69 - "SecretManager"
Cohesion: 0.11
Nodes (10): Server, NewHandler(), NewServer(), NewServerWithComponentNamespace(), Handler, generateRandomSecret(), getSecretKeys(), SecretManager (+2 more)

### Community 71 - "start_operator.sh"
Cohesion: 0.41
Nodes (12): build_operator(), check_prerequisites(), cleanup(), create_service_account(), install_crds(), main(), print_error(), print_info() (+4 more)

### Community 72 - ".Get"
Cohesion: 0.22
Nodes (13): countingGetClient, TestConcurrentAccess(), TestConcurrentSnapshot(), TestDelete(), TestDelete_NonExistent(), TestEmptyValue(), TestExists(), TestGet_Singleton() (+5 more)

### Community 73 - "PROGRESS-MVP.md"
Cohesion: 0.04
Nodes (41): Architecture Decisions, Data Provider Architecture, Environment Variables, Future Work (Not in Current Scope), GET /clusters, GET /nodes, GET /targets/{UUID}, Implementation Files (+33 more)

### Community 74 - "Compute"
Cohesion: 0.22
Nodes (5): ComputeJobStats(), TestComputeJobStats(), Compute(), JobCounter, Summary

### Community 75 - "sync_test.go"
Cohesion: 0.09
Nodes (16): ConfigStoreInitializer, NewConfigStoreInitializer(), TestConfigStoreInitializer_NeedLeaderElection(), TestConfigStoreInitializer_Start_ConfigMapNotFound(), TestConfigStoreInitializer_Start_Success(), Store, SyncConfigMapToStore(), TestRoundTrip() (+8 more)

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

### Community 82 - ".updateClusterJobStatuses"
Cohesion: 0.12
Nodes (12): preStartFailureReason(), TestPreStartFailureReason(), backfillLegacyMaxRetries(), TestBackfillLegacyMaxRetries(), TestFailedAttemptRetainsRollbackIdentityBeforeRetry(), TestShouldRetryJobHonorsConfiguredRetryLimit(), TestUpdateClusterJobStatuses_ClusterNameEmpty(), TestUpdateClusterJobStatuses_PreservesExplicitZero() (+4 more)

### Community 84 - "GraphClusterScore"
Cohesion: 0.14
Nodes (6): convertNodeStatusesWithScores(), convertNodeStatusesWithScores(), buildGraphRunResponse(), convertGraphClusterScoresForSnapshot(), GraphClusterScore, GraphRunResponse

### Community 85 - "Krkn-Operator Roadmap"
Cohesion: 0.29
Nodes (6): Krkn-Operator Roadmap, Ongoing (not release-blocked), Release 1 — 2026-08-18, Release 2 — 2026-10-18, Release 3 — 2026-12-22, Release 4 — 2027-02-23

### Community 86 - ".loadCategoryResiliencyHistories"
Cohesion: 0.25
Nodes (10): categoryHistoryScore, containsString(), filterCategoryHistoryScoresByCluster(), filterGraphHistoryScoresByPermission(), graphHistoryScores(), runHasAnyCategory(), scenarioHistoryScores(), graphHistoryRun() (+2 more)

### Community 87 - "KrknOperatorTargetProviderConfigStatus"
Cohesion: 0.29
Nodes (3): getProviderNames(), KrknOperatorTargetProviderConfigStatus, ProviderConfigData

### Community 88 - "Publisher"
Cohesion: 0.22
Nodes (5): Publisher, TestPublisherReconcile(), TestPublisherReconcileReturnsTransientErrors(), TestPublisherRecoversFromConcurrentCreateAndDelete(), TestPublisherStartRetriesAndReturnsCancellation()

### Community 89 - "ClusterTarget"
Cohesion: 0.18
Nodes (7): ClustersResponse, countClustersFromTargetData(), TestCountClustersFromTargetData(), TestFilterClustersByPermission(), ClusterHealthStatus, ClusterTarget, KrknTargetRequestStatus

### Community 90 - "category_run_configuration.go"
Cohesion: 0.15
Nodes (26): scenarioFileBehavior, GraphScenarioNode, appendGraphSignaturePart(), compareCategoryRunConfigurations(), compareGraphConfigurations(), compareGraphConfigurationsWithStableNodeIDs(), compareGraphNodeConfigurations(), compareGraphRunConfigurations() (+18 more)

### Community 91 - "convertClusterJobs"
Cohesion: 0.16
Nodes (11): ClusterJobStatusResponse, RestoreResponse, RollbackRunSummary, ScenarioRunStatusResponse, convertClusterJobs(), convertMetaTime(), TestConvertClusterJobs_EmptyInput(), TestConvertClusterJobs_NilInput() (+3 more)

### Community 92 - "KrknOperatorTargetProviderList"
Cohesion: 0.21
Nodes (7): KrknOperatorTargetProviderConfigReconciler, Architecture Decision, Components Requiring Leader Election (Leader-Only), Components Running on All Replicas (Multi-Active), isConflictError(), countActiveProviders(), KrknOperatorTargetProviderList

### Community 93 - "watcher_test.go"
Cohesion: 0.19
Nodes (18): hasGraphRunStatusChanged(), hasScenarioRunStatusChanged(), TestHasGraphRunStatusChanged_NewNodeAdded(), TestHasGraphRunStatusChanged_NoChange(), TestHasGraphRunStatusChanged_NodeOrderChanged(), TestHasGraphRunStatusChanged_NodeStatusChange(), TestHasGraphRunStatusChanged_PhaseChange(), TestHasGraphRunStatusChanged_ResiliencyScoresChange() (+10 more)

### Community 95 - "ProviderRegistration"
Cohesion: 0.15
Nodes (12): API Reference, CreateProviderConfigRequest, Interfaces, NewProviderRegistration, NewProviderRegistrationWithConfig, Provider Configuration Functions, Provider Registration Functions, UpdateProviderConfig (+4 more)

### Community 96 - "Provider Configuration API Specification"
Cohesion: 0.08
Nodes (24): 1. List Available Providers (Optional), 2. Create Config Request, 3. Poll for Completion, 4. Parse Schemas and Build UI, 5. Submit Configuration, Base URL, Change Log, Common Error Patterns (+16 more)

### Community 97 - ".extractAndStoreReports"
Cohesion: 0.17
Nodes (10): decodeReportPayloads(), extractDelimitedContent(), TestDecodeReportPayloadsPreservesValidReport(), TestDecodeReportPayloadsReportsInvalidHTML(), TestExtractAndStoreReportsRejectsMultiClusterRun(), TestExtractDelimitedContent(), TestTargetClusterCount(), TestValidateReportPayloadSize() (+2 more)

### Community 99 - "TestKrknGraphRun_ValidateGraph"
Cohesion: 0.33
Nodes (5): SanitizeNodeIDForKubernetes(), stringPtr(), TestCollisionDetection(), TestKrknGraphRun_ValidateGraph(), TestSanitizeNodeIDForKubernetes()

### Community 100 - "makeScenarioRun"
Cohesion: 0.24
Nodes (13): makeGraphRun(), makeScenarioRun(), TestListJobs_GroupedRollbacksBeforePagination(), TestListJobs_IncludesStats(), TestListJobs_LimitOnlyReturnsAll(), TestListJobs_NoPagination(), TestListJobs_Paginated(), TestListJobs_SkipsGraphRunChildren() (+5 more)

### Community 101 - "Authentication API Specification"
Cohesion: 0.09
Nodes (22): 1. Check Admin Registration, 1. Initial Load, 2. First Admin Registration, 2. Register User, 3. Login, 3. Login, 4. Authenticated Requests, 5. Role-Based UI (+14 more)

### Community 102 - "net/http.Handler"
Cohesion: 0.20
Nodes (7): Role, loggingMiddleware(), maxBodySizeMiddleware(), TestMaxBodySizeMiddleware(), handlerOrStatus(), TestCheckClusterLiveness(), testKubeconfig()

### Community 103 - "buildClusterAPIURLMap"
Cohesion: 0.25
Nodes (8): buildClusterAPIURLMap(), contains(), containsInMiddle(), TestBuildClusterAPIURLMap(), TestBuildClusterAPIURLMap_Collision(), TestBuildClusterAPIURLMap_SameNameSameURL(), TestValidateScenarioRunAccess(), TestValidateScenarioRunAccess_NoGroups()

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

### Community 112 - "TokenGenerator"
Cohesion: 0.24
Nodes (4): Claims, mockTokenGen(), mockTokenGenInvalid(), TokenGenerator

### Community 117 - "CLAUDE.md - Project Instructions"
Cohesion: 0.10
Nodes (19): Architecture Principles, Before Committing, Before Writing Code, CLAUDE.md - Project Instructions, Code Organization & Reusability, Code Quality Standards, Communication, Critical Rules (+11 more)

### Community 128 - "Setup"
Cohesion: 0.40
Nodes (5): 1. Create Virtual Environment (Python 3.11), 2. Install Dependencies, 3. Verify Generated Stubs, 4. Start gRPC Server, Setup

### Community 129 - "Implementation Details"
Cohesion: 0.12
Nodes (18): Code Refactoring: getKubeconfigFromTarget() Helper ✅, Implementation Details, Phase 10: POST /scenarios/detail/{scenario_name} Endpoint ✅, Phase 11: POST /scenarios/globals/{scenario_name} Endpoint ✅, Phase 1: REST API Setup ✅, Phase 2: GET /clusters Endpoint ✅, Phase 3: Integration ✅, Phase 4: Testing ✅ (+10 more)

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

### Community 140 - ".translateVolumesToFileMounts"
Cohesion: 0.29
Nodes (3): TestLoadFileConfigMapByID(), TestTranslateVolumesToFileMounts(), FileMount

### Community 141 - "newProvidersHandler"
Cohesion: 0.40
Nodes (5): newProvider(), newProvidersHandler(), TestListProviders_Empty(), TestListProviders_ScopedToNamespace(), TestUpdateProviderStatus()

### Community 142 - "KrknScenarioRunReconciler"
Cohesion: 0.15
Nodes (10): imageResolver, imageSignatureVerifier, KrknScenarioRunReconciler, TestScenarioRunReconcileSkipsPendingMigration(), countJobPhaseChanges(), timeEqual(), backfillLegacyGraphResiliencyScoreStatuses(), TestBackfillLegacyGraphResiliencyScoreStatuses() (+2 more)

### Community 144 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.17
Nodes (12): API and operator refactoring to support multiple clusters, API Endpoints Structure (Nested Approach), BREAKING CHANGES, Implementation Details, ✅ Implemented, KrknTargetRequest controller in operator, Moving from direct scenario creation to CRD approach, Overview (+4 more)

### Community 145 - ".flatten"
Cohesion: 0.18
Nodes (12): rawTelemetrySource, intPtr(), TestFlattenRepresentativeScenario(), TestRawTelemetrySourceFlatten(), contains(), containsAt(), TestMetadataJSONMarshaling(), TestMetadataPresenceTracking() (+4 more)

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

### Community 155 - "Troubleshooting"
Cohesion: 0.40
Nodes (5): Error: "Command not found: kubectl", Error: "failed to connect to gRPC server", Error: "Failed to get kubeconfig", Error: "no module named 'krkn_lib'", Troubleshooting

### Community 156 - ".GetScenarioRunLogs"
Cohesion: 0.33
Nodes (6): filterReportLines(), isReportMarkerEnd(), isReportMarkerStart(), TestFilterReportLines(), TestIsReportMarkerEnd(), TestIsReportMarkerStart()

### Community 159 - ".QueryTelemetryPage"
Cohesion: 0.47
Nodes (7): FacetOption, QueryTelemetryResponse, TelemetryDocument, TelemetryStats, decodeTelemetry(), facetsFromBuckets(), statsFromBuckets()

### Community 160 - "bootstrapOLMResources"
Cohesion: 0.67
Nodes (3): bootstrapOLMResources(), TestBootstrapOLMResourcesPropagatesClientError(), TestBootstrapOLMResourcesWiring()

### Community 163 - "files/validation_test.go"
Cohesion: 0.31
Nodes (8): contains(), findSubstring(), TestValidateFileContent(), TestValidateFileGroups(), TestValidateUserFilePermissions(), ValidateFileContent(), ValidateFileGroups(), ValidateUserFilePermissions()

### Community 164 - "BuildUnifiedJobList"
Cohesion: 0.16
Nodes (11): convertClusterResiliencyScores(), averageResiliencyScore(), convertMetaTime(), BuildUnifiedJobList(), getDefaultPageSize(), GroupRollbackJobs(), runCategoryNames(), scenarioRunImage() (+3 more)

### Community 166 - "krkngraphrun_controller_sanitization_test.go"
Cohesion: 0.83
Nodes (3): isAlphanumeric(), isValidLabelChar(), TestSanitizeNodeID()

### Community 167 - "Pushing Images"
Cohesion: 0.50
Nodes (4): Custom Push, Push Both Images, Push Individual Images, Pushing Images

### Community 168 - "providerConfigTestScheme"
Cohesion: 0.18
Nodes (13): addKnownTypes(), decodeErrorResponse(), providerConfigTestScheme(), TestGetProviderConfigByUUID_CompletedReturnsConfigData(), TestGetProviderConfigByUUID_InternalErrorReturnsSanitized500(), TestGetProviderConfigByUUID_NotFoundReturns404(), TestGetProviderConfigByUUID_PendingReturnsAccepted(), TestPostProviderConfig_FailureReturnsSanitizedError() (+5 more)

### Community 170 - "elasticsearch/labels_test.go"
Cohesion: 0.33
Nodes (5): TestBuildAnnotations(), TestBuildLabels(), TestBuildLabelsAccessControl(), TestUpdateAnnotations(), UpdateAnnotations()

### Community 171 - "generator_test.go"
Cohesion: 0.14
Nodes (22): CreateTargetRequest, UpdateTargetRequest, generateKubeconfigFromCredentialsType(), generateKubeconfigFromKubeconfigType(), generateKubeconfigFromRequest(), generateKubeconfigFromTokenType(), TestCreateTarget_WithKubeconfig(), TestUpdateTarget() (+14 more)

### Community 172 - "sanitizeHeaders"
Cohesion: 0.40
Nodes (4): maskToken(), sanitizeHeaders(), TestMaskToken(), TestSanitizeHeaders()

### Community 173 - "WorkflowResponse"
Cohesion: 0.67
Nodes (3): workflowResponsesHaveCategories(), WorkflowResponse, ListWorkflowsResponse

## Knowledge Gaps
- **17 isolated node(s):** `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME`, `ICON_BASE64`, `ICON_MEDIATYPE` (+12 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 734 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **30 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Handler` connect `Handler` to `main.go`, `ConnectionParams`, `Handler`, `GetClaimsFromContext`, `.UpdateRegistry`, `context.Context`, `BuildUnifiedJobList`, `SecretManager`, `JobTracker`, `k8s.io/api/core/v1.Secret`, `GetClaimsFromContext`, `bootstrap.go`, `.loadCategoryResiliencyHistories`, `GetEnabled`, `github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2`?**
  _High betweenness centrality (0.057) - this node is a cross-community bridge._
- **Are the 160 inferred relationships involving `NewTestHandler()` (e.g. with `setupAuthTestHandler()` and `TestLogin_InactiveUser()`) actually correct?**
  _`NewTestHandler()` has 160 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME` to the rest of the system?**
  _17 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.09254858616560745 - nodes in this community are weakly interconnected._
- **Why does `krkn-operator` connect `krkn-operator` to `.flatten`, `Running Locally`, `Running gRPC Data Provider Locally for Development`?**
  _High betweenness centrality (0.039) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.014632005072428424 - nodes in this community are weakly interconnected._
- **Why does `KrknWorkspace` connect `KrknWorkspace` to `main.go`, `Handler`, `.DeepCopy`, `context.Context`, `k8s.io/apimachinery/pkg/apis/meta/v1.Time`, `GroupLabelKey`, `Deployment`, `.DeepCopyInto`, `GetEnabled`, `.DeepCopyInto`?**
  _High betweenness centrality (0.025) - this node is a cross-community bridge._