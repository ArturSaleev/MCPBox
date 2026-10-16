import { useState } from 'react';
import type { FormEvent } from 'react';

import { Copy, Database, LoaderCircle, Pencil, Plus, Shield, Trash2 } from 'lucide-react';

import type { ProRagBoxWriteResponse } from '../pro/ragbox';
import type {
  ProStorageConfig,
  ProStorageRuntimeStatus,
  ProStorageShadowStatus,
} from '../pro/storage';

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './ui/dialog';
import { Textarea } from './ui/textarea';

type ProCopy = {
  title: string;
  subtitle: string;
  localLogin: string;
  password: string;
  login: string;
  loginHint: string;
  signInSSO: string;
  ssoAvailable: string;
  ssoIssuer: string;
  ssoRedirectURL: string;
  ssoScopes: string;
  ssoDomainHint: string;
  ssoDefaultRole: string;
  ssoSessionDays: string;
  ssoAutoCreate: string;
  ssoEnabledLabel: string;
  ssoDisabledLabel: string;
  connectedAs: string;
  scopes: string;
  createToken: string;
  createTokenHint: string;
  tokenName: string;
  tokenNamePlaceholder: string;
  tokenScopes: string;
  tokenScopesPlaceholder: string;
  expiresDays: string;
  expiryPolicy: string;
  expiryHint: string;
  noExpiry: string;
  oneDay: string;
  sevenDays: string;
  thirtyDays: string;
  ninetyDays: string;
  oneYear: string;
  activeTokens: string;
  noTokens: string;
  revoke: string;
  adminOnly: string;
  oneTimeSecret: string;
  copyToken: string;
  authHint: string;
  users: string;
  rolesTitle: string;
  sessionsTitle: string;
  noUsers: string;
  noSessions: string;
  createUser: string;
  editUserRoles: string;
  changeUserPassword: string;
  disableUser: string;
  enableUser: string;
  deleteUser: string;
  createSession: string;
  displayName: string;
  email: string;
  newPassword: string;
  confirmPassword: string;
  roleNames: string;
  sessionLabel: string;
  issueSession: string;
  createdSessionToken: string;
  currentSession: string;
  revokeCurrentSession: string;
  settingsTabAccess: string;
  settingsTabStorage: string;
  settingsTabPolicybox: string;
  storageTitle: string;
  storageDescription: string;
  storageBootstrapBackend: string;
  storageOperationalBackend: string;
  storageOperationalBackendSQLite: string;
  storageOperationalBackendPostgres: string;
  storageTabBackends: string;
  storageTabLogs: string;
  storagePostgresHost: string;
  storagePostgresPort: string;
  storagePostgresDatabase: string;
  storagePostgresSchema: string;
  storagePostgresUsername: string;
  storagePostgresPassword: string;
  storagePostgresConfigured: string;
  storagePostgresPasswordRequired: string;
  storageClearPostgresDSN: string;
  storageLogsBackend: string;
  storageLogsBackendOperational: string;
  storageLogsBackendClickHouse: string;
  storageLogsBackendElasticsearch: string;
  storageClickHouseURL: string;
  storageClickHouseDatabase: string;
  storageClickHouseUsername: string;
  storageClickHousePassword: string;
  storageClickHouseConfigured: string;
  storageClearClickHousePassword: string;
  storageElasticsearchURL: string;
  storageElasticsearchUsername: string;
  storageElasticsearchPassword: string;
  storageElasticsearchIndex: string;
  storageElasticsearchConfigured: string;
  storageClearElasticsearchPassword: string;
  storageSave: string;
  storageSaved: string;
  storageSaveFailed: string;
  storageRuntimeTitle: string;
  storageDesiredBackend: string;
  storageActiveBackend: string;
  storageFallbackReason: string;
  storageConnectionConfigured: string;
  storageConnectionReachable: string;
  storageConnectionUnreachable: string;
  storageConnectionNotConfigured: string;
  storageShadowsTitle: string;
  storageShadowProjects: string;
  storageShadowKnowledgeBases: string;
  storageShadowLinks: string;
  storageLastSync: string;
  storageNeverSynced: string;
  policyboxTitle: string;
  policyboxDescription: string;
  policyboxStartupTokensHint: string;
  policyboxJWTSecretHint: string;
  policyboxBaseURL: string;
  policyboxAdminToken: string;
  policyboxAdminTokenPlaceholder: string;
  policyboxAdminTokenConfigured: string;
  policyboxSave: string;
  policyboxConfigured: string;
  policyboxNotConfigured: string;
  policyboxAdminReady: string;
  policyboxRuntimeReady: string;
  policyboxAdminMissing: string;
  policyboxRuntimeMissing: string;
  policyboxRuntimeManage: string;
  policyboxRuntimeDialogTitle: string;
  policyboxRuntimeDialogDescription: string;
  policyboxRuntimeClientId: string;
  policyboxRuntimeClientSecret: string;
  policyboxRuntimeRequestJson: string;
  policyboxRuntimeTicket: string;
  policyboxRuntimeToken: string;
  policyboxRuntimeResponseJson: string;
  policyboxRuntimeIssueTicket: string;
  policyboxRuntimeExchangeToken: string;
  policyboxRuntimeValidateToken: string;
  ragboxTitle: string;
  ragboxDescription: string;
  ragboxEnabled: string;
  ragboxBaseURL: string;
  ragboxAdminToken: string;
  ragboxAdminTokenPlaceholder: string;
  ragboxAdminTokenConfigured: string;
  ragboxTimeout: string;
  ragboxWriteMode: string;
  ragboxSearchMode: string;
  ragboxWriteModeBleveOnly: string;
  ragboxWriteModeDualWrite: string;
  ragboxSearchModeBleveOnly: string;
  ragboxSearchModeDualSearch: string;
  ragboxSearchModeHybrid: string;
  ragboxModesHint: string;
  ragboxWorkspaceBindingTitle: string;
  ragboxCollectionBindingTitle: string;
  ragboxProjectLabel: string;
  ragboxKnowledgeBaseLabel: string;
  ragboxEnsureWorkspace: string;
  ragboxEnsureCollection: string;
  ragboxBindingHint: string;
  ragboxSave: string;
  ragboxConfigured: string;
  ragboxNotConfigured: string;
  ragboxClearAdminToken: string;
  ragboxAdminReady: string;
  ragboxAdminMissing: string;
  ragboxVectorConnectionsTitle: string;
  ragboxVectorConnectionsHint: string;
  ragboxVectorConnectionsFallback: string;
  ragboxVectorConnectionsEmpty: string;
  ragboxVectorConnectionsLoadFailed: string;
  ragboxVectorBackendType: string;
  ragboxVectorConnectionCreateTitle: string;
  ragboxVectorConnectionCreate: string;
  ragboxVectorConnectionCreateDefault: string;
  ragboxVectorConnectionCreated: string;
  ragboxVectorConnectionCreateFailed: string;
  ragboxVectorConnectionDefault: string;
  ragboxVectorConnectionMakeDefault: string;
  ragboxVectorConnectionDefaultSet: string;
  ragboxVectorConnectionDefaultFailed: string;
  ragboxQdrantURL: string;
  ragboxQdrantAPIKey: string;
  ragboxQdrantCollectionPrefix: string;
  ragboxHost: string;
  ragboxPort: string;
  ragboxProtocol: string;
  ragboxUsername: string;
  ragboxPassword: string;
  ragboxDatabaseName: string;
  ragboxPgvectorDatabaseURL: string;
  ragboxPgvectorSchema: string;
  ragboxPgvectorTablePrefix: string;
  ragboxBackendConfigJson: string;
  ragboxBackendConfigJsonHint: string;
  ragboxWorkspaceEnsured: string;
  ragboxCollectionEnsured: string;
  ragboxEnsureWorkspaceFailed: string;
  ragboxEnsureCollectionFailed: string;
  ragboxFileUploadTitle: string;
  ragboxRecordsUpsertTitle: string;
  ragboxChooseFile: string;
  ragboxUploadFile: string;
  ragboxUploadFileFailed: string;
  ragboxFileUploaded: string;
  ragboxRecordsPayload: string;
  ragboxUpsertRecords: string;
  ragboxUpsertRecordsFailed: string;
  ragboxRecordsUpserted: string;
  ragboxSourceName: string;
  ragboxExternalId: string;
  ragboxTitleField: string;
  ragboxMetadataJson: string;
  ragboxPayloadPlaceholder: string;
  ragboxOperationResultTitle: string;
  ragboxOperationResultEmpty: string;
  ragboxLocalIndexed: string;
  ragboxLocalNotIndexed: string;
  ragboxLocalDirPath: string;
  ragboxLocalWrittenFiles: string;
  ragboxRemoteSkipped: string;
  ragboxRemoteJobId: string;
  ragboxRemoteStatus: string;
  ragboxRemoteDocumentId: string;
  ragboxRemoteAcceptedRecords: string;
  ragboxRemoteCollectionId: string;
  ragboxKBVectorConnection: string;
  ragboxKBVectorConnectionDefault: string;
  authMethod: string;
  rolesLabel: string;
  userId: string;
  sessionId: string;
  allUsers: string;
  sessionsFilter: string;
  createUserFailed: string;
  updateUserRolesFailed: string;
  updateUserPasswordFailed: string;
  passwordChanged: string;
  passwordsDoNotMatch: string;
  disableUserFailed: string;
  enableUserFailed: string;
  deleteUserFailed: string;
  disableUserConfirm: string;
  enableUserConfirm: string;
  deleteUserConfirm: string;
  createSessionFailed: string;
  revokeSessionFailed: string;
  revokeCurrentSessionConfirm: string;
  me: string;
  statusReady: string;
  notConnected: string;
  name: string;
  description: string;
  loading: string;
};

type ProPrincipal = {
  name: string;
  scopes: string[];
  roles: string[];
  user_id?: number;
  session_id?: number;
  auth_method?: string;
  is_bootstrap: boolean;
};

type ProTokenRecord = {
  id: number;
  name: string;
  scopes: string[];
  expires_at?: string;
  last_used_at?: string;
  revoked_at?: string;
  created_at: string;
};

type ProUserRecord = {
  id: number;
  email: string;
  display_name: string;
  auth_provider: string;
  external_id: string;
  is_bootstrap: boolean;
  roles: string[];
  scopes: string[];
  last_login_at?: string;
  disabled_at?: string;
  created_at: string;
};

type ProRoleRecord = {
  id: number;
  name: string;
  display_name: string;
  description: string;
  scopes: string[];
  is_system: boolean;
  created_at: string;
};

type ProSessionRecord = {
  id: number;
  user_id: number;
  user_name: string;
  label: string;
  auth_method: string;
  roles: string[];
  scopes: string[];
  expires_at?: string;
  last_used_at?: string;
  revoked_at?: string;
  created_at: string;
};

type ProScopePreset = {
  id: 'reader' | 'operator' | 'admin';
  label: string;
  scopes: string[];
  description: string;
};

type ProjectOption = {
  project_id: number;
  name: string;
};

type ProjectKnowledgeBaseOption = {
  project_id: number;
  collection_id: string;
  name: string;
};

type RagBoxWorkspaceOption = {
  id: string;
  name: string;
  vector_connection_id: string;
  access_mode: string;
  status: string;
};

type SettingsTabID = 'access' | 'storage' | 'policybox' | 'ragbox' | 'users' | 'sessions';

type ProAccessViewProps = {
  logoSrc: string;
  editionName: string;
  proCopy: ProCopy;
  proPrincipal: ProPrincipal | null;
  proLoading: boolean;
  proLocalLoginLoading: boolean;
  proLoginEmail: string;
  proLoginPassword: string;
  proSSOEnabled: boolean;
  proSSOProviderName?: string;
  proSSOIssuerURL?: string;
  proSSORedirectURL?: string;
  proSSOScopes?: string[];
  proSSOHostedDomain?: string;
  proSSODefaultRole?: string;
  proSSOSessionDays?: number;
  proSSOAutoCreateUsers?: boolean;
  proCreateOpen: boolean;
  proCreatingToken: boolean;
  proRevokingTokenId: number | null;
  proNewTokenName: string;
  proNewTokenScopes: string;
  proNewTokenExpiresDays: string;
  proCreatedTokenValue: string | null;
  proCreatedSessionToken: string | null;
  proStorageConfig: ProStorageConfig | null;
  proStorageRuntimeStatus: ProStorageRuntimeStatus | null;
  proStorageShadowStatus: ProStorageShadowStatus | null;
  proStorageOperationalBackend: 'sqlite' | 'postgres';
  proStoragePostgresHost: string;
  proStoragePostgresPort: string;
  proStoragePostgresDatabase: string;
  proStoragePostgresSchema: string;
  proStoragePostgresUsername: string;
  proStoragePostgresPassword: string;
  proStorageClearPostgresDSN: boolean;
  proStorageLogsBackend: 'operational' | 'clickhouse' | 'elasticsearch';
  proStorageClickHouseURL: string;
  proStorageClickHouseDatabase: string;
  proStorageClickHouseUsername: string;
  proStorageClickHousePassword: string;
  proStorageClearClickHousePassword: boolean;
  proStorageElasticsearchURL: string;
  proStorageElasticsearchUsername: string;
  proStorageElasticsearchPassword: string;
  proStorageElasticsearchIndex: string;
  proStorageClearElasticsearchPassword: boolean;
  proStorageSaving: boolean;
  proPolicyBoxBaseURL: string;
  proPolicyBoxAdminToken: string;
  proPolicyBoxAdminTokenConfigured: boolean;
  proPolicyBoxAdminTokenPreview?: string;
  proPolicyBoxSaving: boolean;
  proPolicyBoxConfigured: boolean;
  proPolicyBoxAdminConfigured: boolean;
  proPolicyBoxRuntimeConfigured: boolean;
  proPolicyBoxRuntimeDialogOpen: boolean;
  proPolicyBoxRuntimePayload: string;
  proPolicyBoxRuntimeTicket: string;
  proPolicyBoxRuntimeToken: string;
  proPolicyBoxRuntimeResponseJSON: string;
  proPolicyBoxIssuingTicket: boolean;
  proPolicyBoxExchangingTicket: boolean;
  proPolicyBoxValidatingToken: boolean;
  proRagBoxBaseURL: string;
  proRagBoxAdminToken: string;
  proRagBoxAdminTokenConfigured: boolean;
  proRagBoxAdminTokenPreview?: string;
  proRagBoxClearAdminToken: boolean;
  proRagBoxTimeoutSeconds: string;
  proRagBoxWriteMode: 'bleve_only' | 'dual_write';
  proRagBoxSearchMode: 'bleve_only' | 'dual_search' | 'hybrid';
  proRagBoxSaving: boolean;
  proRagBoxAdminConfigured: boolean;
  proRagBoxVectorConnections: Array<{
    id: string;
    name: string;
    backend_type: string;
    status: string;
    is_default: boolean;
  }>;
  proRagBoxWorkspaces: RagBoxWorkspaceOption[];
  proRagBoxBuiltinFallbackBackend: string;
  proRagBoxLoadingVectorConnections: boolean;
  proRagBoxLoadingWorkspaces: boolean;
  proRagBoxCreatingVectorConnection: boolean;
  proRagBoxDefaultingVectorConnectionId: string | null;
  proRagBoxVectorConnectionForm: {
    name: string;
    backend_type: 'sqlite' | 'qdrant' | 'pgvector' | 'milvus' | 'weaviate' | 'opensearch' | 'elasticsearch' | 'pinecone' | 'chroma';
    description: string;
    is_default: boolean;
    qdrant_scheme: 'http' | 'https';
    qdrant_host: string;
    qdrant_port: string;
    qdrant_api_key: string;
    qdrant_collection_prefix: string;
    pgvector_host: string;
    pgvector_port: string;
    pgvector_username: string;
    pgvector_password: string;
    pgvector_database_name: string;
    pgvector_schema: string;
    pgvector_table_prefix: string;
    backend_config_json: string;
  };
  proRagBoxConfigured: boolean;
  proRagBoxBindingProjectId: string;
  proRagBoxBindingVectorConnectionId: string;
  proRagBoxBindingExistingWorkspaceId: string;
  proRagBoxBindingCollectionProjectId: string;
  proRagBoxBindingCollectionId: string;
  proRagBoxEnsuringWorkspace: boolean;
  proRagBoxEnsuringCollection: boolean;
  proRagBoxUploadProjectId: string;
  proRagBoxUploadCollectionId: string;
  proRagBoxUploadFileName: string;
  proRagBoxUploadSourceName: string;
  proRagBoxUploadExternalId: string;
  proRagBoxUploadTitle: string;
  proRagBoxUploadMetadata: string;
  proRagBoxUploadingFile: boolean;
  proRagBoxRecordsProjectId: string;
  proRagBoxRecordsCollectionId: string;
  proRagBoxRecordsPayload: string;
  proRagBoxUpsertingRecords: boolean;
  proRagBoxLastOperation: ProRagBoxWriteResponse | null;
  ragBoxProjects: ProjectOption[];
  ragBoxProjectKnowledgeBases: ProjectKnowledgeBaseOption[];
  proTokens: ProTokenRecord[];
  proUsers: ProUserRecord[];
  proRoles: ProRoleRecord[];
  proSessions: ProSessionRecord[];
  proScopePresets: ProScopePreset[];
  canWritePro: boolean;
  canAdminPro: boolean;
  proCreateUserOpen: boolean;
  proCreatingUser: boolean;
  proEditUserOpen: boolean;
  proUpdatingUserRoles: boolean;
  proPasswordUserOpen: boolean;
  proUpdatingUserPassword: boolean;
  proDisablingUserId: number | null;
  proEnablingUserId: number | null;
  proDeletingUserId: number | null;
  proSessionsFilterUserId: string;
  proEditingUserId: string;
  proEditingUserName: string;
  proEditingUserRoles: string;
  proPasswordUserId: string;
  proPasswordUserName: string;
  proPasswordUserPassword: string;
  proPasswordUserPasswordConfirm: string;
  proCreateSessionOpen: boolean;
  proCreatingSession: boolean;
  proRevokingSessionId: number | null;
  proNewUserName: string;
  proNewUserEmail: string;
  proNewUserRoles: string;
  proNewSessionUserId: string;
  proNewSessionLabel: string;
  proNewSessionExpiresDays: string;
  onSetProCreateOpen: (open: boolean) => void;
  onSetProCreateUserOpen: (open: boolean) => void;
  onSetProEditUserOpen: (open: boolean) => void;
  onSetProPasswordUserOpen: (open: boolean) => void;
  onSetProCreateSessionOpen: (open: boolean) => void;
  onSetProLoginEmail: (value: string) => void;
  onSetProLoginPassword: (value: string) => void;
  onSetProNewTokenName: (value: string) => void;
  onSetProNewTokenScopes: (value: string) => void;
  onSetProNewTokenExpiresDays: (value: string) => void;
  onSetProNewUserName: (value: string) => void;
  onSetProNewUserEmail: (value: string) => void;
  onSetProNewUserRoles: (value: string) => void;
  onSetProEditingUserRoles: (value: string) => void;
  onSetProPasswordUserPassword: (value: string) => void;
  onSetProPasswordUserPasswordConfirm: (value: string) => void;
  onSetProSessionsFilterUserId: (value: string) => void;
  onSetProNewSessionUserId: (value: string) => void;
  onSetProNewSessionLabel: (value: string) => void;
  onSetProNewSessionExpiresDays: (value: string) => void;
  onSetProStorageOperationalBackend: (value: 'sqlite' | 'postgres') => void;
  onSetProStoragePostgresHost: (value: string) => void;
  onSetProStoragePostgresPort: (value: string) => void;
  onSetProStoragePostgresDatabase: (value: string) => void;
  onSetProStoragePostgresSchema: (value: string) => void;
  onSetProStoragePostgresUsername: (value: string) => void;
  onSetProStoragePostgresPassword: (value: string) => void;
  onSetProStorageClearPostgresDSN: (value: boolean) => void;
  onSetProStorageLogsBackend: (value: 'operational' | 'clickhouse' | 'elasticsearch') => void;
  onSetProStorageClickHouseURL: (value: string) => void;
  onSetProStorageClickHouseDatabase: (value: string) => void;
  onSetProStorageClickHouseUsername: (value: string) => void;
  onSetProStorageClickHousePassword: (value: string) => void;
  onSetProStorageClearClickHousePassword: (value: boolean) => void;
  onSetProStorageElasticsearchURL: (value: string) => void;
  onSetProStorageElasticsearchUsername: (value: string) => void;
  onSetProStorageElasticsearchPassword: (value: string) => void;
  onSetProStorageElasticsearchIndex: (value: string) => void;
  onSetProStorageClearElasticsearchPassword: (value: boolean) => void;
  onSetProPolicyBoxBaseURL: (value: string) => void;
  onSetProPolicyBoxAdminToken: (value: string) => void;
  onSetProPolicyBoxRuntimeDialogOpen: (open: boolean) => void;
  onSetProPolicyBoxRuntimePayload: (value: string) => void;
  onSetProPolicyBoxRuntimeTicket: (value: string) => void;
  onSetProPolicyBoxRuntimeToken: (value: string) => void;
  onSetProRagBoxBaseURL: (value: string) => void;
  onSetProRagBoxAdminToken: (value: string) => void;
  onSetProRagBoxClearAdminToken: (value: boolean) => void;
  onSetProRagBoxTimeoutSeconds: (value: string) => void;
  onSetProRagBoxWriteMode: (value: 'bleve_only' | 'dual_write') => void;
  onSetProRagBoxSearchMode: (value: 'bleve_only' | 'dual_search' | 'hybrid') => void;
  onSetProRagBoxVectorConnectionForm: (updater: (current: ProAccessViewProps['proRagBoxVectorConnectionForm']) => ProAccessViewProps['proRagBoxVectorConnectionForm']) => void;
  onSetProRagBoxBindingProjectId: (value: string) => void;
  onSetProRagBoxBindingVectorConnectionId: (value: string) => void;
  onSetProRagBoxBindingExistingWorkspaceId: (value: string) => void;
  onSetProRagBoxBindingCollectionProjectId: (value: string) => void;
  onSetProRagBoxBindingCollectionId: (value: string) => void;
  onSetProRagBoxUploadProjectId: (value: string) => void;
  onSetProRagBoxUploadCollectionId: (value: string) => void;
  onSetProRagBoxUploadFile: (file: File | null) => void;
  onSetProRagBoxUploadSourceName: (value: string) => void;
  onSetProRagBoxUploadExternalId: (value: string) => void;
  onSetProRagBoxUploadTitle: (value: string) => void;
  onSetProRagBoxUploadMetadata: (value: string) => void;
  onSetProRagBoxRecordsProjectId: (value: string) => void;
  onSetProRagBoxRecordsCollectionId: (value: string) => void;
  onSetProRagBoxRecordsPayload: (value: string) => void;
  onLoginProLocal: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onStartProSSOSignIn: () => void;
  onCreateProToken: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onCreateProUser: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onStartEditProUser: (user: ProUserRecord) => void;
  onUpdateProUserRoles: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onStartProPasswordChange: (user: ProUserRecord) => void;
  onUpdateProUserPassword: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onDisableProUser: (user: ProUserRecord) => Promise<void>;
  onEnableProUser: (user: ProUserRecord) => Promise<void>;
  onDeleteProUser: (user: ProUserRecord) => Promise<void>;
  onCreateProSession: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onRevokeCurrentProSession: () => Promise<void>;
  onSaveProStorageConfig: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onSaveProPolicyBoxConfig: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onIssueProPolicyBoxRuntimeTicket: () => Promise<void>;
  onExchangeProPolicyBoxRuntimeTicket: () => Promise<void>;
  onValidateProPolicyBoxRuntimeToken: () => Promise<void>;
  onSaveProRagBoxConfig: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onCreateProRagBoxVectorConnection: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onSetDefaultProRagBoxVectorConnection: (connectionId: string) => Promise<void>;
  onEnsureRagBoxWorkspace: () => Promise<void>;
  onEnsureRagBoxCollection: () => Promise<void>;
  onUploadRagBoxFile: () => Promise<void>;
  onUpsertRagBoxRecords: () => Promise<void>;
  onCopyToken: (value: string) => Promise<void>;
  onRevokeProToken: (tokenId: number) => Promise<void>;
  onRevokeProSession: (sessionId: number) => Promise<void>;
};

export function ProAccessView({
  logoSrc,
  editionName,
  proCopy,
  proPrincipal,
  proLoading,
  proLocalLoginLoading,
  proLoginEmail,
  proLoginPassword,
  proSSOEnabled,
  proSSOProviderName,
  proSSOIssuerURL,
  proSSORedirectURL,
  proSSOScopes,
  proSSOHostedDomain,
  proSSODefaultRole,
  proSSOSessionDays,
  proSSOAutoCreateUsers,
  proCreateOpen,
  proCreatingToken,
  proRevokingTokenId,
  proNewTokenName,
  proNewTokenScopes,
  proNewTokenExpiresDays,
  proCreatedTokenValue,
  proCreatedSessionToken,
  proStorageConfig,
  proStorageRuntimeStatus,
  proStorageShadowStatus,
  proStorageOperationalBackend,
  proStoragePostgresHost,
  proStoragePostgresPort,
  proStoragePostgresDatabase,
  proStoragePostgresSchema,
  proStoragePostgresUsername,
  proStoragePostgresPassword,
  proStorageClearPostgresDSN,
  proStorageLogsBackend,
  proStorageClickHouseURL,
  proStorageClickHouseDatabase,
  proStorageClickHouseUsername,
  proStorageClickHousePassword,
  proStorageClearClickHousePassword,
  proStorageElasticsearchURL,
  proStorageElasticsearchUsername,
  proStorageElasticsearchPassword,
  proStorageElasticsearchIndex,
  proStorageClearElasticsearchPassword,
  proStorageSaving,
  proPolicyBoxBaseURL,
  proPolicyBoxAdminToken,
  proPolicyBoxAdminTokenConfigured,
  proPolicyBoxAdminTokenPreview,
  proPolicyBoxSaving,
  proPolicyBoxConfigured,
  proPolicyBoxAdminConfigured,
  proPolicyBoxRuntimeConfigured,
  proPolicyBoxRuntimeDialogOpen,
  proPolicyBoxRuntimePayload,
  proPolicyBoxRuntimeTicket,
  proPolicyBoxRuntimeToken,
  proPolicyBoxRuntimeResponseJSON,
  proPolicyBoxIssuingTicket,
  proPolicyBoxExchangingTicket,
  proPolicyBoxValidatingToken,
  proRagBoxBaseURL,
  proRagBoxAdminToken,
  proRagBoxAdminTokenConfigured,
  proRagBoxAdminTokenPreview,
  proRagBoxClearAdminToken,
  proRagBoxTimeoutSeconds,
  proRagBoxWriteMode,
  proRagBoxSearchMode,
  proRagBoxSaving,
  proRagBoxAdminConfigured,
  proRagBoxVectorConnections,
  proRagBoxWorkspaces,
  proRagBoxBuiltinFallbackBackend,
  proRagBoxLoadingVectorConnections,
  proRagBoxLoadingWorkspaces,
  proRagBoxCreatingVectorConnection,
  proRagBoxDefaultingVectorConnectionId,
  proRagBoxVectorConnectionForm,
  proRagBoxConfigured,
  proRagBoxBindingProjectId,
  proRagBoxBindingVectorConnectionId,
  proRagBoxBindingExistingWorkspaceId,
  proRagBoxBindingCollectionProjectId,
  proRagBoxBindingCollectionId,
  proRagBoxEnsuringWorkspace,
  proRagBoxEnsuringCollection,
  proRagBoxUploadProjectId,
  proRagBoxUploadCollectionId,
  proRagBoxUploadFileName,
  proRagBoxUploadSourceName,
  proRagBoxUploadExternalId,
  proRagBoxUploadTitle,
  proRagBoxUploadMetadata,
  proRagBoxUploadingFile,
  proRagBoxRecordsProjectId,
  proRagBoxRecordsCollectionId,
  proRagBoxRecordsPayload,
  proRagBoxUpsertingRecords,
  proRagBoxLastOperation,
  ragBoxProjects,
  ragBoxProjectKnowledgeBases,
  proTokens,
  proUsers,
  proRoles,
  proSessions,
  proScopePresets,
  canWritePro,
  canAdminPro,
  proCreateUserOpen,
  proCreatingUser,
  proEditUserOpen,
  proUpdatingUserRoles,
  proPasswordUserOpen,
  proUpdatingUserPassword,
  proDisablingUserId,
  proEnablingUserId,
  proDeletingUserId,
  proSessionsFilterUserId,
  proEditingUserId,
  proEditingUserName,
  proEditingUserRoles,
  proPasswordUserId,
  proPasswordUserName,
  proPasswordUserPassword,
  proPasswordUserPasswordConfirm,
  proCreateSessionOpen,
  proCreatingSession,
  proRevokingSessionId,
  proNewUserName,
  proNewUserEmail,
  proNewUserRoles,
  proNewSessionUserId,
  proNewSessionLabel,
  proNewSessionExpiresDays,
  onSetProCreateOpen,
  onSetProCreateUserOpen,
  onSetProEditUserOpen,
  onSetProPasswordUserOpen,
  onSetProCreateSessionOpen,
  onSetProLoginEmail,
  onSetProLoginPassword,
  onSetProNewTokenName,
  onSetProNewTokenScopes,
  onSetProNewTokenExpiresDays,
  onSetProNewUserName,
  onSetProNewUserEmail,
  onSetProNewUserRoles,
  onSetProEditingUserRoles,
  onSetProPasswordUserPassword,
  onSetProPasswordUserPasswordConfirm,
  onSetProSessionsFilterUserId,
  onSetProNewSessionUserId,
  onSetProNewSessionLabel,
  onSetProNewSessionExpiresDays,
  onSetProStorageOperationalBackend,
  onSetProStoragePostgresHost,
  onSetProStoragePostgresPort,
  onSetProStoragePostgresDatabase,
  onSetProStoragePostgresSchema,
  onSetProStoragePostgresUsername,
  onSetProStoragePostgresPassword,
  onSetProStorageClearPostgresDSN,
  onSetProStorageLogsBackend,
  onSetProStorageClickHouseURL,
  onSetProStorageClickHouseDatabase,
  onSetProStorageClickHouseUsername,
  onSetProStorageClickHousePassword,
  onSetProStorageClearClickHousePassword,
  onSetProStorageElasticsearchURL,
  onSetProStorageElasticsearchUsername,
  onSetProStorageElasticsearchPassword,
  onSetProStorageElasticsearchIndex,
  onSetProStorageClearElasticsearchPassword,
  onSetProPolicyBoxBaseURL,
  onSetProPolicyBoxAdminToken,
  onSetProPolicyBoxRuntimeDialogOpen,
  onSetProPolicyBoxRuntimePayload,
  onSetProPolicyBoxRuntimeTicket,
  onSetProPolicyBoxRuntimeToken,
  onSetProRagBoxBaseURL,
  onSetProRagBoxAdminToken,
  onSetProRagBoxClearAdminToken,
  onSetProRagBoxTimeoutSeconds,
  onSetProRagBoxWriteMode,
  onSetProRagBoxSearchMode,
  onSetProRagBoxVectorConnectionForm,
  onSetProRagBoxBindingProjectId,
  onSetProRagBoxBindingVectorConnectionId,
  onSetProRagBoxBindingExistingWorkspaceId,
  onSetProRagBoxBindingCollectionProjectId,
  onSetProRagBoxBindingCollectionId,
  onSetProRagBoxUploadProjectId,
  onSetProRagBoxUploadCollectionId,
  onSetProRagBoxUploadFile,
  onSetProRagBoxUploadSourceName,
  onSetProRagBoxUploadExternalId,
  onSetProRagBoxUploadTitle,
  onSetProRagBoxUploadMetadata,
  onSetProRagBoxRecordsProjectId,
  onSetProRagBoxRecordsCollectionId,
  onSetProRagBoxRecordsPayload,
  onLoginProLocal,
  onStartProSSOSignIn,
  onCreateProToken,
  onCreateProUser,
  onStartEditProUser,
  onUpdateProUserRoles,
  onStartProPasswordChange,
  onUpdateProUserPassword,
  onDisableProUser,
  onEnableProUser,
  onDeleteProUser,
  onCreateProSession,
  onRevokeCurrentProSession,
  onSaveProStorageConfig,
  onSaveProPolicyBoxConfig,
  onIssueProPolicyBoxRuntimeTicket,
  onExchangeProPolicyBoxRuntimeTicket,
  onValidateProPolicyBoxRuntimeToken,
  onSaveProRagBoxConfig,
  onCreateProRagBoxVectorConnection,
  onSetDefaultProRagBoxVectorConnection,
  onEnsureRagBoxWorkspace,
  onEnsureRagBoxCollection,
  onUploadRagBoxFile,
  onUpsertRagBoxRecords,
  onCopyToken,
  onRevokeProToken,
  onRevokeProSession,
}: ProAccessViewProps) {
  const [activeSettingsTab, setActiveSettingsTab] = useState<SettingsTabID>('storage');
  const [activeStorageTab, setActiveStorageTab] = useState<'backends' | 'logs'>('backends');
  const [createVectorConnectionOpen, setCreateVectorConnectionOpen] = useState(false);
  const filteredKnowledgeBases = ragBoxProjectKnowledgeBases.filter(
    (item) => String(item.project_id) === proRagBoxBindingCollectionProjectId,
  );
  const filteredUploadKnowledgeBases = ragBoxProjectKnowledgeBases.filter(
    (item) => String(item.project_id) === proRagBoxUploadProjectId,
  );
  const filteredRecordKnowledgeBases = ragBoxProjectKnowledgeBases.filter(
    (item) => String(item.project_id) === proRagBoxRecordsProjectId,
  );
  const formatTimestamp = (value?: string) => (
    value ? new Date(value).toLocaleString() : proCopy.storageNeverSynced
  );

  if (!proPrincipal) {
    return (
      <section className="flex min-h-screen items-center justify-center px-6 py-10">
        <div className="w-full max-w-md rounded-[28px] border border-border bg-card p-6 shadow-[0_20px_60px_rgba(15,23,42,0.08)]">
          <div className="flex flex-col items-center text-center">
            <img src={logoSrc} alt={editionName || proCopy.title} className="h-20 w-20 object-contain" />
            <h2 className="mt-3 text-3xl font-semibold text-electric-blue">{proCopy.title}</h2>
          </div>

          <form className="mt-8 space-y-4" onSubmit={onLoginProLocal}>
            <div className="text-sm font-medium text-foreground mt-3 text-center">{proCopy.localLogin}</div>
            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">{proCopy.email}</span>
              <input
                required
                type="email"
                autoComplete="username"
                value={proLoginEmail}
                onChange={(event) => onSetProLoginEmail(event.target.value)}
                className="h-11 w-full rounded-xl border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
              />
            </label>
            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">{proCopy.password}</span>
              <input
                required
                type="password"
                autoComplete="current-password"
                value={proLoginPassword}
                onChange={(event) => onSetProLoginPassword(event.target.value)}
                className="h-11 w-full rounded-xl border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
              />
            </label>
            <button
              type="submit"
              disabled={proLocalLoginLoading}
              className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-xl bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
            >
              {proLocalLoginLoading ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" />}
              {proCopy.login}
            </button>
          </form>

          {proSSOEnabled ? (
            <div className="mt-6 border-t border-border pt-6">
              <button
                onClick={onStartProSSOSignIn}
                className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-xl border border-electric-blue/30 bg-background px-4 text-sm font-medium text-electric-blue transition-colors hover:bg-electric-blue/12"
              >
                <Shield className="h-4 w-4" />
                {proCopy.signInSSO}
                {proSSOProviderName ? ` · ${proSSOProviderName}` : ''}
              </button>
              {proSSOHostedDomain ? (
                <div className="mt-3 text-center text-xs text-muted-foreground">
                  {proCopy.ssoDomainHint}: <span className="font-medium text-foreground">{proSSOHostedDomain}</span>
                </div>
              ) : null}
            </div>
          ) : null}
        </div>
      </section>
    );
  }

  const expiryPresets = [
    { value: '0', label: proCopy.noExpiry },
    { value: '1', label: proCopy.oneDay },
    { value: '7', label: proCopy.sevenDays },
    { value: '30', label: proCopy.thirtyDays },
    { value: '90', label: proCopy.ninetyDays },
    { value: '365', label: proCopy.oneYear },
  ];
  const settingsTabs: Array<{ id: SettingsTabID; label: string }> = [
    { id: 'storage', label: proCopy.settingsTabStorage },
    { id: 'policybox', label: proCopy.settingsTabPolicybox },
    { id: 'ragbox', label: proCopy.settingsTabRagbox },
    { id: 'access', label: proCopy.settingsTabAccess },
    { id: 'users', label: proCopy.settingsTabUsers },
    { id: 'sessions', label: proCopy.settingsTabSessions },
  ];
  return (
    <>
      <section className="space-y-6">
        <div className="rounded-2xl border border-border bg-card p-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <p className="text-sm uppercase tracking-[0.24em] text-electric-blue">
              MCPBOX
            </p>
            <h2 className="mt-2 text-3xl font-semibold">{proCopy.settingsTitle}</h2>
            <p className="mt-2 max-w-3xl text-muted-foreground">{proCopy.settingsSubtitle}</p>
          </div>
          <div className="rounded-xl border border-border bg-background px-4 py-3">
            <div className="text-sm text-muted-foreground">{proCopy.me}</div>
            <div className="mt-1 text-lg font-semibold">
              {proPrincipal ? proPrincipal.name : proCopy.statusReady}
            </div>
          </div>
        </div>
        <div className="mt-6 flex flex-wrap gap-2">
          {settingsTabs.map((tab) => {
            const isActive = activeSettingsTab === tab.id;
            return (
              <button
                key={tab.id}
                type="button"
                onClick={() => setActiveSettingsTab(tab.id)}
                className={`inline-flex h-10 items-center justify-center rounded-xl border px-4 text-sm font-medium transition-colors ${
                  isActive
                    ? 'border-electric-blue bg-electric-blue/8 text-electric-blue'
                    : 'border-border bg-background text-foreground hover:bg-accent'
                }`}
              >
                {tab.label}
              </button>
            );
          })}
        </div>
      </div>

      <div className={`${activeSettingsTab === 'sessions' ? 'hidden' : 'grid'} gap-6 xl:grid-cols-[1.1fr_0.9fr]`}>
        <section className={`${activeSettingsTab === 'storage' ? 'block' : 'hidden'} rounded-2xl border border-border bg-card p-6 xl:col-span-2`}>
          <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <h3 className="text-xl font-semibold">{proCopy.storageTitle}</h3>
              <p className="mt-2 max-w-3xl text-sm text-muted-foreground">{proCopy.storageDescription}</p>
            </div>
          </div>

          <div className="mt-6 grid gap-4 lg:grid-cols-[0.95fr_1.05fr]">
            <div className="rounded-2xl border border-border bg-background p-4">
              <div className="text-sm text-muted-foreground">{proCopy.storageBootstrapBackend}</div>
              <div className="mt-1 text-2xl font-semibold">{proStorageConfig?.bootstrap_backend ?? 'sqlite'}</div>
              <div className="mt-3 text-xs text-muted-foreground">
                {proCopy.storageBootstrapHint}
              </div>
            </div>
            <div className="rounded-2xl border border-border bg-background p-4">
              <div className="text-lg font-semibold">{proCopy.storageShadowsTitle}</div>
              <div className="mt-4 grid gap-3 sm:grid-cols-3">
                <div className="rounded-xl border border-border bg-card px-4 py-3 text-sm">
                  <div className="text-muted-foreground">{proCopy.storageShadowProjects}</div>
                  <div className="mt-1 text-lg font-semibold">{proStorageShadowStatus?.projects_count ?? 0}</div>
                  <div className="mt-2 text-xs text-muted-foreground">
                    {proCopy.storageLastSync}: {formatTimestamp(proStorageShadowStatus?.last_project_sync_at)}
                  </div>
                </div>
                <div className="rounded-xl border border-border bg-card px-4 py-3 text-sm">
                  <div className="text-muted-foreground">{proCopy.storageShadowKnowledgeBases}</div>
                  <div className="mt-1 text-lg font-semibold">{proStorageShadowStatus?.rag_collections_count ?? 0}</div>
                  <div className="mt-2 text-xs text-muted-foreground">
                    {proCopy.storageLastSync}: {formatTimestamp(proStorageShadowStatus?.last_rag_collection_sync_at)}
                  </div>
                </div>
                <div className="rounded-xl border border-border bg-card px-4 py-3 text-sm">
                  <div className="text-muted-foreground">{proCopy.storageShadowLinks}</div>
                  <div className="mt-1 text-lg font-semibold">{proStorageShadowStatus?.project_rag_links_count ?? 0}</div>
                  <div className="mt-2 text-xs text-muted-foreground">
                    {proCopy.storageLastSync}: {formatTimestamp(proStorageShadowStatus?.last_project_rag_link_sync_at)}
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className="mt-6 flex flex-wrap gap-2">
            {[
              { id: 'backends', label: proCopy.storageTabBackends },
              { id: 'logs', label: proCopy.storageTabLogs },
            ].map((tab) => {
              const isActive = activeStorageTab === tab.id;
              return (
                <button
                  key={tab.id}
                  type="button"
                  onClick={() => setActiveStorageTab(tab.id as 'backends' | 'logs')}
                  className={`inline-flex h-10 items-center justify-center rounded-xl border px-4 text-sm font-medium transition-colors ${
                    isActive
                      ? 'border-electric-blue bg-electric-blue/8 text-electric-blue'
                      : 'border-border bg-background text-foreground hover:bg-accent'
                  }`}
                >
                  {tab.label}
                </button>
              );
            })}
          </div>

          <form className="mt-6 grid gap-4 lg:grid-cols-2" onSubmit={onSaveProStorageConfig}>
            <div className={`${activeStorageTab === 'backends' ? 'contents' : 'hidden'}`}>
              <label className="block space-y-2 lg:col-span-2">
                <span className="text-sm text-muted-foreground">{proCopy.storageOperationalBackend}</span>
                <select
                  value={proStorageOperationalBackend}
                  onChange={(event) => onSetProStorageOperationalBackend(event.target.value as 'sqlite' | 'postgres')}
                  disabled={!canAdminPro}
                  className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                >
                  <option value="sqlite">{proCopy.storageOperationalBackendSQLite}</option>
                  <option value="postgres">{proCopy.storageOperationalBackendPostgres}</option>
                </select>
              </label>

              <div className="rounded-xl border border-border bg-background px-4 py-3 text-sm text-muted-foreground lg:col-span-2">
                {proCopy.storageBackendHint}
              </div>

              <label className="block space-y-2">
                <span className="text-sm text-muted-foreground">{proCopy.storagePostgresHost}</span>
                <input
                  value={proStoragePostgresHost}
                  onChange={(event) => onSetProStoragePostgresHost(event.target.value)}
                  disabled={!canAdminPro || proStorageOperationalBackend !== 'postgres'}
                  className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                  placeholder="127.0.0.1"
                />
              </label>

              <label className="block space-y-2">
                <span className="text-sm text-muted-foreground">{proCopy.storagePostgresPort}</span>
                <input
                  value={proStoragePostgresPort}
                  onChange={(event) => onSetProStoragePostgresPort(event.target.value)}
                  disabled={!canAdminPro || proStorageOperationalBackend !== 'postgres'}
                  className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                  placeholder="5432"
                />
              </label>

              <label className="block space-y-2">
                <span className="text-sm text-muted-foreground">{proCopy.storagePostgresDatabase}</span>
                <input
                  value={proStoragePostgresDatabase}
                  onChange={(event) => onSetProStoragePostgresDatabase(event.target.value)}
                  disabled={!canAdminPro || proStorageOperationalBackend !== 'postgres'}
                  className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                />
              </label>

              <label className="block space-y-2">
                <span className="text-sm text-muted-foreground">{proCopy.storagePostgresSchema}</span>
                <input
                  value={proStoragePostgresSchema}
                  onChange={(event) => onSetProStoragePostgresSchema(event.target.value)}
                  disabled={!canAdminPro || proStorageOperationalBackend !== 'postgres'}
                  className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                  placeholder="public"
                />
              </label>

              <label className="block space-y-2">
                <span className="text-sm text-muted-foreground">{proCopy.storagePostgresUsername}</span>
                <input
                  value={proStoragePostgresUsername}
                  onChange={(event) => onSetProStoragePostgresUsername(event.target.value)}
                  disabled={!canAdminPro || proStorageOperationalBackend !== 'postgres'}
                  className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                />
              </label>

              <label className="block space-y-2 lg:col-span-2">
                <span className="text-sm text-muted-foreground">{proCopy.storagePostgresPassword}</span>
                <input
                  type="password"
                  value={proStoragePostgresPassword}
                  onChange={(event) => onSetProStoragePostgresPassword(event.target.value)}
                  disabled={!canAdminPro || proStorageOperationalBackend !== 'postgres'}
                  className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                />
                {proStorageConfig?.operational_postgres_preview ? (
                  <div className="text-xs text-muted-foreground">
                    {proCopy.storagePostgresConfigured}: {proStorageConfig.operational_postgres_preview}
                  </div>
                ) : null}
                <div className="text-xs text-muted-foreground">{proCopy.storagePostgresPasswordRequired}</div>
              </label>

              <label className="flex items-center gap-3 rounded-md border border-border bg-background px-3 py-3 text-sm lg:col-span-2">
                <input
                  type="checkbox"
                  checked={proStorageClearPostgresDSN}
                  onChange={(event) => onSetProStorageClearPostgresDSN(event.target.checked)}
                  disabled={!canAdminPro || !proStorageConfig?.operational_postgres_configured}
                  className="h-4 w-4 rounded border-border"
                />
                <span>{proCopy.storageClearPostgresDSN}</span>
              </label>
            </div>

            <div className={`${activeStorageTab === 'logs' ? 'contents' : 'hidden'}`}>
              <label className="block space-y-2 lg:col-span-2">
                <span className="text-sm text-muted-foreground">{proCopy.storageLogsBackend}</span>
                <select
                  value={proStorageLogsBackend}
                  onChange={(event) => onSetProStorageLogsBackend(event.target.value as 'operational' | 'clickhouse' | 'elasticsearch')}
                  disabled={!canAdminPro}
                  className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                >
                  <option value="operational">{proCopy.storageLogsBackendOperational}</option>
                  <option value="clickhouse">{proCopy.storageLogsBackendClickHouse}</option>
                  <option value="elasticsearch">{proCopy.storageLogsBackendElasticsearch}</option>
                </select>
              </label>

              <div className="rounded-xl border border-border bg-background px-4 py-3 text-sm text-muted-foreground lg:col-span-2">
                {proCopy.storageLogsHint}
              </div>

              {proStorageLogsBackend === 'operational' ? (
                <div className="rounded-xl border border-border bg-background px-4 py-4 text-sm text-muted-foreground lg:col-span-2">
                  {proCopy.storageLogsOperationalOnly}
                </div>
              ) : null}

              {proStorageLogsBackend === 'clickhouse' ? (
                <>
                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.storageClickHouseURL}</span>
                    <input
                      value={proStorageClickHouseURL}
                      onChange={(event) => onSetProStorageClickHouseURL(event.target.value)}
                      disabled={!canAdminPro}
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                      placeholder="http://127.0.0.1:8123"
                    />
                  </label>

                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.storageClickHouseDatabase}</span>
                    <input
                      value={proStorageClickHouseDatabase}
                      onChange={(event) => onSetProStorageClickHouseDatabase(event.target.value)}
                      disabled={!canAdminPro}
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                    />
                  </label>

                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.storageClickHouseUsername}</span>
                    <input
                      value={proStorageClickHouseUsername}
                      onChange={(event) => onSetProStorageClickHouseUsername(event.target.value)}
                      disabled={!canAdminPro}
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                    />
                  </label>

                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.storageClickHousePassword}</span>
                    <input
                      type="password"
                      value={proStorageClickHousePassword}
                      onChange={(event) => onSetProStorageClickHousePassword(event.target.value)}
                      disabled={!canAdminPro}
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                    />
                    {proStorageConfig?.clickhouse_password_preview ? (
                      <div className="text-xs text-muted-foreground">
                        {proCopy.storageClickHouseConfigured}: {proStorageConfig.clickhouse_password_preview}
                      </div>
                    ) : null}
                  </label>

                  <label className="flex items-center gap-3 rounded-md border border-border bg-background px-3 py-3 text-sm lg:col-span-2">
                    <input
                      type="checkbox"
                      checked={proStorageClearClickHousePassword}
                      onChange={(event) => onSetProStorageClearClickHousePassword(event.target.checked)}
                      disabled={!canAdminPro || !proStorageConfig?.clickhouse_configured}
                      className="h-4 w-4 rounded border-border"
                    />
                    <span>{proCopy.storageClearClickHousePassword}</span>
                  </label>
                </>
              ) : null}

              {proStorageLogsBackend === 'elasticsearch' ? (
                <>
                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.storageElasticsearchURL}</span>
                    <input
                      value={proStorageElasticsearchURL}
                      onChange={(event) => onSetProStorageElasticsearchURL(event.target.value)}
                      disabled={!canAdminPro}
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                      placeholder="http://127.0.0.1:9200"
                    />
                  </label>

                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.storageElasticsearchIndex}</span>
                    <input
                      value={proStorageElasticsearchIndex}
                      onChange={(event) => onSetProStorageElasticsearchIndex(event.target.value)}
                      disabled={!canAdminPro}
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                    />
                  </label>

                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.storageElasticsearchUsername}</span>
                    <input
                      value={proStorageElasticsearchUsername}
                      onChange={(event) => onSetProStorageElasticsearchUsername(event.target.value)}
                      disabled={!canAdminPro}
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                    />
                  </label>

                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.storageElasticsearchPassword}</span>
                    <input
                      type="password"
                      value={proStorageElasticsearchPassword}
                      onChange={(event) => onSetProStorageElasticsearchPassword(event.target.value)}
                      disabled={!canAdminPro}
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                    />
                    {proStorageConfig?.elasticsearch_password_preview ? (
                      <div className="text-xs text-muted-foreground">
                        {proCopy.storageElasticsearchConfigured}: {proStorageConfig.elasticsearch_password_preview}
                      </div>
                    ) : null}
                  </label>

                  <label className="flex items-center gap-3 rounded-md border border-border bg-background px-3 py-3 text-sm lg:col-span-2">
                    <input
                      type="checkbox"
                      checked={proStorageClearElasticsearchPassword}
                      onChange={(event) => onSetProStorageClearElasticsearchPassword(event.target.checked)}
                      disabled={!canAdminPro || !proStorageConfig?.elasticsearch_configured}
                      className="h-4 w-4 rounded border-border"
                    />
                    <span>{proCopy.storageClearElasticsearchPassword}</span>
                  </label>
                </>
              ) : null}
            </div>

            <div className="grid gap-3 lg:col-span-2 lg:grid-cols-3">
              <div className="rounded-xl border border-border bg-background px-4 py-3 text-sm">
                <div className="text-muted-foreground">{proCopy.storageDesiredBackend}</div>
                <div className="mt-1 font-medium">
                  {proStorageRuntimeStatus?.desired_operational_backend ?? proStorageConfig?.operational_backend ?? 'sqlite'}
                </div>
              </div>
              <div className="rounded-xl border border-border bg-background px-4 py-3 text-sm">
                <div className="text-muted-foreground">{proCopy.storageActiveBackend}</div>
                <div className="mt-1 font-medium">
                  {proStorageRuntimeStatus?.active_operational_backend ?? proStorageConfig?.operational_backend ?? 'sqlite'}
                </div>
              </div>
              <div className="rounded-xl border border-border bg-background px-4 py-3 text-sm">
                <div className="text-muted-foreground">{proCopy.storageLogsBackend}</div>
                <div className="mt-1 font-medium">
                  {proStorageRuntimeStatus?.active_logs_backend ?? proStorageConfig?.effective_logs_backend ?? 'operational'}
                </div>
              </div>
            </div>

            {proStorageRuntimeStatus?.operational_fallback_reason || proStorageRuntimeStatus?.logs_fallback_reason ? (
              <div className="rounded-xl border border-amber-500/30 bg-amber-500/10 px-4 py-3 text-sm text-amber-700 lg:col-span-2">
                <div>{proStorageRuntimeStatus.operational_fallback_reason ?? ''}</div>
                <div>{proStorageRuntimeStatus.logs_fallback_reason ?? ''}</div>
              </div>
            ) : null}

            <div className="grid gap-3 lg:col-span-2 lg:grid-cols-3">
              {[
                ['PostgreSQL', proStorageRuntimeStatus?.postgres],
                ['ClickHouse', proStorageRuntimeStatus?.clickhouse],
                ['Elasticsearch', proStorageRuntimeStatus?.elasticsearch],
              ].map(([label, status]) => (
                <div key={label} className="rounded-xl border border-border bg-background px-4 py-3 text-sm">
                  <div className="font-medium">{label}</div>
                  <div className="mt-2 text-muted-foreground">
                    {!status?.configured
                      ? proCopy.storageConnectionNotConfigured
                      : status.reachable
                        ? proCopy.storageConnectionReachable
                        : proCopy.storageConnectionUnreachable}
                  </div>
                  {status?.message ? (
                    <div className="mt-2 text-xs text-muted-foreground">{status.message}</div>
                  ) : null}
                </div>
              ))}
            </div>

            <div className="lg:col-span-2">
              <button
                type="submit"
                disabled={!canAdminPro || proStorageSaving}
                className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
              >
                {proStorageSaving ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Database className="h-4 w-4" />}
                {proCopy.storageSave}
              </button>
            </div>
          </form>
        </section>

        <section className={`${activeSettingsTab === 'policybox' ? 'block' : 'hidden'} rounded-2xl border border-border bg-card p-6 xl:col-span-2`}>
          <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <h3 className="text-xl font-semibold">{proCopy.policyboxTitle}</h3>
              <p className="mt-2 max-w-3xl text-sm text-muted-foreground">{proCopy.policyboxDescription}</p>
              <div className="mt-4 grid gap-3 lg:grid-cols-2">
                <div className="rounded-xl border border-border bg-background px-4 py-3 text-sm text-muted-foreground">
                  {proCopy.policyboxStartupTokensHint}
                </div>
                <div className="rounded-xl border border-border bg-background px-4 py-3 text-sm text-muted-foreground">
                  {proCopy.policyboxJWTSecretHint}
                </div>
              </div>
            </div>
            <div className="rounded-xl border border-border bg-background px-4 py-3">
              <div className="text-sm text-muted-foreground">{proCopy.statusReady}</div>
              <div className="mt-1 text-lg font-semibold">
                {proPolicyBoxConfigured ? proCopy.policyboxConfigured : proCopy.policyboxNotConfigured}
              </div>
            </div>
          </div>

          <form className="mt-6 grid gap-4 lg:grid-cols-2" onSubmit={onSaveProPolicyBoxConfig}>
            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">{proCopy.policyboxBaseURL}</span>
              <input
                value={proPolicyBoxBaseURL}
                onChange={(event) => onSetProPolicyBoxBaseURL(event.target.value)}
                disabled={!canAdminPro}
                className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                placeholder="http://127.0.0.1:38180"
              />
            </label>

            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">{proCopy.policyboxAdminToken}</span>
              <input
                type="password"
                value={proPolicyBoxAdminToken}
                onChange={(event) => onSetProPolicyBoxAdminToken(event.target.value)}
                disabled={!canAdminPro}
                className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                placeholder={proCopy.policyboxAdminTokenPlaceholder}
              />
              {proPolicyBoxAdminTokenPreview ? (
                <div className="text-xs text-muted-foreground">{proCopy.policyboxAdminTokenConfigured}: {proPolicyBoxAdminTokenPreview}</div>
              ) : null}
            </label>

            <div className="rounded-md border border-border bg-background px-3 py-3 text-sm text-muted-foreground">
              {proPolicyBoxAdminConfigured ? proCopy.policyboxAdminReady : proCopy.policyboxAdminMissing}
            </div>

            <div className="rounded-md border border-border bg-background px-3 py-3 text-sm text-muted-foreground">
              {proPolicyBoxRuntimeConfigured ? proCopy.policyboxRuntimeReady : proCopy.policyboxRuntimeMissing}
            </div>

            <Dialog open={proPolicyBoxRuntimeDialogOpen} onOpenChange={onSetProPolicyBoxRuntimeDialogOpen}>
              <DialogTrigger asChild>
                <button
                  type="button"
                  disabled={!canAdminPro || !proPolicyBoxAdminConfigured}
                  className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                >
                  <Shield className="h-4 w-4" />
                  {proCopy.policyboxRuntimeManage}
                </button>
              </DialogTrigger>
              <DialogContent className="w-[80vw] max-w-[80vw] min-w-[80vw]">
                <DialogHeader>
                  <DialogTitle>{proCopy.policyboxRuntimeDialogTitle}</DialogTitle>
                  <DialogDescription>{proCopy.policyboxRuntimeDialogDescription}</DialogDescription>
                </DialogHeader>
                <div className="space-y-4">
                  <div className="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
                    <div className="space-y-3 rounded-xl border border-border bg-background p-4">
                      <div className="text-sm font-medium">{proCopy.policyboxRuntimeRequestJson}</div>
                      <Textarea
                        value={proPolicyBoxRuntimePayload}
                        onChange={(event) => onSetProPolicyBoxRuntimePayload(event.target.value)}
                        disabled={!canAdminPro}
                        className="min-h-[460px] font-mono text-xs"
                        spellCheck={false}
                      />
                      <button
                        type="button"
                        onClick={() => void onIssueProPolicyBoxRuntimeTicket()}
                        disabled={!canAdminPro || proPolicyBoxIssuingTicket}
                        className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                      >
                        {proPolicyBoxIssuingTicket ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" />}
                        {proCopy.policyboxRuntimeIssueTicket}
                      </button>
                    </div>
                    <div className="space-y-3 rounded-xl border border-border bg-background p-4">
                      <label className="block space-y-2">
                        <span className="text-sm text-muted-foreground">{proCopy.policyboxRuntimeTicket}</span>
                        <input
                          value={proPolicyBoxRuntimeTicket}
                          onChange={(event) => onSetProPolicyBoxRuntimeTicket(event.target.value)}
                          disabled={!canAdminPro}
                          className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                        />
                      </label>
                      <button
                        type="button"
                        onClick={() => void onExchangeProPolicyBoxRuntimeTicket()}
                        disabled={!canAdminPro || proPolicyBoxExchangingTicket}
                        className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                      >
                        {proPolicyBoxExchangingTicket ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" />}
                        {proCopy.policyboxRuntimeExchangeToken}
                      </button>
                      <label className="block space-y-2">
                        <span className="text-sm text-muted-foreground">{proCopy.policyboxRuntimeToken}</span>
                        <Textarea
                          value={proPolicyBoxRuntimeToken}
                          onChange={(event) => onSetProPolicyBoxRuntimeToken(event.target.value)}
                          disabled={!canAdminPro}
                          className="min-h-[140px] font-mono text-xs"
                          spellCheck={false}
                        />
                      </label>
                      <button
                        type="button"
                        onClick={() => void onValidateProPolicyBoxRuntimeToken()}
                        disabled={!canAdminPro || proPolicyBoxValidatingToken}
                        className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                      >
                        {proPolicyBoxValidatingToken ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" />}
                        {proCopy.policyboxRuntimeValidateToken}
                      </button>
                      <div className="text-sm font-medium">{proCopy.policyboxRuntimeResponseJson}</div>
                      <Textarea
                        value={proPolicyBoxRuntimeResponseJSON}
                        readOnly
                        className="min-h-[230px] font-mono text-xs"
                        spellCheck={false}
                      />
                    </div>
                  </div>
                </div>
              </DialogContent>
            </Dialog>

            <div className="lg:col-span-2">
              <button
                type="submit"
                disabled={!canAdminPro || proPolicyBoxSaving}
                className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
              >
                {proPolicyBoxSaving ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" />}
                {proCopy.policyboxSave}
              </button>
            </div>
          </form>
        </section>

        <section className={`${activeSettingsTab === 'ragbox' ? 'block' : 'hidden'} rounded-2xl border border-border bg-card p-6 xl:col-span-2`}>
          <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
            <div>
              <h3 className="text-xl font-semibold">{proCopy.ragboxTitle}</h3>
              <p className="mt-2 max-w-3xl text-sm text-muted-foreground">{proCopy.ragboxDescription}</p>
            </div>
            <div className="rounded-xl border border-border bg-background px-4 py-3">
              <div className="text-sm text-muted-foreground">{proCopy.statusReady}</div>
              <div className="mt-1 text-lg font-semibold">
                {proRagBoxConfigured ? proCopy.ragboxConfigured : proCopy.ragboxNotConfigured}
              </div>
            </div>
          </div>

          <form className="mt-6 grid gap-4 lg:grid-cols-2" onSubmit={onSaveProRagBoxConfig}>
            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">{proCopy.ragboxBaseURL}</span>
              <input
                value={proRagBoxBaseURL}
                onChange={(event) => onSetProRagBoxBaseURL(event.target.value)}
                disabled={!canAdminPro}
                className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                placeholder="http://127.0.0.1:18092"
              />
            </label>

            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">{proCopy.ragboxAdminToken}</span>
              <input
                type="password"
                value={proRagBoxAdminToken}
                onChange={(event) => onSetProRagBoxAdminToken(event.target.value)}
                disabled={!canAdminPro}
                className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                placeholder={proCopy.ragboxAdminTokenPlaceholder}
              />
              {proRagBoxAdminTokenPreview ? (
                <div className="text-xs text-muted-foreground">{proCopy.ragboxAdminTokenConfigured}: {proRagBoxAdminTokenPreview}</div>
              ) : null}
            </label>

            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">{proCopy.ragboxTimeout}</span>
              <input
                type="number"
                min="1"
                max="300"
                value={proRagBoxTimeoutSeconds}
                onChange={(event) => onSetProRagBoxTimeoutSeconds(event.target.value)}
                disabled={!canAdminPro}
                className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
              />
            </label>

            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">{proCopy.ragboxWriteMode}</span>
              <select
                value={proRagBoxWriteMode}
                onChange={(event) => onSetProRagBoxWriteMode(event.target.value as 'bleve_only' | 'dual_write')}
                disabled={!canAdminPro}
                className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
              >
                <option value="bleve_only">{proCopy.ragboxWriteModeBleveOnly}</option>
                <option value="dual_write">{proCopy.ragboxWriteModeDualWrite}</option>
              </select>
            </label>

            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">{proCopy.ragboxSearchMode}</span>
              <select
                value={proRagBoxSearchMode}
                onChange={(event) => onSetProRagBoxSearchMode(event.target.value as 'bleve_only' | 'dual_search' | 'hybrid')}
                disabled={!canAdminPro}
                className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
              >
                <option value="bleve_only">{proCopy.ragboxSearchModeBleveOnly}</option>
                <option value="dual_search">{proCopy.ragboxSearchModeDualSearch}</option>
                <option value="hybrid">{proCopy.ragboxSearchModeHybrid}</option>
              </select>
            </label>

            <label className="flex items-center gap-3 rounded-md border border-border bg-background px-3 py-3 text-sm lg:col-span-2">
              <input
                type="checkbox"
                checked={proRagBoxClearAdminToken}
                onChange={(event) => onSetProRagBoxClearAdminToken(event.target.checked)}
                disabled={!canAdminPro || !proRagBoxAdminTokenConfigured}
                className="h-4 w-4 rounded border-border"
              />
              <span>{proCopy.ragboxClearAdminToken}</span>
            </label>

            <div className="rounded-md border border-border bg-background px-3 py-3 text-sm text-muted-foreground lg:col-span-2">
              {proCopy.ragboxModesHint}
            </div>

            <div className="rounded-md border border-border bg-background px-3 py-3 text-sm text-muted-foreground">
              {proRagBoxConfigured ? proCopy.ragboxConfigured : proCopy.ragboxNotConfigured}
            </div>

            <div className="rounded-md border border-border bg-background px-3 py-3 text-sm text-muted-foreground">
              {proRagBoxAdminConfigured ? proCopy.ragboxAdminReady : proCopy.ragboxAdminMissing}
            </div>

            <div className="lg:col-span-2">
              <button
                type="submit"
                disabled={!canAdminPro || proRagBoxSaving}
                className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
              >
                {proRagBoxSaving ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" />}
                {proCopy.ragboxSave}
              </button>
            </div>
          </form>

          <div className="mt-6 grid gap-6 lg:grid-cols-[1.1fr_0.9fr]">
            <section className="rounded-2xl border border-border bg-background p-5">
              <div className="flex items-center justify-between gap-4">
                <div>
                  <h4 className="text-base font-semibold">{proCopy.ragboxVectorConnectionsTitle}</h4>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {proCopy.ragboxVectorConnectionsHint}
                  </p>
                </div>
                <div className="text-xs text-muted-foreground">
                  {proCopy.ragboxVectorConnectionsFallback}: {proRagBoxBuiltinFallbackBackend}
                </div>
              </div>

              <div className="mt-4">
                <Dialog open={createVectorConnectionOpen} onOpenChange={setCreateVectorConnectionOpen}>
                  <DialogTrigger asChild>
                    <button
                      type="button"
                      disabled={!canAdminPro}
                      className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      <Plus className="h-4 w-4" />
                      {proCopy.ragboxVectorConnectionCreate}
                    </button>
                  </DialogTrigger>
                  <DialogContent className="sm:max-w-2xl">
                    <DialogHeader>
                      <DialogTitle>{proCopy.ragboxVectorConnectionCreateTitle}</DialogTitle>
                      <DialogDescription>{proCopy.ragboxVectorConnectionsHint}</DialogDescription>
                    </DialogHeader>
                    <form className="space-y-4" onSubmit={onCreateProRagBoxVectorConnection}>
                      <div className="grid gap-4 md:grid-cols-2">
                        <label className="block space-y-2">
                          <span className="text-sm text-muted-foreground">{proCopy.name}</span>
                          <input
                            required
                            value={proRagBoxVectorConnectionForm.name}
                            onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, name: event.target.value }))}
                            disabled={!canAdminPro}
                            className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                          />
                        </label>

                        <label className="block space-y-2">
                          <span className="text-sm text-muted-foreground">{proCopy.ragboxVectorBackendType}</span>
                          <select
                            value={proRagBoxVectorConnectionForm.backend_type}
                            onChange={(event) =>
                              onSetProRagBoxVectorConnectionForm((current) => ({
                                ...current,
                                backend_type: event.target.value as 'sqlite' | 'qdrant' | 'pgvector' | 'milvus' | 'weaviate' | 'opensearch' | 'elasticsearch' | 'pinecone' | 'chroma',
                              }))
                            }
                            disabled={!canAdminPro}
                            className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                          >
                            <option value="sqlite">sqlite</option>
                            <option value="qdrant">qdrant</option>
                            <option value="pgvector">pgvector</option>
                            <option value="milvus">milvus</option>
                            <option value="weaviate">weaviate</option>
                            <option value="opensearch">opensearch</option>
                            <option value="elasticsearch">elasticsearch</option>
                            <option value="pinecone">pinecone</option>
                            <option value="chroma">chroma</option>
                          </select>
                        </label>
                      </div>

                      <label className="block space-y-2">
                        <span className="text-sm text-muted-foreground">{proCopy.description}</span>
                        <input
                          value={proRagBoxVectorConnectionForm.description}
                          onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, description: event.target.value }))}
                          disabled={!canAdminPro}
                          className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                        />
                      </label>

                      {proRagBoxVectorConnectionForm.backend_type === 'qdrant' ? (
                        <div className="grid gap-4 md:grid-cols-2">
                          <label className="block space-y-2">
                            <span className="text-sm text-muted-foreground">{proCopy.ragboxProtocol}</span>
                            <select
                              value={proRagBoxVectorConnectionForm.qdrant_scheme}
                              onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, qdrant_scheme: event.target.value as 'http' | 'https' }))}
                              disabled={!canAdminPro}
                              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                            >
                              <option value="http">http</option>
                              <option value="https">https</option>
                            </select>
                          </label>
                          <label className="block space-y-2">
                            <span className="text-sm text-muted-foreground">{proCopy.ragboxHost}</span>
                            <input
                              required
                              value={proRagBoxVectorConnectionForm.qdrant_host}
                              onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, qdrant_host: event.target.value }))}
                              disabled={!canAdminPro}
                              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                            />
                          </label>
                          <label className="block space-y-2">
                            <span className="text-sm text-muted-foreground">{proCopy.ragboxPort}</span>
                            <input
                              required
                              value={proRagBoxVectorConnectionForm.qdrant_port}
                              onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, qdrant_port: event.target.value }))}
                              disabled={!canAdminPro}
                              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                            />
                          </label>
                          <label className="block space-y-2">
                            <span className="text-sm text-muted-foreground">{proCopy.ragboxQdrantAPIKey}</span>
                            <input
                              value={proRagBoxVectorConnectionForm.qdrant_api_key}
                              onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, qdrant_api_key: event.target.value }))}
                              disabled={!canAdminPro}
                              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                            />
                          </label>
                          <label className="block space-y-2 md:col-span-2">
                            <span className="text-sm text-muted-foreground">{proCopy.ragboxQdrantCollectionPrefix}</span>
                            <input
                              required
                              value={proRagBoxVectorConnectionForm.qdrant_collection_prefix}
                              onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, qdrant_collection_prefix: event.target.value }))}
                              disabled={!canAdminPro}
                              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                            />
                          </label>
                        </div>
                      ) : null}

                      {proRagBoxVectorConnectionForm.backend_type === 'pgvector' ? (
                        <div className="grid gap-4 md:grid-cols-2">
                          <label className="block space-y-2">
                            <span className="text-sm text-muted-foreground">{proCopy.ragboxHost}</span>
                            <input
                              required
                              value={proRagBoxVectorConnectionForm.pgvector_host}
                              onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, pgvector_host: event.target.value }))}
                              disabled={!canAdminPro}
                              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                            />
                          </label>
                          <label className="block space-y-2">
                            <span className="text-sm text-muted-foreground">{proCopy.ragboxPort}</span>
                            <input
                              required
                              value={proRagBoxVectorConnectionForm.pgvector_port}
                              onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, pgvector_port: event.target.value }))}
                              disabled={!canAdminPro}
                              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                            />
                          </label>
                          <label className="block space-y-2">
                            <span className="text-sm text-muted-foreground">{proCopy.ragboxUsername}</span>
                            <input
                              required
                              value={proRagBoxVectorConnectionForm.pgvector_username}
                              onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, pgvector_username: event.target.value }))}
                              disabled={!canAdminPro}
                              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                            />
                          </label>
                          <label className="block space-y-2">
                            <span className="text-sm text-muted-foreground">{proCopy.ragboxPassword}</span>
                            <input
                              required
                              type="password"
                              value={proRagBoxVectorConnectionForm.pgvector_password}
                              onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, pgvector_password: event.target.value }))}
                              disabled={!canAdminPro}
                              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                            />
                          </label>
                          <label className="block space-y-2">
                            <span className="text-sm text-muted-foreground">{proCopy.ragboxDatabaseName}</span>
                            <input
                              required
                              value={proRagBoxVectorConnectionForm.pgvector_database_name}
                              onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, pgvector_database_name: event.target.value }))}
                              disabled={!canAdminPro}
                              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                            />
                          </label>
                          <label className="block space-y-2">
                            <span className="text-sm text-muted-foreground">{proCopy.ragboxPgvectorSchema}</span>
                            <input
                              required
                              value={proRagBoxVectorConnectionForm.pgvector_schema}
                              onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, pgvector_schema: event.target.value }))}
                              disabled={!canAdminPro}
                              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                            />
                          </label>
                          <label className="block space-y-2 md:col-span-2">
                            <span className="text-sm text-muted-foreground">{proCopy.ragboxPgvectorTablePrefix}</span>
                            <input
                              value={proRagBoxVectorConnectionForm.pgvector_table_prefix}
                              onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, pgvector_table_prefix: event.target.value }))}
                              disabled={!canAdminPro}
                              className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                            />
                          </label>
                        </div>
                      ) : null}

                      {!['sqlite', 'qdrant', 'pgvector'].includes(proRagBoxVectorConnectionForm.backend_type) ? (
                        <label className="block space-y-2">
                          <span className="text-sm text-muted-foreground">{proCopy.ragboxBackendConfigJson}</span>
                          <textarea
                            required
                            rows={9}
                            value={proRagBoxVectorConnectionForm.backend_config_json}
                            onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, backend_config_json: event.target.value }))}
                            disabled={!canAdminPro}
                            placeholder={proCopy.ragboxBackendConfigJsonHint}
                            className="w-full rounded-md border border-border bg-input-background px-3 py-2 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                          />
                        </label>
                      ) : null}

                      <label className="flex items-center gap-3 rounded-md border border-border bg-card px-3 py-3 text-sm">
                        <input
                          type="checkbox"
                          checked={proRagBoxVectorConnectionForm.is_default}
                          onChange={(event) => onSetProRagBoxVectorConnectionForm((current) => ({ ...current, is_default: event.target.checked }))}
                          disabled={!canAdminPro}
                          className="h-4 w-4 rounded border-border"
                        />
                        <span>{proCopy.ragboxVectorConnectionCreateDefault}</span>
                      </label>

                      <div className="flex justify-end">
                        <button
                          type="submit"
                          disabled={!canAdminPro || proRagBoxCreatingVectorConnection}
                          className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                        >
                          {proRagBoxCreatingVectorConnection ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                          {proCopy.ragboxVectorConnectionCreate}
                        </button>
                      </div>
                    </form>
                  </DialogContent>
                </Dialog>
              </div>

              <div className="mt-4 space-y-3">
                {proRagBoxLoadingVectorConnections ? (
                  <div className="text-sm text-muted-foreground">{proCopy.loading}</div>
                ) : proRagBoxVectorConnections.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-border px-4 py-5 text-sm text-muted-foreground">
                    {proCopy.ragboxVectorConnectionsEmpty}
                  </div>
                ) : (
                  proRagBoxVectorConnections.map((connection) => (
                    <div key={connection.id} className="rounded-xl border border-border px-4 py-4">
                      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                        <div>
                          <div className="font-medium">{connection.name}</div>
                          <div className="mt-1 text-xs text-muted-foreground">
                            {connection.backend_type} · {connection.status} · {connection.id}
                          </div>
                        </div>
                        <button
                          type="button"
                          onClick={() => void onSetDefaultProRagBoxVectorConnection(connection.id)}
                          disabled={!canAdminPro || connection.is_default || proRagBoxDefaultingVectorConnectionId === connection.id}
                          className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                        >
                          {proRagBoxDefaultingVectorConnectionId === connection.id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" />}
                          {connection.is_default ? proCopy.ragboxVectorConnectionDefault : proCopy.ragboxVectorConnectionMakeDefault}
                        </button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </section>

            <section className="rounded-2xl border border-border bg-background p-5">
              <div>
                <h4 className="text-base font-semibold">{proCopy.ragboxWorkspaceBindingTitle}</h4>
                <p className="mt-1 text-sm text-muted-foreground">
                  {proCopy.ragboxBindingHint}
                </p>
              </div>

              <div className="mt-4 space-y-4">
                <label className="block space-y-2">
                  <span className="text-sm text-muted-foreground">{proCopy.ragboxProjectLabel}</span>
                  <select
                    value={proRagBoxBindingProjectId}
                    onChange={(event) => onSetProRagBoxBindingProjectId(event.target.value)}
                    disabled={!canAdminPro}
                    className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    <option value="">{proCopy.notConnected}</option>
                    {ragBoxProjects.map((project) => (
                      <option key={project.project_id} value={String(project.project_id)}>
                        {project.name}
                      </option>
                    ))}
                  </select>
                </label>

                <label className="block space-y-2">
                  <span className="text-sm text-muted-foreground">Existing RagBox workspace</span>
                  <select
                    value={proRagBoxBindingExistingWorkspaceId}
                    onChange={(event) => onSetProRagBoxBindingExistingWorkspaceId(event.target.value)}
                    disabled={!canAdminPro}
                    className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    <option value="">Create or use MCPBox-managed workspace</option>
                    {proRagBoxWorkspaces.map((workspace) => (
                      <option key={workspace.id} value={workspace.id}>
                        {workspace.name} · {workspace.vector_connection_id} · {workspace.access_mode}
                      </option>
                    ))}
                  </select>
                  <p className="text-xs text-muted-foreground">
                    {proRagBoxLoadingWorkspaces
                      ? proCopy.loading
                      : 'Choose an existing RagBox workspace to bind this project, or leave empty to create a new workspace from MCPBox.'}
                  </p>
                </label>

                <label className="block space-y-2">
                  <span className="text-sm text-muted-foreground">{proCopy.ragboxKBVectorConnection}</span>
                  <select
                    value={proRagBoxBindingVectorConnectionId}
                    onChange={(event) => onSetProRagBoxBindingVectorConnectionId(event.target.value)}
                    disabled={!canAdminPro || !!proRagBoxBindingExistingWorkspaceId}
                    className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    <option value="">{proCopy.ragboxKBVectorConnectionDefault}: {proRagBoxBuiltinFallbackBackend}</option>
                    {proRagBoxVectorConnections.map((connection) => (
                      <option key={connection.id} value={connection.id}>
                        {connection.name} · {connection.backend_type}{connection.is_default ? ' · default' : ''}
                      </option>
                    ))}
                  </select>
                  {proRagBoxBindingExistingWorkspaceId ? (
                    <p className="text-xs text-muted-foreground">
                      Existing workspace selected, so its current vector connection will be used.
                    </p>
                  ) : null}
                </label>

                <button
                  type="button"
                  onClick={() => void onEnsureRagBoxWorkspace()}
                  disabled={!canAdminPro || !proRagBoxBindingProjectId || proRagBoxEnsuringWorkspace}
                  className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                >
                  {proRagBoxEnsuringWorkspace ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" />}
                  {proCopy.ragboxEnsureWorkspace}
                </button>
              </div>
            </section>
          </div>

        </section>

        <section className={`${activeSettingsTab === 'access' ? 'block' : 'hidden'} rounded-2xl border border-border bg-card p-6`}>
          <div className="flex items-start justify-between gap-4">
            <div>
              <h3 className="text-xl font-semibold">{proCopy.activeTokens}</h3>
              <p className="mt-2 text-sm text-muted-foreground">{proCopy.authHint}</p>
            </div>
            <Dialog open={proCreateOpen} onOpenChange={onSetProCreateOpen}>
              <DialogTrigger asChild>
                <button
                  disabled={!canWritePro}
                  className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                >
                  <Plus className="h-4 w-4" />
                  {proCopy.createToken}
                </button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-xl">
                <DialogHeader>
                  <DialogTitle>{proCopy.createToken}</DialogTitle>
                  <DialogDescription>
                    {proCopy.oneTimeSecret}. {proCopy.createTokenHint}
                  </DialogDescription>
                </DialogHeader>
                <form className="space-y-4" onSubmit={onCreateProToken}>
                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.tokenName}</span>
                    <input
                      required
                      value={proNewTokenName}
                      onChange={(event) => onSetProNewTokenName(event.target.value)}
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                      placeholder={proCopy.tokenNamePlaceholder}
                    />
                  </label>
                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.tokenScopes}</span>
                    <div className="grid gap-2 sm:grid-cols-3">
                      {proScopePresets.map((preset) => {
                        const presetValue = preset.scopes.join(', ');
                        const isActive = proNewTokenScopes.trim() === presetValue;
                        const requiresAdmin = preset.scopes.includes('pro:admin');
                        const isDisabled = requiresAdmin && !canAdminPro;

                        return (
                          <button
                            key={preset.id}
                            type="button"
                            onClick={() => {
                              if (!isDisabled) {
                                onSetProNewTokenScopes(presetValue);
                              }
                            }}
                            disabled={isDisabled}
                            className={`rounded-xl border p-3 text-left transition-colors ${
                              isActive
                                ? 'border-electric-blue bg-electric-blue/8'
                                : 'border-border bg-background hover:bg-accent'
                            } ${isDisabled ? 'cursor-not-allowed opacity-55' : ''}`}
                          >
                            <div className="flex items-center justify-between gap-2">
                              <div className="font-medium">{preset.label}</div>
                              {requiresAdmin ? (
                                <span className="rounded-full border border-amber-500/30 bg-amber-500/10 px-2 py-0.5 text-[10px] font-medium text-amber-600">
                                  {proCopy.adminOnly}
                                </span>
                              ) : null}
                            </div>
                            <div className="mt-1 text-xs text-muted-foreground">{preset.description}</div>
                            <code className="mt-2 block overflow-x-auto text-[11px] text-electric-blue">
                              {presetValue}
                            </code>
                          </button>
                        );
                      })}
                    </div>
                    <input
                      required
                      value={proNewTokenScopes}
                      onChange={(event) => onSetProNewTokenScopes(event.target.value)}
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                      placeholder={proCopy.tokenScopesPlaceholder}
                    />
                  </label>
                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.expiresDays}</span>
                    <div className="grid gap-2 sm:grid-cols-3">
                      {expiryPresets.map((preset) => (
                        <button
                          key={`token-expiry-${preset.value}`}
                          type="button"
                          onClick={() => onSetProNewTokenExpiresDays(preset.value)}
                          className={`rounded-xl border px-3 py-2 text-sm transition-colors ${
                            proNewTokenExpiresDays.trim() === preset.value
                              ? 'border-electric-blue bg-electric-blue/8 text-electric-blue'
                              : 'border-border bg-background hover:bg-accent'
                          }`}
                        >
                          {preset.label}
                        </button>
                      ))}
                    </div>
                    <input
                      min="0"
                      type="number"
                      value={proNewTokenExpiresDays}
                      onChange={(event) => onSetProNewTokenExpiresDays(event.target.value)}
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                    />
                    <div className="text-xs text-muted-foreground">{proCopy.expiryHint}</div>
                  </label>
                  <button
                    type="submit"
                    disabled={proCreatingToken || !canWritePro}
                    className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    {proCreatingToken ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                    {proCopy.createToken}
                  </button>
                </form>
              </DialogContent>
            </Dialog>
          </div>

          {proCreatedTokenValue ? (
            <div className="mt-5 rounded-xl border border-electric-blue/30 bg-electric-blue/8 p-4">
              <div className="text-sm font-medium">{proCopy.oneTimeSecret}</div>
              <code className="mt-2 block overflow-x-auto rounded-lg bg-background px-3 py-3 text-xs text-electric-blue">
                {proCreatedTokenValue}
              </code>
              <button
                onClick={() => void onCopyToken(proCreatedTokenValue)}
                className="mt-3 inline-flex h-9 items-center justify-center gap-2 rounded-md border border-border px-3 text-sm font-medium transition-colors hover:bg-accent"
              >
                <Copy className="h-4 w-4" />
                {proCopy.copyToken}
              </button>
            </div>
          ) : null}

          <div className="mt-5 space-y-3">
            {proLoading ? (
              <div className="flex items-center gap-2 rounded-lg border border-border bg-background px-4 py-5 text-sm text-muted-foreground">
                <LoaderCircle className="h-4 w-4 animate-spin" />
                {proCopy.activeTokens}
              </div>
            ) : proTokens.length === 0 ? (
              <div className="rounded-lg border border-dashed border-border bg-background px-4 py-5 text-sm text-muted-foreground">
                {proCopy.noTokens}
              </div>
            ) : (
              proTokens.map((token) => (
                <div key={token.id} className="rounded-xl border border-border bg-background p-4">
                  <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div>
                      <div className="font-medium">{token.name}</div>
                      <div className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">
                        #{token.id}
                      </div>
                      <div className="mt-3 flex flex-wrap gap-2">
                        {token.scopes.map((scope) => (
                          <span
                            key={`${token.id}-${scope}`}
                            className="rounded-full border border-electric-blue/20 bg-electric-blue/8 px-2.5 py-1 text-[11px] font-medium text-electric-blue"
                          >
                            {scope}
                          </span>
                        ))}
                      </div>
                      <div className="mt-3 text-sm text-muted-foreground">
                        {token.expires_at ? `Expires: ${new Date(token.expires_at).toLocaleString()}` : 'No expiry'}
                      </div>
                      {token.last_used_at ? (
                        <div className="mt-1 text-sm text-muted-foreground">
                          Last used: {new Date(token.last_used_at).toLocaleString()}
                        </div>
                      ) : null}
                    </div>
                    <button
                      onClick={() => void onRevokeProToken(token.id)}
                      disabled={!!token.revoked_at || proRevokingTokenId === token.id || !canAdminPro}
                      className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-destructive/30 px-4 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      {proRevokingTokenId === token.id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                      {proCopy.revoke}
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </section>

        <aside className={`${activeSettingsTab === 'access' || activeSettingsTab === 'users' ? 'space-y-6' : 'hidden'} ${activeSettingsTab === 'users' ? 'xl:col-span-2' : ''}`}>
          <div className={`${activeSettingsTab === 'access' ? 'block' : 'hidden'} rounded-2xl border border-border bg-card p-6`}>
            <div className="text-sm text-muted-foreground">{proCopy.connectedAs}</div>
            <div className="mt-2 text-xl font-semibold">
              {proPrincipal?.name ?? proCopy.notConnected}
            </div>
            {proPrincipal ? (
              <>
                <div className="mt-4 grid gap-3 sm:grid-cols-2">
                  <div className="rounded-xl border border-border bg-background px-3 py-3">
                    <div className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground">
                      {proCopy.authMethod}
                    </div>
                    <div className="mt-1 text-sm font-medium">{proPrincipal.auth_method ?? 'unknown'}</div>
                  </div>
                  <div className="rounded-xl border border-border bg-background px-3 py-3">
                    <div className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground">
                      {proCopy.currentSession}
                    </div>
                    <div className="mt-1 text-sm font-medium">
                      {proPrincipal.session_id ? `#${proPrincipal.session_id}` : proPrincipal.is_bootstrap ? 'bootstrap' : 'agent token'}
                    </div>
                  </div>
                </div>
                <div className="mt-3 grid gap-3 sm:grid-cols-2">
                  <div className="rounded-xl border border-border bg-background px-3 py-3">
                    <div className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground">
                      {proCopy.userId}
                    </div>
                    <div className="mt-1 text-sm font-medium">
                      {proPrincipal.user_id ? `#${proPrincipal.user_id}` : 'n/a'}
                    </div>
                  </div>
                  <div className="rounded-xl border border-border bg-background px-3 py-3">
                    <div className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground">
                      {proCopy.rolesLabel}
                    </div>
                    <div className="mt-2 flex flex-wrap gap-2">
                      {proPrincipal.roles.map((role) => (
                        <span
                          key={`principal-role-${role}`}
                          className="rounded-full border border-border bg-card px-2.5 py-1 text-[11px] font-medium text-foreground"
                        >
                          {role}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>
                <div className="mt-4 text-sm text-muted-foreground">{proCopy.scopes}</div>
                <div className="mt-2 flex flex-wrap gap-2">
                  {proPrincipal.scopes.map((scope) => (
                    <span
                      key={`principal-${scope}`}
                      className="rounded-full border border-electric-blue/20 bg-electric-blue/8 px-2.5 py-1 text-[11px] font-medium text-electric-blue"
                    >
                      {scope}
                    </span>
                  ))}
                </div>
                {proPrincipal.session_id ? (
                  <button
                    onClick={() => void onRevokeCurrentProSession()}
                    disabled={proRevokingSessionId === proPrincipal.session_id}
                    className="mt-4 inline-flex h-10 items-center justify-center gap-2 rounded-md border border-destructive/30 px-4 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    {proRevokingSessionId === proPrincipal.session_id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                    {proCopy.revokeCurrentSession}
                  </button>
                ) : null}
              </>
            ) : null}
          </div>

          <div className={`${activeSettingsTab === 'users' ? 'block' : 'hidden'} rounded-2xl border border-border bg-card p-6`}>
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="text-lg font-semibold">{proCopy.users}</div>
                <div className="mt-1 text-sm text-muted-foreground">{proUsers.length}</div>
              </div>
              <Dialog open={proCreateUserOpen} onOpenChange={onSetProCreateUserOpen}>
                <DialogTrigger asChild>
                  <button
                    disabled={!canAdminPro}
                    className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    <Plus className="h-4 w-4" />
                    {proCopy.createUser}
                  </button>
                </DialogTrigger>
                <DialogContent className="sm:max-w-xl">
                  <DialogHeader>
                    <DialogTitle>{proCopy.createUser}</DialogTitle>
                  </DialogHeader>
                  <form className="space-y-4" onSubmit={onCreateProUser}>
                    <label className="block space-y-2">
                      <span className="text-sm text-muted-foreground">{proCopy.displayName}</span>
                      <input
                        required
                        value={proNewUserName}
                        onChange={(event) => onSetProNewUserName(event.target.value)}
                        className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                      />
                    </label>
                    <label className="block space-y-2">
                      <span className="text-sm text-muted-foreground">{proCopy.email}</span>
                      <input
                        value={proNewUserEmail}
                        onChange={(event) => onSetProNewUserEmail(event.target.value)}
                        className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                      />
                    </label>
                    <label className="block space-y-2">
                      <span className="text-sm text-muted-foreground">{proCopy.roleNames}</span>
                      <input
                        value={proNewUserRoles}
                        onChange={(event) => onSetProNewUserRoles(event.target.value)}
                        className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                        placeholder="reader"
                      />
                    </label>
                    <button
                      type="submit"
                      disabled={proCreatingUser || !canAdminPro}
                      className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      {proCreatingUser ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                      {proCopy.createUser}
                    </button>
                  </form>
                </DialogContent>
              </Dialog>
            </div>

            <div className="mt-4 space-y-3">
              {proUsers.length === 0 ? (
                <div className="rounded-lg border border-dashed border-border bg-background px-4 py-5 text-sm text-muted-foreground">
                  {proCopy.noUsers}
                </div>
              ) : (
                proUsers.map((user) => (
                  <div key={user.id} className="rounded-xl border border-border bg-background p-4">
                    <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                      <div>
                        <div className="font-medium">{user.display_name}</div>
                        <div className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">
                          #{user.id}{user.email ? ` · ${user.email}` : ''}{user.is_bootstrap ? ' · bootstrap' : ''}
                        </div>
                        {user.disabled_at ? (
                          <div className="mt-2 inline-flex rounded-full border border-amber-500/30 bg-amber-500/10 px-2.5 py-1 text-[11px] font-medium text-amber-600">
                            disabled
                          </div>
                        ) : null}
                        <div className="mt-3 flex flex-wrap gap-2">
                          {user.roles.map((role) => (
                            <span key={`${user.id}-role-${role}`} className="rounded-full border border-border bg-card px-2.5 py-1 text-[11px] font-medium text-foreground">
                              {role}
                            </span>
                          ))}
                        </div>
                      </div>
                      <div className="flex flex-wrap gap-2">
                        <button
                          onClick={() => onStartProPasswordChange(user)}
                          disabled={!canAdminPro || user.auth_provider !== 'local'}
                          className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                        >
                          <Shield className="h-4 w-4" />
                          {proCopy.changeUserPassword}
                        </button>
                        <button
                          onClick={() => onStartEditProUser(user)}
                          disabled={!canAdminPro || !!user.disabled_at}
                          className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-border px-4 text-sm font-medium transition-colors hover:bg-accent disabled:cursor-not-allowed disabled:opacity-70"
                        >
                          <Pencil className="h-4 w-4" />
                          {proCopy.editUserRoles}
                        </button>
                        <button
                          onClick={() => void onEnableProUser(user)}
                          disabled={!canAdminPro || !user.disabled_at || proEnablingUserId === user.id}
                          className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-emerald-500/30 px-4 text-sm font-medium text-emerald-600 transition-colors hover:bg-emerald-500/10 disabled:cursor-not-allowed disabled:opacity-70"
                        >
                          {proEnablingUserId === user.id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
                          {proCopy.enableUser}
                        </button>
                        <button
                          onClick={() => void onDisableProUser(user)}
                          disabled={!canAdminPro || !!user.disabled_at || user.is_bootstrap || proDisablingUserId === user.id}
                          className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-amber-500/30 px-4 text-sm font-medium text-amber-600 transition-colors hover:bg-amber-500/10 disabled:cursor-not-allowed disabled:opacity-70"
                        >
                          {proDisablingUserId === user.id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : null}
                          {proCopy.disableUser}
                        </button>
                        <button
                          onClick={() => void onDeleteProUser(user)}
                          disabled={!canAdminPro || user.is_bootstrap || proDeletingUserId === user.id}
                          className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-destructive/30 px-4 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-70"
                        >
                          {proDeletingUserId === user.id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                          {proCopy.deleteUser}
                        </button>
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        </aside>
      </div>

      <div className={`${activeSettingsTab === 'users' || activeSettingsTab === 'sessions' ? 'grid' : 'hidden'} gap-6 ${activeSettingsTab === 'users' ? 'xl:grid-cols-2' : 'xl:grid-cols-1'}`}>
        <section className={`${activeSettingsTab === 'users' ? 'block' : 'hidden'} rounded-2xl border border-border bg-card p-6`}>
          <div className="text-lg font-semibold">{proCopy.rolesTitle}</div>
          <div className="mt-4 space-y-3">
            {proRoles.map((role) => (
              <div key={role.id} className="rounded-xl border border-border bg-background p-4">
                <div className="font-medium">{role.display_name}</div>
                <div className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">{role.name}</div>
                <div className="mt-2 text-sm text-muted-foreground">{role.description}</div>
                <div className="mt-3 flex flex-wrap gap-2">
                  {role.scopes.map((scope) => (
                    <span key={`${role.id}-${scope}`} className="rounded-full border border-electric-blue/20 bg-electric-blue/8 px-2.5 py-1 text-[11px] font-medium text-electric-blue">
                      {scope}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </section>

        <section className={`${activeSettingsTab === 'sessions' ? 'block' : 'hidden'} rounded-2xl border border-border bg-card p-6`}>
          <div className="flex items-start justify-between gap-3">
            <div>
              <div className="text-lg font-semibold">{proCopy.sessionsTitle}</div>
              <div className="mt-1 text-sm text-muted-foreground">{proSessions.length}</div>
            </div>
            <div className="flex flex-col gap-2 sm:items-end">
              <label className="block">
                <span className="mb-1 block text-xs uppercase tracking-[0.18em] text-muted-foreground">
                  {proCopy.sessionsFilter}
                </span>
                <select
                  value={proSessionsFilterUserId}
                  onChange={(event) => onSetProSessionsFilterUserId(event.target.value)}
                  className="h-10 min-w-[200px] rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                >
                  <option value="all">{proCopy.allUsers}</option>
                  {proUsers.map((user) => (
                    <option key={`filter-user-${user.id}`} value={String(user.id)}>
                      {user.display_name} #{user.id}
                    </option>
                  ))}
                </select>
              </label>
              <Dialog open={proCreateSessionOpen} onOpenChange={onSetProCreateSessionOpen}>
                <DialogTrigger asChild>
                  <button
                    disabled={!canAdminPro || proUsers.length === 0}
                    className="inline-flex h-10 items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    <Plus className="h-4 w-4" />
                    {proCopy.createSession}
                  </button>
                </DialogTrigger>
                <DialogContent className="sm:max-w-xl">
                <DialogHeader>
                  <DialogTitle>{proCopy.createSession}</DialogTitle>
                </DialogHeader>
                <form className="space-y-4" onSubmit={onCreateProSession}>
                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.users}</span>
                    <select
                      required
                      value={proNewSessionUserId}
                      onChange={(event) => onSetProNewSessionUserId(event.target.value)}
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                    >
                      <option value="">Select user</option>
                      {proUsers.map((user) => (
                        <option key={`session-user-${user.id}`} value={String(user.id)}>
                          {user.display_name} #{user.id}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.sessionLabel}</span>
                    <input
                      value={proNewSessionLabel}
                      onChange={(event) => onSetProNewSessionLabel(event.target.value)}
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                    />
                  </label>
                  <label className="block space-y-2">
                    <span className="text-sm text-muted-foreground">{proCopy.expiresDays}</span>
                    <div className="grid gap-2 sm:grid-cols-3">
                      {expiryPresets.map((preset) => (
                        <button
                          key={`session-expiry-${preset.value}`}
                          type="button"
                          onClick={() => onSetProNewSessionExpiresDays(preset.value)}
                          className={`rounded-xl border px-3 py-2 text-sm transition-colors ${
                            proNewSessionExpiresDays.trim() === preset.value
                              ? 'border-electric-blue bg-electric-blue/8 text-electric-blue'
                              : 'border-border bg-background hover:bg-accent'
                          }`}
                        >
                          {preset.label}
                        </button>
                      ))}
                    </div>
                    <input
                      min="0"
                      type="number"
                      value={proNewSessionExpiresDays}
                      onChange={(event) => onSetProNewSessionExpiresDays(event.target.value)}
                      className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                    />
                    <div className="text-xs text-muted-foreground">{proCopy.expiryHint}</div>
                  </label>
                  <button
                    type="submit"
                    disabled={proCreatingSession || !canAdminPro}
                    className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
                  >
                    {proCreatingSession ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                    {proCopy.issueSession}
                  </button>
                </form>
                </DialogContent>
              </Dialog>
            </div>
          </div>

          {proCreatedSessionToken ? (
            <div className="mt-5 rounded-xl border border-electric-blue/30 bg-electric-blue/8 p-4">
              <div className="text-sm font-medium">{proCopy.createdSessionToken}</div>
              <code className="mt-2 block overflow-x-auto rounded-lg bg-background px-3 py-3 text-xs text-electric-blue">
                {proCreatedSessionToken}
              </code>
              <button
                onClick={() => void onCopyToken(proCreatedSessionToken)}
                className="mt-3 inline-flex h-9 items-center justify-center gap-2 rounded-md border border-border px-3 text-sm font-medium transition-colors hover:bg-accent"
              >
                <Copy className="h-4 w-4" />
                {proCopy.copyToken}
              </button>
            </div>
          ) : null}

          <div className="mt-4 space-y-3">
            {proSessions.filter((session) => proSessionsFilterUserId === 'all' || String(session.user_id) === proSessionsFilterUserId).length === 0 ? (
              <div className="rounded-lg border border-dashed border-border bg-background px-4 py-5 text-sm text-muted-foreground">
                {proCopy.noSessions}
              </div>
            ) : (
              proSessions
                .filter((session) => proSessionsFilterUserId === 'all' || String(session.user_id) === proSessionsFilterUserId)
                .map((session) => (
                <div
                  key={session.id}
                  className={`rounded-xl border bg-background p-4 ${
                    proPrincipal?.session_id === session.id
                      ? 'border-electric-blue/50 shadow-[0_0_0_1px_rgba(37,99,235,0.15)]'
                      : 'border-border'
                  }`}
                >
                  <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div>
                      <div className="font-medium">{session.user_name}</div>
                      <div className="mt-1 text-xs uppercase tracking-[0.18em] text-muted-foreground">
                        #{session.id} · user #{session.user_id} · {session.auth_method}
                      </div>
                      {proPrincipal?.session_id === session.id ? (
                        <div className="mt-2 inline-flex rounded-full border border-electric-blue/20 bg-electric-blue/8 px-2.5 py-1 text-[11px] font-medium text-electric-blue">
                          {proCopy.currentSession}
                        </div>
                      ) : null}
                      {session.label ? (
                        <div className="mt-2 text-sm text-muted-foreground">{session.label}</div>
                      ) : null}
                      <div className="mt-3 flex flex-wrap gap-2">
                        {session.roles.map((role) => (
                          <span key={`${session.id}-role-${role}`} className="rounded-full border border-border bg-card px-2.5 py-1 text-[11px] font-medium text-foreground">
                            {role}
                          </span>
                        ))}
                      </div>
                      <div className="mt-3 text-sm text-muted-foreground">
                        {session.expires_at ? `Expires: ${new Date(session.expires_at).toLocaleString()}` : 'No expiry'}
                      </div>
                      {session.last_used_at ? (
                        <div className="mt-1 text-sm text-muted-foreground">
                          Last used: {new Date(session.last_used_at).toLocaleString()}
                        </div>
                      ) : null}
                    </div>
                    <button
                      onClick={() => void onRevokeProSession(session.id)}
                      disabled={!!session.revoked_at || proRevokingSessionId === session.id || !canAdminPro}
                      className="inline-flex h-10 items-center justify-center gap-2 rounded-md border border-destructive/30 px-4 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-70"
                    >
                      {proRevokingSessionId === session.id ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Trash2 className="h-4 w-4" />}
                      {proCopy.revoke}
                    </button>
                  </div>
                </div>
              ))
            )}
          </div>
        </section>
      </div>
      </section>
      <Dialog open={proEditUserOpen} onOpenChange={onSetProEditUserOpen}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>{proCopy.editUserRoles}</DialogTitle>
            <DialogDescription>{proEditingUserName || `#${proEditingUserId}`}</DialogDescription>
          </DialogHeader>
          <form className="space-y-4" onSubmit={onUpdateProUserRoles}>
            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">{proCopy.roleNames}</span>
              <input
                required
                value={proEditingUserRoles}
                onChange={(event) => onSetProEditingUserRoles(event.target.value)}
                className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
                placeholder="reader, operator"
              />
            </label>
            <button
              type="submit"
              disabled={proUpdatingUserRoles || !canAdminPro}
              className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
            >
              {proUpdatingUserRoles ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Pencil className="h-4 w-4" />}
              {proCopy.editUserRoles}
            </button>
          </form>
        </DialogContent>
      </Dialog>
      <Dialog open={proPasswordUserOpen} onOpenChange={onSetProPasswordUserOpen}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>{proCopy.changeUserPassword}</DialogTitle>
            <DialogDescription>{proPasswordUserName || `#${proPasswordUserId}`}</DialogDescription>
          </DialogHeader>
          <form className="space-y-4" onSubmit={onUpdateProUserPassword}>
            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">{proCopy.newPassword}</span>
              <input
                required
                minLength={8}
                type="password"
                value={proPasswordUserPassword}
                onChange={(event) => onSetProPasswordUserPassword(event.target.value)}
                className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
              />
            </label>
            <label className="block space-y-2">
              <span className="text-sm text-muted-foreground">{proCopy.confirmPassword}</span>
              <input
                required
                minLength={8}
                type="password"
                value={proPasswordUserPasswordConfirm}
                onChange={(event) => onSetProPasswordUserPasswordConfirm(event.target.value)}
                className="h-10 w-full rounded-md border border-border bg-input-background px-3 text-sm outline-none transition-colors focus:border-electric-blue"
              />
            </label>
            <button
              type="submit"
              disabled={proUpdatingUserPassword || !canAdminPro}
              className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-md bg-electric-blue px-4 text-sm font-medium text-white transition-colors hover:bg-electric-blue/90 disabled:cursor-not-allowed disabled:opacity-70"
            >
              {proUpdatingUserPassword ? <LoaderCircle className="h-4 w-4 animate-spin" /> : <Shield className="h-4 w-4" />}
              {proCopy.changeUserPassword}
            </button>
          </form>
        </DialogContent>
      </Dialog>
    </>
  );
}
