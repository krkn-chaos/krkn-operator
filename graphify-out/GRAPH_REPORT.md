# Graph Report - krkn-operator  (2026-11-04)

## Corpus Check
- 335 files · ~428,496 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 19 file(s) not represented in the graph (top: (none) 13, .jsonl 2, .toml 1)

## Summary
- 3625 nodes · 14601 edges · 161 communities (133 shown, 28 thin omitted)
- Extraction: 91% EXTRACTED · 9% INFERRED · 0% AMBIGUOUS · INFERRED: 1358 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `77aa3b52`
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
- .prepareJobResources
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
- categoryHistoryRun
- client.go
- backup_restore_handlers_test.go
- .updateClusterJobStatuses
- Handler
- ConnectionParams
- KrknOperatorTargetProviderConfig
- injection.go
- .DeepCopy
- k8s.io/api/core/v1.Secret
- NewServerWithComponentNamespace
- time.Time
- ComponentManifest
- bootstrap.go
- UnifiedJobItem
- installer_test.go
- parseResiliencyHistoryQuery
- migration_test.go
- notify-operator-docs.test.js
- .Get
- category_run_configuration_test.go
- ValidatePVC
- Hub
- buildScenarioRunResponse
- k8s.io/apimachinery/pkg/runtime.Scheme
- ScenarioReference
- Handler
- github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2
- cloudcreds/types.go
- GetNodesRequest
- dataprovider_grpc.pb.go
- GetClaimsFromContext
- converter_test.go
- .DeepCopyInto
- ScenarioRunRequest
- category_run_configuration.go
- .extractAndStoreReports
- sanitizeUsername
- 🔧 TODO: Pod Recreation and Retry Logic
- metadata_presence_test.go
- rate_limiter_test.go
- SecretManager
- sigs.k8s.io/controller-runtime.Manager
- start_operator.sh
- .getClusterAPIURL
- PROGRESS-MVP.md
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
- publicScenarioReference
- KrknOperatorTargetProviderConfigStatus
- CategoryResiliencyHistoryResponse
- GET methods
- ResolveGraph
- ScenarioAttemptStatus
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
- Troubleshooting
- Pushing Images
- setup-ocm-and-operator.sh
- File Management API Specification
- API compatibility notes
- commands.go
- Cluster Segregation Troubleshooting
- channel-head.sh
- Running gRPC Data Provider Locally for Development
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
- KrknWorkspace
- JobTracker
- KrknScenarioRunReconciler
- 🔧 TODO: Pod Recreation and Retry Logic
- Running Locally
- install-yq.sh
- test-release-workflow-gates.sh
- Deployment
- Available Make Targets
- Building Images
- Troubleshooting
- Setup
- KrknUserGroup
- .GetScenarioRunLogs
- assets/README.md
- buildClusterAPIURLMap
- dataProviderServiceClient
- file_proto_dataprovider_proto_rawDescGZIP
- files/validation_test.go
- DataProviderServiceServer
- krkngraphrun_controller_sanitization_test.go
- bootstrapOLMResources
- generator.go

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
- `3. Create Provider Config Request` --references--> `KrknOperatorTargetProviderConfig`  [INFERRED]
  docs/api-provider-config-spec.md → api/v1alpha1/krknoperatortargetproviderconfig_types.go
- `Overview` --references--> `KrknOperatorTargetProviderConfig`  [INFERRED]
  docs/api-provider-config-spec.md → api/v1alpha1/krknoperatortargetproviderconfig_types.go

## Import Cycles
- None detected.

## Communities (161 total, 28 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.09
Nodes (15): availableWorkspacePVC, CategoryEntityAssociationResponse, CreateCategoryRequest, ResiliencyHistoryQueryRequest, scenarioFileBehavior, UpdateCategoryRequest, workspacePublishRequest, ContextKey (+7 more)

### Community 1 - "testing.T"
Cohesion: 0.02
Nodes (192): TestKrknUserCreation(), TestKrknUserList(), TestKrknUserRoles(), TestKrknUserStatus(), TestKrknUserValidation(), TestScenarioReferenceValidate(), setupAuthTestHandler(), TestIsRegistered_MethodNotAllowed() (+184 more)

### Community 2 - "Handler"
Cohesion: 0.06
Nodes (29): ErrorResponse, writeCategoryRunNotFoundOrInternalError(), buildCloudCredentialResponse(), convertClusterResiliencyScores(), convertGraphClusterScores(), extractGroupNameAndUserIDFromPath(), extractGroupNameFromMembersPath(), validateClusterPermissions() (+21 more)

### Community 3 - "rollbackTestHandler"
Cohesion: 0.18
Nodes (24): RollbackAvailability, rollbackProbeName(), rollbackRunAttemptName(), rollbackRunName(), decodeRollbackResponse(), rollbackAvailableProbe(), rollbackRegistrySecret(), rollbackRequest() (+16 more)

### Community 4 - "GetClaimsFromContext"
Cohesion: 0.05
Nodes (45): AvailableFilesResponse, DeleteFileResponse, ListFilesResponse, UpdateFileResponse, buildFileInfo(), buildFileResponse(), deriveLogicalName(), extractLogicalName() (+37 more)

### Community 5 - "context.Context"
Cohesion: 0.05
Nodes (51): historyQueryCountingClient, mockUserStatusChecker, BackupConfig, resourceType, RestoreConfig, KrknWorkspaceReconciler, removeCategoryLabel(), isValidConfigMapKey() (+43 more)

### Community 6 - "NewTestHandler"
Cohesion: 0.10
Nodes (68): esDoerFunc, boolPtr(), newEsAlertsTestSecretWithHost(), newEsScheme(), newEsTestSecret(), newEsTestSecretWithHost(), TestCreateElasticsearchConfig_DuplicateName(), TestCreateElasticsearchConfig_Forbidden() (+60 more)

### Community 7 - "createAdminContext"
Cohesion: 0.08
Nodes (56): TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(), TestCloudCredentialContract_GraphRunStripsNodeEnv(), newAWSTestSecret(), newCloudCredScheme(), newGCPTestSecret(), newGroupedAWSSecret(), TestCloudCredentialsRouter_MethodNotAllowed(), TestCloudCredentialsRouter_SubpathMethodNotAllowed() (+48 more)

### Community 8 - "backup.go"
Cohesion: 0.14
Nodes (16): tokenRequest, TestE2E(), getMetricsOutput(), GetNonEmptyLines(), GetProjectDir(), InstallCertManager(), InstallPrometheusOperator(), IsCertManagerCRDsInstalled() (+8 more)

### Community 9 - ".prepareJobResources"
Cohesion: 0.17
Nodes (8): preparedJobResources, TestAppendCloudCredentialInjection_EmptyRefNoop(), getOwnerLabel(), rollbackConfigTemplateCheck(), TestPrepareJobResourcesRejectsUnavailableWorkspaceBeforeCreatingPod(), TestPrepareJobResourcesRequiresWorkspaceForRollback(), TestRollbackConfigTemplateCheck(), TestScenarioPodMountsOnlyAssignedWorkspaceDirectory()

### Community 10 - "api/types.go"
Cohesion: 0.05
Nodes (46): ActiveRunsOverviewResponse, AddGroupMemberRequest, AddGroupMemberResponse, ChangePasswordRequest, ChangePasswordResponse, ClusterPermissionSet, ClusterResiliencyScoreResponse, CreateTargetResponse (+38 more)

### Community 11 - "Broadcaster"
Cohesion: 0.17
Nodes (6): hashBytes(), roleFromAdmin(), AuthorizationCheckFunc, Broadcaster, graphRunEventHandler, scenarioRunEventHandler

### Community 12 - "migration.go"
Cohesion: 0.12
Nodes (44): clusterJobImage, clusterJobProvider, clusterJobRetries, graphScenarioReferences, migrationBlockedError, migrationPlan, migrationResult, migrationStats (+36 more)

### Community 13 - "addAdminContext"
Cohesion: 0.12
Nodes (41): addAdminContext(), addUserContext(), createTestAdminUser(), setupFilesTestHandler(), TestCanAccessFile(), TestCreateFile(), TestCreateFile_DefaultFilePurpose(), TestCreateFile_DuplicateName() (+33 more)

### Community 14 - "handlers_test.go"
Cohesion: 0.07
Nodes (40): TestLogin_InactiveUser(), TestLogin_InvalidCredentials(), TestLogin_Success(), setupScenarioRunTestHandler(), TestDeleteTargetByUUID_AdminCanDeleteAnyResource(), TestDeleteTargetByUUID_InvalidUUID(), TestDeleteTargetByUUID_NonOwner_Forbidden(), TestDeleteTargetByUUID_NotFound() (+32 more)

### Community 15 - "KrknCategory"
Cohesion: 0.15
Nodes (13): CategoryListResponse, CategoryResponse, buildCategoryResponse(), buildCategoryVisibilityLabels(), categoryErrorCode(), categoryGroups(), categoryIsPublic(), categoryVisibleToUser() (+5 more)

### Community 16 - "render-bundle.sh"
Cohesion: 0.07
Nodes (32): channel_has_entries(), ensure_catalog_channel(), catalog-channel.sh script, update_default_channel(), write_release_config(), catalog_submit_checkout(), catalog_submit_open_pr(), catalog-submit.sh script (+24 more)

### Community 17 - "server.py"
Cohesion: 0.06
Nodes (6): add_DataProviderServiceServicer_to_server(), DataProviderService, DataProviderServiceServicer, DataProviderServiceStub, DataProviderServicer, serve()

### Community 18 - "NewConfigStoreInitializer"
Cohesion: 0.09
Nodes (16): ConfigStoreInitializer, NewConfigStoreInitializer(), TestConfigStoreInitializer_NeedLeaderElection(), TestConfigStoreInitializer_Start_ConfigMapNotFound(), TestConfigStoreInitializer_Start_Success(), Store, SyncConfigMapToStore(), TestRoundTrip() (+8 more)

### Community 19 - "NewHub"
Cohesion: 0.10
Nodes (33): NewHandler(), NewBroadcaster(), TestBroadcastDashboardUpdate(), TestBroadcastGraphRunDeleted(), TestBroadcastGraphRunUpdate(), TestBroadcastOnlyToSubscribedClients(), TestBroadcastScenarioRunDeleted(), TestBroadcastScenarioRunUpdate() (+25 more)

### Community 20 - "WSUnifiedJobItem"
Cohesion: 0.07
Nodes (21): TestFilterJobsByCategoriesMatchesAnySelectedCategory(), TestPaginateJobItems(), buildGraphRunResponse(), computeWSJobStats(), convertGraphClusterScoresForSnapshot(), filterJobCategoriesByVisibility(), filterJobsByCategories(), paginateJobItems() (+13 more)

### Community 21 - "user_handlers_test.go"
Cohesion: 0.14
Nodes (34): createTestUser(), kvValue(), setupUserTestHandler(), TestChangePassword_AdminChange_NoCurrentPassword(), TestChangePassword_AdminChangesOtherUser_EmitsAuditLog(), TestChangePassword_SamePassword_Rejected(), TestChangePassword_SelfChange_NoAdminAuditLog(), TestChangePassword_SelfChange_RequiresCurrentPassword() (+26 more)

### Community 22 - "client_test.go"
Cohesion: 0.10
Nodes (23): doerFunc, roundTripperFunc, NewClient(), scenarioParametersSchema(), TestBaseURL(), TestClientTransportDefaultNotHTTPTransport(), TestClientTransportPooling(), TestConnectionParamsTLSConfig() (+15 more)

### Community 23 - "KrknGraphRun"
Cohesion: 0.13
Nodes (12): KrknGraphRunReconciler, TestGraphRunReconcileSkipsPendingMigration(), TestLoadFileConfigMapByID(), TestTranslateVolumesToFileMounts(), sanitizeNodeID(), TestCalculateResiliencyScore_StoresNodeScores(), overrideRef(), TestGraphRunInvalidImageSignatureFailsWithoutRetry() (+4 more)

### Community 24 - "elasticsearch/types.go"
Cohesion: 0.14
Nodes (23): AffectedPods, ClusterMetadata, CreateElasticsearchConfigResponse, DeleteElasticsearchConfigResponse, FacetOption, ListElasticsearchConfigsResponse, NodeSummaryInfo, QueryAlertsRequest (+15 more)

### Community 25 - "InjectCredentials"
Cohesion: 0.41
Nodes (11): InjectCredentials(), makeSecret(), TestInjectCredentialsAWS(), TestInjectCredentialsAzure(), TestInjectCredentialsBaremetal(), TestInjectCredentialsGCP(), TestInjectCredentialsIBMCloud(), TestInjectCredentialsMissingProviderLabel() (+3 more)

### Community 26 - "installer.go"
Cohesion: 0.09
Nodes (21): main(), resolveMigrationNamespace(), runCRDMigrationMode(), TestResolveMigrationNamespacePrefersHookNamespace(), TestRunCRDMigrationMode(), configMapClient, crdManifest, deploymentClient (+13 more)

### Community 27 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.10
Nodes (9): ClustersResponse, KrknTargetRequestReconciler, ClusterHealthStatus, ClusterTarget, KrknOperatorTarget, KrknOperatorTargetSpec, KrknTargetRequest, KrknTargetRequestSpec (+1 more)

### Community 28 - "categoryHistoryRun"
Cohesion: 0.14
Nodes (16): categoryEnvironmentParameter, categoryHistoryRun, categoryParameterProfileFingerprint(), categoryParameterProfileName(), TestCategoryParameterProfileFingerprintForEmptyEnvironment(), TestCategoryParameterProfileFingerprintGraphEnvironmentMultiset(), TestCategoryParameterProfileFingerprintIgnoresRunType(), TestCategoryParameterProfileFingerprintScenarioEnvironment() (+8 more)

### Community 29 - "client.go"
Cohesion: 0.10
Nodes (27): AlertDocument, esAlertSearchResponse, esSearchResponse, facetField, filterAggregation, Option, StatusError, termsAggregation (+19 more)

### Community 30 - "backup_restore_handlers_test.go"
Cohesion: 0.14
Nodes (23): NewJobTracker(), createMultipartBody(), createTestHandler(), TestGetRestoreStatusAdminOnly(), TestGetRestoreStatusTrackedJob(), TestGetRestoreStatusUnknownJob(), TestHandlerShutdownCancelsContext(), TestJobTrackerEviction() (+15 more)

### Community 31 - ".updateClusterJobStatuses"
Cohesion: 0.14
Nodes (9): backfillLegacyMaxRetries(), TestBackfillLegacyMaxRetries(), TestFailedAttemptRetainsRollbackIdentityBeforeRetry(), TestShouldRetryJobHonorsConfiguredRetryLimit(), TestUpdateClusterJobStatuses_ClusterNameEmpty(), TestUpdateClusterJobStatuses_PreservesExplicitZero(), TestUpdateClusterJobStatuses_ProviderNameEmpty(), TestScenarioRunInvalidImageSignatureIsNotRetried() (+1 more)

### Community 32 - "Handler"
Cohesion: 0.23
Nodes (4): visibleCategoryNames(), AuthorizationChecker, categoryVisibilityChecker, Handler

### Community 33 - "ConnectionParams"
Cohesion: 0.23
Nodes (7): Doer, executeSearch(), Client, ConnectionParams, newSearchRequest(), TestTransportKey(), transportKey()

### Community 34 - "KrknOperatorTargetProviderConfig"
Cohesion: 0.10
Nodes (15): KrknOperatorTargetProviderConfigReconciler, Architecture Decision, Components Requiring Leader Election (Leader-Only), Components Running on All Replicas (Multi-Active), isConflictError(), countActiveProviders(), Configuration Functions, CreateProviderConfigRequest (+7 more)

### Community 35 - "injection.go"
Cohesion: 0.40
Nodes (8): injectAWS(), injectAzure(), injectBaremetal(), injectGCP(), injectIBMCloud(), injectOpenStack(), injectVMware(), secretKeyRef()

### Community 37 - "k8s.io/api/core/v1.Secret"
Cohesion: 0.11
Nodes (13): cloudCredentialAccessContext, updateCloudCredentialSecretData(), buildConnectionParams(), buildConnectionParamsForIndex(), buildElasticsearchConfigResponse(), canAccessElasticsearchConfig(), buildRegistryInfo(), ExtractGroupsFromLabels() (+5 more)

### Community 38 - "NewServerWithComponentNamespace"
Cohesion: 0.07
Nodes (24): CachedUserStatusChecker, Claims, countingUserStatusChecker, Role, userStatusCacheEntry, UserStatusChecker, loggingMiddleware(), maxBodySizeMiddleware() (+16 more)

### Community 39 - "time.Time"
Cohesion: 0.11
Nodes (18): GraphRunDetailResponse, GraphRunListItem, GraphRunListResponse, GraphRunSpecResponse, GraphRunStatusResponse, RestoreResponse, 1.1 Resiliency weight per graph node, 1. Request - Create GraphRun (POST /api/v1/graphruns) (+10 more)

### Community 40 - "ComponentManifest"
Cohesion: 0.15
Nodes (7): ListComponentsResponse, Load(), TestLoad(), TestLoadRejectsInvalidManifest(), ComponentDescription, ComponentManifest, KrknComponentStatus

### Community 41 - "bootstrap.go"
Cohesion: 0.17
Nodes (13): RemoveElasticsearchGrafanaURLAnnotations(), TestRemoveElasticsearchGrafanaURLAnnotations(), ensureJWTSecret(), EnsureResources(), ensureSCCBinding(), ensureScenarioRunnerRBAC(), ensureServiceAccount(), ensureServices() (+5 more)

### Community 42 - "UnifiedJobItem"
Cohesion: 0.12
Nodes (7): PaginationMeta, ScenarioRunListItem, ScenarioRunListResponse, UnifiedJobItem, UnifiedJobsResponse, ComputeJobStats(), TestComputeJobStats()

### Community 43 - "installer_test.go"
Cohesion: 0.18
Nodes (12): crdClient, applyCRD(), applyCRDs(), TestApplyCRDsCreatesMissingAndUpdatesChangedSpecs(), TestApplyCRDsDoesNotPatchUnchangedSpecs(), TestApplyCRDsLabelsExistingCRDsWithoutChangingOtherLabels(), TestApplyCRDsRejectsNonCRDManifest(), testCRD() (+4 more)

### Community 44 - "parseResiliencyHistoryQuery"
Cohesion: 0.18
Nodes (10): resiliencyHistoryQuerySelections, normalizeResiliencyHistoryProviders(), normalizeResiliencyHistorySelections(), parseResiliencyHistoryQuery(), createTarGz(), TestCreateAndExtractTarGz(), TestCreateTarGzFlushErrors(), TestExtractTarGzPathTraversal() (+2 more)

### Community 45 - "migration_test.go"
Cohesion: 0.20
Nodes (25): Complete(), Stage(), containsSubstring(), countPatchActions(), jsonMarshal(), migrationTestLogger(), newMigrationClient(), scenarioRunFixture() (+17 more)

### Community 46 - "notify-operator-docs.test.js"
Cohesion: 0.10
Nodes (11): assert, chartWorkflow, { execFileSync, spawnSync }, fs, notifier, notifyWorkflow, os, path (+3 more)

### Community 47 - ".Get"
Cohesion: 0.22
Nodes (13): countingGetClient, TestConcurrentAccess(), TestConcurrentSnapshot(), TestDelete(), TestDelete_NonExistent(), TestEmptyValue(), TestExists(), TestGet_Singleton() (+5 more)

### Community 48 - "category_run_configuration_test.go"
Cohesion: 0.33
Nodes (14): TestBuildCategoryResiliencyHistoryGroupsTypedConfigurationsAndSortsPerCluster(), TestGraphHistoryScoresOnlyKeepsFinalScoresIncludingZero(), compareCategoryRunConfigurations(), assertConfigurationDifferences(), floatPointer(), graphRunConfigurationFixture(), scenarioReference(), scenarioRunConfigurationFixture() (+6 more)

### Community 49 - "ValidatePVC"
Cohesion: 0.20
Nodes (7): workspaceInfo, workspaceAvailable(), workspaceInfoFromCR(), boundPVC(), TestValidatePVC(), ValidatePVC(), WorkspaceReference

### Community 50 - "Hub"
Cohesion: 0.36
Nodes (4): BroadcastMessage, Client, Hub, PaginationClientState

### Community 51 - "buildScenarioRunResponse"
Cohesion: 0.17
Nodes (12): TestBuildUnifiedJobList_SkipsGraphRunChildren(), averageResiliencyScore(), buildScenarioRunDetailResponse(), buildScenarioRunResponse(), buildUnifiedJobList(), convertClusterResiliencyScoresWS(), runCategoryNames(), sanitizedClusterJobs() (+4 more)

### Community 52 - "k8s.io/apimachinery/pkg/runtime.Scheme"
Cohesion: 0.14
Nodes (17): addKnownTypes(), decodeErrorResponse(), providerConfigTestScheme(), TestGetProviderConfigByUUID_CompletedReturnsConfigData(), TestGetProviderConfigByUUID_InternalErrorReturnsSanitized500(), TestGetProviderConfigByUUID_NotFoundReturns404(), TestGetProviderConfigByUUID_PendingReturnsAccepted(), TestPostProviderConfig_FailureReturnsSanitizedError() (+9 more)

### Community 53 - "ScenarioReference"
Cohesion: 0.18
Nodes (13): publicScenarioReference(), TestCreateScenarioRun_ResiliencyScore(), buildContainerImage(), buildContainerImageFromReference(), getTestConfig(), privateReference(), publicReference(), TestBuildContainerImage_PublicRegistryUsesKrknctl() (+5 more)

### Community 54 - "Handler"
Cohesion: 0.11
Nodes (8): Handler, BuildUnifiedJobList(), runCategoryNames(), scenarioRunImage(), TestBuildUnifiedJobList_IdentifiesRollbackFromSource(), TestBuildUnifiedJobList_IncludesResolvedScenarioImage(), TestBuildUnifiedJobList_SortOrder(), buildTargetResponse()

### Community 55 - "github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2"
Cohesion: 0.10
Nodes (16): mockScenarioProvider, InvalidImageSignatureError, convertInputFields(), createScenarioProvider(), filterScenariosByIsAScenario(), getScenarioSignatureStatus(), resolveRollbackDigest(), TestFilterScenariosByIsAScenario() (+8 more)

### Community 56 - "cloudcreds/types.go"
Cohesion: 0.16
Nodes (15): CreateCloudCredentialResponse, DeleteCloudCredentialResponse, ListCloudCredentialsResponse, UpdateCloudCredentialResponse, buildCloudCredentialSecretData(), CloudCredentialResponse, CreateCloudCredentialRequest, UpdateCloudCredentialRequest (+7 more)

### Community 57 - "GetNodesRequest"
Cohesion: 0.15
Nodes (3): DataProviderServiceClient, GetNodesRequest, GetNodesResponse

### Community 58 - "dataprovider_grpc.pb.go"
Cohesion: 0.16
Nodes (5): UnimplementedDataProviderServiceServer, UnsafeDataProviderServiceServer, _DataProviderService_ExecuteKubectl_Handler(), _DataProviderService_GetNodes_Handler(), RegisterDataProviderServiceServer()

### Community 59 - "GetClaimsFromContext"
Cohesion: 0.06
Nodes (27): TerminalRequest, TerminalResponse, Code References, writeScenarioRunAccessError(), parseBoolHeader(), TestParseBoolHeader(), sanitizeRunNameLabel(), resolveMaxRetries() (+19 more)

### Community 60 - "converter_test.go"
Cohesion: 0.11
Nodes (22): rollbackImageCache, rollbackImageCheck, Publish and select a workspace, Replay and unavailable storage, Rollback artifact persistence, Scenario workspaces on a shared PVC, Storage validation, BuildDockerConfigJSON() (+14 more)

### Community 61 - ".DeepCopyInto"
Cohesion: 0.05
Nodes (21): publicHistoryCategory(), filterUsers(), newKrknUser(), GraphScenario, KrknCategory, KrknCategoryList, KrknCategorySpec, KrknComponent (+13 more)

### Community 62 - "ScenarioRunRequest"
Cohesion: 0.19
Nodes (13): Phase 12: Scenario Job Management - /scenarios/run Endpoints ✅, Phase 9: POST /scenarios Endpoint ✅, FileMount, GlobalsRequest, JobsListResponse, JobStatusResponse, ScenarioRunRequest, ScenariosRequest (+5 more)

### Community 63 - "category_run_configuration.go"
Cohesion: 0.15
Nodes (20): GraphRunCreateRequest, GraphScenarioNode, appendGraphSignaturePart(), compareGraphConfigurations(), compareGraphConfigurationsWithStableNodeIDs(), compareGraphNodeConfigurations(), graphConfigurationSignature(), graphNodeConfigurationSignature() (+12 more)

### Community 64 - ".extractAndStoreReports"
Cohesion: 0.17
Nodes (10): decodeReportPayloads(), extractDelimitedContent(), TestDecodeReportPayloadsPreservesValidReport(), TestDecodeReportPayloadsReportsInvalidHTML(), TestExtractAndStoreReportsRejectsMultiClusterRun(), TestExtractDelimitedContent(), TestTargetClusterCount(), TestValidateReportPayloadSize() (+2 more)

### Community 65 - "sanitizeUsername"
Cohesion: 0.32
Nodes (5): k8sUserStatusChecker, sanitizeResourceName(), sanitizeUsername(), ValidateEmail(), ValidatePassword()

### Community 66 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.22
Nodes (9): Configuration, Current Behavior, Implementation Plan (TODO), Option A: Cancellation Field in Status (Recommended), Option B: Finalizers for Cancellation Tracking, Option C: Separate CancellationRequest CR, Proposed Solution Options, Requirements (+1 more)

### Community 67 - "metadata_presence_test.go"
Cohesion: 0.21
Nodes (11): rawTelemetrySource, intPtr(), TestFlattenRepresentativeScenario(), TestRawTelemetrySourceFlatten(), contains(), containsAt(), TestMetadataJSONMarshaling(), TestMetadataPresenceTracking() (+3 more)

### Community 68 - "rate_limiter_test.go"
Cohesion: 0.06
Nodes (43): ipRateLimiter, rateLimiterEntry, responseWriter, clientIPFromHeaders(), extractClientIP(), hostFromRemoteAddr(), isTrustedProxy(), newIPRateLimiter() (+35 more)

### Community 69 - "SecretManager"
Cohesion: 0.09
Nodes (9): Server, getTokenDuration(), TestGetTokenDuration(), NewServer(), Handler, generateRandomSecret(), getSecretKeys(), SecretManager (+1 more)

### Community 70 - "sigs.k8s.io/controller-runtime.Manager"
Cohesion: 0.11
Nodes (7): Publisher, NewNamespaceFilter(), TestPublisherReconcile(), TestPublisherReconcileReturnsTransientErrors(), TestPublisherRecoversFromConcurrentCreateAndDelete(), TestPublisherStartRetriesAndReturnsCancellation(), ProviderConfigContributorReconciler

### Community 71 - "start_operator.sh"
Cohesion: 0.41
Nodes (12): build_operator(), check_prerequisites(), cleanup(), create_service_account(), install_crds(), main(), print_error(), print_info() (+4 more)

### Community 72 - ".getClusterAPIURL"
Cohesion: 0.46
Nodes (7): newTargetRequest(), newTestScheme(), TestGetClusterAPIURL_ClusterNotFound(), TestGetClusterAPIURL_Found(), TestGetClusterAPIURL_ProviderNotFound(), TestGetClusterAPIURL_TargetRequestNotFound(), TestReconcile_FailedJobStatus_IncludesClusterAPIURL()

### Community 73 - "PROGRESS-MVP.md"
Cohesion: 0.04
Nodes (41): Architecture Decisions, Data Provider Architecture, Environment Variables, Future Work (Not in Current Scope), GET /clusters, GET /nodes, GET /targets/{UUID}, Implementation Files (+33 more)

### Community 74 - "Compute"
Cohesion: 0.30
Nodes (3): Compute(), JobCounter, Summary

### Community 75 - "KrknScenarioRunSpec"
Cohesion: 0.18
Nodes (8): KrknScenarioRunSpec, compareScenarioRunConfigurations(), compareStringMaps(), optionalBoolValue(), sameScenarioFiles(), sameScenarioReference(), FileMount, RollbackSource

### Community 76 - "KrknOperatorTargetProvider"
Cohesion: 0.06
Nodes (27): Completed, Current Status, In Progress, Pending, 1. List Providers, 2. Get Provider Config Status, 2. Update Provider Status, 3. Create Provider Config Request (+19 more)

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
Cohesion: 0.17
Nodes (13): CreateElasticsearchConfigRequest, QueryTelemetryRequest, UpdateElasticsearchConfigRequest, buildInlineConnectionParams(), InlineConnection, ValidateQueryRequest(), strPtr(), TestValidateUpdateRequest() (+5 more)

### Community 84 - "GraphClusterScore"
Cohesion: 0.14
Nodes (6): convertNodeStatusesWithScores(), convertNodeStatusesWithScores(), GraphClusterScore, GraphRunSummary, KrknGraphRunStatus, NodeStatus

### Community 85 - "Krkn-Operator Roadmap"
Cohesion: 0.25
Nodes (6): Krkn-Operator Roadmap, Ongoing (not release-blocked), Release 1 — 2026-08-18, Release 2 — 2026-10-18, Release 3 — 2026-12-22, Release 4 — 2027-02-23

### Community 86 - "publicScenarioReference"
Cohesion: 0.05
Nodes (73): newCategoryRequest(), newCategoryRequestAs(), newCategoryTestGroup(), newCategoryTestHandler(), newCategoryTestUser(), TestCategoriesRouterCRUD(), TestCategoriesRouterGroupCreationRequiresMembership(), TestCategoriesRouterGroupVisibility() (+65 more)

### Community 87 - "KrknOperatorTargetProviderConfigStatus"
Cohesion: 0.29
Nodes (3): getProviderNames(), KrknOperatorTargetProviderConfigStatus, ProviderConfigData

### Community 88 - "CategoryResiliencyHistoryResponse"
Cohesion: 0.38
Nodes (7): CategoryConfigurationGroup, CategoryResiliencyDataPoint, CategoryResiliencyHistoryResponse, ResiliencyHistoryQueryResponse, datapointForRunID(), datapointRunIDs(), assembleResiliencyHistoryQueryResponse()

### Community 89 - "GET methods"
Cohesion: 0.06
Nodes (35): API Requirements, /clusters, DELETE methods, GET methods, GET methods, get nodes, get pods, Grpc python service requirement (+27 more)

### Community 90 - "ResolveGraph"
Cohesion: 0.15
Nodes (13): FromKrknctlScenarioSet(), publicScenarioReference(), publicScenario(), TestFromKrknctlScenarioSetCreatesPublicReferences(), TestToKrknctlScenarioSetUsesScenarioReference(), ToKrknctlScenarioSet(), MapScenarioNodeToScenarioRunSpec(), ResolveGraph() (+5 more)

### Community 91 - "ScenarioAttemptStatus"
Cohesion: 0.29
Nodes (4): ClusterJobStatusResponse, NodeStatusResponse, ScenarioRunStatusResponse, ScenarioAttemptStatus

### Community 92 - "elasticsearch/labels_test.go"
Cohesion: 0.33
Nodes (5): TestBuildAnnotations(), TestBuildLabels(), TestBuildLabelsAccessControl(), TestUpdateAnnotations(), UpdateAnnotations()

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

### Community 99 - "TestKrknGraphRun_ValidateGraph"
Cohesion: 0.33
Nodes (5): SanitizeNodeIDForKubernetes(), stringPtr(), TestCollisionDetection(), TestKrknGraphRun_ValidateGraph(), TestSanitizeNodeIDForKubernetes()

### Community 100 - "makeScenarioRun"
Cohesion: 0.27
Nodes (12): makeGraphRun(), makeScenarioRun(), TestListJobs_IncludesStats(), TestListJobs_LimitOnlyReturnsAll(), TestListJobs_NoPagination(), TestListJobs_Paginated(), TestListJobs_SkipsGraphRunChildren(), TestListJobs_TypeDiscriminator() (+4 more)

### Community 101 - "Authentication API Specification"
Cohesion: 0.09
Nodes (22): 1. Check Admin Registration, 1. Initial Load, 2. First Admin Registration, 2. Register User, 3. Login, 3. Login, 4. Authenticated Requests, 5. Role-Based UI (+14 more)

### Community 102 - "Troubleshooting"
Cohesion: 0.40
Nodes (5): Error: "Command not found: kubectl", Error: "failed to connect to gRPC server", Error: "Failed to get kubeconfig", Error: "no module named 'krkn_lib'", Troubleshooting

### Community 103 - "Pushing Images"
Cohesion: 0.50
Nodes (4): Custom Push, Push Both Images, Push Individual Images, Pushing Images

### Community 104 - "setup-ocm-and-operator.sh"
Cohesion: 0.67
Nodes (5): log_error(), log_info(), log_warn(), setup-ocm-and-operator.sh script, wait_for_condition()

### Community 105 - "File Management API Specification"
Cohesion: 0.09
Nodes (22): Access Control, API Endpoints, ConfigMap Structure, Content Validation, Create a Public Configuration File, Create File, Delete File, Error Responses (+14 more)

### Community 107 - "API compatibility notes"
Cohesion: 0.50
Nodes (4): API compatibility notes, Graph node resiliency weights, Log streaming, Retry limits

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

### Community 117 - "CLAUDE.md - Project Instructions"
Cohesion: 0.10
Nodes (19): Architecture Principles, Before Committing, Before Writing Code, CLAUDE.md - Project Instructions, Code Organization & Reusability, Code Quality Standards, Communication, Critical Rules (+11 more)

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

### Community 140 - "KrknWorkspace"
Cohesion: 0.11
Nodes (7): CanAccess(), TestCanAccess(), KrknScenarioRunStatus, KrknWorkspace, KrknWorkspaceSpec, KrknWorkspaceStatus, ReportStatus

### Community 142 - "KrknScenarioRunReconciler"
Cohesion: 0.14
Nodes (10): imageResolver, imageSignatureVerifier, KrknScenarioRunReconciler, migrationPending(), TestScenarioRunReconcileSkipsPendingMigration(), countJobPhaseChanges(), backfillLegacyGraphResiliencyScoreStatuses(), TestBackfillLegacyGraphResiliencyScoreStatuses() (+2 more)

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

### Community 153 - "Setup"
Cohesion: 0.40
Nodes (5): 1. Create Virtual Environment (Python 3.11), 2. Install Dependencies, 3. Verify Generated Stubs, 4. Start gRPC Server, Setup

### Community 154 - "KrknUserGroup"
Cohesion: 0.11
Nodes (18): categoryHistoryScore, containsString(), filterCategoryHistoryScoresByCluster(), filterGraphHistoryScoresByPermission(), filterScenarioHistoryScoresByPermission(), graphHistoryScores(), runHasAnyCategory(), scenarioHistoryScores() (+10 more)

### Community 156 - ".GetScenarioRunLogs"
Cohesion: 0.16
Nodes (12): filterReportLines(), isReportMarkerEnd(), isReportMarkerStart(), isWebSocketDisconnectError(), maskToken(), preStartFailureReason(), sanitizeHeaders(), writeWSError() (+4 more)

### Community 159 - "buildClusterAPIURLMap"
Cohesion: 0.25
Nodes (8): buildClusterAPIURLMap(), contains(), containsInMiddle(), TestBuildClusterAPIURLMap(), TestBuildClusterAPIURLMap_Collision(), TestBuildClusterAPIURLMap_SameNameSameURL(), TestValidateScenarioRunAccess(), TestValidateScenarioRunAccess_NoGroups()

### Community 163 - "files/validation_test.go"
Cohesion: 0.31
Nodes (8): contains(), findSubstring(), TestValidateFileContent(), TestValidateFileGroups(), TestValidateUserFilePermissions(), ValidateFileContent(), ValidateFileGroups(), ValidateUserFilePermissions()

### Community 166 - "krkngraphrun_controller_sanitization_test.go"
Cohesion: 0.83
Nodes (3): isAlphanumeric(), isValidLabelChar(), TestSanitizeNodeID()

### Community 167 - "bootstrapOLMResources"
Cohesion: 0.67
Nodes (3): bootstrapOLMResources(), TestBootstrapOLMResourcesPropagatesClientError(), TestBootstrapOLMResourcesWiring()

### Community 171 - "generator.go"
Cohesion: 0.18
Nodes (16): CreateTargetRequest, UpdateTargetRequest, generateKubeconfigFromCredentialsType(), generateKubeconfigFromKubeconfigType(), generateKubeconfigFromRequest(), generateKubeconfigFromTokenType(), TestCreateTarget_DuplicateClusterName(), TestCreateTarget_WithKubeconfig() (+8 more)

## Knowledge Gaps
- **17 isolated node(s):** `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME`, `ICON_BASE64`, `ICON_MEDIATYPE` (+12 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 734 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **28 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Handler` connect `Handler` to `main.go`, `sanitizeUsername`, `Handler`, `ConnectionParams`, `GetClaimsFromContext`, `k8s.io/api/core/v1.Secret`, `context.Context`, `SecretManager`, `bootstrap.go`, `JobTracker`, `KrknCategory`, `ValidatePVC`, `.GetScenarioRunLogs`, `github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2`, `KrknUserGroup`, `GetClaimsFromContext`, `converter_test.go`, `backup_restore_handlers_test.go`?**
  _High betweenness centrality (0.061) - this node is a cross-community bridge._
- **Are the 159 inferred relationships involving `NewTestHandler()` (e.g. with `setupAuthTestHandler()` and `TestLogin_InactiveUser()`) actually correct?**
  _`NewTestHandler()` has 159 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME` to the rest of the system?**
  _17 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.09432867011087247 - nodes in this community are weakly interconnected._
- **Why does `krkn-operator` connect `krkn-operator` to `Running Locally`, `API compatibility notes`?**
  _High betweenness centrality (0.032) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.015226011102299762 - nodes in this community are weakly interconnected._
- **Why does `KrknUserGroup` connect `KrknUserGroup` to `main.go`, `.DeepCopy`, `context.Context`, `k8s.io/api/core/v1.Secret`, `k8s.io/apimachinery/pkg/apis/meta/v1.Time`, `File Management API Specification`, `Handler`, `publicScenarioReference`, `GetClaimsFromContext`, `.DeepCopyInto`?**
  _High betweenness centrality (0.025) - this node is a cross-community bridge._