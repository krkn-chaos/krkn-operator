# Graph Report - krkn-operator  (2026-11-11)

## Corpus Check
- 337 files · ~434,088 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 18 file(s) not represented in the graph (top: (none) 12, .jsonl 2, .toml 1)

## Summary
- 3661 nodes · 14847 edges · 170 communities (140 shown, 30 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 1389 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `68cf697d`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- main.go
- testing.T
- Handler
- .PostScenarioRun
- GetClaimsFromContext
- context.Context
- NewTestHandler
- cloudcreds_handlers_test.go
- utils.go
- rollbackTestHandler
- api/types.go
- Broadcaster
- migration.go
- addAdminContext
- Handler
- GroupLabelKey
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
- mockScenarioProvider
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- KrknUserGroupSpec
- client.go
- backup_restore_handlers_test.go
- applyResource
- Handler
- ConnectionParams
- KrknOperatorTargetProviderConfig
- injection.go
- registry/labels_test.go
- NewCachedUserStatusChecker
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- ComponentManifest
- bootstrap.go
- UnifiedJobItem
- wsorigin.go
- handlers_test.go
- migration_test.go
- notify-operator-docs.test.js
- converter_test.go
- ssrf.go
- KrknScenarioRunSpec
- Hub
- KrknWorkspace
- NewTokenGenerator
- github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2
- rate_limiter.go
- newReportTestHandler
- cloudcreds/types.go
- GetNodesRequest
- dataprovider_grpc.pb.go
- .checkRollbackImage
- parseResiliencyHistoryQuery
- .DeepCopyInto
- ScenarioRunRequest
- .PostScenarioDetail
- GET methods
- installer_test.go
- 🔧 TODO: Pod Recreation and Retry Logic
- buildContainerImageFromReference
- PROGRESS-MVP.md
- SecretManager
- Publisher
- start_operator.sh
- workspace_handlers_test.go
- Agent Specifications - MVP Archive
- Compute
- sync_test.go
- KrknOperatorTargetProvider
- registry/types.go
- test-api.sh
- Frontend Authentication Implementation Tasks
- ExecuteKubectlRequest
- mockJob
- KrknScenarioRun
- captureLogSink
- krkn-operator Development Progress (Post-MVP)
- Krkn-Operator Roadmap
- .loadCategoryResiliencyHistories
- newCategoryTestUser
- REST API Endpoints
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- ResolveGraph
- buildUnifiedJobList
- SecretManager
- watcher_test.go
- ExecuteKubectlResponse
- ProviderRegistration
- Provider Configuration API Specification
- NewServerWithComponentNamespace
- dataprovider.pb.go
- TestKrknGraphRun_ValidateGraph
- makeScenarioRun
- Authentication API Specification
- Technical Notes
- Post-MVP Development
- setup-ocm-and-operator.sh
- File Management API Specification
- JobTracker
- commands.go
- Cluster Segregation Troubleshooting
- channel-head.sh
- Running gRPC Data Provider Locally for Development
- category_run_configuration_test.go
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
- files/types.go
- TokenGenerator
- .ExecuteTerminal
- CategoryResiliencyHistoryResponse
- 🔧 TODO: Pod Recreation and Retry Logic
- .flatten
- Running Locally
- install-yq.sh
- test-release-workflow-gates.sh
- Deployment
- Available Make Targets
- Building Images
- Troubleshooting
- decodeTelemetry
- KrknOperatorTargetProviderConfigStatus
- Troubleshooting
- .GetScenarioRunLogs
- assets/README.md
- GraphRunResponse
- dataProviderServiceClient
- DataProviderServiceServer
- StripCloudEnvVars
- WorkflowResponse
- files/validation_test.go
- BuildUnifiedJobList
- migrationBlockedError
- krkngraphrun_controller_sanitization_test.go
- Pushing Images
- providerConfigTestScheme
- GenerateFromToken

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

## Communities (170 total, 30 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.09
Nodes (18): availableWorkspacePVC, CategoryEntityAssociationResponse, CreateCategoryRequest, ResiliencyHistoryQueryRequest, UpdateCategoryRequest, workspacePublishRequest, ContextKey, clusterTarget (+10 more)

### Community 1 - "testing.T"
Cohesion: 0.01
Nodes (182): TestKrknUserCreation(), TestKrknUserList(), TestKrknUserRoles(), TestKrknUserStatus(), TestKrknUserValidation(), TestScenarioReferenceValidate(), setupAuthTestHandler(), TestIsRegistered_MethodNotAllowed() (+174 more)

### Community 2 - "Handler"
Cohesion: 0.08
Nodes (19): workspaceInfo, writeCategoryRunNotFoundOrInternalError(), updateCloudCredentialSecretData(), convertGraphClusterScores(), extractGroupNameAndUserIDFromPath(), extractGroupNameFromMembersPath(), convertMetaTime(), extractPathSuffix() (+11 more)

### Community 3 - ".PostScenarioRun"
Cohesion: 0.24
Nodes (7): k8sUserStatusChecker, writeScenarioRunAccessError(), sanitizeResourceName(), sanitizeRunNameLabel(), resolveMaxRetries(), sanitizeUsername(), ConflictsWithMount()

### Community 4 - "GetClaimsFromContext"
Cohesion: 0.06
Nodes (37): buildFileInfo(), buildFileResponse(), deriveLogicalName(), extractLogicalName(), convertConfigMapToWorkflowInfo(), convertFileResponseToWorkflow(), extractIDFromPath(), filterVisibleWorkflowCategories() (+29 more)

### Community 5 - "context.Context"
Cohesion: 0.05
Nodes (57): historyQueryCountingClient, mockUserStatusChecker, BackupConfig, resourceType, RestoreConfig, Code References, removeCategoryLabel(), filterScenarioHistoryScoresByPermission() (+49 more)

### Community 6 - "NewTestHandler"
Cohesion: 0.09
Nodes (64): countingGetClient, esDoerFunc, boolPtr(), newEsAlertsTestSecretWithHost(), newEsScheme(), newEsTestSecret(), newEsTestSecretWithHost(), TestCreateElasticsearchConfig_DuplicateName() (+56 more)

### Community 7 - "cloudcreds_handlers_test.go"
Cohesion: 0.06
Nodes (83): TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(), TestCloudCredentialContract_GraphRunStripsNodeEnv(), newAWSTestSecret(), newCloudCredScheme(), newGCPTestSecret(), newGroupedAWSSecret(), TestCloudCredentialsRouter_MethodNotAllowed(), TestCloudCredentialsRouter_SubpathMethodNotAllowed() (+75 more)

### Community 8 - "utils.go"
Cohesion: 0.14
Nodes (16): tokenRequest, TestE2E(), getMetricsOutput(), GetNonEmptyLines(), GetProjectDir(), InstallCertManager(), InstallPrometheusOperator(), IsCertManagerCRDsInstalled() (+8 more)

### Community 9 - "rollbackTestHandler"
Cohesion: 0.21
Nodes (26): RollbackAvailability, rollbackProbeName(), rollbackRunAttemptName(), rollbackRunName(), decodeRollbackResponse(), rollbackAvailableProbe(), rollbackRegistrySecret(), rollbackRequest() (+18 more)

### Community 10 - "api/types.go"
Cohesion: 0.05
Nodes (44): ActiveRunsOverviewResponse, AddGroupMemberRequest, AddGroupMemberResponse, ChangePasswordRequest, ChangePasswordResponse, ClusterPermissionSet, ClusterResiliencyScoreResponse, CreateTargetResponse (+36 more)

### Community 11 - "Broadcaster"
Cohesion: 0.12
Nodes (11): Handler, NewHandler(), convertNodeStatusesWithScores(), hashBytes(), roleFromAdmin(), visibleCategoryNames(), AuthorizationChecker, Broadcaster (+3 more)

### Community 12 - "migration.go"
Cohesion: 0.13
Nodes (42): clusterJobImage, clusterJobProvider, clusterJobRetries, graphScenarioReferences, migrationPlan, migrationResult, migrationStats, resiliencyScoreStatus (+34 more)

### Community 13 - "addAdminContext"
Cohesion: 0.11
Nodes (42): addAdminContext(), addUserContext(), createTestAdminUser(), setupFilesTestHandler(), TestCanAccessFile(), TestCreateFile(), TestCreateFile_DefaultFilePurpose(), TestCreateFile_DuplicateName() (+34 more)

### Community 14 - "Handler"
Cohesion: 0.08
Nodes (17): cloudCredentialAccessContext, Handler, buildCloudCredentialResponse(), buildConnectionParams(), buildConnectionParamsForIndex(), buildElasticsearchConfigResponse(), canAccessElasticsearchConfig(), validateClusterPermissions() (+9 more)

### Community 15 - "GroupLabelKey"
Cohesion: 0.10
Nodes (19): CategoryListResponse, CategoryResponse, ErrorResponse, buildCategoryResponse(), buildCategoryVisibilityLabels(), categoryErrorCode(), categoryGroups(), categoryIsPublic() (+11 more)

### Community 16 - "render-bundle.sh"
Cohesion: 0.07
Nodes (32): channel_has_entries(), ensure_catalog_channel(), catalog-channel.sh script, update_default_channel(), write_release_config(), catalog_submit_checkout(), catalog_submit_open_pr(), catalog-submit.sh script (+24 more)

### Community 17 - "server.py"
Cohesion: 0.06
Nodes (6): add_DataProviderServiceServicer_to_server(), DataProviderService, DataProviderServiceServicer, DataProviderServiceStub, DataProviderServicer, serve()

### Community 18 - "category_run_configuration.go"
Cohesion: 0.14
Nodes (27): GraphRunCreateRequest, scenarioFileBehavior, GraphScenarioNode, appendGraphSignaturePart(), compareCategoryRunConfigurations(), compareGraphConfigurations(), compareGraphConfigurationsWithStableNodeIDs(), compareGraphNodeConfigurations() (+19 more)

### Community 19 - "NewHub"
Cohesion: 0.10
Nodes (35): NewBroadcaster(), TestBroadcastDashboardUpdate(), TestBroadcastGraphRunDeleted(), TestBroadcastGraphRunUpdate(), TestBroadcastOnlyToSubscribedClients(), TestBroadcastScenarioRunDeleted(), TestBroadcastScenarioRunUpdate(), TestBroadcastScenarioRunUpdate_IncludesSanitizedClusterJobs() (+27 more)

### Community 20 - "WSUnifiedJobItem"
Cohesion: 0.10
Nodes (14): TestFilterJobsByCategoriesMatchesAnySelectedCategory(), TestPaginateJobItems(), computeWSJobStats(), filterJobCategoriesByVisibility(), filterJobsByCategories(), groupRollbackJobs(), paginateJobItems(), TestGroupRollbackJobsWebSocketKeepsFlatViewAndChecksUID() (+6 more)

### Community 21 - "user_handlers_test.go"
Cohesion: 0.14
Nodes (34): createTestUser(), kvValue(), setupUserTestHandler(), TestChangePassword_AdminChange_NoCurrentPassword(), TestChangePassword_AdminChangesOtherUser_EmitsAuditLog(), TestChangePassword_SamePassword_Rejected(), TestChangePassword_SelfChange_NoAdminAuditLog(), TestChangePassword_SelfChange_RequiresCurrentPassword() (+26 more)

### Community 22 - "client_test.go"
Cohesion: 0.10
Nodes (22): doerFunc, roundTripperFunc, NewClient(), scenarioParametersSchema(), TestBaseURL(), TestClientTransportDefaultNotHTTPTransport(), TestClientTransportPooling(), TestConnectionParamsTLSConfig() (+14 more)

### Community 23 - "KrknGraphRun"
Cohesion: 0.16
Nodes (9): KrknGraphRunReconciler, TestGraphRunReconcileSkipsPendingMigration(), TestLoadFileConfigMapByID(), TestTranslateVolumesToFileMounts(), sanitizeNodeID(), TestCalculateResiliencyScore_StoresNodeScores(), overrideRef(), TestGraphRunInvalidImageSignatureFailsWithoutRetry() (+1 more)

### Community 24 - "elasticsearch/types.go"
Cohesion: 0.10
Nodes (27): AffectedPods, CreateElasticsearchConfigRequest, CreateElasticsearchConfigResponse, DeleteElasticsearchConfigResponse, ListElasticsearchConfigsResponse, QueryAlertsRequest, QueryTelemetryRequest, RawJSON (+19 more)

### Community 25 - "installer.go"
Cohesion: 0.09
Nodes (24): bootstrapOLMResources(), main(), resolveMigrationNamespace(), runCRDMigrationMode(), TestResolveMigrationNamespacePrefersHookNamespace(), TestRunCRDMigrationMode(), TestBootstrapOLMResourcesPropagatesClientError(), TestBootstrapOLMResourcesWiring() (+16 more)

### Community 26 - "mockScenarioProvider"
Cohesion: 0.12
Nodes (13): POST methods, refactoring, /scenarios ✅ COMPLETED, /scenarios/detail/{scenario_name} ✅ COMPLETED, /scenarios/globals/{scenario_name} ✅ COMPLETED, /scenarios/run/{UUID}, /targets ✅ COMPLETED, mockScenarioProvider (+5 more)

### Community 27 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.09
Nodes (12): ClustersResponse, KrknTargetRequestReconciler, countClustersFromTargetData(), TestCountClustersFromTargetData(), TestFilterClustersByPermission(), ClusterHealthStatus, ClusterTarget, KrknOperatorTarget (+4 more)

### Community 29 - "client.go"
Cohesion: 0.10
Nodes (27): AlertDocument, esAlertSearchResponse, facetField, filterAggregation, Option, QueryAlertsResponse, StatusError, termsAggregation (+19 more)

### Community 30 - "backup_restore_handlers_test.go"
Cohesion: 0.14
Nodes (23): NewJobTracker(), createMultipartBody(), createTestHandler(), TestGetRestoreStatusAdminOnly(), TestGetRestoreStatusTrackedJob(), TestGetRestoreStatusUnknownJob(), TestHandlerShutdownCancelsContext(), TestJobTrackerEviction() (+15 more)

### Community 31 - "applyResource"
Cohesion: 0.50
Nodes (4): TestApplyResourceRestoresArchivedStatus(), applyResource(), isKrknCRD(), isNotFound()

### Community 33 - "ConnectionParams"
Cohesion: 0.21
Nodes (7): Doer, executeSearch(), Client, ConnectionParams, newSearchRequest(), TestTransportKey(), transportKey()

### Community 34 - "KrknOperatorTargetProviderConfig"
Cohesion: 0.08
Nodes (23): KrknOperatorTargetProviderConfigReconciler, 1. List Providers, 2. Get Provider Config Status, 2. Update Provider Status, 3. Create Provider Config Request, 3. Update Provider Configuration, API Endpoints, Architecture Decision (+15 more)

### Community 35 - "injection.go"
Cohesion: 0.17
Nodes (21): preparedJobResources, TestAppendCloudCredentialInjection_EmptyRefNoop(), injectAWS(), injectAzure(), injectBaremetal(), InjectCredentials(), injectGCP(), injectIBMCloud() (+13 more)

### Community 37 - "registry/labels_test.go"
Cohesion: 0.29
Nodes (6): BuildRegistryAnnotations(), TestBuildRegistryAnnotations(), TestBuildRegistryLabels(), TestExtractGroupsFromLabels(), TestUpdateRegistryAnnotations(), UpdateRegistryAnnotations()

### Community 38 - "NewCachedUserStatusChecker"
Cohesion: 0.16
Nodes (10): CachedUserStatusChecker, countingUserStatusChecker, userStatusCacheEntry, UserStatusChecker, NewCachedUserStatusChecker(), TestCachedUserStatusChecker(), TestCachedUserStatusChecker_EvictsExpiredEntries(), TestCachedUserStatusChecker_Invalidation() (+2 more)

### Community 39 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.18
Nodes (10): GraphRunDetailResponse, GraphRunListItem, GraphRunListResponse, GraphRunSpecResponse, GraphRunStatusResponse, 1.1 Resiliency weight per graph node, 1. Request - Create GraphRun (POST /api/v1/graphruns), 2. Response - List GraphRuns (GET /api/v1/graphruns) (+2 more)

### Community 40 - "ComponentManifest"
Cohesion: 0.15
Nodes (7): ListComponentsResponse, Load(), TestLoad(), TestLoadRejectsInvalidManifest(), ComponentDescription, ComponentManifest, KrknComponentStatus

### Community 41 - "bootstrap.go"
Cohesion: 0.12
Nodes (15): categoryEnvironmentParameter, writeCategoryProfileField(), RemoveElasticsearchGrafanaURLAnnotations(), TestRemoveElasticsearchGrafanaURLAnnotations(), ensureJWTSecret(), EnsureResources(), ensureSCCBinding(), ensureScenarioRunnerRBAC() (+7 more)

### Community 42 - "UnifiedJobItem"
Cohesion: 0.08
Nodes (17): ClusterJobStatusResponse, NodeStatusResponse, PaginationMeta, RestoreResponse, RollbackRunSummary, ScenarioRunListItem, ScenarioRunListResponse, ScenarioRunStatusResponse (+9 more)

### Community 43 - "wsorigin.go"
Cohesion: 0.22
Nodes (13): checkWebSocketOriginV2(), TestCheckWebSocketOriginV2(), checkWebSocketOrigin(), TestCheckWebSocketOrigin(), canonicalOrigin(), defaultPort(), IsAllowedOrigin(), portsMatch() (+5 more)

### Community 44 - "handlers_test.go"
Cohesion: 0.04
Nodes (63): TestLogin_InactiveUser(), TestLogin_InvalidCredentials(), TestLogin_Success(), contains(), containsMiddle(), publicScenarioReference(), TestCreateGraphRun_WithFileReferences(), TestCreateGraphRun_ResiliencyScore() (+55 more)

### Community 45 - "migration_test.go"
Cohesion: 0.20
Nodes (25): Complete(), Stage(), containsSubstring(), countPatchActions(), jsonMarshal(), migrationTestLogger(), newMigrationClient(), scenarioRunFixture() (+17 more)

### Community 46 - "notify-operator-docs.test.js"
Cohesion: 0.10
Nodes (11): assert, chartWorkflow, { execFileSync, spawnSync }, fs, notifier, notifyWorkflow, os, path (+3 more)

### Community 47 - "converter_test.go"
Cohesion: 0.20
Nodes (15): BuildDockerConfigJSON(), ExtractRegistryV2FromSecret(), TestBuildDockerConfigJSON_MissingCredentials(), TestBuildDockerConfigJSON_OAuth(), TestBuildDockerConfigJSON_Password(), TestExtractRegistryV2FromSecret_InvalidType(), TestExtractRegistryV2FromSecret_MissingAnnotations(), TestExtractRegistryV2FromSecret_MissingComponent() (+7 more)

### Community 48 - "ssrf.go"
Cohesion: 0.15
Nodes (10): responseWriter, guardedCheckRedirect(), guardedDialContext(), TestGuardedCheckRedirect(), TestQueryTelemetryAllowsUnrestrictedLoopback(), TestValidateDestinationIP(), TestValidateDestinationURL(), validateDestinationIP() (+2 more)

### Community 49 - "KrknScenarioRunSpec"
Cohesion: 0.25
Nodes (4): KrknScenarioRunSpec, MapScenarioNodeToScenarioRunSpec(), FileMount, RollbackSource

### Community 50 - "Hub"
Cohesion: 0.23
Nodes (7): isWebSocketDisconnectError(), writeWSError(), AuthorizationCheckFunc, BroadcastMessage, Client, Hub, PaginationClientState

### Community 51 - "KrknWorkspace"
Cohesion: 0.09
Nodes (26): Publish and select a workspace, Replay and unavailable storage, Reuse from another operator, Scenario workspaces on a shared PVC, Storage validation, migrationPending(), Available(), Mount() (+18 more)

### Community 52 - "NewTokenGenerator"
Cohesion: 0.08
Nodes (33): TestGroupNameValidation(), TestGroupsRouterEdgeCases(), TestCheckScenarioRunAccess(), TestFilterScenarioRunsByGroupPermission(), TestPostScenarioRun_ClusterNameCollision_Returns500(), TestPostScenarioRun_SameClusterSameURL_NoCollision(), TestPostScenarioRunSetsOwner(), TestClusterSegregationLegacyRunsWithoutJobs() (+25 more)

### Community 53 - "github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2"
Cohesion: 0.21
Nodes (10): resolveRollbackDigest(), resolveImmutableImage(), resolvePrivateImageDigest(), TestResolvePrivateImageDigestUsesManifestDigest(), ResolvePrivateTagDigest(), ResolvePublicTagDigest(), TestResolvePrivateTagDigestMissing(), TestResolvePrivateTagDigestWithAuthentication() (+2 more)

### Community 54 - "rate_limiter.go"
Cohesion: 0.14
Nodes (20): ipRateLimiter, rateLimiterEntry, clientIPFromHeaders(), extractClientIP(), hostFromRemoteAddr(), isTrustedProxy(), newIPRateLimiter(), parseTrustedProxyCIDRs() (+12 more)

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

### Community 59 - ".checkRollbackImage"
Cohesion: 0.39
Nodes (4): rollbackImageCache, rollbackImageCheck, Rollback artifact persistence, Rollback history in Jobs

### Community 60 - "parseResiliencyHistoryQuery"
Cohesion: 0.33
Nodes (5): resiliencyHistoryQuerySelections, normalizeResiliencyHistoryProviders(), normalizeResiliencyHistorySelections(), parseResiliencyHistoryQuery(), extractFile()

### Community 61 - ".DeepCopyInto"
Cohesion: 0.05
Nodes (19): filterUsers(), newKrknUser(), GraphScenario, KrknCategoryList, KrknCategorySpec, KrknComponent, KrknComponentList, KrknGraphRunList (+11 more)

### Community 62 - "ScenarioRunRequest"
Cohesion: 0.16
Nodes (15): Phase 12: Scenario Job Management - /scenarios/run Endpoints ✅, Phase 9: POST /scenarios Endpoint ✅, FileMount, GlobalsRequest, JobsListResponse, JobStatusResponse, ScenarioRunRequest, ScenarioRunResponse (+7 more)

### Community 63 - ".PostScenarioDetail"
Cohesion: 0.31
Nodes (3): convertInputFields(), createScenarioProvider(), getScenarioSignatureStatus()

### Community 64 - "GET methods"
Cohesion: 0.07
Nodes (28): API Requirements, /clusters, DELETE methods, GET methods, GET methods, get nodes, get pods, Grpc python service requirement (+20 more)

### Community 65 - "installer_test.go"
Cohesion: 0.18
Nodes (12): crdClient, applyCRD(), applyCRDs(), TestApplyCRDsCreatesMissingAndUpdatesChangedSpecs(), TestApplyCRDsDoesNotPatchUnchangedSpecs(), TestApplyCRDsLabelsExistingCRDsWithoutChangingOtherLabels(), TestApplyCRDsRejectsNonCRDManifest(), testCRD() (+4 more)

### Community 66 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.22
Nodes (9): Configuration, Current Behavior, Implementation Plan (TODO), Option A: Cancellation Field in Status (Recommended), Option B: Finalizers for Cancellation Tracking, Option C: Separate CancellationRequest CR, Proposed Solution Options, Requirements (+1 more)

### Community 67 - "buildContainerImageFromReference"
Cohesion: 0.17
Nodes (15): publicScenarioReference(), TestCreateScenarioRun_ResiliencyScore(), buildContainerImage(), buildContainerImageFromReference(), getTestConfig(), privateReference(), publicReference(), TestBuildContainerImage_PublicRegistryUsesKrknctl() (+7 more)

### Community 68 - "PROGRESS-MVP.md"
Cohesion: 0.22
Nodes (8): Architecture Decisions, Data Provider Architecture, Future Work (Not in Current Scope), Integration Points, Namespace Configuration, Next Steps, Requirements Overview, REST API Framework

### Community 70 - "Publisher"
Cohesion: 0.09
Nodes (10): Publisher, KrknWorkspaceReconciler, NewNamespaceFilter(), TestWorkspaceReconcilerMapsPVCWithoutOwningIt(), TestWorkspaceReconcilerTracksPVCState(), TestPublisherReconcile(), TestPublisherReconcileReturnsTransientErrors(), TestPublisherRecoversFromConcurrentCreateAndDelete() (+2 more)

### Community 71 - "start_operator.sh"
Cohesion: 0.41
Nodes (12): build_operator(), check_prerequisites(), cleanup(), create_service_account(), install_crds(), main(), print_error(), print_info() (+4 more)

### Community 72 - "workspace_handlers_test.go"
Cohesion: 0.21
Nodes (20): failingWorkspaceGetClient, newCategoryRequestAs(), setupScenarioRunTestHandler(), graphRunCategoryTestHandler(), TestCreateGraphRun_RejectsMissingCategoryBeforeCreatingCR(), TestPostScenarioRun_CategoriesArePersistedOnCreatedCR(), TestPostScenarioRun_RejectsInvalidCategoryNameBeforeCreatingCR(), readyWorkspace() (+12 more)

### Community 73 - "Agent Specifications - MVP Archive"
Cohesion: 0.20
Nodes (7): Agent Specifications - MVP Archive, Files, MVP Completion Date, MVP Features Delivered, Post-MVP Development, create configmap for operator based on KrknOperatorTargetProviderConfig, KrknOperatorTargetProviderConfig

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

### Community 82 - "KrknScenarioRun"
Cohesion: 0.06
Nodes (27): imageResolver, imageSignatureVerifier, KrknScenarioRunReconciler, scenarioRunImage(), TestScenarioRunReconcileSkipsPendingMigration(), backfillLegacyMaxRetries(), countJobPhaseChanges(), effectiveScenarioMaxRetries() (+19 more)

### Community 84 - "krkn-operator Development Progress (Post-MVP)"
Cohesion: 0.22
Nodes (9): API Endpoints Completed, Core Infrastructure, Current Status, Development Phases Template, Future Considerations, Key Technical Achievements, krkn-operator Development Progress (Post-MVP), MVP Completion Summary (+1 more)

### Community 85 - "Krkn-Operator Roadmap"
Cohesion: 0.29
Nodes (6): Krkn-Operator Roadmap, Ongoing (not release-blocked), Release 1 — 2026-08-18, Release 2 — 2026-10-18, Release 3 — 2026-12-22, Release 4 — 2027-02-23

### Community 86 - ".loadCategoryResiliencyHistories"
Cohesion: 0.10
Nodes (25): categoryHistoryRun, categoryHistoryScore, categoryParameterProfileFingerprint(), categoryParameterProfileName(), TestCategoryParameterProfileFingerprintForEmptyEnvironment(), TestCategoryParameterProfileFingerprintGraphEnvironmentMultiset(), TestCategoryParameterProfileFingerprintIgnoresRunType(), TestCategoryParameterProfileFingerprintScenarioEnvironment() (+17 more)

### Community 87 - "newCategoryTestUser"
Cohesion: 0.19
Nodes (21): newCategoryRequest(), newCategoryTestGroup(), newCategoryTestHandler(), newCategoryTestUser(), TestCategoriesRouterCRUD(), TestCategoriesRouterGroupCreationRequiresMembership(), TestCategoriesRouterGroupVisibility(), TestCategoriesRouterNameMaximumLength() (+13 more)

### Community 88 - "REST API Endpoints"
Cohesion: 0.29
Nodes (7): GET /clusters, GET /nodes, GET /targets/{UUID}, POST /scenarios, POST /scenarios/detail/{scenario_name}, POST /targets, REST API Endpoints

### Community 89 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.08
Nodes (10): timeEqual(), GraphRunSummary, KrknGraphRunStatus, KrknOperatorTargetProviderStatus, KrknOperatorTargetStatus, KrknWorkspaceStatus, NodeStatus, ReportStatus (+2 more)

### Community 90 - "ResolveGraph"
Cohesion: 0.17
Nodes (12): FromKrknctlScenarioSet(), publicScenarioReference(), publicScenario(), TestFromKrknctlScenarioSetCreatesPublicReferences(), TestToKrknctlScenarioSetUsesScenarioReference(), ToKrknctlScenarioSet(), ResolveGraph(), contains() (+4 more)

### Community 91 - "buildUnifiedJobList"
Cohesion: 0.17
Nodes (12): TestBuildUnifiedJobList_SkipsGraphRunChildren(), averageResiliencyScore(), buildScenarioRunDetailResponse(), buildScenarioRunResponse(), buildUnifiedJobList(), convertClusterResiliencyScoresWS(), runCategoryNames(), sanitizedClusterJobs() (+4 more)

### Community 92 - "SecretManager"
Cohesion: 0.14
Nodes (7): getTokenDuration(), TestGetTokenDuration(), generateRandomSecret(), getSecretKeys(), SecretManager, TestGenerateRandomSecret(), NewTestSecretManager()

### Community 93 - "watcher_test.go"
Cohesion: 0.19
Nodes (18): hasGraphRunStatusChanged(), hasScenarioRunStatusChanged(), TestHasGraphRunStatusChanged_NewNodeAdded(), TestHasGraphRunStatusChanged_NoChange(), TestHasGraphRunStatusChanged_NodeOrderChanged(), TestHasGraphRunStatusChanged_NodeStatusChange(), TestHasGraphRunStatusChanged_PhaseChange(), TestHasGraphRunStatusChanged_ResiliencyScoresChange() (+10 more)

### Community 95 - "ProviderRegistration"
Cohesion: 0.15
Nodes (12): API Reference, CreateProviderConfigRequest, Interfaces, NewProviderRegistration, NewProviderRegistrationWithConfig, Provider Configuration Functions, Provider Registration Functions, UpdateProviderConfig (+4 more)

### Community 96 - "Provider Configuration API Specification"
Cohesion: 0.08
Nodes (24): 1. List Available Providers (Optional), 2. Create Config Request, 3. Poll for Completion, 4. Parse Schemas and Build UI, 5. Submit Configuration, Base URL, Change Log, Common Error Patterns (+16 more)

### Community 97 - "NewServerWithComponentNamespace"
Cohesion: 0.16
Nodes (12): Role, TestListComponentsRouteAndAuth(), loggingMiddleware(), maxBodySizeMiddleware(), NewServer(), NewServerWithComponentNamespace(), TestMaxBodySizeMiddleware(), Middleware (+4 more)

### Community 98 - "dataprovider.pb.go"
Cohesion: 0.20
Nodes (3): file_proto_dataprovider_proto_init(), file_proto_dataprovider_proto_rawDescGZIP(), init()

### Community 99 - "TestKrknGraphRun_ValidateGraph"
Cohesion: 0.33
Nodes (5): SanitizeNodeIDForKubernetes(), stringPtr(), TestCollisionDetection(), TestKrknGraphRun_ValidateGraph(), TestSanitizeNodeIDForKubernetes()

### Community 100 - "makeScenarioRun"
Cohesion: 0.24
Nodes (13): makeGraphRun(), makeScenarioRun(), TestListJobs_GroupedRollbacksBeforePagination(), TestListJobs_IncludesStats(), TestListJobs_LimitOnlyReturnsAll(), TestListJobs_NoPagination(), TestListJobs_Paginated(), TestListJobs_SkipsGraphRunChildren() (+5 more)

### Community 101 - "Authentication API Specification"
Cohesion: 0.09
Nodes (22): 1. Check Admin Registration, 1. Initial Load, 2. First Admin Registration, 2. Register User, 3. Login, 3. Login, 4. Authenticated Requests, 5. Role-Based UI (+14 more)

### Community 102 - "Technical Notes"
Cohesion: 0.40
Nodes (5): Environment Variables, Implementation Files, KrknTargetRequest Structure, Server Configuration, Technical Notes

### Community 103 - "Post-MVP Development"
Cohesion: 0.40
Nodes (5): Completed, In Progress, Pending, Phase 1: KrknOperatorTarget CRD and CRUD API, Post-MVP Development

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

### Community 112 - "category_run_configuration_test.go"
Cohesion: 0.35
Nodes (13): TestBuildCategoryResiliencyHistoryGroupsTypedConfigurationsAndSortsPerCluster(), TestGraphHistoryScoresOnlyKeepsFinalScoresIncludingZero(), assertConfigurationDifferences(), floatPointer(), graphRunConfigurationFixture(), scenarioReference(), scenarioRunConfigurationFixture(), stringPointer() (+5 more)

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

### Community 140 - "files/types.go"
Cohesion: 0.20
Nodes (9): AvailableFilesResponse, DeleteFileResponse, ListFilesResponse, UpdateFileResponse, TestBuildFileInfo_ExcludesStudioLayout(), TestBuildFileResponse_ContentByPurpose(), TestBuildFileResponse_WorkflowNameFallback(), CreateFileResponse (+1 more)

### Community 141 - "TokenGenerator"
Cohesion: 0.31
Nodes (4): Claims, mockTokenGen(), mockTokenGenInvalid(), TokenGenerator

### Community 142 - ".ExecuteTerminal"
Cohesion: 0.25
Nodes (3): TerminalRequest, TerminalResponse, NewDataProviderServiceClient()

### Community 143 - "CategoryResiliencyHistoryResponse"
Cohesion: 0.38
Nodes (7): CategoryConfigurationGroup, CategoryResiliencyDataPoint, CategoryResiliencyHistoryResponse, ResiliencyHistoryQueryResponse, datapointForRunID(), datapointRunIDs(), assembleResiliencyHistoryQueryResponse()

### Community 144 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.17
Nodes (12): API and operator refactoring to support multiple clusters, API Endpoints Structure (Nested Approach), BREAKING CHANGES, Implementation Details, ✅ Implemented, KrknTargetRequest controller in operator, Moving from direct scenario creation to CRD approach, Overview (+4 more)

### Community 145 - ".flatten"
Cohesion: 0.16
Nodes (15): ClusterMetadata, NodeSummaryInfo, rawTelemetrySource, TelemetryDocument, intPtr(), TestFlattenRepresentativeScenario(), TestRawTelemetrySourceFlatten(), contains() (+7 more)

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

### Community 153 - "decodeTelemetry"
Cohesion: 0.48
Nodes (7): esSearchResponse, FacetOption, QueryTelemetryResponse, TelemetryStats, decodeTelemetry(), facetsFromBuckets(), statsFromBuckets()

### Community 154 - "KrknOperatorTargetProviderConfigStatus"
Cohesion: 0.29
Nodes (3): getProviderNames(), KrknOperatorTargetProviderConfigStatus, ProviderConfigData

### Community 155 - "Troubleshooting"
Cohesion: 0.40
Nodes (5): Error: "Command not found: kubectl", Error: "failed to connect to gRPC server", Error: "Failed to get kubeconfig", Error: "no module named 'krkn_lib'", Troubleshooting

### Community 156 - ".GetScenarioRunLogs"
Cohesion: 0.33
Nodes (6): filterReportLines(), isReportMarkerEnd(), isReportMarkerStart(), TestFilterReportLines(), TestIsReportMarkerEnd(), TestIsReportMarkerStart()

### Community 158 - "GraphRunResponse"
Cohesion: 0.29
Nodes (3): buildGraphRunResponse(), convertGraphClusterScoresForSnapshot(), GraphRunResponse

### Community 161 - "StripCloudEnvVars"
Cohesion: 0.50
Nodes (4): IsCloudEnvVar(), StripCloudEnvVars(), TestIsCloudEnvVar(), TestStripCloudEnvVars()

### Community 162 - "WorkflowResponse"
Cohesion: 0.67
Nodes (3): workflowResponsesHaveCategories(), WorkflowResponse, ListWorkflowsResponse

### Community 163 - "files/validation_test.go"
Cohesion: 0.31
Nodes (8): contains(), findSubstring(), TestValidateFileContent(), TestValidateFileGroups(), TestValidateUserFilePermissions(), ValidateFileContent(), ValidateFileGroups(), ValidateUserFilePermissions()

### Community 164 - "BuildUnifiedJobList"
Cohesion: 0.18
Nodes (9): convertClusterResiliencyScores(), averageResiliencyScore(), BuildUnifiedJobList(), getDefaultPageSize(), GroupRollbackJobs(), runCategoryNames(), TestGroupRollbackJobs_UsesSourceUIDAndKeepsOrphans(), PaginateSlice() (+1 more)

### Community 166 - "krkngraphrun_controller_sanitization_test.go"
Cohesion: 0.83
Nodes (3): isAlphanumeric(), isValidLabelChar(), TestSanitizeNodeID()

### Community 167 - "Pushing Images"
Cohesion: 0.50
Nodes (4): Custom Push, Push Both Images, Push Individual Images, Pushing Images

### Community 168 - "providerConfigTestScheme"
Cohesion: 0.11
Nodes (24): addKnownTypes(), decodeErrorResponse(), providerConfigTestScheme(), TestGetProviderConfigByUUID_CompletedReturnsConfigData(), TestGetProviderConfigByUUID_InternalErrorReturnsSanitized500(), TestGetProviderConfigByUUID_NotFoundReturns404(), TestGetProviderConfigByUUID_PendingReturnsAccepted(), TestPostProviderConfig_FailureReturnsSanitizedError() (+16 more)

### Community 171 - "GenerateFromToken"
Cohesion: 0.14
Nodes (22): CreateTargetRequest, UpdateTargetRequest, generateKubeconfigFromCredentialsType(), generateKubeconfigFromKubeconfigType(), generateKubeconfigFromRequest(), generateKubeconfigFromTokenType(), TestCreateTarget_DuplicateClusterName(), TestUpdateTarget() (+14 more)

## Knowledge Gaps
- **17 isolated node(s):** `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME`, `ICON_BASE64`, `ICON_MEDIATYPE` (+12 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 732 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **30 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Handler` connect `Handler` to `main.go`, `Handler`, `.PostScenarioRun`, `GetClaimsFromContext`, `context.Context`, `.ExecuteTerminal`, `GroupLabelKey`, `backup_restore_handlers_test.go`, `ConnectionParams`, `BuildUnifiedJobList`, `bootstrap.go`, `github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2`, `.checkRollbackImage`, `.PostScenarioDetail`, `SecretManager`, `KrknScenarioRun`, `.loadCategoryResiliencyHistories`, `SecretManager`, `JobTracker`?**
  _High betweenness centrality (0.053) - this node is a cross-community bridge._
- **Are the 160 inferred relationships involving `NewTestHandler()` (e.g. with `setupAuthTestHandler()` and `TestLogin_InactiveUser()`) actually correct?**
  _`NewTestHandler()` has 160 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME` to the rest of the system?**
  _17 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.09163123741329156 - nodes in this community are weakly interconnected._
- **Why does `krkn-operator` connect `krkn-operator` to `.flatten`, `Running Locally`, `Running gRPC Data Provider Locally for Development`?**
  _High betweenness centrality (0.030) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.014620548412788714 - nodes in this community are weakly interconnected._
- **Why does `KrknWorkspace` connect `KrknWorkspace` to `main.go`, `Handler`, `.DeepCopy`, `context.Context`, `workspace_handlers_test.go`, `Deployment`, `k8s.io/apimachinery/pkg/apis/meta/v1.Time`, `k8s.io/apimachinery/pkg/apis/meta/v1.Time`, `.DeepCopyInto`?**
  _High betweenness centrality (0.023) - this node is a cross-community bridge._