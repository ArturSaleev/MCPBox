import type { FormEvent } from 'react';

import { Copy, LoaderCircle, Pencil, Plus, Shield, Trash2 } from 'lucide-react';

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from './ui/dialog';

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
  disableUser: string;
  enableUser: string;
  deleteUser: string;
  createSession: string;
  displayName: string;
  email: string;
  roleNames: string;
  sessionLabel: string;
  issueSession: string;
  createdSessionToken: string;
  currentSession: string;
  revokeCurrentSession: string;
  authMethod: string;
  rolesLabel: string;
  userId: string;
  sessionId: string;
  allUsers: string;
  sessionsFilter: string;
  createUserFailed: string;
  updateUserRolesFailed: string;
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
  proDisablingUserId: number | null;
  proEnablingUserId: number | null;
  proDeletingUserId: number | null;
  proSessionsFilterUserId: string;
  proEditingUserId: string;
  proEditingUserName: string;
  proEditingUserRoles: string;
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
  onSetProSessionsFilterUserId: (value: string) => void;
  onSetProNewSessionUserId: (value: string) => void;
  onSetProNewSessionLabel: (value: string) => void;
  onSetProNewSessionExpiresDays: (value: string) => void;
  onLoginProLocal: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onStartProSSOSignIn: () => void;
  onCreateProToken: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onCreateProUser: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onStartEditProUser: (user: ProUserRecord) => void;
  onUpdateProUserRoles: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onDisableProUser: (user: ProUserRecord) => Promise<void>;
  onEnableProUser: (user: ProUserRecord) => Promise<void>;
  onDeleteProUser: (user: ProUserRecord) => Promise<void>;
  onCreateProSession: (event: FormEvent<HTMLFormElement>) => Promise<void>;
  onRevokeCurrentProSession: () => Promise<void>;
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
  proDisablingUserId,
  proEnablingUserId,
  proDeletingUserId,
  proSessionsFilterUserId,
  proEditingUserId,
  proEditingUserName,
  proEditingUserRoles,
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
  onSetProSessionsFilterUserId,
  onSetProNewSessionUserId,
  onSetProNewSessionLabel,
  onSetProNewSessionExpiresDays,
  onLoginProLocal,
  onStartProSSOSignIn,
  onCreateProToken,
  onCreateProUser,
  onStartEditProUser,
  onUpdateProUserRoles,
  onDisableProUser,
  onEnableProUser,
  onDeleteProUser,
  onCreateProSession,
  onRevokeCurrentProSession,
  onCopyToken,
  onRevokeProToken,
  onRevokeProSession,
}: ProAccessViewProps) {
  if (!proPrincipal) {
    return (
      <section className="flex min-h-screen items-center justify-center px-6 py-10">
        <div className="w-full max-w-md rounded-[28px] border border-border bg-card p-8 shadow-[0_20px_60px_rgba(15,23,42,0.08)]">
          <div className="flex flex-col items-center text-center">
            <img src={logoSrc} alt={editionName || proCopy.title} className="h-20 w-20 object-contain" />
            <p className="mt-4 text-xs uppercase tracking-[0.28em] text-electric-blue">MCPBox PRO v1</p>
            <h2 className="mt-3 text-3xl font-semibold">{proCopy.title}</h2>
            <p className="mt-2 max-w-sm text-sm text-muted-foreground">{proCopy.loginHint}</p>
          </div>

          <form className="mt-8 space-y-4" onSubmit={onLoginProLocal}>
            <div className="text-sm font-medium text-foreground">{proCopy.localLogin}</div>
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
  return (
    <>
      <section className="space-y-6">
        <div className="rounded-2xl border border-border bg-card p-6">
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <p className="text-sm uppercase tracking-[0.24em] text-electric-blue">
              {editionName || proCopy.title}
            </p>
            <h2 className="mt-2 text-3xl font-semibold">{proCopy.title}</h2>
            <p className="mt-2 max-w-3xl text-muted-foreground">{proCopy.subtitle}</p>
          </div>
          <div className="rounded-xl border border-border bg-background px-4 py-3">
            <div className="text-sm text-muted-foreground">{proCopy.me}</div>
            <div className="mt-1 text-lg font-semibold">
              {proPrincipal ? proPrincipal.name : proCopy.statusReady}
            </div>
          </div>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
        <section className="rounded-2xl border border-border bg-card p-6">
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

        <aside className="space-y-6">
          <div className="rounded-2xl border border-border bg-card p-6">
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

          <div className="rounded-2xl border border-border bg-card p-6">
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

      <div className="grid gap-6 xl:grid-cols-2">
        <section className="rounded-2xl border border-border bg-card p-6">
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

        <section className="rounded-2xl border border-border bg-card p-6">
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
    </>
  );
}
