# Graph Report - krkn-operator  (2026-11-05)

## Corpus Check
- 335 files · ~429,169 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 18 file(s) not represented in the graph (top: (none) 12, .jsonl 2, .toml 1)

## Summary
- 3630 nodes · 14655 edges · 169 communities (142 shown, 27 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 1366 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `58efc7e3`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- main.go
- testing.T
- Handler
- rollbackTestHandler
- GetClaimsFromContext
- context.Context
- NewTestHandler
- createAdminContext
- backup.go
- publicScenarioReference
- api/types.go
- Broadcaster
- migration.go
- addAdminContext
- handlers_test.go
- KrknCategory
- render-bundle.sh
- server.py
- sync_test.go
- NewHub
- WSUnifiedJobItem
- user_handlers_test.go
- client_test.go
- KrknGraphRun
- elasticsearch/types.go
- installer.go
- installer_test.go
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- scenarioRunConfigurationFixture
- client.go
- backup_restore_handlers_test.go
- newCategoryRequestAs
- Handler
- ConnectionParams
- KrknOperatorTargetProviderConfig
- injection.go
- .DeepCopy
- k8s.io/api/core/v1.Secret
- Middleware
- time.Time
- ComponentManifest
- k8s.io/apimachinery/pkg/apis/meta/v1/unstructured.Unstructured
- UnifiedJobItem
- wsorigin.go
- parseResiliencyHistoryQuery
- migration_test.go
- notify-operator-docs.test.js
- converter_test.go
- ssrf.go
- .canAccessWorkspace
- Hub
- buildScenarioRunResponse
- k8s.io/apimachinery/pkg/runtime.Scheme
- ScenarioReference
- BuildUnifiedJobList
- files/types.go
- cloudcreds/types.go
- GetNodesRequest
- dataprovider_grpc.pb.go
- .PostScenarioRun
- .checkRollbackImage
- .DeepCopyInto
- ScenarioRunRequest
- category_run_configuration.go
- ValidateCreateRequest
- compareGraphNodeConfigurations
- 🔧 TODO: Pod Recreation and Retry Logic
- metadata_presence_test.go
- rate_limiter_test.go
- SecretManager
- sigs.k8s.io/controller-runtime.Manager
- start_operator.sh
- buildClusterAPIURLMap
- Agent Specifications - MVP Archive
- Compute
- KrknScenarioRunSpec
- KrknOperatorTargetProvider
- registry/types.go
- test-api.sh
- Frontend Authentication Implementation Tasks
- ExecuteKubectlRequest
- mockJob
- .ExecuteTerminal
- captureLogSink
- GraphClusterScore
- Krkn-Operator Roadmap
- GroupLabelKey
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- CategoryResiliencyHistoryResponse
- GET methods
- ResolveGraph
- time.Time
- elasticsearch/labels_test.go
- watcher_test.go
- ExecuteKubectlResponse
- ProviderRegistration
- Provider Configuration API Specification
- newProvidersHandler
- dataprovider.pb.go
- TestKrknGraphRun_ValidateGraph
- makeScenarioRun
- Authentication API Specification
- generateKubeconfigFromRequest
- SanitizeUserIDForLabel
- setup-ocm-and-operator.sh
- File Management API Specification
- TokenGenerator
- commands.go
- Cluster Segregation Troubleshooting
- channel-head.sh
- Running gRPC Data Provider Locally for Development
- ScenarioReference
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
- PROGRESS-MVP.md
- JobTracker
- KrknScenarioRun
- extractTarGz
- 🔧 TODO: Pod Recreation and Retry Logic
- Scenario workspaces on a shared PVC
- Running Locally
- install-yq.sh
- test-release-workflow-gates.sh
- Deployment
- Available Make Targets
- Building Images
- Troubleshooting
- krkn-operator Development Progress (Post-MVP)
- KrknUserGroup
- .GetScenarioRunLogs
- assets/README.md
- REST API Endpoints
- GraphRunResponse
- dataProviderServiceClient
- Technical Notes
- Post-MVP Development
- files/validation_test.go
- DataProviderServiceServer
- manifest_test.go
- krkngraphrun_controller_sanitization_test.go
- WorkflowResponse
- setupTestHandler

## God Nodes (most connected - your core abstractions)
1. `Handler` - 212 edges
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

## Communities (169 total, 27 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.09
Nodes (19): availableWorkspacePVC, CategoryEntityAssociationResponse, CreateCategoryRequest, ResiliencyHistoryQueryRequest, scenarioFileBehavior, UpdateCategoryRequest, workspacePublishRequest, ContextKey (+11 more)

### Community 1 - "testing.T"
Cohesion: 0.01
Nodes (226): TestKrknUserCreation(), TestKrknUserList(), TestKrknUserRoles(), TestKrknUserStatus(), TestKrknUserValidation(), TestScenarioReferenceValidate(), bootstrapOLMResources(), NewConfigStoreInitializer() (+218 more)

### Community 2 - "Handler"
Cohesion: 0.08
Nodes (13): Handler, writeCategoryRunNotFoundOrInternalError(), extractGroupNameAndUserIDFromPath(), extractGroupNameFromMembersPath(), convertMetaTime(), createScenarioProvider(), extractPathSuffix(), writeJSON() (+5 more)

### Community 3 - "rollbackTestHandler"
Cohesion: 0.21
Nodes (25): RollbackAvailability, rollbackProbeName(), rollbackRunAttemptName(), rollbackRunName(), decodeRollbackResponse(), rollbackAvailableProbe(), rollbackRegistrySecret(), rollbackRequest() (+17 more)

### Community 4 - "GetClaimsFromContext"
Cohesion: 0.06
Nodes (39): buildFileInfo(), buildFileResponse(), deriveLogicalName(), extractLogicalName(), isValidConfigMapKey(), validateCreateFileRequest(), validateFileFields(), validateUpdateFileRequest() (+31 more)

### Community 5 - "context.Context"
Cohesion: 0.04
Nodes (57): historyQueryCountingClient, k8sUserStatusChecker, mockScenarioProvider, mockUserStatusChecker, BackupConfig, resourceType, RestoreConfig, InvalidImageSignatureError (+49 more)

### Community 6 - "NewTestHandler"
Cohesion: 0.08
Nodes (82): countingGetClient, esDoerFunc, boolPtr(), newEsAlertsTestSecretWithHost(), newEsScheme(), newEsTestSecret(), newEsTestSecretWithHost(), TestCreateElasticsearchConfig_DuplicateName() (+74 more)

### Community 7 - "createAdminContext"
Cohesion: 0.08
Nodes (57): TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(), TestCloudCredentialContract_GraphRunStripsNodeEnv(), newAWSTestSecret(), newCloudCredScheme(), newGCPTestSecret(), newGroupedAWSSecret(), TestCloudCredentialsRouter_MethodNotAllowed(), TestCloudCredentialsRouter_SubpathMethodNotAllowed() (+49 more)

### Community 8 - "backup.go"
Cohesion: 0.14
Nodes (16): tokenRequest, TestE2E(), getMetricsOutput(), GetNonEmptyLines(), GetProjectDir(), InstallCertManager(), InstallPrometheusOperator(), IsCertManagerCRDsInstalled() (+8 more)

### Community 9 - "publicScenarioReference"
Cohesion: 0.08
Nodes (30): contains(), containsMiddle(), publicScenarioReference(), TestCreateGraphRun_WithFileReferences(), TestCreateGraphRun_ResiliencyScore(), TestGetScenarioRunStatus_IncludesCustomRunName(), TestListScenarioRuns_FilterByScenarioName(), TestListScenarioRuns_IncludesCustomRunName() (+22 more)

### Community 10 - "api/types.go"
Cohesion: 0.05
Nodes (43): ActiveRunsOverviewResponse, AddGroupMemberRequest, AddGroupMemberResponse, ChangePasswordRequest, ChangePasswordResponse, ClusterPermissionSet, ClusterResiliencyScoreResponse, ClustersResponse (+35 more)

### Community 11 - "Broadcaster"
Cohesion: 0.17
Nodes (6): hashBytes(), roleFromAdmin(), AuthorizationCheckFunc, Broadcaster, graphRunEventHandler, scenarioRunEventHandler

### Community 12 - "migration.go"
Cohesion: 0.15
Nodes (35): clusterJobImage, clusterJobProvider, clusterJobRetries, graphScenarioReferences, migrationBlockedError, migrationPlan, resiliencyScoreStatus, addProvider() (+27 more)

### Community 13 - "addAdminContext"
Cohesion: 0.11
Nodes (43): addAdminContext(), addUserContext(), createTestAdminUser(), setupFilesTestHandler(), TestCanAccessFile(), TestCreateFile(), TestCreateFile_DefaultFilePurpose(), TestCreateFile_DuplicateName() (+35 more)

### Community 14 - "handlers_test.go"
Cohesion: 0.05
Nodes (43): TestLogin_InactiveUser(), TestLogin_InvalidCredentials(), TestLogin_Success(), convertInputFields(), maskToken(), sanitizeHeaders(), TestConvertInputFields(), TestDeleteTargetByUUID_AdminCanDeleteAnyResource() (+35 more)

### Community 15 - "KrknCategory"
Cohesion: 0.12
Nodes (18): CategoryListResponse, CategoryResponse, ErrorResponse, buildCategoryResponse(), buildCategoryVisibilityLabels(), categoryErrorCode(), categoryGroups(), categoryIsPublic() (+10 more)

### Community 16 - "render-bundle.sh"
Cohesion: 0.07
Nodes (32): channel_has_entries(), ensure_catalog_channel(), catalog-channel.sh script, update_default_channel(), write_release_config(), catalog_submit_checkout(), catalog_submit_open_pr(), catalog-submit.sh script (+24 more)

### Community 17 - "server.py"
Cohesion: 0.06
Nodes (6): add_DataProviderServiceServicer_to_server(), DataProviderService, DataProviderServiceServicer, DataProviderServiceStub, DataProviderServicer, serve()

### Community 18 - "sync_test.go"
Cohesion: 0.16
Nodes (10): ConfigStoreInitializer, SyncConfigMapToStore(), TestRoundTrip(), TestSyncConfigMapToStore(), TestSyncConfigMapToStore_NilStore(), TestWriteConfigMapData(), WriteConfigMapData(), BackfillLegacyProviderConfigLabel() (+2 more)

### Community 19 - "NewHub"
Cohesion: 0.14
Nodes (28): NewHandler(), NewBroadcaster(), TestBroadcastDashboardUpdate(), TestBroadcastGraphRunDeleted(), TestBroadcastGraphRunUpdate(), TestBroadcastOnlyToSubscribedClients(), TestBroadcastScenarioRunDeleted(), TestBroadcastScenarioRunUpdate() (+20 more)

### Community 20 - "WSUnifiedJobItem"
Cohesion: 0.12
Nodes (11): TestFilterJobsByCategoriesMatchesAnySelectedCategory(), TestPaginateJobItems(), computeWSJobStats(), filterJobCategoriesByVisibility(), filterJobsByCategories(), paginateJobItems(), ServerMessage, WSJobStatsSummary (+3 more)

### Community 21 - "user_handlers_test.go"
Cohesion: 0.14
Nodes (34): createTestUser(), kvValue(), setupUserTestHandler(), TestChangePassword_AdminChange_NoCurrentPassword(), TestChangePassword_AdminChangesOtherUser_EmitsAuditLog(), TestChangePassword_SamePassword_Rejected(), TestChangePassword_SelfChange_NoAdminAuditLog(), TestChangePassword_SelfChange_RequiresCurrentPassword() (+26 more)

### Community 22 - "client_test.go"
Cohesion: 0.11
Nodes (24): doerFunc, Option, roundTripperFunc, NewClient(), scenarioParametersSchema(), TestBaseURL(), TestClientTransportDefaultNotHTTPTransport(), TestClientTransportPooling() (+16 more)

### Community 23 - "KrknGraphRun"
Cohesion: 0.16
Nodes (9): KrknGraphRunReconciler, TestGraphRunReconcileSkipsPendingMigration(), TestLoadFileConfigMapByID(), TestTranslateVolumesToFileMounts(), sanitizeNodeID(), TestCalculateResiliencyScore_StoresNodeScores(), overrideRef(), TestGraphRunInvalidImageSignatureFailsWithoutRetry() (+1 more)

### Community 24 - "elasticsearch/types.go"
Cohesion: 0.14
Nodes (20): AffectedPods, AlertDocument, CreateElasticsearchConfigResponse, DeleteElasticsearchConfigResponse, ListElasticsearchConfigsResponse, QueryAlertsRequest, QueryAlertsResponse, QueryTelemetryRequest (+12 more)

### Community 25 - "installer.go"
Cohesion: 0.12
Nodes (17): main(), resolveMigrationNamespace(), runCRDMigrationMode(), TestResolveMigrationNamespacePrefersHookNamespace(), TestRunCRDMigrationMode(), configMapClient, crdManifest, createMissingCRDs() (+9 more)

### Community 26 - "installer_test.go"
Cohesion: 0.13
Nodes (16): crdClient, deploymentClient, applyCRD(), applyCRDs(), TestApplyCRDsCreatesMissingAndUpdatesChangedSpecs(), TestApplyCRDsDoesNotPatchUnchangedSpecs(), TestApplyCRDsLabelsExistingCRDsWithoutChangingOtherLabels(), TestApplyCRDsRejectsNonCRDManifest() (+8 more)

### Community 27 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.12
Nodes (8): KrknTargetRequestReconciler, TestMarshalUnmarshalSecretData(), TestUnmarshalSecretData_Invalid(), UnmarshalSecretData(), KrknOperatorTarget, KrknOperatorTargetSpec, KrknTargetRequest, KrknTargetRequestSpec

### Community 28 - "scenarioRunConfigurationFixture"
Cohesion: 0.13
Nodes (30): categoryHistoryRun, categoryParameterProfileFingerprint(), categoryParameterProfileName(), TestCategoryParameterProfileFingerprintForEmptyEnvironment(), TestCategoryParameterProfileFingerprintGraphEnvironmentMultiset(), TestCategoryParameterProfileFingerprintIgnoresRunType(), TestCategoryParameterProfileFingerprintScenarioEnvironment(), TestCategoryParameterProfileNameIsDeterministicAndReadable() (+22 more)

### Community 29 - "client.go"
Cohesion: 0.11
Nodes (24): esAlertSearchResponse, esSearchResponse, facetField, filterAggregation, StatusError, termsAggregation, buildAlertsSearchBody(), buildFacetAggs() (+16 more)

### Community 30 - "backup_restore_handlers_test.go"
Cohesion: 0.06
Nodes (38): categoryEnvironmentParameter, NewJobTracker(), createMultipartBody(), createTestHandler(), TestGetRestoreStatusAdminOnly(), TestGetRestoreStatusTrackedJob(), TestGetRestoreStatusUnknownJob(), TestHandlerShutdownCancelsContext() (+30 more)

### Community 31 - "newCategoryRequestAs"
Cohesion: 0.22
Nodes (15): newCategoryRequestAs(), setupScenarioRunTestHandler(), TestPostScenarioRun_CategoriesArePersistedOnCreatedCR(), TestPostScenarioRun_RejectsInvalidCategoryNameBeforeCreatingCR(), readyWorkspace(), TestGraphRunPinsNodeWorkspace(), TestListAvailableWorkspacePVCs(), TestScenarioRollbackRequiresWorkspaceAndPersistsSelection() (+7 more)

### Community 32 - "Handler"
Cohesion: 0.22
Nodes (5): visibleCategoryNames(), AuthorizationChecker, categoryVisibilityChecker, ClientMessage, Handler

### Community 33 - "ConnectionParams"
Cohesion: 0.17
Nodes (14): Doer, FacetOption, QueryTelemetryResponse, TelemetryDocument, TelemetryStats, decodeTelemetry(), executeSearch(), facetsFromBuckets() (+6 more)

### Community 34 - "KrknOperatorTargetProviderConfig"
Cohesion: 0.08
Nodes (21): KrknOperatorTargetProviderConfigReconciler, 1. List Providers, 2. Get Provider Config Status, 2. Update Provider Status, 3. Create Provider Config Request, 3. Update Provider Configuration, API Endpoints, Architecture Decision (+13 more)

### Community 35 - "injection.go"
Cohesion: 0.17
Nodes (22): preparedJobResources, TestAppendCloudCredentialInjection_EmptyRefNoop(), TestAppendCloudCredentialInjection_UsesSecretKeyRef(), injectAWS(), injectAzure(), injectBaremetal(), InjectCredentials(), injectGCP() (+14 more)

### Community 37 - "k8s.io/api/core/v1.Secret"
Cohesion: 0.14
Nodes (15): cloudCredentialAccessContext, buildCloudCredentialResponse(), updateCloudCredentialSecretData(), buildConnectionParams(), buildConnectionParamsForIndex(), buildElasticsearchConfigResponse(), canAccessElasticsearchConfig(), buildRegistryInfo() (+7 more)

### Community 38 - "Middleware"
Cohesion: 0.25
Nodes (8): countingUserStatusChecker, UserStatusChecker, NewCachedUserStatusChecker(), TestCachedUserStatusChecker(), TestCachedUserStatusChecker_EvictsExpiredEntries(), TestCachedUserStatusChecker_Invalidation(), TestNewCachedUserStatusChecker_NilCheckerPanics(), TestNewCachedUserStatusChecker_NonPositiveTTLDefaults()

### Community 39 - "time.Time"
Cohesion: 0.18
Nodes (10): GraphRunDetailResponse, GraphRunListItem, GraphRunListResponse, GraphRunSpecResponse, GraphRunStatusResponse, 1.1 Resiliency weight per graph node, 1. Request - Create GraphRun (POST /api/v1/graphruns), 2. Response - List GraphRuns (GET /api/v1/graphruns) (+2 more)

### Community 40 - "ComponentManifest"
Cohesion: 0.20
Nodes (4): ListComponentsResponse, ComponentDescription, ComponentManifest, KrknComponentStatus

### Community 41 - "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured.Unstructured"
Cohesion: 0.17
Nodes (13): migrationResult, migrationStats, resourceDefinition, checkAnnotationSize(), forEachResource(), logSummary(), patchMigrationResourceWithRetry(), writeMigrationAnnotation() (+5 more)

### Community 42 - "UnifiedJobItem"
Cohesion: 0.11
Nodes (8): NodeStatusResponse, PaginationMeta, ScenarioRunListItem, ScenarioRunListResponse, UnifiedJobItem, UnifiedJobsResponse, ComputeJobStats(), TestComputeJobStats()

### Community 43 - "wsorigin.go"
Cohesion: 0.22
Nodes (13): checkWebSocketOriginV2(), TestCheckWebSocketOriginV2(), checkWebSocketOrigin(), TestCheckWebSocketOrigin(), canonicalOrigin(), defaultPort(), IsAllowedOrigin(), portsMatch() (+5 more)

### Community 44 - "parseResiliencyHistoryQuery"
Cohesion: 0.33
Nodes (5): resiliencyHistoryQuerySelections, normalizeResiliencyHistoryProviders(), normalizeResiliencyHistorySelections(), parseResiliencyHistoryQuery(), extractFile()

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

### Community 49 - ".canAccessWorkspace"
Cohesion: 0.31
Nodes (4): workspaceInfo, workspaceAvailable(), workspaceInfoFromCR(), WorkspaceReference

### Community 50 - "Hub"
Cohesion: 0.13
Nodes (8): Store, isWebSocketDisconnectError(), writeWSError(), Get(), BroadcastMessage, Client, Hub, PaginationClientState

### Community 51 - "buildScenarioRunResponse"
Cohesion: 0.17
Nodes (12): TestBuildUnifiedJobList_SkipsGraphRunChildren(), averageResiliencyScore(), buildScenarioRunDetailResponse(), buildScenarioRunResponse(), buildUnifiedJobList(), convertClusterResiliencyScoresWS(), runCategoryNames(), sanitizedClusterJobs() (+4 more)

### Community 52 - "k8s.io/apimachinery/pkg/runtime.Scheme"
Cohesion: 0.10
Nodes (24): addKnownTypes(), decodeErrorResponse(), providerConfigTestScheme(), TestGetProviderConfigByUUID_CompletedReturnsConfigData(), TestGetProviderConfigByUUID_InternalErrorReturnsSanitized500(), TestGetProviderConfigByUUID_NotFoundReturns404(), TestGetProviderConfigByUUID_PendingReturnsAccepted(), TestPostProviderConfig_FailureReturnsSanitizedError() (+16 more)

### Community 53 - "ScenarioReference"
Cohesion: 0.29
Nodes (10): buildContainerImage(), buildContainerImageFromReference(), getTestConfig(), privateReference(), publicReference(), TestBuildContainerImage_PublicRegistryUsesKrknctl(), TestBuildContainerImage_RejectsPrivateWithoutResolvedRegistry(), TestBuildContainerImageFromReference_PrivateRegistry() (+2 more)

### Community 54 - "BuildUnifiedJobList"
Cohesion: 0.14
Nodes (10): convertClusterResiliencyScores(), convertGraphClusterScores(), averageResiliencyScore(), BuildUnifiedJobList(), getDefaultPageSize(), runCategoryNames(), scenarioRunImage(), PaginateSlice() (+2 more)

### Community 55 - "files/types.go"
Cohesion: 0.20
Nodes (9): AvailableFilesResponse, DeleteFileResponse, ListFilesResponse, UpdateFileResponse, TestBuildFileInfo_ExcludesStudioLayout(), TestBuildFileResponse_ContentByPurpose(), TestBuildFileResponse_WorkflowNameFallback(), CreateFileResponse (+1 more)

### Community 56 - "cloudcreds/types.go"
Cohesion: 0.14
Nodes (15): CreateCloudCredentialResponse, DeleteCloudCredentialResponse, ListCloudCredentialsResponse, UpdateCloudCredentialResponse, buildCloudCredentialSecretData(), CloudCredentialResponse, CreateCloudCredentialRequest, UpdateCloudCredentialRequest (+7 more)

### Community 57 - "GetNodesRequest"
Cohesion: 0.15
Nodes (3): DataProviderServiceClient, GetNodesRequest, GetNodesResponse

### Community 58 - "dataprovider_grpc.pb.go"
Cohesion: 0.16
Nodes (5): UnimplementedDataProviderServiceServer, UnsafeDataProviderServiceServer, _DataProviderService_ExecuteKubectl_Handler(), _DataProviderService_GetNodes_Handler(), RegisterDataProviderServiceServer()

### Community 59 - ".PostScenarioRun"
Cohesion: 0.08
Nodes (20): Code References, writeScenarioRunAccessError(), validateClusterPermissions(), preStartFailureReason(), sanitizeRunNameLabel(), TestPreStartFailureReason(), resolveMaxRetries(), AggregateClusterPermissions() (+12 more)

### Community 60 - ".checkRollbackImage"
Cohesion: 0.48
Nodes (3): rollbackImageCache, rollbackImageCheck, Rollback artifact persistence

### Community 61 - ".DeepCopyInto"
Cohesion: 0.05
Nodes (22): filterUsers(), CanAccess(), TestCanAccess(), GraphScenario, KrknCategoryList, KrknCategorySpec, KrknComponent, KrknComponentList (+14 more)

### Community 62 - "ScenarioRunRequest"
Cohesion: 0.16
Nodes (15): Phase 12: Scenario Job Management - /scenarios/run Endpoints ✅, Phase 9: POST /scenarios Endpoint ✅, FileMount, GlobalsRequest, JobsListResponse, JobStatusResponse, ScenarioRunRequest, ScenarioRunResponse (+7 more)

### Community 63 - "category_run_configuration.go"
Cohesion: 0.23
Nodes (13): GraphRunCreateRequest, GraphScenarioNode, appendGraphSignaturePart(), compareGraphConfigurations(), graphConfigurationSignature(), graphNodeConfigurationSignature(), sameGraphConfigurationIgnoringNodeIDs(), sameGraphNodeIDs() (+5 more)

### Community 64 - "ValidateCreateRequest"
Cohesion: 0.25
Nodes (9): CreateElasticsearchConfigRequest, UpdateElasticsearchConfigRequest, strPtr(), TestValidateUpdateRequest(), validateAccess(), ValidateCreateRequest(), ValidateMergedConnection(), validateTLSSettings() (+1 more)

### Community 65 - "compareGraphNodeConfigurations"
Cohesion: 0.25
Nodes (9): compareGraphConfigurationsWithStableNodeIDs(), compareGraphNodeConfigurations(), compareScenarioRunConfigurations(), compareStringMaps(), optionalBoolValue(), optionalStringValue(), sameGraphVolumeMounts(), sameScenarioFiles() (+1 more)

### Community 66 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.22
Nodes (9): Configuration, Current Behavior, Implementation Plan (TODO), Option A: Cancellation Field in Status (Recommended), Option B: Finalizers for Cancellation Tracking, Option C: Separate CancellationRequest CR, Proposed Solution Options, Requirements (+1 more)

### Community 67 - "metadata_presence_test.go"
Cohesion: 0.18
Nodes (13): ClusterMetadata, NodeSummaryInfo, rawTelemetrySource, intPtr(), TestFlattenRepresentativeScenario(), TestRawTelemetrySourceFlatten(), contains(), containsAt() (+5 more)

### Community 68 - "rate_limiter_test.go"
Cohesion: 0.13
Nodes (20): ipRateLimiter, rateLimiterEntry, clientIPFromHeaders(), extractClientIP(), hostFromRemoteAddr(), isTrustedProxy(), newIPRateLimiter(), parseTrustedProxyCIDRs() (+12 more)

### Community 69 - "SecretManager"
Cohesion: 0.11
Nodes (10): Server, TestListComponentsRouteAndAuth(), NewServer(), NewServerWithComponentNamespace(), Handler, generateRandomSecret(), getSecretKeys(), SecretManager (+2 more)

### Community 70 - "sigs.k8s.io/controller-runtime.Manager"
Cohesion: 0.09
Nodes (10): Publisher, KrknWorkspaceReconciler, NewNamespaceFilter(), TestWorkspaceReconcilerMapsPVCWithoutOwningIt(), TestWorkspaceReconcilerTracksPVCState(), TestPublisherReconcile(), TestPublisherReconcileReturnsTransientErrors(), TestPublisherRecoversFromConcurrentCreateAndDelete() (+2 more)

### Community 71 - "start_operator.sh"
Cohesion: 0.41
Nodes (12): build_operator(), check_prerequisites(), cleanup(), create_service_account(), install_crds(), main(), print_error(), print_info() (+4 more)

### Community 72 - "buildClusterAPIURLMap"
Cohesion: 0.25
Nodes (8): buildClusterAPIURLMap(), contains(), containsInMiddle(), TestBuildClusterAPIURLMap(), TestBuildClusterAPIURLMap_Collision(), TestBuildClusterAPIURLMap_SameNameSameURL(), TestValidateScenarioRunAccess(), TestValidateScenarioRunAccess_NoGroups()

### Community 73 - "Agent Specifications - MVP Archive"
Cohesion: 0.20
Nodes (7): Agent Specifications - MVP Archive, Files, MVP Completion Date, MVP Features Delivered, Post-MVP Development, create configmap for operator based on KrknOperatorTargetProviderConfig, KrknOperatorTargetProviderConfig

### Community 74 - "Compute"
Cohesion: 0.30
Nodes (3): Compute(), JobCounter, Summary

### Community 75 - "KrknScenarioRunSpec"
Cohesion: 0.29
Nodes (3): KrknScenarioRunSpec, FileMount, RollbackSource

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

### Community 82 - ".ExecuteTerminal"
Cohesion: 0.29
Nodes (3): TerminalRequest, TerminalResponse, NewDataProviderServiceClient()

### Community 84 - "GraphClusterScore"
Cohesion: 0.09
Nodes (8): convertNodeStatusesWithScores(), convertNodeStatusesWithScores(), GraphClusterScore, GraphRunSummary, KrknGraphRunStatus, KrknScenarioRunStatus, NodeStatus, ReportStatus

### Community 85 - "Krkn-Operator Roadmap"
Cohesion: 0.29
Nodes (6): Krkn-Operator Roadmap, Ongoing (not release-blocked), Release 1 — 2026-08-18, Release 2 — 2026-10-18, Release 3 — 2026-12-22, Release 4 — 2027-02-23

### Community 86 - "GroupLabelKey"
Cohesion: 0.23
Nodes (19): newCategoryRequest(), newCategoryTestGroup(), newCategoryTestHandler(), newCategoryTestUser(), TestCategoriesRouterCRUD(), TestCategoriesRouterGroupCreationRequiresMembership(), TestCategoriesRouterGroupVisibility(), TestCategoriesRouterNameMaximumLength() (+11 more)

### Community 87 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.09
Nodes (10): getProviderNames(), ClusterHealthStatus, ClusterTarget, KrknOperatorTargetProviderConfigStatus, KrknOperatorTargetProviderStatus, KrknOperatorTargetStatus, KrknTargetRequestStatus, ProviderConfigData (+2 more)

### Community 88 - "CategoryResiliencyHistoryResponse"
Cohesion: 0.38
Nodes (7): CategoryConfigurationGroup, CategoryResiliencyDataPoint, CategoryResiliencyHistoryResponse, ResiliencyHistoryQueryResponse, datapointForRunID(), datapointRunIDs(), assembleResiliencyHistoryQueryResponse()

### Community 89 - "GET methods"
Cohesion: 0.06
Nodes (35): API Requirements, /clusters, DELETE methods, GET methods, GET methods, get nodes, get pods, Grpc python service requirement (+27 more)

### Community 90 - "ResolveGraph"
Cohesion: 0.15
Nodes (13): FromKrknctlScenarioSet(), publicScenarioReference(), publicScenario(), TestFromKrknctlScenarioSetCreatesPublicReferences(), TestToKrknctlScenarioSetUsesScenarioReference(), ToKrknctlScenarioSet(), MapScenarioNodeToScenarioRunSpec(), ResolveGraph() (+5 more)

### Community 91 - "time.Time"
Cohesion: 0.12
Nodes (14): ClusterJobStatusResponse, ListTargetsResponse, RestoreResponse, ScenarioRunStatusResponse, TargetResponse, CachedUserStatusChecker, userStatusCacheEntry, convertClusterJobs() (+6 more)

### Community 92 - "elasticsearch/labels_test.go"
Cohesion: 0.33
Nodes (5): TestBuildAnnotations(), TestBuildLabels(), TestBuildLabelsAccessControl(), TestUpdateAnnotations(), UpdateAnnotations()

### Community 93 - "watcher_test.go"
Cohesion: 0.19
Nodes (18): hasGraphRunStatusChanged(), hasScenarioRunStatusChanged(), TestHasGraphRunStatusChanged_NewNodeAdded(), TestHasGraphRunStatusChanged_NoChange(), TestHasGraphRunStatusChanged_NodeOrderChanged(), TestHasGraphRunStatusChanged_NodeStatusChange(), TestHasGraphRunStatusChanged_PhaseChange(), TestHasGraphRunStatusChanged_ResiliencyScoresChange() (+10 more)

### Community 95 - "ProviderRegistration"
Cohesion: 0.11
Nodes (14): getTokenDuration(), TestGetTokenDuration(), API Reference, CreateProviderConfigRequest, Interfaces, NewProviderRegistration, NewProviderRegistrationWithConfig, Provider Configuration Functions (+6 more)

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

### Community 102 - "generateKubeconfigFromRequest"
Cohesion: 0.53
Nodes (6): CreateTargetRequest, UpdateTargetRequest, generateKubeconfigFromCredentialsType(), generateKubeconfigFromKubeconfigType(), generateKubeconfigFromRequest(), generateKubeconfigFromTokenType()

### Community 103 - "SanitizeUserIDForLabel"
Cohesion: 0.40
Nodes (6): TestSanitizeUserIDForLabel(), SanitizeUserIDForLabel(), SanitizeUserIDForResourceName(), TestSanitizeUserIDConsistency(), TestSanitizeUserIDForLabel(), TestSanitizeUserIDForResourceName()

### Community 104 - "setup-ocm-and-operator.sh"
Cohesion: 0.67
Nodes (5): log_error(), log_info(), log_warn(), setup-ocm-and-operator.sh script, wait_for_condition()

### Community 105 - "File Management API Specification"
Cohesion: 0.09
Nodes (22): Access Control, API Endpoints, ConfigMap Structure, Content Validation, Create a Public Configuration File, Create File, Delete File, Error Responses (+14 more)

### Community 107 - "TokenGenerator"
Cohesion: 0.11
Nodes (13): Claims, Role, loggingMiddleware(), maxBodySizeMiddleware(), TestMaxBodySizeMiddleware(), mockTokenGen(), mockTokenGenInvalid(), TokenGenerator (+5 more)

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

### Community 112 - "ScenarioReference"
Cohesion: 0.33
Nodes (3): publicScenarioReference(), TestCreateScenarioRun_ResiliencyScore(), ScenarioReference

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

### Community 140 - "PROGRESS-MVP.md"
Cohesion: 0.22
Nodes (8): Architecture Decisions, Data Provider Architecture, Future Work (Not in Current Scope), Integration Points, Namespace Configuration, Next Steps, Requirements Overview, REST API Framework

### Community 142 - "KrknScenarioRun"
Cohesion: 0.06
Nodes (28): imageResolver, imageSignatureVerifier, KrknScenarioRunReconciler, migrationPending(), TestScenarioRunReconcileSkipsPendingMigration(), backfillLegacyMaxRetries(), countJobPhaseChanges(), getOwnerLabel() (+20 more)

### Community 143 - "extractTarGz"
Cohesion: 0.40
Nodes (5): createTarGz(), TestCreateAndExtractTarGz(), TestCreateTarGzFlushErrors(), TestExtractTarGzPathTraversal(), extractTarGz()

### Community 144 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.17
Nodes (12): API and operator refactoring to support multiple clusters, API Endpoints Structure (Nested Approach), BREAKING CHANGES, Implementation Details, ✅ Implemented, KrknTargetRequest controller in operator, Moving from direct scenario creation to CRD approach, Overview (+4 more)

### Community 145 - "Scenario workspaces on a shared PVC"
Cohesion: 0.22
Nodes (4): Publish and select a workspace, Replay and unavailable storage, Scenario workspaces on a shared PVC, Storage validation

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

### Community 153 - "krkn-operator Development Progress (Post-MVP)"
Cohesion: 0.22
Nodes (9): API Endpoints Completed, Core Infrastructure, Current Status, Development Phases Template, Future Considerations, Key Technical Achievements, krkn-operator Development Progress (Post-MVP), MVP Completion Summary (+1 more)

### Community 154 - "KrknUserGroup"
Cohesion: 0.15
Nodes (12): categoryHistoryScore, containsString(), filterCategoryHistoryScoresByCluster(), filterGraphHistoryScoresByPermission(), filterScenarioHistoryScoresByPermission(), graphHistoryScores(), runHasAnyCategory(), scenarioHistoryScores() (+4 more)

### Community 156 - ".GetScenarioRunLogs"
Cohesion: 0.33
Nodes (6): filterReportLines(), isReportMarkerEnd(), isReportMarkerStart(), TestFilterReportLines(), TestIsReportMarkerEnd(), TestIsReportMarkerStart()

### Community 158 - "REST API Endpoints"
Cohesion: 0.29
Nodes (7): GET /clusters, GET /nodes, GET /targets/{UUID}, POST /scenarios, POST /scenarios/detail/{scenario_name}, POST /targets, REST API Endpoints

### Community 159 - "GraphRunResponse"
Cohesion: 0.29
Nodes (3): buildGraphRunResponse(), convertGraphClusterScoresForSnapshot(), GraphRunResponse

### Community 161 - "Technical Notes"
Cohesion: 0.40
Nodes (5): Environment Variables, Implementation Files, KrknTargetRequest Structure, Server Configuration, Technical Notes

### Community 162 - "Post-MVP Development"
Cohesion: 0.40
Nodes (5): Completed, In Progress, Pending, Phase 1: KrknOperatorTarget CRD and CRUD API, Post-MVP Development

### Community 163 - "files/validation_test.go"
Cohesion: 0.31
Nodes (8): contains(), findSubstring(), TestValidateFileContent(), TestValidateFileGroups(), TestValidateUserFilePermissions(), ValidateFileContent(), ValidateFileGroups(), ValidateUserFilePermissions()

### Community 165 - "manifest_test.go"
Cohesion: 0.67
Nodes (3): Load(), TestLoad(), TestLoadRejectsInvalidManifest()

### Community 166 - "krkngraphrun_controller_sanitization_test.go"
Cohesion: 0.83
Nodes (3): isAlphanumeric(), isValidLabelChar(), TestSanitizeNodeID()

### Community 167 - "WorkflowResponse"
Cohesion: 0.67
Nodes (3): workflowResponsesHaveCategories(), WorkflowResponse, ListWorkflowsResponse

### Community 171 - "setupTestHandler"
Cohesion: 0.10
Nodes (21): setupTestHandler(), TestCreateTarget_DuplicateClusterName(), TestCreateTarget_MissingRequiredFields(), TestCreateTarget_WithCredentials(), TestCreateTarget_WithKubeconfig(), TestCreateTarget_WithToken(), TestDeleteQuietly_MissingObjectIsNoOp(), TestDeleteQuietly_RunsWithCanceledContext() (+13 more)

## Knowledge Gaps
- **17 isolated node(s):** `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME`, `ICON_BASE64`, `ICON_MEDIATYPE` (+12 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 734 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **27 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Handler` connect `Handler` to `main.go`, `ConnectionParams`, `GetClaimsFromContext`, `k8s.io/api/core/v1.Secret`, `context.Context`, `SecretManager`, `JobTracker`, `KrknScenarioRun`, `KrknCategory`, `.canAccessWorkspace`, `.ExecuteTerminal`, `BuildUnifiedJobList`, `KrknUserGroup`, `.PostScenarioRun`, `.checkRollbackImage`, `backup_restore_handlers_test.go`?**
  _High betweenness centrality (0.068) - this node is a cross-community bridge._
- **Are the 159 inferred relationships involving `NewTestHandler()` (e.g. with `setupAuthTestHandler()` and `TestLogin_InactiveUser()`) actually correct?**
  _`NewTestHandler()` has 159 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME` to the rest of the system?**
  _17 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.09120394395433316 - nodes in this community are weakly interconnected._
- **Why does `krkn-operator` connect `krkn-operator` to `Scenario workspaces on a shared PVC`, `Running Locally`?**
  _High betweenness centrality (0.031) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.012651141961486789 - nodes in this community are weakly interconnected._
- **Why does `Running gRPC Data Provider Locally for Development` connect `Running gRPC Data Provider Locally for Development` to `Setup`, `Scenario workspaces on a shared PVC`?**
  _High betweenness centrality (0.025) - this node is a cross-community bridge._