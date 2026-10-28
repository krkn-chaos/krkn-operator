# Graph Report - krkn-operator  (2026-10-21)

## Corpus Check
- cluster-only mode — file stats not available

## Summary
- 2913 nodes · 12713 edges · 128 communities (98 shown, 30 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 1232 edges (avg confidence: 0.85)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `482eee5f`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Community 0
- Community 1
- Community 2
- Community 3
- Community 4
- Community 5
- Community 6
- Community 7
- Community 8
- Community 9
- Community 10
- Community 11
- Community 12
- Community 13
- Community 14
- Community 15
- Community 16
- Community 17
- Community 18
- Community 19
- Community 20
- Community 21
- Community 22
- Community 23
- Community 24
- Community 25
- Community 26
- Community 27
- Community 28
- Community 29
- Community 30
- Community 31
- Community 32
- Community 33
- Community 34
- Community 35
- Community 36
- Community 37
- Community 38
- Community 39
- Community 40
- Community 41
- Community 42
- Community 43
- Community 44
- Community 45
- Community 46
- Community 47
- Community 48
- Community 49
- Community 50
- Community 51
- Community 52
- Community 53
- Community 54
- Community 55
- Community 56
- Community 57
- Community 58
- Community 59
- Community 60
- Community 61
- Community 62
- Community 63
- Community 64
- Community 65
- Community 66
- Community 67
- Community 68
- Community 69
- Community 70
- Community 71
- Community 72
- Community 73
- Community 74
- Community 75
- Community 76
- Community 77
- Community 78
- Community 79
- Community 80
- Community 81
- Community 82
- Community 83
- Community 84
- Community 85
- Community 86
- Community 87
- Community 88
- Community 89
- Community 90
- Community 91
- Community 92
- Community 93
- Community 94
- Community 95
- Community 96
- Community 97
- Community 98
- Community 99
- Community 100
- Community 101
- Community 102
- Community 103
- Community 104
- Community 105
- Community 107
- Community 108
- Community 109
- Community 110
- Community 111
- Community 112
- Community 113
- Community 115
- Community 116
- Community 117
- Community 118
- Community 119
- Community 120
- Community 121
- Community 122
- Community 123
- Community 124
- Community 125
- Community 126
- Community 127

## God Nodes (most connected - your core abstractions)
1. `Handler` - 192 edges
2. `NewTestHandler()` - 166 edges
3. `writeJSONError()` - 124 edges
4. `writeJSON()` - 95 edges
5. `createAdminContext()` - 91 edges
6. `GetClaimsFromContext()` - 61 edges
7. `newEsScheme()` - 57 edges
8. `KrknScenarioRun` - 51 edges
9. `createUserContext()` - 43 edges
10. `KrknGraphRun` - 41 edges

## Surprising Connections (you probably didn't know these)
- `validateCreateRegistryRequest()` --references--> `CreateRegistryRequest`  [EXTRACTED]
  internal/api/registry_handlers.go → pkg/registry/types.go
- `validateUpdateRegistryRequest()` --references--> `UpdateRegistryRequest`  [EXTRACTED]
  internal/api/registry_handlers.go → pkg/registry/types.go
- `Handler` --references--> `SecretManager`  [EXTRACTED]
  internal/api/handlers.go → pkg/auth/secret_manager.go
- `Handler` --references--> `Client`  [EXTRACTED]
  internal/api/handlers.go → pkg/elasticsearch/client.go
- `assembleResiliencyHistoryQueryResponse()` --references--> `KrknCategory`  [EXTRACTED]
  internal/api/resiliency_history_query.go → api/v1alpha1/krkncategory_types.go

## Import Cycles
- None detected.

## Communities (128 total, 30 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.10
Nodes (12): CategoryEntityAssociationResponse, CreateCategoryRequest, ResiliencyHistoryQueryRequest, scenarioFileBehavior, UpdateCategoryRequest, ContextKey, clusterTarget, jobOutcome (+4 more)

### Community 1 - "Community 1"
Cohesion: 0.02
Nodes (187): TestKrknUserCreation(), TestKrknUserList(), TestKrknUserRoles(), TestKrknUserStatus(), TestKrknUserValidation(), TestScenarioReferenceValidate(), setupAuthTestHandler(), TestIsRegistered_MethodNotAllowed() (+179 more)

### Community 2 - "Community 2"
Cohesion: 0.09
Nodes (16): writeCategoryRunNotFoundOrInternalError(), buildCloudCredentialResponse(), updateCloudCredentialSecretData(), convertGraphClusterScores(), buildUserGroupResponse(), extractGroupNameAndUserIDFromPath(), extractGroupNameFromMembersPath(), extractPathSuffix() (+8 more)

### Community 3 - "Community 3"
Cohesion: 0.04
Nodes (23): filterUsers(), GraphRunSummary, GraphScenario, KrknCategoryList, KrknCategorySpec, KrknGraphRunList, KrknGraphRunStatus, KrknOperatorTargetList (+15 more)

### Community 4 - "Community 4"
Cohesion: 0.05
Nodes (47): AvailableFilesResponse, DeleteFileResponse, ListFilesResponse, UpdateFileResponse, buildFileInfo(), buildFileResponse(), deriveLogicalName(), extractLogicalName() (+39 more)

### Community 5 - "Community 5"
Cohesion: 0.06
Nodes (53): k8sUserStatusChecker, mockUserStatusChecker, RestoreConfig, removeCategoryLabel(), isValidConfigMapKey(), validateCreateFileRequest(), validateFileFields(), validateUpdateFileRequest() (+45 more)

### Community 6 - "Community 6"
Cohesion: 0.11
Nodes (67): esDoerFunc, boolPtr(), newEsAlertsTestSecretWithHost(), newEsScheme(), newEsTestSecret(), newEsTestSecretWithHost(), TestCreateElasticsearchConfig_DuplicateName(), TestCreateElasticsearchConfig_Forbidden() (+59 more)

### Community 7 - "Community 7"
Cohesion: 0.07
Nodes (62): TestCloudCredentialContract_ConsolePayloadToCRDToSecretKeyRef(), TestCloudCredentialContract_GraphRunStripsNodeEnv(), newAWSTestSecret(), newCloudCredScheme(), newGCPTestSecret(), newGroupedAWSSecret(), TestCloudCredentialsRouter_MethodNotAllowed(), TestCloudCredentialsRouter_SubpathMethodNotAllowed() (+54 more)

### Community 8 - "Community 8"
Cohesion: 0.07
Nodes (35): AvailableCommandsResponse, tokenRequest, TestExecuteTerminal_CommandParsing(), TestExecuteTerminal_ValidationLogic(), GetAllowedCommands(), GetBlockedFlags(), CommandSpec, IsCommandAllowed() (+27 more)

### Community 9 - "Community 9"
Cohesion: 0.07
Nodes (27): imageResolver, imageSignatureVerifier, KrknScenarioRunReconciler, TestScenarioRunReconcileSkipsPendingMigration(), backfillLegacyMaxRetries(), countJobPhaseChanges(), extractDelimitedContent(), getOwnerLabel() (+19 more)

### Community 10 - "Community 10"
Cohesion: 0.05
Nodes (47): ActiveRunsOverviewResponse, AddGroupMemberRequest, AddGroupMemberResponse, ChangePasswordRequest, ChangePasswordResponse, ClusterPermissionSet, ClusterResiliencyScoreResponse, CreateTargetResponse (+39 more)

### Community 11 - "Community 11"
Cohesion: 0.07
Nodes (17): Handler, TerminalRequest, TerminalResponse, writeScenarioRunAccessError(), parseBoolHeader(), TestParseBoolHeader(), convertInputFields(), createScenarioProvider() (+9 more)

### Community 12 - "Community 12"
Cohesion: 0.13
Nodes (42): clusterJobImage, clusterJobProvider, clusterJobRetries, graphScenarioReferences, migrationPlan, migrationResult, migrationStats, resiliencyScoreStatus (+34 more)

### Community 13 - "Community 13"
Cohesion: 0.11
Nodes (43): addAdminContext(), addUserContext(), createTestAdminUser(), setupFilesTestHandler(), TestCanAccessFile(), TestCreateFile(), TestCreateFile_DefaultFilePurpose(), TestCreateFile_DuplicateName() (+35 more)

### Community 14 - "Community 14"
Cohesion: 0.07
Nodes (40): TestLogin_InactiveUser(), TestLogin_InvalidCredentials(), TestLogin_Success(), maskToken(), sanitizeHeaders(), setupScenarioRunTestHandler(), TestDeleteTargetByUUID_AdminCanDeleteAnyResource(), TestDeleteTargetByUUID_InvalidUUID() (+32 more)

### Community 15 - "Community 15"
Cohesion: 0.08
Nodes (19): CategoryListResponse, CategoryResponse, ErrorResponse, buildCategoryResponse(), buildCategoryVisibilityLabels(), categoryErrorCode(), categoryGroups(), categoryIsPublic() (+11 more)

### Community 16 - "Community 16"
Cohesion: 0.07
Nodes (32): channel_has_entries(), ensure_catalog_channel(), catalog-channel.sh script, update_default_channel(), write_release_config(), catalog_submit_checkout(), catalog_submit_open_pr(), catalog-submit.sh script (+24 more)

### Community 17 - "Community 17"
Cohesion: 0.07
Nodes (6): add_DataProviderServiceServicer_to_server(), DataProviderService, DataProviderServiceServicer, DataProviderServiceStub, DataProviderServicer, serve()

### Community 18 - "Community 18"
Cohesion: 0.06
Nodes (26): ConfigStoreInitializer, bootstrapOLMResources(), main(), NewConfigStoreInitializer(), resolveMigrationNamespace(), runCRDMigrationMode(), TestConfigStoreInitializer_NeedLeaderElection(), TestConfigStoreInitializer_Start_ConfigMapNotFound() (+18 more)

### Community 19 - "Community 19"
Cohesion: 0.10
Nodes (36): NewHandler(), NewBroadcaster(), TestBroadcastDashboardUpdate(), TestBroadcastGraphRunDeleted(), TestBroadcastGraphRunUpdate(), TestBroadcastOnlyToSubscribedClients(), TestBroadcastScenarioRunDeleted(), TestBroadcastScenarioRunUpdate() (+28 more)

### Community 20 - "Community 20"
Cohesion: 0.06
Nodes (24): TestBuildUnifiedJobList_SkipsGraphRunChildren(), TestFilterJobsByCategoriesMatchesAnySelectedCategory(), TestPaginateJobItems(), buildGraphRunResponse(), buildUnifiedJobList(), computeWSJobStats(), convertGraphClusterScoresForSnapshot(), filterJobCategoriesByVisibility() (+16 more)

### Community 21 - "Community 21"
Cohesion: 0.14
Nodes (34): createTestUser(), kvValue(), setupUserTestHandler(), TestChangePassword_AdminChange_NoCurrentPassword(), TestChangePassword_AdminChangesOtherUser_EmitsAuditLog(), TestChangePassword_SamePassword_Rejected(), TestChangePassword_SelfChange_NoAdminAuditLog(), TestChangePassword_SelfChange_RequiresCurrentPassword() (+26 more)

### Community 22 - "Community 22"
Cohesion: 0.10
Nodes (22): doerFunc, roundTripperFunc, NewClient(), scenarioParametersSchema(), TestBaseURL(), TestClientTransportDefaultNotHTTPTransport(), TestClientTransportPooling(), TestConnectionParamsTLSConfig() (+14 more)

### Community 23 - "Community 23"
Cohesion: 0.14
Nodes (10): KrknGraphRunReconciler, migrationPending(), TestGraphRunReconcileSkipsPendingMigration(), TestLoadFileConfigMapByID(), TestTranslateVolumesToFileMounts(), sanitizeNodeID(), TestCalculateResiliencyScore_StoresNodeScores(), overrideRef() (+2 more)

### Community 24 - "Community 24"
Cohesion: 0.10
Nodes (29): AffectedPods, AlertDocument, CreateElasticsearchConfigRequest, CreateElasticsearchConfigResponse, DeleteElasticsearchConfigResponse, ListElasticsearchConfigsResponse, QueryAlertsRequest, QueryAlertsResponse (+21 more)

### Community 25 - "Community 25"
Cohesion: 0.17
Nodes (22): preparedJobResources, TestAppendCloudCredentialInjection_EmptyRefNoop(), TestAppendCloudCredentialInjection_UsesSecretKeyRef(), injectAWS(), injectAzure(), injectBaremetal(), InjectCredentials(), injectGCP() (+14 more)

### Community 26 - "Community 26"
Cohesion: 0.12
Nodes (12): configMapClient, deploymentClient, ensureMigrationGuard(), logStaleMigrationGuard(), markMigrationStarted(), removeMigrationGuard(), Stage(), Sync() (+4 more)

### Community 27 - "Community 27"
Cohesion: 0.20
Nodes (25): Complete(), Stage(), containsSubstring(), countPatchActions(), jsonMarshal(), migrationTestLogger(), newMigrationClient(), scenarioRunFixture() (+17 more)

### Community 28 - "Community 28"
Cohesion: 0.11
Nodes (8): ClustersResponse, KrknTargetRequestReconciler, ClusterHealthStatus, ClusterTarget, KrknOperatorTarget, KrknOperatorTargetSpec, KrknTargetRequest, KrknTargetRequestSpec

### Community 29 - "Community 29"
Cohesion: 0.13
Nodes (21): esAlertSearchResponse, esSearchResponse, facetField, filterAggregation, Option, StatusError, termsAggregation, buildAlertsSearchBody() (+13 more)

### Community 30 - "Community 30"
Cohesion: 0.14
Nodes (23): NewJobTracker(), createMultipartBody(), createTestHandler(), TestGetRestoreStatusAdminOnly(), TestGetRestoreStatusTrackedJob(), TestGetRestoreStatusUnknownJob(), TestHandlerShutdownCancelsContext(), TestJobTrackerEviction() (+15 more)

### Community 31 - "Community 31"
Cohesion: 0.14
Nodes (23): CreateTargetRequest, UpdateTargetRequest, generateKubeconfigFromCredentialsType(), generateKubeconfigFromKubeconfigType(), generateKubeconfigFromRequest(), generateKubeconfigFromTokenType(), TestCreateTarget_WithKubeconfig(), TestUpdateTarget() (+15 more)

### Community 32 - "Community 32"
Cohesion: 0.17
Nodes (6): hashBytes(), roleFromAdmin(), AuthorizationCheckFunc, Broadcaster, graphRunEventHandler, scenarioRunEventHandler

### Community 33 - "Community 33"
Cohesion: 0.17
Nodes (14): Doer, FacetOption, QueryTelemetryResponse, TelemetryDocument, TelemetryStats, decodeTelemetry(), executeSearch(), facetsFromBuckets() (+6 more)

### Community 34 - "Community 34"
Cohesion: 0.11
Nodes (9): KrknOperatorTargetProviderConfigReconciler, getProviderNames(), isConflictError(), countActiveProviders(), KrknOperatorTargetProviderConfig, KrknOperatorTargetProviderConfigSpec, KrknOperatorTargetProviderConfigStatus, KrknOperatorTargetProviderList (+1 more)

### Community 35 - "Community 35"
Cohesion: 0.16
Nodes (17): checkWebSocketOriginV2(), TestCheckWebSocketOriginV2(), checkWebSocketOrigin(), TestCheckWebSocketOrigin(), guardedCheckRedirect(), TestGuardedCheckRedirect(), TestValidateDestinationURL(), validateDestinationURL() (+9 more)

### Community 37 - "Community 37"
Cohesion: 0.16
Nodes (8): cloudCredentialAccessContext, buildConnectionParams(), buildConnectionParamsForIndex(), buildElasticsearchConfigResponse(), canAccessElasticsearchConfig(), buildRegistryInfo(), ExtractGroupsFromLabels(), SanitizeGroupName()

### Community 38 - "Community 38"
Cohesion: 0.17
Nodes (12): crdClient, applyCRD(), applyCRDs(), TestApplyCRDsCreatesMissingAndUpdatesChangedSpecs(), TestApplyCRDsDoesNotPatchUnchangedSpecs(), TestApplyCRDsLabelsExistingCRDsWithoutChangingOtherLabels(), TestApplyCRDsRejectsNonCRDManifest(), testCRD() (+4 more)

### Community 39 - "Community 39"
Cohesion: 0.13
Nodes (13): convertClusterResiliencyScores(), averageResiliencyScore(), convertMetaTime(), BuildUnifiedJobList(), getDefaultPageSize(), runCategoryNames(), scenarioRunImage(), PaginateSlice() (+5 more)

### Community 40 - "Community 40"
Cohesion: 0.14
Nodes (16): categoryEnvironmentParameter, categoryHistoryRun, categoryParameterProfileFingerprint(), categoryParameterProfileName(), TestCategoryParameterProfileFingerprintForEmptyEnvironment(), TestCategoryParameterProfileFingerprintGraphEnvironmentMultiset(), TestCategoryParameterProfileFingerprintIgnoresRunType(), TestCategoryParameterProfileFingerprintScenarioEnvironment() (+8 more)

### Community 41 - "Community 41"
Cohesion: 0.13
Nodes (14): categoryHistoryScore, containsString(), filterCategoryHistoryScoresByCluster(), filterGraphHistoryScoresByPermission(), filterScenarioHistoryScoresByPermission(), graphHistoryScores(), runHasAnyCategory(), scenarioHistoryScores() (+6 more)

### Community 42 - "Community 42"
Cohesion: 0.10
Nodes (10): ClusterJobStatusResponse, NodeStatusResponse, PaginationMeta, ScenarioRunListItem, ScenarioRunListResponse, ScenarioRunStatusResponse, UnifiedJobItem, UnifiedJobsResponse (+2 more)

### Community 43 - "Community 43"
Cohesion: 0.12
Nodes (12): getTokenDuration(), TestGetTokenDuration(), newResiliencyTestScheme(), newTestClientsetWithLogs(), TestCalculateResiliencyScores(), TestReconcile_ResiliencyScoreFailure_DoesNotBlockTerminalPhase(), generateRandomSecret(), getSecretKeys() (+4 more)

### Community 44 - "Community 44"
Cohesion: 0.23
Nodes (4): visibleCategoryNames(), AuthorizationChecker, categoryVisibilityChecker, Handler

### Community 45 - "Community 45"
Cohesion: 0.22
Nodes (20): newCategoryRequest(), newCategoryTestGroup(), newCategoryTestHandler(), newCategoryTestUser(), TestCategoriesRouterCRUD(), TestCategoriesRouterGroupCreationRequiresMembership(), TestCategoriesRouterGroupVisibility(), TestCategoriesRouterNameMaximumLength() (+12 more)

### Community 46 - "Community 46"
Cohesion: 0.10
Nodes (11): assert, chartWorkflow, { execFileSync, spawnSync }, fs, notifier, notifyWorkflow, os, path (+3 more)

### Community 47 - "Community 47"
Cohesion: 0.20
Nodes (11): ensureJWTSecret(), EnsureResources(), ensureSCCBinding(), ensureScenarioRunnerRBAC(), ensureServiceAccount(), ensureServices(), intstrFromInt(), namespaceHash() (+3 more)

### Community 48 - "Community 48"
Cohesion: 0.19
Nodes (18): hasGraphRunStatusChanged(), hasScenarioRunStatusChanged(), TestHasGraphRunStatusChanged_NewNodeAdded(), TestHasGraphRunStatusChanged_NoChange(), TestHasGraphRunStatusChanged_NodeOrderChanged(), TestHasGraphRunStatusChanged_NodeStatusChange(), TestHasGraphRunStatusChanged_PhaseChange(), TestHasGraphRunStatusChanged_ResiliencyScoresChange() (+10 more)

### Community 49 - "Community 49"
Cohesion: 0.12
Nodes (15): JobsListResponse, JobStatusResponse, ListTargetsResponse, RestoreResponse, ScenariosResponse, ScenarioTag, TargetResponse, CachedUserStatusChecker (+7 more)

### Community 50 - "Community 50"
Cohesion: 0.18
Nodes (6): JobStatus, JobTracker, BroadcastMessage, Client, Hub, PaginationClientState

### Community 51 - "Community 51"
Cohesion: 0.17
Nodes (17): GraphRunCreateRequest, GraphScenarioNode, appendGraphSignaturePart(), compareGraphConfigurations(), compareGraphConfigurationsWithStableNodeIDs(), graphConfigurationSignature(), graphNodeConfigurationSignature(), sameGraphConfigurationIgnoringNodeIDs() (+9 more)

### Community 52 - "Community 52"
Cohesion: 0.21
Nodes (9): ipRateLimiter, rateLimiterEntry, clientIPFromHeaders(), extractClientIP(), hostFromRemoteAddr(), isTrustedProxy(), parseTrustedProxyCIDRs(), TestExtractClientIP() (+1 more)

### Community 53 - "Community 53"
Cohesion: 0.20
Nodes (15): publicScenarioReference(), TestCreateScenarioRun_ResiliencyScore(), buildContainerImage(), buildContainerImageFromReference(), getTestConfig(), privateReference(), publicReference(), TestBuildContainerImage_PublicRegistryUsesKrknctl() (+7 more)

### Community 54 - "Community 54"
Cohesion: 0.22
Nodes (13): countingGetClient, TestConcurrentAccess(), TestConcurrentSnapshot(), TestDelete(), TestDelete_NonExistent(), TestEmptyValue(), TestExists(), TestGet_Singleton() (+5 more)

### Community 55 - "Community 55"
Cohesion: 0.21
Nodes (5): mockScenarioProvider, InvalidImageSignatureError, resolveImmutableImage(), resolvePrivateImageDigest(), TestResolvePrivateImageDigestUsesManifestDigest()

### Community 56 - "Community 56"
Cohesion: 0.14
Nodes (15): CreateCloudCredentialResponse, DeleteCloudCredentialResponse, ListCloudCredentialsResponse, UpdateCloudCredentialResponse, buildCloudCredentialSecretData(), CloudCredentialResponse, CreateCloudCredentialRequest, UpdateCloudCredentialRequest (+7 more)

### Community 57 - "Community 57"
Cohesion: 0.15
Nodes (3): DataProviderServiceClient, GetNodesRequest, GetNodesResponse

### Community 58 - "Community 58"
Cohesion: 0.16
Nodes (5): UnimplementedDataProviderServiceServer, UnsafeDataProviderServiceServer, _DataProviderService_ExecuteKubectl_Handler(), _DataProviderService_GetNodes_Handler(), RegisterDataProviderServiceServer()

### Community 59 - "Community 59"
Cohesion: 0.33
Nodes (14): TestBuildCategoryResiliencyHistoryGroupsTypedConfigurationsAndSortsPerCluster(), TestGraphHistoryScoresOnlyKeepsFinalScoresIncludingZero(), compareCategoryRunConfigurations(), assertConfigurationDifferences(), floatPointer(), graphRunConfigurationFixture(), scenarioReference(), scenarioRunConfigurationFixture() (+6 more)

### Community 60 - "Community 60"
Cohesion: 0.20
Nodes (15): BuildDockerConfigJSON(), ExtractRegistryV2FromSecret(), TestBuildDockerConfigJSON_MissingCredentials(), TestBuildDockerConfigJSON_OAuth(), TestBuildDockerConfigJSON_Password(), TestExtractRegistryV2FromSecret_InvalidType(), TestExtractRegistryV2FromSecret_MissingAnnotations(), TestExtractRegistryV2FromSecret_MissingComponent() (+7 more)

### Community 61 - "Community 61"
Cohesion: 0.17
Nodes (7): buildRegistryResponse(), BuildRegistryAnnotations(), BuildRegistryLabels(), ExtractGroupsFromLabels(), TestBuildRegistryAnnotations(), TestUpdateRegistryAnnotations(), UpdateRegistryAnnotations()

### Community 62 - "Community 62"
Cohesion: 0.13
Nodes (7): BackupConfig, resourceType, stripMetadata(), TestStripMetadata(), TestStripMetadataRemovesEmptyAnnotations(), TestValidateBackupName(), validateBackupName()

### Community 63 - "Community 63"
Cohesion: 0.16
Nodes (10): Role, loggingMiddleware(), maxBodySizeMiddleware(), NewServer(), TestMaxBodySizeMiddleware(), NewLazyMiddleware(), CheckClusterLiveness(), handlerOrStatus() (+2 more)

### Community 64 - "Community 64"
Cohesion: 0.21
Nodes (12): addKnownTypes(), newKrknUser(), newUserStatusScheme(), TestK8sUserStatusChecker_IsUserActive(), TestK8sUserStatusChecker_LookupErrorPropagates(), newTargetRequest(), newTestScheme(), TestGetClusterAPIURL_ClusterNotFound() (+4 more)

### Community 65 - "Community 65"
Cohesion: 0.15
Nodes (13): FromKrknctlScenarioSet(), publicScenarioReference(), publicScenario(), TestFromKrknctlScenarioSetCreatesPublicReferences(), TestToKrknctlScenarioSetUsesScenarioReference(), ToKrknctlScenarioSet(), MapScenarioNodeToScenarioRunSpec(), ResolveGraph() (+5 more)

### Community 66 - "Community 66"
Cohesion: 0.21
Nodes (9): countingUserStatusChecker, UserStatusChecker, Middleware, NewCachedUserStatusChecker(), TestCachedUserStatusChecker(), TestCachedUserStatusChecker_EvictsExpiredEntries(), TestCachedUserStatusChecker_Invalidation(), TestNewCachedUserStatusChecker_NilCheckerPanics() (+1 more)

### Community 67 - "Community 67"
Cohesion: 0.20
Nodes (10): ClusterMetadata, NodeSummaryInfo, rawTelemetrySource, isSensitiveKey(), redactSensitiveParameters(), redactValueWithTracking(), TestFlattenRepresentativeScenario(), TestFlattenRedactsParameters() (+2 more)

### Community 68 - "Community 68"
Cohesion: 0.25
Nodes (11): newIPRateLimiter(), rateLimitMiddleware(), TestIPRateLimiter_AllowsUnderLimit(), TestIPRateLimiter_BlocksOverLimit(), TestIPRateLimiter_Cleanup(), TestIPRateLimiter_CleanupPreservesFresh(), TestIPRateLimiter_EvictsOldestWhenAtCapacity(), TestIPRateLimiter_SeparateLimitersPerIP() (+3 more)

### Community 69 - "Community 69"
Cohesion: 0.19
Nodes (10): filterReportLines(), isReportMarkerEnd(), isReportMarkerStart(), isWebSocketDisconnectError(), preStartFailureReason(), writeWSError(), TestFilterReportLines(), TestIsReportMarkerEnd() (+2 more)

### Community 71 - "Community 71"
Cohesion: 0.41
Nodes (12): build_operator(), check_prerequisites(), cleanup(), create_service_account(), install_crds(), main(), print_error(), print_info() (+4 more)

### Community 72 - "Community 72"
Cohesion: 0.18
Nodes (10): resiliencyHistoryQuerySelections, normalizeResiliencyHistoryProviders(), normalizeResiliencyHistorySelections(), parseResiliencyHistoryQuery(), createTarGz(), TestCreateAndExtractTarGz(), TestCreateTarGzFlushErrors(), TestExtractTarGzPathTraversal() (+2 more)

### Community 73 - "Community 73"
Cohesion: 0.27
Nodes (12): makeGraphRun(), makeScenarioRun(), TestListJobs_IncludesStats(), TestListJobs_LimitOnlyReturnsAll(), TestListJobs_NoPagination(), TestListJobs_Paginated(), TestListJobs_SkipsGraphRunChildren(), TestListJobs_TypeDiscriminator() (+4 more)

### Community 74 - "Community 74"
Cohesion: 0.30
Nodes (3): Compute(), JobCounter, Summary

### Community 75 - "Community 75"
Cohesion: 0.23
Nodes (8): averageResiliencyScore(), buildScenarioRunDetailResponse(), buildScenarioRunResponse(), convertClusterResiliencyScoresWS(), sanitizedClusterJobs(), TestBuildScenarioRunResponse_IncludesClusterJobs(), TestBuildScenarioRunResponse_NoClusterJobs(), ClusterResiliencyScore

### Community 76 - "Community 76"
Cohesion: 0.29
Nodes (12): NewProviderRegistration(), NewProviderRegistrationWithConfig(), setupTestClient(), TestDeactivateProvider_HandlesNotFound(), TestDeactivateProvider_SetsActiveToFalse(), TestEnsureProvider_CreatesNewProvider(), TestEnsureProvider_UpdatesExistingProvider(), TestNeedLeaderElection_ReturnsTrue() (+4 more)

### Community 77 - "Community 77"
Cohesion: 0.21
Nodes (11): CreateRegistryRequest, RegistryInfo, RegistryResponse, UpdateRegistryRequest, AuthEntry, AvailableRegistriesResponse, CreateRegistryResponse, DeleteRegistryResponse (+3 more)

### Community 78 - "Community 78"
Cohesion: 0.39
Nodes (11): api_call(), cleanup(), extract_body(), extract_http_code(), log_error(), log_info(), log_step(), log_success() (+3 more)

### Community 79 - "Community 79"
Cohesion: 0.20
Nodes (9): historyQueryCountingClient, CleanupOldResources(), ExampleCleanupOldResources_cronJob(), ExampleCleanupOldResources_providerConfig(), ExampleCleanupOldResources_targetRequest(), extractItemsFromList(), TestCleanupOldResources(), TestExtractItemsFromList() (+1 more)

### Community 82 - "Community 82"
Cohesion: 0.29
Nodes (10): newReportTestHandler(), reportRequest(), TestDownloadReportDeniesUserWithoutReportClusterAccess(), TestDownloadReportHTMLThroughV2Router(), TestDownloadReportPDF(), TestDownloadReportRejectsMalformedPath(), TestDownloadReportUnavailable(), TestGetReportStatusPending() (+2 more)

### Community 85 - "Community 85"
Cohesion: 0.27
Nodes (4): Claims, mockTokenGen(), mockTokenGenInvalid(), TokenGenerator

### Community 86 - "Community 86"
Cohesion: 0.24
Nodes (9): newCategoryRequestAs(), graphRunCategoryTestHandler(), runCategoryFixture(), TestCreateGraphRun_CategoriesArePersistedOnCreatedCR(), TestCreateGraphRun_RejectsMissingCategoryBeforeCreatingCR(), TestPostScenarioRun_CategoriesArePersistedOnCreatedCR(), TestPostScenarioRun_RejectsInvalidCategoryNameBeforeCreatingCR(), TestGetScenarioRunConfig_Success() (+1 more)

### Community 87 - "Community 87"
Cohesion: 0.22
Nodes (4): convertNodeStatusesWithScores(), convertNodeStatusesWithScores(), GraphClusterScore, NodeStatus

### Community 88 - "Community 88"
Cohesion: 0.22
Nodes (5): responseWriter, guardedDialContext(), TestValidateDestinationIP(), validateDestinationIP(), validateInlineDestination()

### Community 89 - "Community 89"
Cohesion: 0.28
Nodes (6): filterScenariosByIsAScenario(), TestFilterScenariosByIsAScenario(), TestFilterScenariosByIsAScenario_CancelledContext(), TestFilterScenariosByIsAScenario_PreservesTagMetadata(), TestValidateValueAgainstSchema(), ValidateValueAgainstSchema()

### Community 90 - "Community 90"
Cohesion: 0.36
Nodes (8): intPtr(), TestRawTelemetrySourceFlatten(), contains(), containsAt(), TestMetadataJSONMarshaling(), TestMetadataPresenceTracking(), boolPtr(), TestValidateCreateRequest()

### Community 91 - "Community 91"
Cohesion: 0.31
Nodes (8): contains(), findSubstring(), TestValidateFileContent(), TestValidateFileGroups(), TestValidateUserFilePermissions(), ValidateFileContent(), ValidateFileGroups(), ValidateUserFilePermissions()

### Community 92 - "Community 92"
Cohesion: 0.29
Nodes (5): GraphRunDetailResponse, GraphRunListItem, GraphRunListResponse, GraphRunSpecResponse, GraphRunStatusResponse

### Community 93 - "Community 93"
Cohesion: 0.29
Nodes (8): KrknScenarioRunSpec, compareGraphNodeConfigurations(), compareScenarioRunConfigurations(), compareStringMaps(), optionalBoolValue(), optionalStringValue(), sameGraphVolumeMounts(), sameScenarioReference()

### Community 95 - "Community 95"
Cohesion: 0.32
Nodes (8): decodeErrorResponse(), providerConfigTestScheme(), TestGetProviderConfigByUUID_CompletedReturnsConfigData(), TestGetProviderConfigByUUID_InternalErrorReturnsSanitized500(), TestGetProviderConfigByUUID_NotFoundReturns404(), TestGetProviderConfigByUUID_PendingReturnsAccepted(), TestPostProviderConfig_FailureReturnsSanitizedError(), TestPostProviderConfig_ReturnsAcceptedWithUUID()

### Community 96 - "Community 96"
Cohesion: 0.38
Nodes (7): CategoryConfigurationGroup, CategoryResiliencyDataPoint, CategoryResiliencyHistoryResponse, ResiliencyHistoryQueryResponse, datapointForRunID(), datapointRunIDs(), assembleResiliencyHistoryQueryResponse()

### Community 97 - "Community 97"
Cohesion: 0.29
Nodes (7): FileMount, GlobalsRequest, ScenarioRunRequest, ScenariosRequest, normalizeLegacyScenarioRunRequest(), TestNormalizeLegacyScenarioRunRequestIgnoresImage(), FileReference

### Community 99 - "Community 99"
Cohesion: 0.33
Nodes (5): SanitizeNodeIDForKubernetes(), stringPtr(), TestCollisionDetection(), TestKrknGraphRun_ValidateGraph(), TestSanitizeNodeIDForKubernetes()

### Community 100 - "Community 100"
Cohesion: 0.40
Nodes (5): newProvider(), newProvidersHandler(), TestListProviders_Empty(), TestListProviders_ScopedToNamespace(), TestUpdateProviderStatus()

### Community 101 - "Community 101"
Cohesion: 0.40
Nodes (6): TestSanitizeUserIDForLabel(), SanitizeUserIDForLabel(), SanitizeUserIDForResourceName(), TestSanitizeUserIDConsistency(), TestSanitizeUserIDForLabel(), TestSanitizeUserIDForResourceName()

### Community 102 - "Community 102"
Cohesion: 0.33
Nodes (5): TestBuildAnnotations(), TestBuildLabels(), TestBuildLabelsAccessControl(), TestUpdateAnnotations(), UpdateAnnotations()

### Community 104 - "Community 104"
Cohesion: 0.67
Nodes (5): log_error(), log_info(), log_warn(), setup-ocm-and-operator.sh script, wait_for_condition()

### Community 110 - "Community 110"
Cohesion: 0.50
Nodes (3): OLM_CHANNEL_NAME, OLM_PACKAGE_NAME, channel-head.sh script

### Community 112 - "Community 112"
Cohesion: 0.83
Nodes (3): isAlphanumeric(), isValidLabelChar(), TestSanitizeNodeID()

## Knowledge Gaps
- **17 isolated node(s):** `OLM_CHANNEL_NAME`, `OLM_PACKAGE_NAME`, `github.com/krkn-chaos/krkn-operator`, `ICON_BASE64`, `ICON_MEDIATYPE` (+12 more)
  These have ≤1 connection - possible missing edges. (Counts symbols only; 342 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **30 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Handler` connect `Community 11` to `Community 0`, `Community 33`, `Community 2`, `Community 4`, `Community 5`, `Community 37`, `Community 39`, `Community 69`, `Community 41`, `Community 43`, `Community 15`, `Community 47`, `Community 50`, `Community 84`, `Community 61`, `Community 30`?**
  _High betweenness centrality (0.061) - this node is a cross-community bridge._
- **Are the 159 inferred relationships involving `NewTestHandler()` (e.g. with `setupAuthTestHandler()` and `TestLogin_InactiveUser()`) actually correct?**
  _`NewTestHandler()` has 159 INFERRED edges - model-reasoned connections that need verification._
- **What connects `OLM_CHANNEL_NAME`, `OLM_PACKAGE_NAME`, `github.com/krkn-chaos/krkn-operator` to the rest of the system?**
  _17 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.10422629638224903 - nodes in this community are weakly interconnected._
- **Why does `WSUnifiedJobItem` connect `Community 20` to `Community 0`, `Community 75`?**
  _High betweenness centrality (0.022) - this node is a cross-community bridge._
- **Should `Community 1` be split into smaller, more focused modules?**
  _Cohesion score 0.016147699090658583 - nodes in this community are weakly interconnected._
- **Why does `KrknCategoryList` connect `Community 3` to `Community 0`, `Community 36`, `Community 15`?**
  _High betweenness centrality (0.017) - this node is a cross-community bridge._