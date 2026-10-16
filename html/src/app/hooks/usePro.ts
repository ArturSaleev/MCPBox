import { FormEvent, useState } from 'react';
import { toast } from 'sonner';
import { apiRequest } from '../utils/api';
import { proAPIRequest, hasProScope } from '../pro/utils';
import { getProCopy } from '../pro/i18n';
import { proAuthStorageKey } from '../pro/constants';
import type {
  ProPrincipal,
  ProTokenRecord,
  ProUserRecord,
  ProRoleRecord,
  ProSessionRecord,
  CreateProTokenResponse,
  CreateProUserResponse,
  UpdateProUserResponse,
  DisableProUserResponse,
  EnableProUserResponse,
  CreateProSessionResponse,
  ProLocalLoginResponse,
  ProSSOConfig,
  ProScopePreset,
} from '../pro/types';
import type { Language } from '../i18n';

export function usePro(language: Language) {
  const [proAuthToken, setProAuthToken] = useState('');
  const [proPrincipal, setProPrincipal] = useState<ProPrincipal | null>(null);
  const [proTokens, setProTokens] = useState<ProTokenRecord[]>([]);
  const [proUsers, setProUsers] = useState<ProUserRecord[]>([]);
  const [proRoles, setProRoles] = useState<ProRoleRecord[]>([]);
  const [proSessions, setProSessions] = useState<ProSessionRecord[]>([]);
  const [proSSOConfig, setProSSOConfig] = useState<ProSSOConfig | null>(null);
  const [proLoading, setProLoading] = useState(false);
  const [proLocalLoginLoading, setProLocalLoginLoading] = useState(false);
  const [proAuthBootstrapping, setProAuthBootstrapping] = useState(false);
  const [proCreateOpen, setProCreateOpen] = useState(false);
  const [proCreatingToken, setProCreatingToken] = useState(false);
  const [proRevokingTokenId, setProRevokingTokenId] = useState<number | null>(null);
  const [proCreateUserOpen, setProCreateUserOpen] = useState(false);
  const [proCreatingUser, setProCreatingUser] = useState(false);
  const [proEditUserOpen, setProEditUserOpen] = useState(false);
  const [proUpdatingUserRoles, setProUpdatingUserRoles] = useState(false);
  const [proDisablingUserId, setProDisablingUserId] = useState<number | null>(null);
  const [proEnablingUserId, setProEnablingUserId] = useState<number | null>(null);
  const [proDeletingUserId, setProDeletingUserId] = useState<number | null>(null);
  const [proSessionsFilterUserId, setProSessionsFilterUserId] = useState('all');
  const [proEditingUserId, setProEditingUserId] = useState<string>('');
  const [proEditingUserName, setProEditingUserName] = useState('');
  const [proEditingUserRoles, setProEditingUserRoles] = useState('reader');
  const [proCreateSessionOpen, setProCreateSessionOpen] = useState(false);
  const [proCreatingSession, setProCreatingSession] = useState(false);
  const [proRevokingSessionId, setProRevokingSessionId] = useState<number | null>(null);
  const [proNewTokenName, setProNewTokenName] = useState('');
  const [proLoginEmail, setProLoginEmail] = useState('');
  const [proLoginPassword, setProLoginPassword] = useState('');
  const [proNewTokenScopes, setProNewTokenScopes] = useState('pro:read, pro:write');
  const [proNewTokenExpiresDays, setProNewTokenExpiresDays] = useState('30');
  const [proCreatedTokenValue, setProCreatedTokenValue] = useState<string | null>(null);
  const [proNewUserName, setProNewUserName] = useState('');
  const [proNewUserEmail, setProNewUserEmail] = useState('');
  const [proNewUserRoles, setProNewUserRoles] = useState('reader');
  const [proNewSessionUserId, setProNewSessionUserId] = useState('');
  const [proNewSessionLabel, setProNewSessionLabel] = useState('');
  const [proNewSessionExpiresDays, setProNewSessionExpiresDays] = useState('30');
  const [proCreatedSessionToken, setProCreatedSessionToken] = useState<string | null>(null);

  const proCopy = getProCopy(language);
  const proScopePresets: ProScopePreset[] =
    language === 'ru'
      ? [
          {
            id: 'reader',
            label: 'Reader',
            scopes: ['pro:read'],
            description: 'Только просмотр Pro API и token activity',
          },
          {
            id: 'operator',
            label: 'Operator',
            scopes: ['pro:read', 'pro:write'],
            description: 'Рабочие операции без полного admin-доступа',
          },
          {
            id: 'admin',
            label: 'Admin',
            scopes: ['pro:admin'],
            description: 'Полный доступ ко всем Pro операциям',
          },
        ]
      : [
          {
            id: 'reader',
            label: 'Reader',
            scopes: ['pro:read'],
            description: 'Read-only access to Pro API and token activity',
          },
          {
            id: 'operator',
            label: 'Operator',
            scopes: ['pro:read', 'pro:write'],
            description: 'Operational access without full admin rights',
          },
          {
            id: 'admin',
            label: 'Admin',
            scopes: ['pro:admin'],
            description: 'Full access to all Pro operations',
          },
        ];

  function resetProAccess(clearStorage = true) {
    setProPrincipal(null);
    setProTokens([]);
    setProUsers([]);
    setProRoles([]);
    setProSessions([]);
    setProSSOConfig(null);
    if (clearStorage && typeof window !== 'undefined') {
      window.localStorage.removeItem(proAuthStorageKey);
    }
  }

  async function loadProSession(tokenOverride?: string) {
    const activeToken = (tokenOverride ?? proAuthToken).trim();
    if (!activeToken) {
      setProPrincipal(null);
      return;
    }

    try {
      setProLoading(true);
      const [principal, tokens, users, roles, sessions, ssoConfig] = await Promise.all([
        proAPIRequest<ProPrincipal>(activeToken, '/api/pro/auth/me', (status) => `Failed to verify Pro token (${status})`),
        proAPIRequest<ProTokenRecord[]>(activeToken, '/api/pro/tokens', (status) => `Failed to load tokens (${status})`),
        proAPIRequest<ProUserRecord[]>(activeToken, '/api/pro/users', (status) => `Failed to load users (${status})`),
        proAPIRequest<ProRoleRecord[]>(activeToken, '/api/pro/roles', (status) => `Failed to load roles (${status})`),
        proAPIRequest<ProSessionRecord[]>(activeToken, '/api/pro/sessions', (status) => `Failed to load sessions (${status})`),
        proAPIRequest<ProSSOConfig>(activeToken, '/api/pro/sso/config', (status) => `Failed to load SSO config (${status})`),
      ]);
      setProPrincipal(principal);
      setProTokens(tokens);
      setProUsers(users);
      setProRoles(roles);
      setProSessions(sessions);
      setProSSOConfig(ssoConfig);
      if (typeof window !== 'undefined') {
        window.localStorage.setItem(proAuthStorageKey, activeToken);
      }
    } catch (loadError) {
      resetProAccess(false);
      if (!tokenOverride) {
        toast.error(proCopy.verifyFailed);
      }
    } finally {
      setProLoading(false);
    }
  }

  async function connectProToken() {
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }
    await loadProSession();
  }

  async function loginProLocal(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    setProLocalLoginLoading(true);
    try {
      const response = await apiRequest<ProLocalLoginResponse>('/api/pro/login', (status) => `Login failed (${status})`, {
        method: 'POST',
        body: JSON.stringify({
          email: proLoginEmail,
          password: proLoginPassword,
        }),
      });
      setProAuthToken(response.token);
      await loadProSession(response.token);
    } catch (loginError) {
      toast.error(proCopy.verifyFailed);
    } finally {
      setProLocalLoginLoading(false);
    }
  }

  async function startProSSO() {
    if (!proSSOConfig?.start_url) {
      return;
    }
    window.location.assign(proSSOConfig.start_url);
  }

  async function createProToken(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProCreatingToken(true);
    try {
      const response = await proAPIRequest<CreateProTokenResponse>(
        proAuthToken,
        '/api/pro/tokens',
        (status) => `Failed to create token (${status})`,
        {
          method: 'POST',
          body: JSON.stringify({
            name: proNewTokenName,
            scopes: proNewTokenScopes.split(',').map((s) => s.trim()),
            expires_days: proNewTokenExpiresDays === '0' ? null : parseInt(proNewTokenExpiresDays, 10) || null,
          }),
        },
      );
      setProCreatedTokenValue(response.token);
      await loadProSession();
      setProCreateOpen(false);
      setProNewTokenName('');
      setProNewTokenScopes('pro:read, pro:write');
      setProNewTokenExpiresDays('30');
    } catch (createError) {
      toast.error(proCopy.createFailed);
    } finally {
      setProCreatingToken(false);
    }
  }

  async function revokeProToken(tokenId: number) {
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProRevokingTokenId(tokenId);
    try {
      await proAPIRequest<void>(proAuthToken, `/api/pro/tokens/${tokenId}`, (status) => `Failed to revoke token (${status})`, {
        method: 'DELETE',
      });
      await loadProSession();
    } catch (revokeError) {
      toast.error(proCopy.revokeFailed);
    } finally {
      setProRevokingTokenId(null);
    }
  }

  async function createProUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProCreatingUser(true);
    try {
      await proAPIRequest<CreateProUserResponse>(
        proAuthToken,
        '/api/pro/users',
        (status) => `Failed to create user (${status})`,
        {
          method: 'POST',
          body: JSON.stringify({
            email: proNewUserEmail,
            display_name: proNewUserName,
            roles: [proNewUserRoles],
          }),
        },
      );
      await loadProSession();
      setProCreateUserOpen(false);
      setProNewUserName('');
      setProNewUserEmail('');
      setProNewUserRoles('reader');
    } catch (createError) {
      toast.error(proCopy.createUserFailed);
    } finally {
      setProCreatingUser(false);
    }
  }

  function openEditUser(user: ProUserRecord) {
    setProEditingUserId(String(user.id));
    setProEditingUserName(user.display_name);
    setProEditingUserRoles(user.roles[0] || 'reader');
    setProEditUserOpen(true);
  }

  async function updateProUserRoles(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProUpdatingUserRoles(true);
    try {
      await proAPIRequest<UpdateProUserResponse>(
        proAuthToken,
        `/api/pro/users/${proEditingUserId}`,
        (status) => `Failed to update user roles (${status})`,
        {
          method: 'PUT',
          body: JSON.stringify({
            roles: [proEditingUserRoles],
          }),
        },
      );
      await loadProSession();
      setProEditUserOpen(false);
      setProEditingUserId('');
      setProEditingUserName('');
      setProEditingUserRoles('reader');
    } catch (updateError) {
      toast.error(proCopy.updateUserRolesFailed);
    } finally {
      setProUpdatingUserRoles(false);
    }
  }

  async function disableProUser(user: ProUserRecord) {
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    if (!window.confirm(proCopy.disableUserConfirm)) {
      return;
    }

    setProDisablingUserId(user.id);
    try {
      await proAPIRequest<DisableProUserResponse>(
        proAuthToken,
        `/api/pro/users/${user.id}/disable`,
        (status) => `Failed to disable user (${status})`,
        {
          method: 'POST',
        },
      );
      await loadProSession();
    } catch (disableError) {
      toast.error(proCopy.disableUserFailed);
    } finally {
      setProDisablingUserId(null);
    }
  }

  async function enableProUser(user: ProUserRecord) {
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    if (!window.confirm(proCopy.enableUserConfirm)) {
      return;
    }

    setProEnablingUserId(user.id);
    try {
      await proAPIRequest<EnableProUserResponse>(
        proAuthToken,
        `/api/pro/users/${user.id}/enable`,
        (status) => `Failed to enable user (${status})`,
        {
          method: 'POST',
        },
      );
      await loadProSession();
    } catch (enableError) {
      toast.error(proCopy.enableUserFailed);
    } finally {
      setProEnablingUserId(null);
    }
  }

  async function deleteProUser(user: ProUserRecord) {
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    if (!window.confirm(proCopy.deleteUserConfirm)) {
      return;
    }

    setProDeletingUserId(user.id);
    try {
      await apiRequest<void>(`/api/pro/users/${user.id}`, (status) => `Failed to delete user (${status})`, {
        method: 'DELETE',
      });
      await loadProSession();
    } catch (deleteError) {
      toast.error(proCopy.deleteUserFailed);
    } finally {
      setProDeletingUserId(null);
    }
  }

  async function createProSession(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProCreatingSession(true);
    try {
      const response = await proAPIRequest<CreateProSessionResponse>(
        proAuthToken,
        '/api/pro/sessions',
        (status) => `Failed to create session (${status})`,
        {
          method: 'POST',
          body: JSON.stringify({
            user_id: proNewSessionUserId ? parseInt(proNewSessionUserId, 10) : null,
            label: proNewSessionLabel,
            expires_days: proNewSessionExpiresDays === '0' ? null : parseInt(proNewSessionExpiresDays, 10) || null,
          }),
        },
      );
      setProCreatedSessionToken(response.token);
      await loadProSession();
      setProCreateSessionOpen(false);
      setProNewSessionUserId('');
      setProNewSessionLabel('');
      setProNewSessionExpiresDays('30');
    } catch (createError) {
      toast.error(proCopy.createSessionFailed);
    } finally {
      setProCreatingSession(false);
    }
  }

  async function revokeProSession(sessionId: number) {
    if (!proAuthToken.trim()) {
      toast.error(proCopy.verifyFailed);
      return;
    }

    setProRevokingSessionId(sessionId);
    try {
      await proAPIRequest<void>(proAuthToken, `/api/pro/sessions/${sessionId}`, (status) => `Failed to revoke session (${status})`, {
        method: 'DELETE',
      });
      await loadProSession();
    } catch (revokeError) {
      toast.error(proCopy.revokeSessionFailed);
    } finally {
      setProRevokingSessionId(null);
    }
  }

  async function revokeCurrentProSession() {
    if (!proPrincipal?.session_id) {
      return;
    }
    await revokeProSession(proPrincipal.session_id);
  }

  async function signOutPro() {
    if (proPrincipal?.session_id) {
      await revokeCurrentProSession();
      return;
    }
    resetProAccess(true);
  }

  function updateAndPersistProAuthToken(value: string) {
    setProAuthToken(value);
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(proAuthStorageKey, value);
    }
  }

  return {
    // State
    proAuthToken,
    setProAuthToken,
    proPrincipal,
    proTokens,
    proUsers,
    proRoles,
    proSessions,
    proSSOConfig,
    proLoading,
    proLocalLoginLoading,
    proAuthBootstrapping,
    proCreateOpen,
    setProCreateOpen,
    proCreatingToken,
    proRevokingTokenId,
    proCreateUserOpen,
    setProCreateUserOpen,
    proCreatingUser,
    proEditUserOpen,
    setProEditUserOpen,
    proUpdatingUserRoles,
    proDisablingUserId,
    proEnablingUserId,
    proDeletingUserId,
    proSessionsFilterUserId,
    setProSessionsFilterUserId,
    proEditingUserId,
    proEditingUserName,
    proEditingUserRoles,
    setProEditingUserRoles,
    proCreateSessionOpen,
    setProCreateSessionOpen,
    proCreatingSession,
    proRevokingSessionId,
    proNewTokenName,
    setProNewTokenName,
    proLoginEmail,
    setProLoginEmail,
    proLoginPassword,
    setProLoginPassword,
    proNewTokenScopes,
    setProNewTokenScopes,
    proNewTokenExpiresDays,
    setProNewTokenExpiresDays,
    proCreatedTokenValue,
    setProCreatedTokenValue,
    proNewUserName,
    setProNewUserName,
    proNewUserEmail,
    setProNewUserEmail,
    proNewUserRoles,
    setProNewUserRoles,
    proNewSessionUserId,
    setProNewSessionUserId,
    proNewSessionLabel,
    setProNewSessionLabel,
    proNewSessionExpiresDays,
    setProNewSessionExpiresDays,
    proCreatedSessionToken,
    setProCreatedSessionToken,
    proCopy,
    proScopePresets,
    // Functions
    resetProAccess,
    loadProSession,
    connectProToken,
    loginProLocal,
    startProSSO,
    createProToken,
    revokeProToken,
    createProUser,
    openEditUser,
    updateProUserRoles,
    disableProUser,
    enableProUser,
    deleteProUser,
    createProSession,
    revokeProSession,
    revokeCurrentProSession,
    signOutPro,
    updateAndPersistProAuthToken,
  };
}
