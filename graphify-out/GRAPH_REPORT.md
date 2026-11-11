# Graph Report - krkn-operator  (2026-11-11)

## Corpus Check
- 337 files · ~433,999 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 18 file(s) not represented in the graph (top: (none) 12, .jsonl 2, .toml 1)

## Summary
- 3660 nodes · 14841 edges · 162 communities (134 shown, 28 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 1386 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `221ff442`
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
- GroupLabelKey
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
- k8s.io/apimachinery/pkg/apis/meta/v1/unstructured.Unstructured
- Handler
- ConnectionParams
- KrknOperatorTargetProviderConfig
- injection.go
- .UpdateRegistry
- NewCachedUserStatusChecker
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- ComponentManifest
- category_run_configuration_test.go
- UnifiedJobItem
- wsorigin.go
- handlers_test.go
- migration_test.go
- notify-operator-docs.test.js
- converter_test.go
- guardedDialContext
- KrknScenarioRunSpec
- Hub
- elasticsearch/validate.go
- .updateClusterJobStatuses
- github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2
- rate_limiter.go
- newReportTestHandler
- cloudcreds/types.go
- GetNodesRequest
- dataprovider_grpc.pb.go
- sigs.k8s.io/controller-runtime/pkg/client.Client
- backup.go
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
- sync_test.go
- Agent Specifications - MVP Archive
- Compute
- sync_test.go
- KrknOperatorTargetProvider
- registry/types.go
- test-api.sh
- Frontend Authentication Implementation Tasks
- ExecuteKubectlRequest
- mockJob
- KrknScenarioRunReconciler
- captureLogSink
- krkn-operator Development Progress (Post-MVP)
- Krkn-Operator Roadmap
- GraphClusterScore
- .getClusterAPIURL
- REST API Endpoints
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- GraphScenarioNode
- ClusterJobStatus
- scenario_filter_test.go
- watcher_test.go
- ExecuteKubectlResponse
- ProviderRegistration
- Provider Configuration API Specification
- .extractAndStoreReports
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
- API compatibility notes
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
- .shouldRetryJob
- newProvidersHandler
- parseBoolHeader
- 🔧 TODO: Pod Recreation and Retry Logic
- .flatten
- Running Locally
- install-yq.sh
- test-release-workflow-gates.sh
- Deployment
- Available Make Targets
- Building Images
- Troubleshooting
- Troubleshooting
- .GetScenarioRunLogs
- assets/README.md
- KrknCategory
- files/validation_test.go
- Handler
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

## Communities (162 total, 28 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.09
Nodes (18): availableWorkspacePVC, CategoryEntityAssociationResponse, CreateCategoryRequest, ResiliencyHistoryQueryRequest, UpdateCategoryRequest, workspacePublishRequest, ContextKey, clusterTarget (+10 more)

### Community 1 - "testing.T"
Cohesion: 0.01
Nodes (208): TestKrknUserCreation(), TestKrknUserList(), TestKrknUserRoles(), TestKrknUserStatus(), TestKrknUserValidation(), TestScenarioReferenceValidate(), NewConfigStoreInitializer(), TestConfigStoreInitializer_NeedLeaderElection() (+200 more)

### Community 2 - "Handler"
Cohesion: 0.07
Nodes (28): ErrorResponse, workspaceInfo, writeCategoryRunNotFoundOrInternalError(), buildCloudCredentialResponse(), updateCloudCredentialSecretData(), convertGraphClusterScores(), buildUserGroupResponse(), extractGroupNameAndUserIDFromPath() (+20 more)

### Community 3 - ".PostScenarioRun"
Cohesion: 0.28
Nodes (7): writeScenarioRunAccessError(), sanitizeRunNameLabel(), resolveMaxRetries(), IsCloudEnvVar(), StripCloudEnvVars(), TestIsCloudEnvVar(), TestStripCloudEnvVars()

### Community 4 - "GetClaimsFromContext"
Cohesion: 0.05
Nodes (48): AvailableFilesResponse, DeleteFileResponse, ListFilesResponse, UpdateFileResponse, buildFileInfo(), buildFileResponse(), deriveLogicalName(), extractLogicalName() (+40 more)

### Community 5 - "context.Context"
Cohesion: 0.07
Nodes (26): mockUserStatusChecker, Code References, filterScenarioHistoryScoresByPermission(), isValidConfigMapKey(), validateFileFields(), effectiveScenarioMaxRetries(), getOwnerLabel(), rollbackConfigTemplateCheck() (+18 more)

### Community 6 - "NewTestHandler"
Cohesion: 0.11
Nodes (69): countingGetClient, esDoerFunc, boolPtr(), newEsAlertsTestSecretWithHost(), newEsScheme(), newEsTestSecret(), newEsTestSecretWithHost(), TestCreateElasticsearchConfig_DuplicateName() (+61 more)

### Community 7 - "cloudcreds_handlers_test.go"
Cohesion: 0.07
Nodes (60): TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(), TestCloudCredentialContract_GraphRunStripsNodeEnv(), newAWSTestSecret(), newCloudCredScheme(), newGCPTestSecret(), newGroupedAWSSecret(), TestCloudCredentialsRouter_MethodNotAllowed(), TestCloudCredentialsRouter_SubpathMethodNotAllowed() (+52 more)

### Community 8 - "utils.go"
Cohesion: 0.14
Nodes (16): tokenRequest, TestE2E(), getMetricsOutput(), GetNonEmptyLines(), GetProjectDir(), InstallCertManager(), InstallPrometheusOperator(), IsCertManagerCRDsInstalled() (+8 more)

### Community 9 - "GroupLabelKey"
Cohesion: 0.06
Nodes (74): RollbackAvailability, newCategoryRequest(), newCategoryRequestAs(), newCategoryTestGroup(), newCategoryTestHandler(), newCategoryTestUser(), TestCategoriesRouterCRUD(), TestCategoriesRouterGroupCreationRequiresMembership() (+66 more)

### Community 10 - "api/types.go"
Cohesion: 0.05
Nodes (45): ActiveRunsOverviewResponse, AddGroupMemberRequest, AddGroupMemberResponse, ChangePasswordRequest, ChangePasswordResponse, ClusterPermissionSet, ClusterResiliencyScoreResponse, CreateTargetResponse (+37 more)

### Community 11 - "Broadcaster"
Cohesion: 0.17
Nodes (6): hashBytes(), roleFromAdmin(), AuthorizationCheckFunc, Broadcaster, graphRunEventHandler, scenarioRunEventHandler

### Community 12 - "migration.go"
Cohesion: 0.15
Nodes (35): clusterJobImage, clusterJobProvider, clusterJobRetries, graphScenarioReferences, migrationBlockedError, migrationPlan, resiliencyScoreStatus, addProvider() (+27 more)

### Community 13 - "addAdminContext"
Cohesion: 0.11
Nodes (42): addAdminContext(), addUserContext(), createTestAdminUser(), setupFilesTestHandler(), TestCanAccessFile(), TestCreateFile(), TestCreateFile_DefaultFilePurpose(), TestCreateFile_DuplicateName() (+34 more)

### Community 14 - "k8s.io/api/core/v1.Secret"
Cohesion: 0.12
Nodes (12): cloudCredentialAccessContext, buildConnectionParams(), buildConnectionParamsForIndex(), buildElasticsearchConfigResponse(), canAccessElasticsearchConfig(), ExtractGroupsFromLabels(), ResolveAccessControlForUpdate(), TestExtractGroupsFromLabels() (+4 more)

### Community 15 - "GetClaimsFromContext"
Cohesion: 0.10
Nodes (14): CategoryListResponse, CategoryResponse, TerminalRequest, TerminalResponse, buildCategoryResponse(), buildCategoryVisibilityLabels(), categoryErrorCode(), replaceCategoryVisibilityLabels() (+6 more)

### Community 16 - "render-bundle.sh"
Cohesion: 0.07
Nodes (32): channel_has_entries(), ensure_catalog_channel(), catalog-channel.sh script, update_default_channel(), write_release_config(), catalog_submit_checkout(), catalog_submit_open_pr(), catalog-submit.sh script (+24 more)

### Community 17 - "server.py"
Cohesion: 0.06
Nodes (6): add_DataProviderServiceServicer_to_server(), DataProviderService, DataProviderServiceServicer, DataProviderServiceStub, DataProviderServicer, serve()

### Community 18 - "category_run_configuration.go"
Cohesion: 0.21
Nodes (20): scenarioFileBehavior, appendGraphSignaturePart(), compareCategoryRunConfigurations(), compareGraphConfigurations(), compareGraphConfigurationsWithStableNodeIDs(), compareGraphNodeConfigurations(), compareGraphRunConfigurations(), compareScenarioRunConfigurations() (+12 more)

### Community 19 - "NewHub"
Cohesion: 0.09
Nodes (37): NewHandler(), NewBroadcaster(), TestBroadcastDashboardUpdate(), TestBroadcastGraphRunDeleted(), TestBroadcastGraphRunUpdate(), TestBroadcastOnlyToSubscribedClients(), TestBroadcastScenarioRunDeleted(), TestBroadcastScenarioRunUpdate() (+29 more)

### Community 20 - "WSUnifiedJobItem"
Cohesion: 0.05
Nodes (28): TestBuildUnifiedJobList_SkipsGraphRunChildren(), TestFilterJobsByCategoriesMatchesAnySelectedCategory(), TestPaginateJobItems(), averageResiliencyScore(), buildGraphRunResponse(), buildScenarioRunDetailResponse(), buildScenarioRunResponse(), buildUnifiedJobList() (+20 more)

### Community 21 - "user_handlers_test.go"
Cohesion: 0.14
Nodes (34): createTestUser(), kvValue(), setupUserTestHandler(), TestChangePassword_AdminChange_NoCurrentPassword(), TestChangePassword_AdminChangesOtherUser_EmitsAuditLog(), TestChangePassword_SamePassword_Rejected(), TestChangePassword_SelfChange_NoAdminAuditLog(), TestChangePassword_SelfChange_RequiresCurrentPassword() (+26 more)

### Community 22 - "client_test.go"
Cohesion: 0.09
Nodes (25): doerFunc, Option, roundTripperFunc, NewClient(), scenarioParametersSchema(), TestBaseURL(), TestClientTransportDefaultNotHTTPTransport(), TestClientTransportPooling() (+17 more)

### Community 23 - "KrknGraphRun"
Cohesion: 0.14
Nodes (11): KrknGraphRunReconciler, TestGraphRunReconcileSkipsPendingMigration(), TestLoadFileConfigMapByID(), TestTranslateVolumesToFileMounts(), publicScenarioReference(), TestCreateScenarioRun_ResiliencyScore(), sanitizeNodeID(), TestCalculateResiliencyScore_StoresNodeScores() (+3 more)

### Community 24 - "elasticsearch/types.go"
Cohesion: 0.13
Nodes (22): AffectedPods, AlertDocument, CreateElasticsearchConfigResponse, DeleteElasticsearchConfigResponse, ListElasticsearchConfigsResponse, QueryAlertsRequest, QueryAlertsResponse, QueryTelemetryRequest (+14 more)

### Community 25 - "installer.go"
Cohesion: 0.09
Nodes (24): bootstrapOLMResources(), main(), resolveMigrationNamespace(), runCRDMigrationMode(), TestResolveMigrationNamespacePrefersHookNamespace(), TestRunCRDMigrationMode(), TestBootstrapOLMResourcesPropagatesClientError(), TestBootstrapOLMResourcesWiring() (+16 more)

### Community 26 - "mockScenarioProvider"
Cohesion: 0.16
Nodes (9): POST methods, refactoring, /scenarios ✅ COMPLETED, /scenarios/detail/{scenario_name} ✅ COMPLETED, /scenarios/globals/{scenario_name} ✅ COMPLETED, /scenarios/run/{UUID}, /targets ✅ COMPLETED, mockScenarioProvider (+1 more)

### Community 27 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.09
Nodes (13): KrknOperatorTargetProviderConfigReconciler, KrknTargetRequestReconciler, Architecture Decision, Components Requiring Leader Election (Leader-Only), Components Running on All Replicas (Multi-Active), isConflictError(), checkProviderActive(), countActiveProviders() (+5 more)

### Community 29 - "client.go"
Cohesion: 0.13
Nodes (22): esAlertSearchResponse, esSearchResponse, facetField, filterAggregation, StatusError, termsAggregation, buildAlertsSearchBody(), buildFacetAggs() (+14 more)

### Community 30 - "backup_restore_handlers_test.go"
Cohesion: 0.15
Nodes (22): NewJobTracker(), createMultipartBody(), createTestHandler(), TestGetRestoreStatusAdminOnly(), TestGetRestoreStatusTrackedJob(), TestGetRestoreStatusUnknownJob(), TestHandlerShutdownCancelsContext(), TestJobTrackerEviction() (+14 more)

### Community 31 - "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured.Unstructured"
Cohesion: 0.18
Nodes (13): migrationResult, migrationStats, resourceDefinition, checkAnnotationSize(), forEachResource(), logSummary(), patchMigrationResourceWithRetry(), writeMigrationAnnotation() (+5 more)

### Community 32 - "Handler"
Cohesion: 0.22
Nodes (5): visibleCategoryNames(), AuthorizationChecker, categoryVisibilityChecker, ClientMessage, Handler

### Community 33 - "ConnectionParams"
Cohesion: 0.17
Nodes (15): Doer, FacetOption, QueryTelemetryResponse, TelemetryDocument, TelemetryStats, decodeTelemetry(), executeSearch(), facetsFromBuckets() (+7 more)

### Community 34 - "KrknOperatorTargetProviderConfig"
Cohesion: 0.11
Nodes (17): 1. List Providers, 2. Get Provider Config Status, 2. Update Provider Status, 3. Create Provider Config Request, 3. Update Provider Configuration, API Endpoints, CreateProviderConfigRequest(), UpdateProviderConfig() (+9 more)

### Community 35 - "injection.go"
Cohesion: 0.07
Nodes (45): preparedJobResources, Publish and select a workspace, Replay and unavailable storage, Reuse from another operator, Scenario workspaces on a shared PVC, Storage validation, TestAppendCloudCredentialInjection_EmptyRefNoop(), migrationPending() (+37 more)

### Community 37 - ".UpdateRegistry"
Cohesion: 0.17
Nodes (7): buildRegistryResponse(), validateCreateRegistryRequest(), validateUpdateRegistryRequest(), BuildRegistryLabels(), ExtractGroupsFromLabels(), TestUpdateRegistryAnnotations(), UpdateRegistryAnnotations()

### Community 38 - "NewCachedUserStatusChecker"
Cohesion: 0.06
Nodes (26): CachedUserStatusChecker, Claims, countingUserStatusChecker, Role, userStatusCacheEntry, UserStatusChecker, getTokenDuration(), TestGetTokenDuration() (+18 more)

### Community 39 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.18
Nodes (10): GraphRunDetailResponse, GraphRunListItem, GraphRunListResponse, GraphRunSpecResponse, GraphRunStatusResponse, 1.1 Resiliency weight per graph node, 1. Request - Create GraphRun (POST /api/v1/graphruns), 2. Response - List GraphRuns (GET /api/v1/graphruns) (+2 more)

### Community 40 - "ComponentManifest"
Cohesion: 0.15
Nodes (7): ListComponentsResponse, Load(), TestLoad(), TestLoadRejectsInvalidManifest(), ComponentDescription, ComponentManifest, KrknComponentStatus

### Community 41 - "category_run_configuration_test.go"
Cohesion: 0.05
Nodes (58): CategoryConfigurationGroup, categoryEnvironmentParameter, categoryHistoryRun, categoryHistoryScore, CategoryResiliencyDataPoint, CategoryResiliencyHistoryResponse, ResiliencyHistoryQueryResponse, categoryParameterProfileFingerprint() (+50 more)

### Community 42 - "UnifiedJobItem"
Cohesion: 0.11
Nodes (10): ClusterJobStatusResponse, NodeStatusResponse, PaginationMeta, RestoreResponse, RollbackRunSummary, ScenarioRunListItem, ScenarioRunListResponse, ScenarioRunStatusResponse (+2 more)

### Community 43 - "wsorigin.go"
Cohesion: 0.16
Nodes (17): checkWebSocketOriginV2(), TestCheckWebSocketOriginV2(), checkWebSocketOrigin(), TestCheckWebSocketOrigin(), guardedCheckRedirect(), TestGuardedCheckRedirect(), TestValidateDestinationURL(), validateDestinationURL() (+9 more)

### Community 44 - "handlers_test.go"
Cohesion: 0.04
Nodes (57): TestLogin_InactiveUser(), TestLogin_InvalidCredentials(), TestLogin_Success(), contains(), containsMiddle(), publicScenarioReference(), TestCreateGraphRun_WithFileReferences(), TestCreateGraphRun_ResiliencyScore() (+49 more)

### Community 45 - "migration_test.go"
Cohesion: 0.20
Nodes (25): Complete(), Stage(), containsSubstring(), countPatchActions(), jsonMarshal(), migrationTestLogger(), newMigrationClient(), scenarioRunFixture() (+17 more)

### Community 46 - "notify-operator-docs.test.js"
Cohesion: 0.10
Nodes (11): assert, chartWorkflow, { execFileSync, spawnSync }, fs, notifier, notifyWorkflow, os, path (+3 more)

### Community 47 - "converter_test.go"
Cohesion: 0.18
Nodes (16): resolvePrivateRegistry(), BuildDockerConfigJSON(), ExtractRegistryV2FromSecret(), TestBuildDockerConfigJSON_MissingCredentials(), TestBuildDockerConfigJSON_OAuth(), TestBuildDockerConfigJSON_Password(), TestExtractRegistryV2FromSecret_InvalidType(), TestExtractRegistryV2FromSecret_MissingAnnotations() (+8 more)

### Community 48 - "guardedDialContext"
Cohesion: 0.25
Nodes (4): responseWriter, guardedDialContext(), TestValidateDestinationIP(), validateDestinationIP()

### Community 49 - "KrknScenarioRunSpec"
Cohesion: 0.13
Nodes (11): publicScenario(), TestToKrknctlScenarioSetUsesScenarioReference(), KrknScenarioRunSpec, MapScenarioNodeToScenarioRunSpec(), contains(), stringContains(), strPtr(), TestMapScenarioNodeToScenarioRunSpec() (+3 more)

### Community 50 - "Hub"
Cohesion: 0.24
Nodes (6): isWebSocketDisconnectError(), writeWSError(), BroadcastMessage, Client, Hub, PaginationClientState

### Community 51 - "elasticsearch/validate.go"
Cohesion: 0.22
Nodes (9): CreateElasticsearchConfigRequest, UpdateElasticsearchConfigRequest, strPtr(), TestValidateUpdateRequest(), validateAccess(), ValidateCreateRequest(), ValidateMergedConnection(), validateTLSSettings() (+1 more)

### Community 52 - ".updateClusterJobStatuses"
Cohesion: 0.15
Nodes (10): preStartFailureReason(), TestPreStartFailureReason(), backfillLegacyMaxRetries(), TestBackfillLegacyMaxRetries(), TestFailedAttemptRetainsRollbackIdentityBeforeRetry(), TestUpdateClusterJobStatuses_ClusterNameEmpty(), TestUpdateClusterJobStatuses_PreservesExplicitZero(), TestUpdateClusterJobStatuses_ProviderNameEmpty() (+2 more)

### Community 53 - "github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2"
Cohesion: 0.21
Nodes (10): resolveRollbackDigest(), resolveImmutableImage(), resolvePrivateImageDigest(), TestResolvePrivateImageDigestUsesManifestDigest(), ResolvePrivateTagDigest(), ResolvePublicTagDigest(), TestResolvePrivateTagDigestMissing(), TestResolvePrivateTagDigestWithAuthentication() (+2 more)

### Community 54 - "rate_limiter.go"
Cohesion: 0.12
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
Cohesion: 0.13
Nodes (6): DataProviderServiceServer, UnimplementedDataProviderServiceServer, UnsafeDataProviderServiceServer, _DataProviderService_ExecuteKubectl_Handler(), _DataProviderService_GetNodes_Handler(), RegisterDataProviderServiceServer()

### Community 59 - "sigs.k8s.io/controller-runtime/pkg/client.Client"
Cohesion: 0.06
Nodes (32): failingWorkspaceGetClient, historyQueryCountingClient, k8sUserStatusChecker, rollbackImageCache, rollbackImageCheck, RestoreConfig, Rollback artifact persistence, Rollback history in Jobs (+24 more)

### Community 60 - "backup.go"
Cohesion: 0.09
Nodes (20): resiliencyHistoryQuerySelections, BackupConfig, resourceType, normalizeResiliencyHistoryProviders(), normalizeResiliencyHistorySelections(), parseResiliencyHistoryQuery(), backupResourceType(), CreateBackup() (+12 more)

### Community 61 - ".DeepCopyInto"
Cohesion: 0.04
Nodes (27): filterUsers(), newKrknUser(), CanAccess(), TestCanAccess(), GraphRunSummary, GraphScenario, KrknCategoryList, KrknComponent (+19 more)

### Community 62 - "ScenarioRunRequest"
Cohesion: 0.15
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
Cohesion: 0.20
Nodes (13): buildContainerImage(), buildContainerImageFromReference(), getTestConfig(), privateReference(), publicReference(), TestBuildContainerImage_PublicRegistryUsesKrknctl(), TestBuildContainerImage_RejectsPrivateWithoutResolvedRegistry(), TestBuildContainerImageFromReference_PrivateRegistry() (+5 more)

### Community 68 - "PROGRESS-MVP.md"
Cohesion: 0.22
Nodes (8): Architecture Decisions, Data Provider Architecture, Future Work (Not in Current Scope), Integration Points, Namespace Configuration, Next Steps, Requirements Overview, REST API Framework

### Community 70 - "Publisher"
Cohesion: 0.09
Nodes (10): Publisher, KrknWorkspaceReconciler, NewNamespaceFilter(), TestWorkspaceReconcilerMapsPVCWithoutOwningIt(), TestWorkspaceReconcilerTracksPVCState(), TestPublisherReconcile(), TestPublisherReconcileReturnsTransientErrors(), TestPublisherRecoversFromConcurrentCreateAndDelete() (+2 more)

### Community 71 - "start_operator.sh"
Cohesion: 0.41
Nodes (12): build_operator(), check_prerequisites(), cleanup(), create_service_account(), install_crds(), main(), print_error(), print_info() (+4 more)

### Community 72 - "sync_test.go"
Cohesion: 0.33
Nodes (6): SyncConfigMapToStore(), TestRoundTrip(), TestSyncConfigMapToStore(), TestSyncConfigMapToStore_NilStore(), TestWriteConfigMapData(), WriteConfigMapData()

### Community 73 - "Agent Specifications - MVP Archive"
Cohesion: 0.20
Nodes (7): Agent Specifications - MVP Archive, Files, MVP Completion Date, MVP Features Delivered, Post-MVP Development, create configmap for operator based on KrknOperatorTargetProviderConfig, KrknOperatorTargetProviderConfig

### Community 74 - "Compute"
Cohesion: 0.22
Nodes (5): ComputeJobStats(), TestComputeJobStats(), Compute(), JobCounter, Summary

### Community 75 - "sync_test.go"
Cohesion: 0.15
Nodes (6): ConfigStoreInitializer, Store, Get(), BackfillLegacyProviderConfigLabel(), TestBackfillLegacyProviderConfigLabel(), TestBackfillLegacyProviderConfigLabelMissingConfigMap()

### Community 76 - "KrknOperatorTargetProvider"
Cohesion: 0.08
Nodes (23): Completed, Current Status, In Progress, Pending, CheckClusterLiveness(), TestCheckClusterLiveness(), testKubeconfig(), Basic Usage (Default Configuration) (+15 more)

### Community 77 - "registry/types.go"
Cohesion: 0.21
Nodes (11): CreateRegistryRequest, RegistryInfo, RegistryResponse, UpdateRegistryRequest, AuthEntry, AvailableRegistriesResponse, CreateRegistryResponse, DeleteRegistryResponse (+3 more)

### Community 78 - "test-api.sh"
Cohesion: 0.39
Nodes (11): api_call(), cleanup(), extract_body(), extract_http_code(), log_error(), log_info(), log_step(), log_success() (+3 more)

### Community 79 - "Frontend Authentication Implementation Tasks"
Cohesion: 0.06
Nodes (30): 1. Read the API Documentation, 1. Start the Operator, 2. Check Available Work, 2. Test Registration (First Admin), 3. Start with Foundation Task, 3. Test Login, 401 on Protected Routes, 4. Claim a Task (+22 more)

### Community 82 - "KrknScenarioRunReconciler"
Cohesion: 0.16
Nodes (9): imageResolver, imageSignatureVerifier, KrknScenarioRunReconciler, TestScenarioRunReconcileSkipsPendingMigration(), countJobPhaseChanges(), backfillLegacyGraphResiliencyScoreStatuses(), TestBackfillLegacyGraphResiliencyScoreStatuses(), TestHasIncompleteResiliencyScores() (+1 more)

### Community 84 - "krkn-operator Development Progress (Post-MVP)"
Cohesion: 0.22
Nodes (9): API Endpoints Completed, Core Infrastructure, Current Status, Development Phases Template, Future Considerations, Key Technical Achievements, krkn-operator Development Progress (Post-MVP), MVP Completion Summary (+1 more)

### Community 85 - "Krkn-Operator Roadmap"
Cohesion: 0.29
Nodes (6): Krkn-Operator Roadmap, Ongoing (not release-blocked), Release 1 — 2026-08-18, Release 2 — 2026-10-18, Release 3 — 2026-12-22, Release 4 — 2027-02-23

### Community 86 - "GraphClusterScore"
Cohesion: 0.22
Nodes (4): convertNodeStatusesWithScores(), convertNodeStatusesWithScores(), GraphClusterScore, NodeStatus

### Community 87 - ".getClusterAPIURL"
Cohesion: 0.46
Nodes (7): newTargetRequest(), newTestScheme(), TestGetClusterAPIURL_ClusterNotFound(), TestGetClusterAPIURL_Found(), TestGetClusterAPIURL_ProviderNotFound(), TestGetClusterAPIURL_TargetRequestNotFound(), TestReconcile_FailedJobStatus_IncludesClusterAPIURL()

### Community 88 - "REST API Endpoints"
Cohesion: 0.29
Nodes (7): GET /clusters, GET /nodes, GET /targets/{UUID}, POST /scenarios, POST /scenarios/detail/{scenario_name}, POST /targets, REST API Endpoints

### Community 89 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.08
Nodes (15): ClustersResponse, getProviderNames(), convertMetaTime(), timeEqual(), countClustersFromTargetData(), TestCountClustersFromTargetData(), TestFilterClustersByPermission(), ClusterHealthStatus (+7 more)

### Community 90 - "GraphScenarioNode"
Cohesion: 0.13
Nodes (14): FromKrknctlScenarioSet(), publicScenarioReference(), TestFromKrknctlScenarioSetCreatesPublicReferences(), ToKrknctlScenarioSet(), GraphScenarioNode, normalizeLegacyGraphScenarioNode(), TestNormalizeLegacyPrivateGraphNode(), workflowResponsesHaveCategories() (+6 more)

### Community 91 - "ClusterJobStatus"
Cohesion: 0.18
Nodes (8): convertClusterJobs(), sanitizedClusterJobs(), TestConvertClusterJobs_EmptyInput(), TestConvertClusterJobs_NilInput(), TestConvertClusterJobs_OmitsClusterAPIURL(), ClusterJobStatus, ScenarioAttemptStatus, ClusterJobResponse

### Community 92 - "scenario_filter_test.go"
Cohesion: 0.47
Nodes (4): filterScenariosByIsAScenario(), TestFilterScenariosByIsAScenario(), TestFilterScenariosByIsAScenario_CancelledContext(), TestFilterScenariosByIsAScenario_PreservesTagMetadata()

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
Cohesion: 0.12
Nodes (16): 1. Login and Get JWT Token, 2. Create KrknTargetRequest (or use existing UUID), 3. Test Terminal Command, Cleanup, Configure Operator to Use Local gRPC Server, Debug gRPC Server in PyCharm, Development Workflow, Notes (+8 more)

### Community 112 - "API compatibility notes"
Cohesion: 0.50
Nodes (4): API compatibility notes, Graph node resiliency weights, Log streaming, Retry limits

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

### Community 141 - "newProvidersHandler"
Cohesion: 0.40
Nodes (5): newProvider(), newProvidersHandler(), TestListProviders_Empty(), TestListProviders_ScopedToNamespace(), TestUpdateProviderStatus()

### Community 144 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.17
Nodes (12): API and operator refactoring to support multiple clusters, API Endpoints Structure (Nested Approach), BREAKING CHANGES, Implementation Details, ✅ Implemented, KrknTargetRequest controller in operator, Moving from direct scenario creation to CRD approach, Overview (+4 more)

### Community 145 - ".flatten"
Cohesion: 0.18
Nodes (13): ClusterMetadata, NodeSummaryInfo, rawTelemetrySource, intPtr(), TestFlattenRepresentativeScenario(), TestRawTelemetrySourceFlatten(), contains(), containsAt() (+5 more)

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
Cohesion: 0.24
Nodes (8): filterReportLines(), isReportMarkerEnd(), isReportMarkerStart(), maskToken(), sanitizeHeaders(), TestFilterReportLines(), TestIsReportMarkerEnd(), TestIsReportMarkerStart()

### Community 161 - "KrknCategory"
Cohesion: 0.21
Nodes (8): categoryGroups(), categoryIsPublic(), categoryVisibleToUser(), publicHistoryCategory(), ExtractGroupNamesFromLabels(), TestExtractGroupNamesFromLabels(), KrknCategory, KrknCategorySpec

### Community 163 - "files/validation_test.go"
Cohesion: 0.31
Nodes (8): contains(), findSubstring(), TestValidateFileContent(), TestValidateFileGroups(), TestValidateUserFilePermissions(), ValidateFileContent(), ValidateFileGroups(), ValidateUserFilePermissions()

### Community 164 - "Handler"
Cohesion: 0.08
Nodes (16): Handler, convertClusterResiliencyScores(), averageResiliencyScore(), convertMetaTime(), BuildUnifiedJobList(), getDefaultPageSize(), GroupRollbackJobs(), runCategoryNames() (+8 more)

### Community 166 - "krkngraphrun_controller_sanitization_test.go"
Cohesion: 0.83
Nodes (3): isAlphanumeric(), isValidLabelChar(), TestSanitizeNodeID()

### Community 167 - "Pushing Images"
Cohesion: 0.50
Nodes (4): Custom Push, Push Both Images, Push Individual Images, Pushing Images

### Community 168 - "providerConfigTestScheme"
Cohesion: 0.18
Nodes (13): addKnownTypes(), decodeErrorResponse(), providerConfigTestScheme(), TestGetProviderConfigByUUID_CompletedReturnsConfigData(), TestGetProviderConfigByUUID_InternalErrorReturnsSanitized500(), TestGetProviderConfigByUUID_NotFoundReturns404(), TestGetProviderConfigByUUID_PendingReturnsAccepted(), TestPostProviderConfig_FailureReturnsSanitizedError() (+5 more)

### Community 171 - "GenerateFromToken"
Cohesion: 0.18
Nodes (16): CreateTargetRequest, UpdateTargetRequest, generateKubeconfigFromCredentialsType(), generateKubeconfigFromKubeconfigType(), generateKubeconfigFromRequest(), generateKubeconfigFromTokenType(), TestCreateTarget_DuplicateClusterName(), TestCreateTarget_WithKubeconfig() (+8 more)

## Knowledge Gaps
- **17 isolated node(s):** `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME`, `ICON_BASE64`, `ICON_MEDIATYPE` (+12 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 732 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **28 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Handler` connect `Handler` to `main.go`, `ConnectionParams`, `Handler`, `.PostScenarioRun`, `GetClaimsFromContext`, `context.Context`, `.UpdateRegistry`, `SecretManager`, `NewCachedUserStatusChecker`, `category_run_configuration_test.go`, `JobTracker`, `k8s.io/api/core/v1.Secret`, `GetClaimsFromContext`, `github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2`, `sigs.k8s.io/controller-runtime/pkg/client.Client`, `.GetScenarioRunLogs`, `.PostScenarioDetail`?**
  _High betweenness centrality (0.056) - this node is a cross-community bridge._
- **Are the 160 inferred relationships involving `NewTestHandler()` (e.g. with `setupAuthTestHandler()` and `TestLogin_InactiveUser()`) actually correct?**
  _`NewTestHandler()` has 160 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME` to the rest of the system?**
  _17 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.09375118708452042 - nodes in this community are weakly interconnected._
- **Why does `krkn-operator` connect `krkn-operator` to `API compatibility notes`, `Running Locally`?**
  _High betweenness centrality (0.029) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.014441563086828221 - nodes in this community are weakly interconnected._
- **Why does `KrknWorkspace` connect `.DeepCopyInto` to `main.go`, `Handler`, `injection.go`, `.DeepCopy`, `context.Context`, `GroupLabelKey`, `Deployment`, `k8s.io/apimachinery/pkg/apis/meta/v1.Time`?**
  _High betweenness centrality (0.023) - this node is a cross-community bridge._