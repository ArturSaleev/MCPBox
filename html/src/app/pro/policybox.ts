export type ProPolicyBoxConfig = {
  enabled: boolean;
  base_url: string;
  admin_token_configured: boolean;
  admin_token_preview?: string;
  validate_client_id: string;
  validate_client_secret_configured: boolean;
  validate_client_secret_preview?: string;
  validate_path: string;
  request_timeout_seconds: number;
  admin_configured: boolean;
  runtime_configured: boolean;
  configured: boolean;
};

export type ProPolicyBoxRuntimeUser = {
  id: string;
  name: string;
  role: string;
  specialization?: string;
  attributes?: Record<string, string>;
};

export type ProPolicyBoxRuntimeTicketPayload = {
  project_slug: string;
  requested_scopes?: string[];
  context?: Record<string, string>;
  metadata?: Record<string, string>;
  user: ProPolicyBoxRuntimeUser;
};

export type ProPolicyBoxRuntimeTicketResponse = {
  success: boolean;
  ticket: string;
  expires_at: string;
};

export type ProPolicyBoxRuntimeTokenResponse = {
  success: boolean;
  token: string;
  session: Record<string, unknown>;
  claims: Record<string, unknown>;
};

export type ProPolicyBoxRuntimeValidateResponse = {
  success: boolean;
  valid: boolean;
  claims: Record<string, unknown>;
  session: Record<string, unknown>;
};
