export type ProPrincipal = {
  name: string;
  scopes: string[];
  roles: string[];
  user_id?: number;
  session_id?: number;
  auth_method?: string;
  is_bootstrap: boolean;
};

export type ProTokenRecord = {
  id: number;
  name: string;
  scopes: string[];
  expires_at?: string;
  last_used_at?: string;
  revoked_at?: string;
  created_at: string;
};

export type CreateProTokenResponse = {
  token: string;
  record: ProTokenRecord;
};

export type ProUserRecord = {
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

export type ProRoleRecord = {
  id: number;
  name: string;
  display_name: string;
  description: string;
  scopes: string[];
  is_system: boolean;
  created_at: string;
};

export type ProSessionRecord = {
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

export type CreateProUserResponse = ProUserRecord;
export type UpdateProUserResponse = ProUserRecord;
export type UpdateProUserPasswordResponse = ProUserRecord;
export type DisableProUserResponse = ProUserRecord;
export type EnableProUserResponse = ProUserRecord;

export type CreateProSessionResponse = {
  token: string;
  record: ProSessionRecord | null;
};

export type ProLocalLoginResponse = {
  token: string;
  user: ProUserRecord;
  session: ProSessionRecord | null;
};

export type ProSSOConfig = {
  enabled: boolean;
  provider_name?: string;
  issuer_url?: string;
  redirect_url?: string;
  start_url?: string;
  allowed_hosted_domain?: string;
  default_role?: string;
  session_days?: number;
  auto_create_users?: boolean;
  scopes?: string[];
};

export type ProScopePreset = {
  id: 'reader' | 'operator' | 'admin';
  label: string;
  scopes: string[];
  description: string;
};
