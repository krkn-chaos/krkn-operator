# Graph Report - krkn-operator  (2026-11-20)

## Corpus Check
- 408 files · ~522,813 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 18 file(s) not represented in the graph (top: (none) 12, .jsonl 2, .toml 1)

## Summary
- 6277 nodes · 21056 edges · 303 communities (259 shown, 44 thin omitted)
- Extraction: 92% EXTRACTED · 8% INFERRED · 0% AMBIGUOUS · INFERRED: 1653 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `717cac14`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- main.go
- testing.T
- Handler
- InjectCredentials
- GetClaimsFromContext
- context.Context
- KrknAIRun
- KrknWorkspace
- utils.go
- KrknTargetRequest
- api/types.go
- Broadcaster
- newCancellationScenarioReconciler
- addAdminContext
- k8s.io/api/core/v1.Secret
- GetClaimsFromContext
- render-bundle.sh
- server.py
- category_run_configuration.go
- contains
- .UpdateCategory
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
- .DeepCopyInto
- Server
- .QueryTelemetryPage
- KrknOperatorTargetProviderConfig
- UnifiedJobItem
- .DeepCopy
- k8s.io/apimachinery/pkg/apis/meta/v1.TypeMeta
- TokenGenerator
- watcher_test.go
- PROGRESS-MVP.md
- ai.go
- github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2
- wsorigin.go
- WSUnifiedJobItem
- migration_test.go
- notify-operator-docs.test.js
- converter_test.go
- ssrf.go
- KrknScenarioRunSpec
- Hub
- POST methods
- POST methods
- captureLogSink
- providerConfigTestScheme
- newReportTestHandler
- cloudcreds/types.go
- .UpdateFile
- dataprovider_grpc.pb.go
- .UpdateRegistry
- mockJob
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- identity_provider_store.go
- scenario_catalog_cache_test.go
- GET methods
- installer_test.go
- buildClusterAPIURLMap
- buildContainerImageFromReference
- wsorigin.go
- Handler
- .loadCategoryResiliencyHistories
- start_operator.sh
- files/validation_test.go
- parseKrknAIScenarioQuery
- scenario_catalog_cache_test.go
- ClusterJobStatus
- KrknOperatorTargetProvider
- .PostScenarioRun
- test-api.sh
- newAIRunFixture
- ExecuteKubectlRequest
- migration_test.go
- KrknScenarioRunReconciler
- .extractAndStoreReports
- category_parameter_profile.go
- Krkn-Operator Roadmap
- time.Time
- InjectCredentials
- ScenariosResponse
- metadata_presence_test.go
- GroupLabelKey
- KrknOperatorTargetProviderConfig
- net/http.Response
- k8s.io/apimachinery/pkg/apis/meta/v1.Time
- Moving from direct scenario creation to CRD approach
- ProviderRegistration
- Provider Configuration API Specification
- .PostScenarioDetail
- dataprovider.pb.go
- TestKrknGraphRun_ValidateGraph
- GraphClusterScore
- Authentication API Specification
- .ListAvailableCloudCredentials
- Publisher
- setup-ocm-and-operator.sh
- File Management API Specification
- google.golang.org/protobuf/reflect/protoreflect.Message
- ProviderRegistration
- commands.go
- Cluster Segregation Troubleshooting
- channel-head.sh
- Running gRPC Data Provider Locally for Development
- docs.go
- api_workflow_test.sh
- select-compatible-release.sh
- test-open-kubernetes-operators-pr.sh
- scenarioRunConfigurationFixture
- post-install.sh
- KrknUser
- ResolvePrivateTagDigest
- test-channel-head.sh
- test-chart-exposure.sh
- test-select-compatible-release.sh
- .PostScenarioRun
- debug-cluster-permissions.sh
- notify-operator-docs.sh
- github.com/krkn-chaos/krkn-operator
- Setup
- GenerateFromToken
- ProviderRegistration
- krkn-operator
- Deployment Guide
- Quick Start - Terminal API Local Testing
- Resiliency Score Feature - Frontend Implementation Guide
- AGENTS.md
- .updateClusterJobStatuses
- krkn-operator-data-provider
- decodeTelemetry
- API Workflow Test
- WSUnifiedJobItem
- sync_test.go
- rate_limiter_test.go
- k8s.io/client-go/kubernetes.Interface
- 🔧 TODO: Pod Recreation and Retry Logic
- API Changes
- Running Locally
- install-yq.sh
- test-release-workflow-gates.sh
- Deployment
- Available Make Targets
- Building Images
- Troubleshooting
- AIConfig
- newProvidersHandler
- Troubleshooting
- .GetScenarioRunLogs
- assets/README.md
- GenerateFromToken
- ScenarioReference
- NewSecretManager
- KrknUserGroupSpec
- ai_test.go
- krkngraphrun_controller_sanitization_test.go
- BuildUnifiedJobList
- WorkflowResponse
- contents
- Pushing Images
- TestValidateValueAgainstSchema
- GET methods
- ComponentManifest
- target
- ExecuteKubectlRequest
- buildUnifiedJobList
- DataProviderServiceServer
- name
- slash-to-add-label.yml
- newReportTestHandler
- API Workflow Test
- .DeepCopy
- captureLogSink
- bootstrap.go
- time.Duration
- TokenGenerator
- manager.yaml
- labels
- workspace_role.yaml
- ValidateCreateRequest
- sortKrknAIScenarios
- gateway
- containers
- labels
- metrics_auth_role.yaml
- role.yaml
- Optional Krkn-AI with OLM on OpenShift
- publish-olm-bundles
- entries
- NewCachedUserStatusChecker
- manager/kustomization.yaml
- spec
- metrics_reader_role.yaml
- config.yaml
- submit-operatorhub-catalog
- buildClusterAPIURLMap
- CategoryResiliencyHistoryResponse
- dataProviderServiceClient
- ScenariosResponse
- volumes
- sync_test.go
- entries
- Running Locally
- Krkn-Operator Roadmap
- TestKrknGraphRun_ValidateGraph
- k8s.io/apimachinery/pkg/apis/meta/v1/unstructured.Unstructured
- submit-community-catalog
- Compute
- KrknGraphRunStatus
- setup-ocm-and-operator.sh
- updateStrategy
- Available Make Targets
- Building Images
- Troubleshooting
- steps
- steps
- Setup
- ExtractRegistryV2FromSecret
- livenessProbe
- readinessProbe
- olm/kustomization.yaml
- Complete Workflow Examples
- .gosec.yaml
- channel-head.sh
- select-ai-release.sh
- krkngraphrun_controller_sanitization_test.go
- ConfigValidationError
- api_workflow_test.sh
- scenarioCatalogError
- resources
- securityContext
- configurationError
- select-compatible-release.sh
- test-open-kubernetes-operators-pr.sh
- test-select-ai-release.sh
- imageProbeError
- installer_test.go
- NewServerWithComponentNamespace
- krknAIArchiveUpstreamError
- crd/kustomization.yaml
- manifests/kustomization.yaml
- network-policy/kustomization.yaml
- assets/README.md
- prometheus/kustomization.yaml
- rbac/kustomization.yaml
- post-install.sh
- generate_proto_go.sh
- ClusterNameCollisionError
- install-yq.sh
- test-channel-head.sh
- test-chart-exposure.sh
- test-release-workflow-gates.sh
- test-select-compatible-release.sh
- rate_limiter_test.go
- generate_proto.sh
- debug-cluster-permissions.sh
- notify-operator-docs.sh
- github.com/krkn-chaos/krkn-operator
- parseResiliencyHistoryQuery
- .reconcileDeletion
- ResolvePrivateTagDigest
- runDeletionToCompletion
- NewServerWithComponentNamespace
- ClusterTarget
- .Reconcile
- newTestScheme
- main
- JobTracker
- guardedDialContext
- metadata_presence_test.go
- .UpdateUserGroup
- KrknOperatorTargetProviderConfigStatus
- KrknAIRunReconciler
- Deployment
- krknairun_config_test.go
- install-ai.sh
- scenario_filter_test.go
- elasticsearch/labels_test.go
- .checkRollbackImage
- DataProviderServiceServer
- OIDCState
- convertClusterJobs
- scenarioImageResolutionError
- GraphRunResponse
- .executeKubectlViaGRPC
- KrknUserGroupSpec
- probeAITags
- validateDestinationURL
- privateAIRunFixture
- validateAIClaim
- stringPtr
- KrknWorkspaceSpec
- krknAIArchiveManifestFile

## God Nodes (most connected - your core abstractions)
1. `Handler` - 281 edges
2. `NewTestHandler()` - 170 edges
3. `writeJSONError()` - 161 edges
4. `writeJSON()` - 119 edges
5. `createAdminContext()` - 99 edges
6. `KrknScenarioRun` - 74 edges
7. `GetClaimsFromContext()` - 71 edges
8. `newEsScheme()` - 57 edges
9. `setupUserTestHandler()` - 43 edges
10. `createUserContext()` - 43 edges

## Surprising Connections (you probably didn't know these)
- `4. Verify a run` --references--> `KrknAIRun`  [INFERRED]
  docs/olm-openshift-krkn-ai.md → api/v1alpha1/krknairun_types.go
- `2. Run the operator` --references--> `KrknComponent`  [INFERRED]
  README.md → api/v1alpha1/krkncomponent_types.go
- `BREAKING CHANGES` --references--> `KrknOperatorTarget`  [INFERRED]
  spec.orig/REQUIREMENTS.md → api/v1alpha1/krknoperatortarget_types.go
- `Overview` --references--> `KrknOperatorTarget`  [INFERRED]
  spec.orig/REQUIREMENTS.md → api/v1alpha1/krknoperatortarget_types.go
- `1. List Providers` --references--> `KrknOperatorTargetProvider`  [INFERRED]
  docs/api-provider-config-spec.md → api/v1alpha1/krknoperatortargetprovider_types.go

## Import Cycles
- None detected.

## Communities (303 total, 44 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.08
Nodes (32): availableWorkspacePVC, CategoryEntityAssociationResponse, CreateCategoryRequest, createIdentityProviderRequest, krknAIScenarioKey, OIDCBlock, patchIdentityProviderRequest, ResiliencyHistoryQueryRequest (+24 more)

### Community 1 - "testing.T"
Cohesion: 0.01
Nodes (253): cancellationValidationObject(), TestRunCancellationValidation(), TestKrknUserCreation(), TestKrknUserList(), TestKrknUserRoles(), TestKrknUserStatus(), TestKrknUserValidation(), TestScenarioReferenceValidate() (+245 more)

### Community 2 - "Handler"
Cohesion: 0.06
Nodes (20): ErrorResponse, Handler, writeCategoryRunNotFoundOrInternalError(), extractGroupNameAndUserIDFromPath(), extractGroupNameFromMembersPath(), convertInputFields(), extractPathSuffix(), writeJSON() (+12 more)

### Community 3 - "InjectCredentials"
Cohesion: 0.08
Nodes (92): countingGetClient, esDoerFunc, TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(), TestCloudCredentialContract_GraphRunStripsNodeEnv(), boolPtr(), newEsAlertsTestSecretWithHost(), newEsScheme(), newEsTestSecret() (+84 more)

### Community 4 - "GetClaimsFromContext"
Cohesion: 0.13
Nodes (16): KrknScenarioRunReconciler, preparedJobResources, TestScenarioRunReconcileSkipsPendingMigration(), effectiveScenarioMaxRetries(), TestCalculateOverallStatus_CreatingReservationIsRunning(), getOwnerLabel(), TestSubmitScenarioPodCopiesAIRunLabels(), rollbackConfigTemplateCheck() (+8 more)

### Community 5 - "context.Context"
Cohesion: 0.03
Nodes (65): assigningUIDClient, completeAIRunOnCancelClient, historyQueryCountingClient, mockUserStatusChecker, BackupConfig, resourceType, RestoreConfig, removeCategoryLabel() (+57 more)

### Community 6 - "KrknAIRun"
Cohesion: 0.21
Nodes (9): artifactsFinalizationPending(), completionTime(), containerTermination(), markAIRunArtifactFinalizationUnavailable(), orchestratorStopWasProven(), setAIRunCancellationConditions(), terminationReason(), KrknAIRun (+1 more)

### Community 7 - "KrknWorkspace"
Cohesion: 0.07
Nodes (28): RollbackAvailability, RollbackState, workspaceInfo, Publish and select a workspace, Replay and unavailable storage, Reuse from another operator, Scenario workspaces on a shared PVC, Storage validation (+20 more)

### Community 8 - "utils.go"
Cohesion: 0.11
Nodes (21): tokenRequest, attemptDirectory(), Inspect(), Run(), TestInspectRequiresReadableDirectory(), TestInspectRetainedRollbackFiles(), TestE2E(), getMetricsOutput() (+13 more)

### Community 9 - "KrknTargetRequest"
Cohesion: 0.12
Nodes (6): KrknTargetRequestReconciler, KrknOperatorTarget, KrknOperatorTargetSpec, KrknOperatorTargetStatus, KrknTargetRequest, KrknTargetRequestSpec

### Community 10 - "api/types.go"
Cohesion: 0.05
Nodes (54): TestLogin_InactiveUser(), TestLogin_InvalidCredentials(), TestLogin_Success(), contains(), containsMiddle(), publicScenarioReference(), TestCreateGraphRun_WithFileReferences(), TestCreateGraphRun_ResiliencyScore() (+46 more)

### Community 11 - "Broadcaster"
Cohesion: 0.05
Nodes (47): ActiveRunsOverviewResponse, AddGroupMemberRequest, AddGroupMemberResponse, ChangePasswordRequest, ChangePasswordResponse, ClusterPermissionSet, ClusterResiliencyScoreResponse, CreateTargetResponse (+39 more)

### Community 12 - "newCancellationScenarioReconciler"
Cohesion: 0.28
Nodes (13): cancellationScenario(), newCancellationScenarioReconciler(), setScenarioPodOwner(), TestKrknAIRunCancellationPropagationUsesOwnerUID(), TestScenarioCancellationDoesNotTrustForeignSameNamePod(), TestScenarioPodProjectsCancellationAnnotationAndForwardsSignal(), TestScenarioRunCancellationDeletesNeverStartedPendingPod(), TestScenarioRunCancellationPreservesCompletedJobs() (+5 more)

### Community 13 - "addAdminContext"
Cohesion: 0.08
Nodes (45): krknAIArchiveTestFile, krknAIRunReadHookClient, krknAIScenarioIndexFixture, krknAIArchiveManifestJSON(), krknAIArchiveTestEntry(), newKrknAIArchiveTestHandler(), TestKrknAIRunArchiveDownloadContainsCommittedTextAndBinaryFiles(), TestKrknAIRunArchiveDownloadPropagatesMissingManifest() (+37 more)

### Community 14 - "k8s.io/api/core/v1.Secret"
Cohesion: 0.06
Nodes (38): cloudCredentialAccessContext, buildCloudCredentialResponse(), updateCloudCredentialSecretData(), buildConnectionParams(), buildConnectionParamsForIndex(), buildElasticsearchConfigResponse(), canAccessElasticsearchConfig(), buildRegistryInfo() (+30 more)

### Community 15 - "GetClaimsFromContext"
Cohesion: 0.04
Nodes (45): required, GH_TOKEN, INPUT_OPERATOR_TAG, OPERATOR_REPO_ROOT, PUSHED_TAG, SOURCE_RELEASE_CHART, WORKFLOW_RUN_HEAD_SHA, operator_tag (+37 more)

### Community 16 - "render-bundle.sh"
Cohesion: 0.06
Nodes (37): channel_has_entries(), ensure_catalog_channel(), catalog-channel.sh script, update_default_channel(), write_release_config(), catalog_submit_checkout(), catalog_submit_open_pr(), catalog-submit.sh script (+29 more)

### Community 17 - "server.py"
Cohesion: 0.06
Nodes (6): add_DataProviderServiceServicer_to_server(), DataProviderService, DataProviderServiceServicer, DataProviderServiceStub, DataProviderServicer, serve()

### Community 18 - "category_run_configuration.go"
Cohesion: 0.12
Nodes (39): addAdminContext(), addUserContext(), createTestAdminUser(), setupFilesTestHandler(), TestCanAccessFile(), TestCreateFile(), TestCreateFile_DefaultFilePurpose(), TestCreateFile_DuplicateName() (+31 more)

### Community 19 - "contains"
Cohesion: 0.33
Nodes (6): contains(), findSubstring(), TestValidateFileGroups(), TestValidateUserFilePermissions(), ValidateFileGroups(), ValidateUserFilePermissions()

### Community 20 - ".UpdateCategory"
Cohesion: 0.33
Nodes (6): buildCategoryVisibilityLabels(), categoryErrorCode(), replaceCategoryVisibilityLabels(), validateCategoryColor(), validateCategoryName(), validateCategoryRequest()

### Community 21 - "user_handlers_test.go"
Cohesion: 0.15
Nodes (10): imageResolver, imageSignatureVerifier, KrknGraphRunReconciler, TestGraphRunReconcileSkipsPendingMigration(), TestGraphRunCancelledChildDoesNotLeaveGraphRunning(), sanitizeNodeID(), TestCalculateResiliencyScore_StoresNodeScores(), overrideRef() (+2 more)

### Community 22 - "client_test.go"
Cohesion: 0.11
Nodes (25): doerFunc, Option, roundTripperFunc, NewClient(), scenarioParametersSchema(), TestBaseURL(), TestClientTransportDefaultNotHTTPTransport(), TestClientTransportPooling() (+17 more)

### Community 23 - "KrknGraphRun"
Cohesion: 0.11
Nodes (37): newAWSTestSecret(), newCloudCredScheme(), newGCPTestSecret(), newGroupedAWSSecret(), TestCloudCredentialsRouter_MethodNotAllowed(), TestCloudCredentialsRouter_SubpathMethodNotAllowed(), TestCreateCloudCredential_AWS_Success(), TestCreateCloudCredential_DuplicateName() (+29 more)

### Community 24 - "elasticsearch/types.go"
Cohesion: 0.05
Nodes (38): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+30 more)

### Community 25 - "installer.go"
Cohesion: 0.05
Nodes (38): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+30 more)

### Community 26 - "Middleware"
Cohesion: 0.15
Nodes (35): clusterJobImage, clusterJobProvider, clusterJobRetries, graphScenarioReferences, migrationBlockedError, migrationPlan, resiliencyScoreStatus, addProvider() (+27 more)

### Community 27 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.05
Nodes (38): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+30 more)

### Community 28 - "GraphClusterScore"
Cohesion: 0.05
Nodes (38): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+30 more)

### Community 29 - "client.go"
Cohesion: 0.12
Nodes (23): esAlertSearchResponse, esSearchResponse, facetField, filterAggregation, StatusError, termsAggregation, buildAlertsSearchBody(), buildFacetAggs() (+15 more)

### Community 30 - "backup_restore_handlers_test.go"
Cohesion: 0.15
Nodes (22): NewJobTracker(), createMultipartBody(), createTestHandler(), TestGetRestoreStatusAdminOnly(), TestGetRestoreStatusTrackedJob(), TestGetRestoreStatusUnknownJob(), TestHandlerShutdownCancelsContext(), TestJobTrackerEviction() (+14 more)

### Community 31 - ".DeepCopyInto"
Cohesion: 0.06
Nodes (13): GraphScenario, KrknAIRunList, KrknCategoryList, KrknComponent, KrknComponentList, KrknGraphRunList, KrknOperatorTargetList, KrknOperatorTargetProviderConfigList (+5 more)

### Community 32 - "Server"
Cohesion: 0.10
Nodes (36): NewHandler(), NewBroadcaster(), TestBroadcastDashboardUpdate(), TestBroadcastGraphRunDeleted(), TestBroadcastGraphRunUpdate(), TestBroadcastOnlyToSubscribedClients(), TestBroadcastScenarioRunDeleted(), TestBroadcastScenarioRunUpdate() (+28 more)

### Community 33 - ".QueryTelemetryPage"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 34 - "KrknOperatorTargetProviderConfig"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 35 - "UnifiedJobItem"
Cohesion: 0.12
Nodes (18): /scenarios/detail/{scenario_name} ✅ COMPLETED, /scenarios/globals/{scenario_name} ✅ COMPLETED, mockScenarioProvider, registryImagesResult, scenarioDetailResult, ScenarioTag, aiRunRegistryImages, createScenarioProvider() (+10 more)

### Community 36 - ".DeepCopy"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 37 - "k8s.io/apimachinery/pkg/apis/meta/v1.TypeMeta"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 38 - "TokenGenerator"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 39 - "watcher_test.go"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 40 - "PROGRESS-MVP.md"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 41 - "ai.go"
Cohesion: 0.17
Nodes (14): aiContainerMatches(), aiDeploymentFieldsMatch(), hasExactlyOneAIContainer(), hasExactlyOneAIVolume(), hasExactlyOneNamedValue(), podSecurityContextMatches(), reconcileAIContainer(), reconcileAIDeployment() (+6 more)

### Community 42 - "github.com/krkn-chaos/krknctl/pkg/provider/models.RegistryV2"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 43 - "wsorigin.go"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 44 - "WSUnifiedJobItem"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 45 - "migration_test.go"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 46 - "notify-operator-docs.test.js"
Cohesion: 0.10
Nodes (11): assert, chartWorkflow, { execFileSync, spawnSync }, fs, notifier, notifyWorkflow, os, path (+3 more)

### Community 47 - "converter_test.go"
Cohesion: 0.05
Nodes (37): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+29 more)

### Community 48 - "ssrf.go"
Cohesion: 0.13
Nodes (16): configMapClient, crdClient, crdManifest, deploymentClient, applyCRD(), createMissingCRDs(), ensureMigrationGuard(), logStaleMigrationGuard() (+8 more)

### Community 49 - "KrknScenarioRunSpec"
Cohesion: 0.05
Nodes (36): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, metadata, annotations (+28 more)

### Community 50 - "Hub"
Cohesion: 0.05
Nodes (36): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, metadata, annotations (+28 more)

### Community 51 - "POST methods"
Cohesion: 0.40
Nodes (5): POST methods, refactoring, /scenarios ✅ COMPLETED, /scenarios/run/{UUID}, /targets ✅ COMPLETED

### Community 52 - "POST methods"
Cohesion: 0.06
Nodes (35): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+27 more)

### Community 53 - "captureLogSink"
Cohesion: 0.06
Nodes (35): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+27 more)

### Community 54 - "providerConfigTestScheme"
Cohesion: 0.06
Nodes (35): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+27 more)

### Community 55 - "newReportTestHandler"
Cohesion: 0.06
Nodes (35): jsonPath, name, type, controller-gen.kubebuilder.io/version, apiVersion, kind, app.kubernetes.io/name, metadata (+27 more)

### Community 56 - "cloudcreds/types.go"
Cohesion: 0.09
Nodes (48): TestGroupDiscoveryUsesSelectedKeycloakInstance(), TestIdentityProviderLifecycleAndSessionRevocation(), TestLegacyKeycloakEnvironmentProviderRemainsAvailable(), TestListUsersFiltersOIDCProvider(), TestSessionExposesMappedGroupsForOIDCEmptyState(), oidcTestAccount(), TestOIDCBlacklistCollisionKeepsAccount(), TestOIDCCheckerUsesUncachedReaderForBlock() (+40 more)

### Community 57 - ".UpdateFile"
Cohesion: 0.04
Nodes (51): AvailableFilesResponse, ListFilesResponse, buildFileInfo(), buildFileResponse(), deriveLogicalName(), extractLogicalName(), validateCreateFileRequest(), validateUpdateFileRequest() (+43 more)

### Community 58 - "dataprovider_grpc.pb.go"
Cohesion: 0.16
Nodes (5): UnimplementedDataProviderServiceServer, UnsafeDataProviderServiceServer, _DataProviderService_ExecuteKubectl_Handler(), _DataProviderService_GetNodes_Handler(), RegisterDataProviderServiceServer()

### Community 59 - ".UpdateRegistry"
Cohesion: 0.19
Nodes (29): rollbackProbeName(), rollbackRunAttemptName(), rollbackRunName(), decodeRollbackResponse(), rollbackAvailableProbe(), rollbackRegistrySecret(), rollbackRequest(), rollbackTestHandler() (+21 more)

### Community 61 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.06
Nodes (34): capabilities, categories, containerImage, description, repository, support, apiVersion, supported (+26 more)

### Community 62 - "identity_provider_store.go"
Cohesion: 0.16
Nodes (19): identityProviderConfig, IdentityProviderResponse, identityProviderStore, k8sUserStatusChecker, TestListComponentsRouteAndAuth(), sanitizeResourceName(), decodeProviderJSON(), providerResponse() (+11 more)

### Community 63 - "scenario_catalog_cache_test.go"
Cohesion: 0.06
Nodes (30): controller-gen.kubebuilder.io/version, apiVersion, kind, metadata, annotations, name, kind, listKind (+22 more)

### Community 64 - "GET methods"
Cohesion: 0.08
Nodes (10): Publisher, KrknWorkspaceReconciler, NewNamespaceFilter(), TestWorkspaceReconcilerMapsPVCWithoutOwningIt(), TestWorkspaceReconcilerTracksPVCState(), TestPublisherReconcile(), TestPublisherReconcileReturnsTransientErrors(), TestPublisherRecoversFromConcurrentCreateAndDelete() (+2 more)

### Community 65 - "installer_test.go"
Cohesion: 0.06
Nodes (30): controller-gen.kubebuilder.io/version, apiVersion, kind, metadata, annotations, name, kind, listKind (+22 more)

### Community 66 - "buildClusterAPIURLMap"
Cohesion: 0.06
Nodes (30): 1. Read the API Documentation, 1. Start the Operator, 2. Check Available Work, 2. Test Registration (First Admin), 3. Start with Foundation Task, 3. Test Login, 401 on Protected Routes, 4. Claim a Task (+22 more)

### Community 67 - "buildContainerImageFromReference"
Cohesion: 0.07
Nodes (29): apiVersion, image, imagePullPolicy, livenessProbe, name, ports, readinessProbe, resources (+21 more)

### Community 68 - "wsorigin.go"
Cohesion: 0.23
Nodes (15): checkWebSocketOriginV2(), TestCheckWebSocketOriginV2(), checkWebSocketOrigin(), TestCheckWebSocketOrigin(), canonicalOrigin(), defaultPort(), IsAllowedCookieOrigin(), IsAllowedOrigin() (+7 more)

### Community 69 - "Handler"
Cohesion: 0.07
Nodes (29): env, CHART_REPO, GH_TOKEN, IMAGE_REGISTRY, REGISTRY, jobs, notify-website, release-chart (+21 more)

### Community 70 - ".loadCategoryResiliencyHistories"
Cohesion: 0.31
Nodes (9): categoryHistoryScore, containsString(), filterCategoryHistoryScoresByCluster(), filterGraphHistoryScoresByPermission(), filterScenarioHistoryScoresByPermission(), graphHistoryScores(), runHasAnyCategory(), scenarioHistoryScores() (+1 more)

### Community 71 - "start_operator.sh"
Cohesion: 0.12
Nodes (22): GraphRunCreateRequest, GraphScenarioNode, appendGraphSignaturePart(), compareGraphConfigurations(), compareGraphConfigurationsWithStableNodeIDs(), compareGraphNodeConfigurations(), compareGraphRunConfigurations(), graphConfigurationSignature() (+14 more)

### Community 72 - "files/validation_test.go"
Cohesion: 0.07
Nodes (28): MIN_KUBE_VERSION, VERSION, version, jobs, render-and-validate, profile, name, on (+20 more)

### Community 73 - "parseKrknAIScenarioQuery"
Cohesion: 0.50
Nodes (3): krknAIScenarioQuery, matchesKrknAIScenarioFilter(), parseKrknAIScenarioQuery()

### Community 74 - "scenario_catalog_cache_test.go"
Cohesion: 0.13
Nodes (15): publicSignatureCache, publicSignatureCacheEntry, publicSignatureCachingProvider, InvalidImageSignatureError, digestPointer(), postScenariosForCacheTest(), TestPostScenariosCachesCatalogWhenTagHasNoScenarioLabels(), TestPostScenariosCachesPublicCatalog() (+7 more)

### Community 75 - "ClusterJobStatus"
Cohesion: 0.10
Nodes (18): TestCancellationCleanupReceiptParsesCoreDecision(), TestCancellationCleanupReceiptRecognizesForcedStop(), cancellationCleanupReceipt(), countJobPhaseChanges(), decodeReportPayloads(), extractDelimitedContent(), TestDecodeReportPayloadsPreservesValidReport(), TestDecodeReportPayloadsReportsInvalidHTML() (+10 more)

### Community 76 - "KrknOperatorTargetProvider"
Cohesion: 0.15
Nodes (7): Handler, hashBytes(), roleFromAdmin(), AuthorizationCheckFunc, Broadcaster, graphRunEventHandler, scenarioRunEventHandler

### Community 77 - ".PostScenarioRun"
Cohesion: 0.16
Nodes (16): categoryEnvironmentParameter, categoryHistoryRun, categoryParameterProfileFingerprint(), categoryParameterProfileName(), TestCategoryParameterProfileFingerprintForEmptyEnvironment(), TestCategoryParameterProfileFingerprintGraphEnvironmentMultiset(), TestCategoryParameterProfileFingerprintIgnoresRunType(), TestCategoryParameterProfileFingerprintScenarioEnvironment() (+8 more)

### Community 78 - "test-api.sh"
Cohesion: 0.14
Nodes (13): convertClusterResiliencyScores(), convertGraphClusterScores(), averageResiliencyScore(), convertMetaTime(), scenarioReferenceForResponse(), BuildUnifiedJobList(), getDefaultPageSize(), runCategoryNames() (+5 more)

### Community 79 - "newAIRunFixture"
Cohesion: 0.13
Nodes (22): clientContainsFinalizer(), setAIRunPodOwner(), TestKrknAIRunCancellationBeforeProvisioningRetainsRun(), TestKrknAIRunCancellationDeletesNeverStartedPendingPod(), TestKrknAIRunCancellationDoesNotTrustForeignSameNamePod(), TestKrknAIRunCancellationSettlesLostUploaderAttempt(), TestKrknAIRunCancellationWaitsForUploaderAfterExecutionStops(), TestKrknAIRunCancellationWithMissingOrchestratorRemainsUnknown() (+14 more)

### Community 80 - "ExecuteKubectlRequest"
Cohesion: 0.08
Nodes (25): description, required, type, BUNDLE_IMAGE, OLM_NAMESPACE, UPGRADE_BUNDLE_IMAGE, bundle_image, upgrade_bundle_image (+17 more)

### Community 81 - "migration_test.go"
Cohesion: 0.21
Nodes (25): Complete(), Stage(), containsSubstring(), countPatchActions(), jsonMarshal(), migrationTestLogger(), newMigrationClient(), scenarioRunFixture() (+17 more)

### Community 82 - "KrknScenarioRunReconciler"
Cohesion: 0.18
Nodes (11): CategoryListResponse, CategoryResponse, buildCategoryResponse(), categoryGroups(), categoryIsPublic(), categoryVisibleToUser(), publicHistoryCategory(), ExtractGroupNamesFromLabels() (+3 more)

### Community 83 - ".extractAndStoreReports"
Cohesion: 0.08
Nodes (24): auth, jwtExpiryHours, jwtSecret, trustedProxyCIDRs, websocketAllowedOrigins, commonAnnotations, commonLabels, crds (+16 more)

### Community 84 - "category_parameter_profile.go"
Cohesion: 0.08
Nodes (24): 1. List Available Providers (Optional), 2. Create Config Request, 3. Poll for Completion, 4. Parse Schemas and Build UI, 5. Submit Configuration, Base URL, Change Log, Common Error Patterns (+16 more)

### Community 85 - "Krkn-Operator Roadmap"
Cohesion: 0.04
Nodes (41): Architecture Decisions, Data Provider Architecture, Environment Variables, Future Work (Not in Current Scope), GET /clusters, GET /nodes, GET /targets/{UUID}, Implementation Files (+33 more)

### Community 86 - "time.Time"
Cohesion: 0.11
Nodes (10): ClusterJobStatusResponse, NodeStatusResponse, PaginationMeta, RestoreResponse, RollbackRunSummary, ScenarioRunListItem, ScenarioRunListResponse, ScenarioRunStatusResponse (+2 more)

### Community 87 - "InjectCredentials"
Cohesion: 0.20
Nodes (20): TestAppendCloudCredentialInjection_EmptyRefNoop(), injectAWS(), injectAzure(), injectBaremetal(), InjectCredentials(), injectGCP(), injectIBMCloud(), injectOpenStack() (+12 more)

### Community 88 - "ScenariosResponse"
Cohesion: 0.09
Nodes (24): console, affinity, enabled, extraEnv, ingress, nodeSelector, replicaCount, route (+16 more)

### Community 89 - "metadata_presence_test.go"
Cohesion: 0.08
Nodes (23): concurrency, cancel-in-progress, group, COLOR, jobs, test, name, on (+15 more)

### Community 90 - "GroupLabelKey"
Cohesion: 0.07
Nodes (57): failingWorkspaceGetClient, newCategoryRequest(), newCategoryRequestAs(), newCategoryTestGroup(), newCategoryTestHandler(), newCategoryTestUser(), TestCategoriesRouterCRUD(), TestCategoriesRouterGroupCreationRequiresMembership() (+49 more)

### Community 91 - "KrknOperatorTargetProviderConfig"
Cohesion: 0.08
Nodes (21): KrknOperatorTargetProviderConfigReconciler, 1. List Providers, 2. Get Provider Config Status, 2. Update Provider Status, 3. Create Provider Config Request, 3. Update Provider Configuration, API Endpoints, Architecture Decision (+13 more)

### Community 92 - "net/http.Response"
Cohesion: 0.06
Nodes (16): krknAIArchiveServiceError, Group, KeycloakDirectory, OIDCClient, ImageRequirements, GroupDirectory, NewKeycloakDirectory(), TestExternalIdentityKeyUsesIssuerAndSubject() (+8 more)

### Community 93 - "k8s.io/apimachinery/pkg/apis/meta/v1.Time"
Cohesion: 0.12
Nodes (14): GraphRunDetailResponse, GraphRunListItem, GraphRunListResponse, GraphRunSpecResponse, GraphRunStatusResponse, 1.1 Resiliency weight per graph node, 1. Request - Create GraphRun (POST /api/v1/graphruns), 2. Response - List GraphRuns (GET /api/v1/graphruns) (+6 more)

### Community 94 - "Moving from direct scenario creation to CRD approach"
Cohesion: 0.10
Nodes (21): API and operator refactoring to support multiple clusters, API Endpoints Structure (Nested Approach), BREAKING CHANGES, Configuration, Current Behavior, Implementation Details, Implementation Plan (TODO), ✅ Implemented (+13 more)

### Community 95 - "ProviderRegistration"
Cohesion: 0.08
Nodes (22): 1. Check Admin Registration, 1. Initial Load, 2. First Admin Registration, 2. Register User, 3. Login, 3. Login, 4. Authenticated Requests, 5. Role-Based UI (+14 more)

### Community 96 - "Provider Configuration API Specification"
Cohesion: 0.09
Nodes (22): Access Control, API Endpoints, ConfigMap Structure, Content Validation, Create a Public Configuration File, Create File, Delete File, Error Responses (+14 more)

### Community 97 - ".PostScenarioDetail"
Cohesion: 0.09
Nodes (21): Access Rules, Cluster Segregation Troubleshooting, Common Issues, Debugging Steps, Example 1: Proper Segregation, Example 2: Multiple Groups, Example 3: Admin User, Expected Behavior Examples (+13 more)

### Community 98 - "dataprovider.pb.go"
Cohesion: 0.20
Nodes (3): file_proto_dataprovider_proto_init(), file_proto_dataprovider_proto_rawDescGZIP(), init()

### Community 99 - "TestKrknGraphRun_ValidateGraph"
Cohesion: 0.09
Nodes (22): name, runs-on, steps, timeout-minutes, concurrency, cancel-in-progress, group, jobs (+14 more)

### Community 100 - "GraphClusterScore"
Cohesion: 0.20
Nodes (5): graphHistoryRun(), convertNodeStatusesWithScores(), convertNodeStatusesWithScores(), GraphClusterScore, NodeStatus

### Community 101 - "Authentication API Specification"
Cohesion: 0.09
Nodes (21): apiVersion, namespaceSelector, from, ports, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, app.kubernetes.io/name (+13 more)

### Community 102 - ".ListAvailableCloudCredentials"
Cohesion: 0.09
Nodes (21): apiVersion, bearerTokenFile, path, port, scheme, tlsConfig, kind, app.kubernetes.io/managed-by (+13 more)

### Community 103 - "Publisher"
Cohesion: 0.08
Nodes (21): Completed, Current Status, In Progress, Pending, Basic Usage (Default Configuration), CleanupOldResources, Config Struct, Configuration Options (+13 more)

### Community 104 - "setup-ocm-and-operator.sh"
Cohesion: 0.10
Nodes (21): image, pullPolicy, image, pullPolicy, image, pullPolicy, image, pullPolicy (+13 more)

### Community 105 - "File Management API Specification"
Cohesion: 0.10
Nodes (20): acm, enabled, console, enabled, gateway, ingress, route, enabled (+12 more)

### Community 106 - "google.golang.org/protobuf/reflect/protoreflect.Message"
Cohesion: 0.10
Nodes (20): apiVersion, kind, app.kubernetes.io/component, app.kubernetes.io/managed-by, app.kubernetes.io/name, control-plane, metadata, labels (+12 more)

### Community 107 - "ProviderRegistration"
Cohesion: 0.15
Nodes (12): API Reference, CreateProviderConfigRequest, Interfaces, NewProviderRegistration, NewProviderRegistrationWithConfig, Provider Configuration Functions, Provider Registration Functions, UpdateProviderConfig (+4 more)

### Community 108 - "commands.go"
Cohesion: 0.10
Nodes (19): Architecture Principles, Before Committing, Before Writing Code, CLAUDE.md - Project Instructions, Code Organization & Reusability, Code Quality Standards, Communication, Critical Rules (+11 more)

### Community 109 - "Cluster Segregation Troubleshooting"
Cohesion: 0.10
Nodes (19): managed-serviceaccount-version, ocm-version, jobs, ocm-setup, include, name, name, runs-on (+11 more)

### Community 110 - "channel-head.sh"
Cohesion: 0.11
Nodes (19): generated, paths, rules, formatters, enable, exclusions, linters, default (+11 more)

### Community 111 - "Running gRPC Data Provider Locally for Development"
Cohesion: 0.14
Nodes (19): resources, resources, resources, krknAI, enabled, orchestrator, resultUploader, cpu (+11 more)

### Community 112 - "docs.go"
Cohesion: 0.11
Nodes (18): acm, enabled, console, enabled, gateway, ingress, route, enabled (+10 more)

### Community 113 - "api_workflow_test.sh"
Cohesion: 0.11
Nodes (18): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, control-plane, metadata, labels, name (+10 more)

### Community 115 - "select-compatible-release.sh"
Cohesion: 0.11
Nodes (18): After (Multi-Active API), API Server Concurrent Operations, API Server Startup Wait, Before (Single Leader), Concurrency Safety, Controller Safety with Leader Election, Deployment Considerations, Expected Startup Logs (+10 more)

### Community 116 - "test-open-kubernetes-operators-pr.sh"
Cohesion: 0.12
Nodes (24): AffectedPods, AlertDocument, ClusterMetadata, CreateElasticsearchConfigResponse, DeleteElasticsearchConfigResponse, ListElasticsearchConfigsResponse, NodeSummaryInfo, QueryAlertsRequest (+16 more)

### Community 117 - "scenarioRunConfigurationFixture"
Cohesion: 0.27
Nodes (16): TestBuildCategoryResiliencyHistoryGroupsTypedConfigurationsAndSortsPerCluster(), TestCategoryResiliencyHistoryRespectsCategoryAndPerClusterVisibility(), TestGraphHistoryScoresOnlyKeepsFinalScoresIncludingZero(), TestScenarioHistoryScoresOnlyKeepsCalculatedClusterScores(), compareCategoryRunConfigurations(), assertConfigurationDifferences(), floatPointer(), graphRunConfigurationFixture() (+8 more)

### Community 118 - "post-install.sh"
Cohesion: 0.11
Nodes (18): Alerts Query API, API compatibility notes, Backup and Restore, Documentation, Ecosystem, Graph node resiliency weights, Krkn-AI Operator API, krkn-operator (+10 more)

### Community 119 - "KrknUser"
Cohesion: 0.11
Nodes (13): Configuration, Current POC limits, Groups and permissions, Identity provider management, Login and sessions, Optional Keycloak OIDC authentication (operator POC), Per-user controls and blacklist, filterUsers() (+5 more)

### Community 120 - "ResolvePrivateTagDigest"
Cohesion: 0.15
Nodes (16): filterReportLines(), isReportMarkerEnd(), isReportMarkerStart(), isWebSocketDisconnectError(), maskToken(), podLogOptions(), preStartFailureReason(), sanitizeHeaders() (+8 more)

### Community 121 - "test-channel-head.sh"
Cohesion: 0.33
Nodes (4): BroadcastMessage, Client, Hub, PaginationClientState

### Community 122 - "test-chart-exposure.sh"
Cohesion: 0.11
Nodes (17): concurrency, cancel-in-progress, group, jobs, security-and-quality, name, on, pull_request (+9 more)

### Community 123 - "test-select-compatible-release.sh"
Cohesion: 0.11
Nodes (17): concurrency, cancel-in-progress, group, jobs, test-e2e, name, on, pull_request (+9 more)

### Community 124 - ".PostScenarioRun"
Cohesion: 0.19
Nodes (11): Code References, writeScenarioRunAccessError(), parseBoolHeader(), TestParseBoolHeader(), sanitizeRunNameLabel(), resolveMaxRetries(), IsCloudEnvVar(), StripCloudEnvVars() (+3 more)

### Community 125 - "debug-cluster-permissions.sh"
Cohesion: 0.17
Nodes (16): KrknAIArtifactSummary, KrknAIConfigRequest, KrknAIConfigValidationError, KrknAIConfigValidationRequest, KrknAIConfigValidationResponse, KrknAIDiscoveryRequest, KrknAIFitnessProgression, KrknAIRunRequest (+8 more)

### Community 126 - "notify-operator-docs.sh"
Cohesion: 0.11
Nodes (17): Architecture, Build Installer Bundle, Custom Push, Deployment Guide, Environment Variables, Git Tag Support, Git Tag Workflow, Image Names (Auto-Generated) (+9 more)

### Community 127 - "github.com/krkn-chaos/krkn-operator"
Cohesion: 0.10
Nodes (17): 1. Setup gRPC Server, 2. Run Operator (new terminal), 3. Test the API (new terminal), "401 Unauthorized", 🧪 Automated Testing, "Command not found: kubectl", 🔥 Common Issues, Debugging (+9 more)

### Community 128 - "Setup"
Cohesion: 0.14
Nodes (16): Code Refactoring: getKubeconfigFromTarget() Helper ✅, Implementation Details, Phase 10: POST /scenarios/detail/{scenario_name} Endpoint ✅, Phase 11: POST /scenarios/globals/{scenario_name} Endpoint ✅, Phase 1: REST API Setup ✅, Phase 2: GET /clusters Endpoint ✅, Phase 3: Integration ✅, Phase 4: Testing ✅ (+8 more)

### Community 129 - "GenerateFromToken"
Cohesion: 0.18
Nodes (16): CreateTargetRequest, UpdateTargetRequest, generateKubeconfigFromCredentialsType(), generateKubeconfigFromKubeconfigType(), generateKubeconfigFromRequest(), generateKubeconfigFromTokenType(), TestCreateTarget_DuplicateClusterName(), TestCreateTarget_WithKubeconfig() (+8 more)

### Community 130 - "ProviderRegistration"
Cohesion: 0.22
Nodes (8): Doer, executeSearch(), Client, ConnectionParams, newSearchRequest(), TestTransportKey(), transportKey(), validateInlineDestination()

### Community 131 - "krkn-operator"
Cohesion: 0.15
Nodes (16): service, service, monitoring, enabled, service, serviceMonitor, service, annotations (+8 more)

### Community 132 - "Deployment Guide"
Cohesion: 0.14
Nodes (15): CreateCloudCredentialResponse, DeleteCloudCredentialResponse, ListCloudCredentialsResponse, UpdateCloudCredentialResponse, buildCloudCredentialSecretData(), CloudCredentialResponse, CreateCloudCredentialRequest, UpdateCloudCredentialRequest (+7 more)

### Community 133 - "Quick Start - Terminal API Local Testing"
Cohesion: 0.12
Nodes (15): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, metadata, labels, name, roleRef (+7 more)

### Community 134 - "Resiliency Score Feature - Frontend Implementation Guide"
Cohesion: 0.12
Nodes (15): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, metadata, labels, name, roleRef (+7 more)

### Community 135 - "AGENTS.md"
Cohesion: 0.15
Nodes (3): DataProviderServiceClient, GetNodesRequest, GetNodesResponse

### Community 136 - ".updateClusterJobStatuses"
Cohesion: 0.10
Nodes (14): containerStatusHasStarted(), podHasStartedContainer(), backfillLegacyMaxRetries(), TestBackfillLegacyMaxRetries(), TestFailedAttemptRetainsRollbackIdentityBeforeRetry(), TestShouldRetryJobDefaultsZeroRetries(), TestShouldRetryJobHonorsConfiguredRetryLimit(), TestUpdateClusterJobStatuses_AIRunDisablesRetries() (+6 more)

### Community 137 - "krkn-operator-data-provider"
Cohesion: 0.12
Nodes (15): Backend Implementation Details, Backend Status: ✅ Completato, Example: Resiliency Score Badge Component, Expected Behavior, Frontend Implementation Guidelines, Manual Testing Steps, Overview, Phase 1: Display Score in List View (+7 more)

### Community 138 - "decodeTelemetry"
Cohesion: 0.53
Nodes (6): FacetOption, QueryTelemetryResponse, TelemetryStats, decodeTelemetry(), facetsFromBuckets(), statsFromBuckets()

### Community 139 - "API Workflow Test"
Cohesion: 0.12
Nodes (15): concurrency, cancel-in-progress, group, jobs, stale, name, on, schedule (+7 more)

### Community 140 - "WSUnifiedJobItem"
Cohesion: 0.10
Nodes (14): TestFilterJobsByCategoriesMatchesAnySelectedCategory(), TestPaginateJobItems(), computeWSJobStats(), filterJobCategoriesByVisibility(), filterJobsByCategories(), groupRollbackJobs(), paginateJobItems(), TestGroupRollbackJobsWebSocketKeepsFlatViewAndChecksUID() (+6 more)

### Community 141 - "sync_test.go"
Cohesion: 0.10
Nodes (21): 1. Login and Get JWT Token, 2. Create KrknTargetRequest (or use existing UUID), 3. Test Terminal Command, Cleanup, Configure Operator to Use Local gRPC Server, Debug gRPC Server in PyCharm, Development Workflow, Error: "Command not found: kubectl" (+13 more)

### Community 142 - "rate_limiter_test.go"
Cohesion: 0.13
Nodes (13): Beads Task Tracking, Database and project identity, Exports and Git, Go Operators â€” Only for `project.type = "go-operator"`, graphify, Handoff, Issue lifecycle, Project Configuration (+5 more)

### Community 143 - "k8s.io/client-go/kubernetes.Interface"
Cohesion: 0.37
Nodes (16): aiCreateRace(), aiDeployment(), aiLabels(), aiPodLabels(), applyLabels(), ensureAIDeployment(), ensureAINetworkPolicy(), ensureAIOrchestratorRole() (+8 more)

### Community 144 - "🔧 TODO: Pod Recreation and Retry Logic"
Cohesion: 0.15
Nodes (15): securityContext, securityContext, operator, affinity, enabled, extraEnv, nodeSelector, olmBootstrap (+7 more)

### Community 145 - "API Changes"
Cohesion: 0.18
Nodes (13): addKnownTypes(), decodeErrorResponse(), providerConfigTestScheme(), TestGetProviderConfigByUUID_CompletedReturnsConfigData(), TestGetProviderConfigByUUID_InternalErrorReturnsSanitized500(), TestGetProviderConfigByUUID_NotFoundReturns404(), TestGetProviderConfigByUUID_PendingReturnsAccepted(), TestPostProviderConfig_FailureReturnsSanitizedError() (+5 more)

### Community 146 - "Running Locally"
Cohesion: 0.15
Nodes (13): FromKrknctlScenarioSet(), publicScenarioReference(), publicScenario(), TestFromKrknctlScenarioSetCreatesPublicReferences(), TestToKrknctlScenarioSetUsesScenarioReference(), ToKrknctlScenarioSet(), MapScenarioNodeToScenarioRunSpec(), ResolveGraph() (+5 more)

### Community 147 - "install-yq.sh"
Cohesion: 0.16
Nodes (14): acm, affinity, config, enabled, extraEnv, logging, nodeSelector, replicaCount (+6 more)

### Community 148 - "test-release-workflow-gates.sh"
Cohesion: 0.14
Nodes (13): apiVersion, images, name, newName, newTag, kind, namePrefix, namespace (+5 more)

### Community 149 - "Deployment"
Cohesion: 0.14
Nodes (13): apiVersion, kind, metadata, name, namespace, roleRef, apiGroup, kind (+5 more)

### Community 151 - "Building Images"
Cohesion: 0.14
Nodes (13): jobs, rebase, name, on, pull_request_target, push, permissions, contents (+5 more)

### Community 152 - "Troubleshooting"
Cohesion: 0.14
Nodes (14): with, build-args, cache-from, cache-to, context, if-no-files-found, name, password (+6 more)

### Community 153 - "AIConfig"
Cohesion: 0.16
Nodes (15): krknAIPodOptions, KrknAIRunPodConfig, AIConfigFromEnv(), decodeStrictJSON(), defaultAIServiceResources(), localObjectReferencesEqual(), parseImagePullSecrets(), parsePullPolicy() (+7 more)

### Community 154 - "newProvidersHandler"
Cohesion: 0.18
Nodes (12): Phase 12: Scenario Job Management - /scenarios/run Endpoints ✅, Phase 9: POST /scenarios Endpoint ✅, FileMount, GlobalsRequest, JobsListResponse, JobStatusResponse, ScenarioRunRequest, ScenariosRequest (+4 more)

### Community 155 - "Troubleshooting"
Cohesion: 0.15
Nodes (12): apiVersion, appVersion, description, home, keywords, maintainers, name, url (+4 more)

### Community 156 - ".GetScenarioRunLogs"
Cohesion: 0.22
Nodes (5): visibleCategoryNames(), AuthorizationChecker, categoryVisibilityChecker, ClientMessage, Handler

### Community 157 - "assets/README.md"
Cohesion: 0.15
Nodes (12): apiVersion, kind, metadata, name, roleRef, apiGroup, kind, name (+4 more)

### Community 158 - "GenerateFromToken"
Cohesion: 0.20
Nodes (15): GroupRollbackJobs(), makeGraphRun(), makeScenarioRun(), TestGroupRollbackJobs_UsesSourceUIDAndKeepsOrphans(), TestListJobs_GroupedRollbacksBeforePagination(), TestListJobs_IncludesStats(), TestListJobs_LimitOnlyReturnsAll(), TestListJobs_NoPagination() (+7 more)

### Community 159 - "ScenarioReference"
Cohesion: 0.09
Nodes (25): KrknScenarioRunSpec, TestLegacyScenarioIdentityRemainsDecodable(), TestResolveScenarioReferenceMigratesOnlyLegacyIdentity(), compareScenarioRunConfigurations(), compareStringMaps(), optionalBoolValue(), sameScenarioFiles(), sameScenarioReference() (+17 more)

### Community 160 - "NewSecretManager"
Cohesion: 0.41
Nodes (12): build_operator(), check_prerequisites(), cleanup(), create_service_account(), install_crds(), main(), print_error(), print_info() (+4 more)

### Community 161 - "KrknUserGroupSpec"
Cohesion: 0.12
Nodes (21): AvailableCommandsResponse, TestExecuteTerminal_CommandParsing(), TestExecuteTerminal_ValidationLogic(), GetAllowedCommands(), GetBlockedFlags(), CommandSpec, IsCommandAllowed(), IsSubcommandAllowed() (+13 more)

### Community 162 - "ai_test.go"
Cohesion: 0.17
Nodes (12): aiOrchestratorSCCBindingName(), newAITokenSecret(), requireOwnedLabels(), testAIConfig(), TestAIResourcesLeavesUserPVCUnchanged(), TestAIResourcesPreservesDefaultedDeploymentFields(), TestAIResourcesReconcilesNetworkPolicyDrift(), TestAIResourcesRefusesForeignObjectsAndPropagatesAPIErrors() (+4 more)

### Community 163 - "krkngraphrun_controller_sanitization_test.go"
Cohesion: 0.17
Nodes (12): aiService, networkPolicy, resources, serviceToken, storage, networkPolicy, egress, enabled (+4 more)

### Community 164 - "BuildUnifiedJobList"
Cohesion: 0.17
Nodes (11): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, metadata, labels, name, rules (+3 more)

### Community 165 - "WorkflowResponse"
Cohesion: 0.17
Nodes (12): Configuration Options, Custom namespace, Disable console (operator only), Install on Kubernetes with Ingress, Install on OpenShift with Route, Install with ACM integration (OpenShift), Install with defaults (console enabled, ACM disabled), Installation Methods (+4 more)

### Community 166 - "contents"
Cohesion: 0.25
Nodes (8): permissions, name, needs, permissions, runs-on, create-manifests, contents, permissions

### Community 167 - "Pushing Images"
Cohesion: 0.17
Nodes (11): API, Dependencies, Development, GetNodes, Installation, krkn-operator-data-provider, Overview, Prerequisites (+3 more)

### Community 168 - "TestValidateValueAgainstSchema"
Cohesion: 0.39
Nodes (11): api_call(), cleanup(), extract_body(), extract_http_code(), log_error(), log_info(), log_step(), log_success() (+3 more)

### Community 169 - "GET methods"
Cohesion: 0.07
Nodes (28): API Requirements, /clusters, DELETE methods, GET methods, GET methods, get nodes, get pods, Grpc python service requirement (+20 more)

### Community 170 - "ComponentManifest"
Cohesion: 0.15
Nodes (7): ListComponentsResponse, Load(), TestLoad(), TestLoadRejectsInvalidManifest(), ComponentDescription, ComponentManifest, KrknComponentStatus

### Community 171 - "target"
Cohesion: 0.18
Nodes (10): apiVersion, kind, patches, path, target, resources, group, kind (+2 more)

### Community 173 - "buildUnifiedJobList"
Cohesion: 0.17
Nodes (12): TestBuildUnifiedJobList_SkipsGraphRunChildren(), averageResiliencyScore(), buildScenarioRunDetailResponse(), buildScenarioRunResponse(), buildUnifiedJobList(), convertClusterResiliencyScoresWS(), runCategoryNames(), sanitizedClusterJobs() (+4 more)

### Community 174 - "DataProviderServiceServer"
Cohesion: 0.18
Nodes (10): name, permissions, runs-on, steps, jobs, build-operator-amd64, name, on (+2 more)

### Community 175 - "name"
Cohesion: 0.23
Nodes (14): name, runs-on, steps, steps, COMMUNITY_OPERATORS_FORK, GH_TOKEN, KUBERNETES_OPERATORS_FORK, build-data-provider-amd64 (+6 more)

### Community 176 - "slash-to-add-label.yml"
Cohesion: 0.18
Nodes (10): permissions, uses, types, jobs, add-label, name, on, issue_comment (+2 more)

### Community 177 - "newReportTestHandler"
Cohesion: 0.33
Nodes (9): newReportTestHandler(), reportRequest(), TestDownloadReportHTMLThroughV2Router(), TestDownloadReportPDF(), TestDownloadReportRejectsMalformedPath(), TestDownloadReportUnavailable(), TestGetReportStatusPending(), TestGetReportStatusRejectsExtraPathSegments() (+1 more)

### Community 178 - "API Workflow Test"
Cohesion: 0.18
Nodes (10): API Workflow Test, Configuration, Error Handling, krkn-operator Test Scripts, Output, Overview, Prerequisites, Testing Against Production (+2 more)

### Community 181 - "bootstrap.go"
Cohesion: 0.21
Nodes (11): ensureJWTSecret(), EnsureResources(), ensureSCCBinding(), ensureScenarioRunnerRBAC(), ensureServiceAccount(), ensureServices(), intstrFromInt(), namespaceHash() (+3 more)

### Community 182 - "time.Duration"
Cohesion: 0.25
Nodes (4): ipRateLimiter, rateLimiterEntry, getTokenDuration(), TestGetTokenDuration()

### Community 183 - "TokenGenerator"
Cohesion: 0.23
Nodes (4): mockTokenGen(), mockTokenGenInvalid(), Claims, TokenGenerator

### Community 184 - "manager.yaml"
Cohesion: 0.20
Nodes (9): apiVersion, kind, app.kubernetes.io/name, control-plane, matchLabels, spec, replicas, selector (+1 more)

### Community 185 - "labels"
Cohesion: 0.20
Nodes (10): kubectl.kubernetes.io/default-container, app.kubernetes.io/managed-by, app.kubernetes.io/name, control-plane, metadata, annotations, labels, name (+2 more)

### Community 186 - "workspace_role.yaml"
Cohesion: 0.20
Nodes (9): apiVersion, kind, metadata, name, namespace, rules, apiGroups, resources (+1 more)

### Community 187 - "ValidateCreateRequest"
Cohesion: 0.25
Nodes (9): CreateElasticsearchConfigRequest, UpdateElasticsearchConfigRequest, strPtr(), TestValidateUpdateRequest(), validateAccess(), ValidateCreateRequest(), ValidateMergedConnection(), validateTLSSettings() (+1 more)

### Community 188 - "sortKrknAIScenarios"
Cohesion: 0.36
Nodes (9): KrknAIScenarioIndexItem, compareInt(), compareKrknAIScenarioIdentity(), compareKrknAIScenarioIDs(), compareKrknAIScenarioMeasurements(), compareKrknAIScenarioStrings(), isKrknAIScenarioNumericID(), krknAIScenarioStatus() (+1 more)

### Community 189 - "gateway"
Cohesion: 0.22
Nodes (9): gateway, annotations, enabled, gatewayName, gatewayNamespace, hostname, path, pathType (+1 more)

### Community 190 - "containers"
Cohesion: 0.22
Nodes (9): args, command, env, image, imagePullPolicy, name, ports, volumeMounts (+1 more)

### Community 191 - "labels"
Cohesion: 0.22
Nodes (8): apiVersion, kind, app.kubernetes.io/managed-by, app.kubernetes.io/name, control-plane, metadata, labels, name

### Community 192 - "metrics_auth_role.yaml"
Cohesion: 0.22
Nodes (8): apiVersion, kind, metadata, name, rules, apiGroups, resources, verbs

### Community 193 - "role.yaml"
Cohesion: 0.22
Nodes (8): apiVersion, kind, metadata, name, rules, apiGroups, resources, verbs

### Community 194 - "Optional Krkn-AI with OLM on OpenShift"
Cohesion: 0.22
Nodes (9): 1. Build a development bundle, 2. Install with AI disabled, 3. Enable AI, 4. Verify a run, Disable, upgrade, and remove, Optional Krkn-AI with OLM on OpenShift, Prerequisites, Reset a disposable test installation (+1 more)

### Community 195 - "publish-olm-bundles"
Cohesion: 0.22
Nodes (9): IMAGE_NAMESPACE, REGISTRY, USE_IMAGE_DIGESTS, publish-olm-bundles, env, name, needs, permissions (+1 more)

### Community 196 - "entries"
Cohesion: 0.25
Nodes (8): entries, defaultChannel, entries, name, package, replaces, schema, schema

### Community 197 - "NewCachedUserStatusChecker"
Cohesion: 0.12
Nodes (13): CachedUserStatusChecker, countingUserStatusChecker, userStatusCacheEntry, UserStatusChecker, ExternalIdentityChecker, Middleware, NewCachedUserStatusChecker(), NewLazyMiddleware() (+5 more)

### Community 198 - "manager/kustomization.yaml"
Cohesion: 0.25
Nodes (7): apiVersion, images, name, newName, newTag, kind, resources

### Community 199 - "spec"
Cohesion: 0.25
Nodes (8): type, runAsNonRoot, seccompProfile, securityContext, serviceAccountName, terminationGracePeriodSeconds, volumes, spec

### Community 200 - "metrics_reader_role.yaml"
Cohesion: 0.25
Nodes (7): apiVersion, kind, metadata, name, rules, nonResourceURLs, verbs

### Community 201 - "config.yaml"
Cohesion: 0.25
Nodes (7): apiVersion, kind, metadata, name, stages, parallel, tests

### Community 202 - "submit-operatorhub-catalog"
Cohesion: 0.22
Nodes (9): RELEASE_TAG, submit-operatorhub-catalog, env, env, if, name, needs, permissions (+1 more)

### Community 203 - "buildClusterAPIURLMap"
Cohesion: 0.25
Nodes (8): buildClusterAPIURLMap(), contains(), containsInMiddle(), TestBuildClusterAPIURLMap(), TestBuildClusterAPIURLMap_Collision(), TestBuildClusterAPIURLMap_SameNameSameURL(), TestValidateScenarioRunAccess(), TestValidateScenarioRunAccess_NoGroups()

### Community 204 - "CategoryResiliencyHistoryResponse"
Cohesion: 0.38
Nodes (7): CategoryConfigurationGroup, CategoryResiliencyDataPoint, CategoryResiliencyHistoryResponse, ResiliencyHistoryQueryResponse, datapointForRunID(), datapointRunIDs(), assembleResiliencyHistoryQueryResponse()

### Community 206 - "ScenariosResponse"
Cohesion: 0.48
Nodes (4): publicScenarioCache, publicScenarioCacheEntry, ScenariosResponse, cloneScenariosResponse()

### Community 207 - "volumes"
Cohesion: 0.29
Nodes (7): volumes, sizeLimit, sizeLimit, sizeLimit, output, tmp, uploadState

### Community 208 - "sync_test.go"
Cohesion: 0.10
Nodes (16): ConfigStoreInitializer, NewConfigStoreInitializer(), TestConfigStoreInitializer_NeedLeaderElection(), TestConfigStoreInitializer_Start_ConfigMapNotFound(), TestConfigStoreInitializer_Start_Success(), Store, SyncConfigMapToStore(), TestRoundTrip() (+8 more)

### Community 209 - "entries"
Cohesion: 0.33
Nodes (6): entries, entries, name, package, schema, schema

### Community 210 - "Running Locally"
Cohesion: 0.29
Nodes (7): 1. Start the gRPC data provider, 2. Run the operator, 3. Create an admin user and log in, 4. (Optional) Run the web console, Architecture overview, Prerequisites, Running Locally

### Community 211 - "Krkn-Operator Roadmap"
Cohesion: 0.29
Nodes (6): Krkn-Operator Roadmap, Ongoing (not release-blocked), Release 1 — 2026-08-18, Release 2 — 2026-10-18, Release 3 — 2026-12-22, Release 4 — 2027-02-23

### Community 212 - "TestKrknGraphRun_ValidateGraph"
Cohesion: 0.33
Nodes (5): SanitizeNodeIDForKubernetes(), stringPtr(), TestCollisionDetection(), TestKrknGraphRun_ValidateGraph(), TestSanitizeNodeIDForKubernetes()

### Community 213 - "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured.Unstructured"
Cohesion: 0.22
Nodes (10): migrationResult, migrationStats, resourceDefinition, checkAnnotationSize(), forEachResource(), logSummary(), patchMigrationResourceWithRetry(), writeMigrationAnnotation() (+2 more)

### Community 214 - "submit-community-catalog"
Cohesion: 0.33
Nodes (6): submit-community-catalog, if, name, needs, runs-on, steps

### Community 215 - "Compute"
Cohesion: 0.22
Nodes (5): ComputeJobStats(), TestComputeJobStats(), Compute(), JobCounter, Summary

### Community 216 - "KrknGraphRunStatus"
Cohesion: 0.15
Nodes (4): GraphRunSummary, KrknAIRunStatus, KrknGraphRunStatus, KrknWorkspaceStatus

### Community 217 - "setup-ocm-and-operator.sh"
Cohesion: 0.67
Nodes (5): log_error(), log_info(), log_warn(), setup-ocm-and-operator.sh script, wait_for_condition()

### Community 218 - "updateStrategy"
Cohesion: 0.40
Nodes (5): maxSurge, maxUnavailable, updateStrategy, rollingUpdate, type

### Community 219 - "Available Make Targets"
Cohesion: 0.40
Nodes (5): Available Make Targets, Build Targets, Deployment Targets, Other Targets, Push Targets

### Community 220 - "Building Images"
Cohesion: 0.40
Nodes (5): Build Both Images, Build Individual Images, Building Images, Custom Registry/Images, Using Podman

### Community 221 - "Troubleshooting"
Cohesion: 0.40
Nodes (5): gRPC Connection Issues, Image Pull Errors, Namespace Issues, Pod Not Starting, Troubleshooting

### Community 222 - "steps"
Cohesion: 0.40
Nodes (5): name, permissions, runs-on, steps, build-data-provider-arm64

### Community 223 - "steps"
Cohesion: 0.40
Nodes (5): name, permissions, runs-on, steps, build-operator-arm64

### Community 224 - "Setup"
Cohesion: 0.40
Nodes (5): 1. Create Virtual Environment (Python 3.11), 2. Install Dependencies, 3. Verify Generated Stubs, 4. Start gRPC Server, Setup

### Community 225 - "ExtractRegistryV2FromSecret"
Cohesion: 0.18
Nodes (13): BuildDockerConfigJSON(), ExtractRegistryV2FromSecret(), TestBuildDockerConfigJSON_MissingCredentials(), TestBuildDockerConfigJSON_OAuth(), TestBuildDockerConfigJSON_Password(), TestExtractRegistryV2FromSecret_InvalidType(), TestExtractRegistryV2FromSecret_MissingAnnotations(), TestExtractRegistryV2FromSecret_MissingComponent() (+5 more)

### Community 226 - "livenessProbe"
Cohesion: 0.50
Nodes (4): livenessProbe, httpGet, initialDelaySeconds, periodSeconds

### Community 227 - "readinessProbe"
Cohesion: 0.50
Nodes (4): readinessProbe, httpGet, initialDelaySeconds, periodSeconds

### Community 228 - "olm/kustomization.yaml"
Cohesion: 0.50
Nodes (3): apiVersion, kind, resources

### Community 229 - "Complete Workflow Examples"
Cohesion: 0.50
Nodes (4): Complete Workflow Examples, Custom Registry, Development (Local), Production (With Git Tag)

### Community 230 - ".gosec.yaml"
Cohesion: 0.50
Nodes (3): exclude-dirs, exclude-generated, excludes

### Community 231 - "channel-head.sh"
Cohesion: 0.50
Nodes (3): OLM_CHANNEL_NAME, OLM_PACKAGE_NAME, channel-head.sh script

### Community 232 - "select-ai-release.sh"
Cohesion: 0.83
Nodes (3): inspect_image(), select-ai-release.sh script, usage()

### Community 233 - "krkngraphrun_controller_sanitization_test.go"
Cohesion: 0.83
Nodes (3): isAlphanumeric(), isValidLabelChar(), TestSanitizeNodeID()

### Community 237 - "resources"
Cohesion: 0.67
Nodes (3): resources, limits, requests

### Community 238 - "securityContext"
Cohesion: 0.67
Nodes (3): securityContext, allowPrivilegeEscalation, capabilities

### Community 244 - "installer_test.go"
Cohesion: 0.23
Nodes (12): applyCRDs(), TestApplyCRDsCreatesMissingAndUpdatesChangedSpecs(), TestApplyCRDsDoesNotPatchUnchangedSpecs(), TestApplyCRDsLabelsExistingCRDsWithoutChangingOtherLabels(), TestApplyCRDsRejectsNonCRDManifest(), testCRD(), TestCreateMissingCRDsBeforeRolloutLeavesExistingSpecsUntouched(), TestWaitForDeploymentReady() (+4 more)

### Community 245 - "NewServerWithComponentNamespace"
Cohesion: 0.17
Nodes (9): Role, discoveryResponseDeadlineMiddleware(), loggingMiddleware(), maxBodySizeMiddleware(), TestDiscoveryResponseDeadlineIsRouteSpecific(), TestMaxBodySizeMiddleware(), handlerOrStatus(), TestCheckClusterLiveness() (+1 more)

### Community 261 - "rate_limiter_test.go"
Cohesion: 0.16
Nodes (18): clientIPFromHeaders(), extractClientIP(), hostFromRemoteAddr(), isTrustedProxy(), newIPRateLimiter(), parseTrustedProxyCIDRs(), rateLimitMiddleware(), TestExtractClientIP() (+10 more)

### Community 266 - "parseResiliencyHistoryQuery"
Cohesion: 0.18
Nodes (10): resiliencyHistoryQuerySelections, normalizeResiliencyHistoryProviders(), normalizeResiliencyHistorySelections(), parseResiliencyHistoryQuery(), createTarGz(), TestCreateAndExtractTarGz(), TestCreateTarGzFlushErrors(), TestExtractTarGzPathTraversal() (+2 more)

### Community 267 - ".reconcileDeletion"
Cohesion: 0.15
Nodes (11): indexChildRun(), formatUIDAnnotation(), hasAIRunOwner(), hasScenarioRunOwner(), parseUIDAnnotation(), sameUIDs(), uniqueUIDs(), SubPathFor() (+3 more)

### Community 268 - "ResolvePrivateTagDigest"
Cohesion: 0.21
Nodes (16): resolveRollbackDigest(), ResolvePrivateTagDigest(), ResolvePublicTagDigest(), serverURL(), testManifestDigest(), testPrivateRegistry(), TestResolvePrivateTagDigestCancellation(), TestResolvePrivateTagDigestHonorsSkipTLS() (+8 more)

### Community 269 - "runDeletionToCompletion"
Cohesion: 0.27
Nodes (10): holdChildPodDeleteClient, aiOrchestratorPodName(), aiResourceName(), TestAIResourceNameKeepsLongRunNamesDistinct(), runDeletionToCompletion(), setAIRunDeleting(), TestReconcileDeletesTerminalRun(), TestReconcileDeletionCleansOnlyUIDOwnedResources() (+2 more)

### Community 270 - "NewServerWithComponentNamespace"
Cohesion: 0.13
Nodes (9): KrknAIServiceConfig, Server, NewHandler(), NewServer(), NewServerWithComponentNamespace(), generateRandomSecret(), getSecretKeys(), SecretManager (+1 more)

### Community 271 - "ClusterTarget"
Cohesion: 0.18
Nodes (7): ClustersResponse, countClustersFromTargetData(), TestCountClustersFromTargetData(), TestFilterClustersByPermission(), ClusterHealthStatus, ClusterTarget, KrknTargetRequestStatus

### Community 272 - ".Reconcile"
Cohesion: 0.26
Nodes (11): aiRunLabelValue(), buildOrchestratorPod(), defaultPullPolicy(), assertAIRunPodRuntimeConfig(), TestEnsureProvisionedUsesEmptyDirAndUploader(), newPendingAIRunFixture(), TestReconcileFailsPermanentProvisioningErrors(), TestReconcileIgnoresLegacyOrchestratorImage() (+3 more)

### Community 273 - "newTestScheme"
Cohesion: 0.36
Nodes (9): newTargetRequest(), newTestScheme(), TestGetClusterAPIURL_ClusterNotFound(), TestGetClusterAPIURL_Found(), TestGetClusterAPIURL_ProviderNotFound(), TestGetClusterAPIURL_TargetRequestNotFound(), TestReconcile_FailedJobStatus_IncludesClusterAPIURL(), TestReconcile_ImageResolutionFailureHasSafeDiagnostic() (+1 more)

### Community 274 - "main"
Cohesion: 0.18
Nodes (11): bootstrapOLMResources(), main(), resolveKrknAIEnabled(), resolveMigrationNamespace(), runCRDMigrationMode(), TestResolveKrknAIEnabledPrecedence(), TestResolveMigrationNamespacePrefersHookNamespace(), TestRunCRDMigrationMode() (+3 more)

### Community 276 - "guardedDialContext"
Cohesion: 0.25
Nodes (4): responseWriter, guardedDialContext(), TestValidateDestinationIP(), validateDestinationIP()

### Community 277 - "metadata_presence_test.go"
Cohesion: 0.19
Nodes (12): rawTelemetrySource, intPtr(), TestFlattenRepresentativeScenario(), TestRawTelemetrySourceFlatten(), contains(), containsAt(), TestMetadataJSONMarshaling(), TestMetadataPresenceTracking() (+4 more)

### Community 278 - ".UpdateUserGroup"
Cohesion: 0.36
Nodes (3): validateClusterPermissions(), validateExternalGroups(), IsValidAction()

### Community 279 - "KrknOperatorTargetProviderConfigStatus"
Cohesion: 0.29
Nodes (3): getProviderNames(), KrknOperatorTargetProviderConfigStatus, ProviderConfigData

### Community 280 - "KrknAIRunReconciler"
Cohesion: 0.33
Nodes (3): KrknAIRunReconciler, oneTarget(), requiredResourceError()

### Community 281 - "Deployment"
Cohesion: 0.33
Nodes (6): Custom Images, Custom Namespace, Deployment, Install CRDs Only, OpenShift Deployment, Quick Deploy

### Community 282 - "krknairun_config_test.go"
Cohesion: 0.47
Nodes (4): KrknAIRunPodConfigFromEnv(), TestKrknAIRunPodConfigPreservesBoundsWithPartialOverrides(), TestKrknAIRunPodConfigRejectsInvalidResources(), TestKrknAIRunPodConfigRejectsInvalidVolumeBounds()

### Community 283 - "install-ai.sh"
Cohesion: 0.33
Nodes (5): NAMESPACE, REGISTRY, install-ai.sh script, STORAGE_CLASS, TAG

### Community 284 - "scenario_filter_test.go"
Cohesion: 0.24
Nodes (6): filterScenariosByIsAScenario(), TestFilterScenariosByIsAScenario(), TestFilterScenariosByIsAScenario_CancelledContext(), TestFilterScenariosByIsAScenario_PreservesTagMetadata(), TestValidateValueAgainstSchema(), ValidateValueAgainstSchema()

### Community 285 - "elasticsearch/labels_test.go"
Cohesion: 0.33
Nodes (5): TestBuildAnnotations(), TestBuildLabels(), TestBuildLabelsAccessControl(), TestUpdateAnnotations(), UpdateAnnotations()

### Community 287 - ".checkRollbackImage"
Cohesion: 0.39
Nodes (4): rollbackImageCache, rollbackImageCheck, Rollback artifact persistence, Rollback history in Jobs

### Community 290 - "convertClusterJobs"
Cohesion: 0.25
Nodes (6): convertClusterJobs(), TestConvertClusterJobs_EmptyInput(), TestConvertClusterJobs_NilInput(), TestConvertClusterJobs_OmitsClusterAPIURL(), ScenarioAttemptStatus, ClusterJobResponse

### Community 293 - "GraphRunResponse"
Cohesion: 0.29
Nodes (3): buildGraphRunResponse(), convertGraphClusterScoresForSnapshot(), GraphRunResponse

### Community 294 - ".executeKubectlViaGRPC"
Cohesion: 0.33
Nodes (3): TerminalRequest, TerminalResponse, NewDataProviderServiceClient()

### Community 296 - "probeAITags"
Cohesion: 0.40
Nodes (3): classifyImageProbeError(), probeAITags(), imageProbeResult

### Community 297 - "validateDestinationURL"
Cohesion: 0.40
Nodes (4): guardedCheckRedirect(), TestGuardedCheckRedirect(), TestValidateDestinationURL(), validateDestinationURL()

### Community 298 - "privateAIRunFixture"
Cohesion: 0.40
Nodes (4): privateAIRunFixture(), TestAIChildCannotChangeParentRegistry(), TestPrivateAIRunUsesSelectedRuntimeImagesAndContext(), TestPrivateAIRunValidatesImagesBeforeCreatingWorkloads()

### Community 300 - "stringPtr"
Cohesion: 0.67
Nodes (3): TestUpdateFileAnnotations_PointerSemantics(), invalidWorkflowGraph(), stringPtr()

## Knowledge Gaps
- **25 isolated node(s):** `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME`, `ICON_BASE64`, `ICON_MEDIATYPE` (+20 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 2028 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **44 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Handler` connect `Handler` to `main.go`, `ProviderRegistration`, `context.Context`, `KrknWorkspace`, `k8s.io/api/core/v1.Secret`, `NewServerWithComponentNamespace`, `k8s.io/client-go/kubernetes.Interface`, `JobTracker`, `.UpdateCategory`, `.UpdateUserGroup`, `.checkRollbackImage`, `UnifiedJobItem`, `.executeKubectlViaGRPC`, `TokenGenerator`, `.UpdateFile`, `identity_provider_store.go`, `NewCachedUserStatusChecker`, `.loadCategoryResiliencyHistories`, `scenario_catalog_cache_test.go`, `test-api.sh`, `ScenariosResponse`, `net/http.Response`, `ResolvePrivateTagDigest`, `.PostScenarioRun`?**
  _High betweenness centrality (0.029) - this node is a cross-community bridge._
- **What connects `github.com/krkn-chaos/krkn-operator`, `OLM_PACKAGE_NAME`, `OLM_CHANNEL_NAME` to the rest of the system?**
  _25 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `main.go` be split into smaller, more focused modules?**
  _Cohesion score 0.07509931670109646 - nodes in this community are weakly interconnected._
- **Why does `ExecuteKubectlResponse` connect `Available Make Targets` to `DataProviderServiceServer`, `dataprovider.pb.go`, `AGENTS.md`, `dataProviderServiceClient`, `dataprovider_grpc.pb.go`, `google.golang.org/protobuf/reflect/protoreflect.Message`?**
  _High betweenness centrality (0.014) - this node is a cross-community bridge._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.011672111672111673 - nodes in this community are weakly interconnected._
- **Why does `KrknScenarioRun` connect `GetClaimsFromContext` to `main.go`, `testing.T`, `Handler`, `context.Context`, `KrknWorkspace`, `.updateClusterJobStatuses`, `KrknTargetRequest`, `.reconcileDeletion`, `WSUnifiedJobItem`, `newCancellationScenarioReconciler`, `newTestScheme`, `user_handlers_test.go`, `.GetScenarioRunLogs`, `GenerateFromToken`, `.checkRollbackImage`, `ScenarioReference`, `.DeepCopyInto`, `buildUnifiedJobList`, `.DeepCopy`, `.loadCategoryResiliencyHistories`, `ClusterJobStatus`, `KrknOperatorTargetProvider`, `.PostScenarioRun`, `test-api.sh`, `Moving from direct scenario creation to CRD approach`, `scenarioRunConfigurationFixture`?**
  _High betweenness centrality (0.010) - this node is a cross-community bridge._
- **Should `Handler` be split into smaller, more focused modules?**
  _Cohesion score 0.06269349845201239 - nodes in this community are weakly interconnected._