// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import {
  CSRF_COOKIE,
  CSRF_HEADER,
  readCookie,
  type Attachment,
} from "@cherrypi/sdk";
import { ref } from "vue";

/** A user or agent actor, as returned by the account endpoints. */
export interface AccountActor {
  id: number;
  kind: string;
  username: string;
  name: string;
  description: string;
  role?: string;
  status?: string;
  password_set: boolean;
  totp_enabled: boolean;
  must_change_password: boolean;
  is_admin: boolean;
  created_at: string;
  last_login_at?: string;
  /** The actor's uploaded picture, unset when it uses the jdenticon fallback. */
  picture?: Attachment;
}

/** One team a user belongs to, as returned by the directory profile. */
export interface DirectoryTeam {
  id: number;
  name: string;
  description: string;
  member_count: number;
  role: string;
  created_at: string;
  updated_at: string;
}

/** One project a user can read, as returned by the directory profile. */
export interface DirectoryProject {
  id: string;
  name: string;
  slug: string;
  description: string;
  created_at: string;
  archived_at?: string;
}

/** The public profile payload for /users/:username. */
export interface DirectoryProfile {
  user: AccountActor;
  teams?: DirectoryTeam[];
  projects?: DirectoryProject[];
  owned_agent_actor_ids?: string[];
}

export interface AccountEmail {
  id: number;
  email: string;
  confirmed: boolean;
  created_at: string;
}

export interface AccountToken {
  id: number;
  name: string;
  prefix: string;
  created_at: string;
  last_used_at?: string;
  expires_at?: string;
}

export interface AccountSession {
  id: number;
  method: string;
  user_agent: string;
  ip: string;
  remember: boolean;
  created_at: string;
  last_seen_at: string;
  expires_at: string;
}

export interface AccountPayload {
  actor: AccountActor;
  emails: AccountEmail[];
  tokens: AccountToken[];
  sessions: AccountSession[];
  recovery_codes_left: number;
}

export interface LoginResult {
  actor?: AccountActor;
  csrf_token?: string;
  totpRequired?: boolean;
  pending_token?: string;
}

/** A federated login method offered on the sign-in page. */
export interface LoginProvider {
  slug: string;
  name: string;
  kind: string;
}

/** An identity provider as returned by the admin surface. */
export interface IdentityProvider {
  id: number;
  kind: string;
  slug: string;
  name: string;
  enabled: boolean;
  issuer: string;
  authorization_endpoint: string;
  token_endpoint: string;
  userinfo_endpoint: string;
  jwks_uri: string;
  client_id: string;
  has_client_secret: boolean;
  scopes: string;
  subject_path: string;
  email_path: string;
  name_path: string;
  created_at: string;
  updated_at: string;
}

/** An LDAP provider as returned by the admin surface. */
export interface LDAPProvider {
  id: number;
  slug: string;
  name: string;
  enabled: boolean;
  host: string;
  port: number;
  encryption: string;
  insecure: boolean;
  base_dn: string;
  bind_dn: string;
  has_bind_password: boolean;
  user_filter: string;
  username_attr: string;
  email_attr: string;
  name_attr: string;
  created_at: string;
  updated_at: string;
}

/** A provider identity linked to the signed-in account. */
export interface ExternalIdentity {
  id: number;
  provider_id: number;
  provider_slug: string;
  provider_name: string;
  kind: string;
  subject: string;
  email: string;
  created_at: string;
}

/** An LDAP username linked to the signed-in account. */
export interface LDAPIdentity {
  id: number;
  provider_id: number;
  provider_slug: string;
  provider_name: string;
  username: string;
  created_at: string;
}

/** The org-wide per-method TOTP and sign-in toggles. */
export interface AuthSettings {
  totp_password: boolean;
  totp_magic_link: boolean;
  totp_oidc: boolean;
  totp_oauth2: boolean;
  totp_ldap: boolean;
  password_login: boolean;
  magic_link_login: boolean;
  email_confirmation: boolean;
  session_ttl_hours: number;
  remember_ttl_hours: number;
  members_can_invite_users: boolean;
  members_can_create_agents: boolean;
  members_can_create_teams: boolean;
  project_deletion_admin_only: boolean;
  team_deletion_admin_only: boolean;
}

/** Public sign-in capabilities, used by the login page. */
export interface AuthConfig {
  registration_open: boolean;
  magic_link: boolean;
  password_login: boolean;
  magic_link_login: boolean;
  email_confirmation: boolean;
}

/** One of the four invitation kinds. */
export type InviteKind = "username" | "email" | "oidc" | "ldap";

/** An invitation, as returned by the admin surface. */
export interface Invitation {
  id: number;
  kind: InviteKind;
  username: string;
  actor_id: number;
  email: string;
  provider_id: number;
  external_username: string;
  role: string;
  status: "pending" | "accepted" | "revoked" | "expired";
  invited_by_username: string;
  created_at: string;
  expires_at: string;
  accepted_at?: string;
  revoked_at?: string;
}

/** A user row, optionally carrying its pending invitation. */
export interface AdminUser extends AccountActor {
  invitation?: Invitation;
}

/** The outbound mail configuration (the password is write-only). */
export interface SMTPSettings {
  host: string;
  port: number;
  username: string;
  from: string;
  encryption: string;
  has_password: boolean;
}

export interface IdentitiesPayload {
  identities: ExternalIdentity[];
  ldap_identities: LDAPIdentity[];
  providers: LoginProvider[];
  ldap_providers: LoginProvider[];
}

/** AccountError carries the HTTP status so callers can branch on 401/409. */
export class AccountError extends Error {
  readonly status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "AccountError";
    this.status = status;
  }
}

const AUTH = "/api/rest/auth";
const ACCOUNT = "/api/rest/account";

export async function request<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<T> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (body !== undefined) headers["Content-Type"] = "application/json";
  const csrf = readCookie(CSRF_COOKIE);
  if (csrf) headers[CSRF_HEADER] = csrf;

  const res = await fetch(path, {
    method,
    headers,
    credentials: "same-origin",
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    throw new AccountError(data?.error ?? res.statusText, res.status);
  }
  return data as T;
}

/** The signed-in actor, or null while anonymous. */
export const currentAccount = ref<AccountActor | null>(null);

let sessionPromise: Promise<AccountActor | null> | undefined;

/**
 * Resolves the current session once per page load; a 401 caches as null. Call
 * `reloadSession()` after login/logout to refresh.
 */
export function ensureSession(): Promise<AccountActor | null> {
  sessionPromise ??= request<{ actor: AccountActor }>("GET", `${AUTH}/session`)
    .then((res) => {
      currentAccount.value = res.actor;
      return res.actor;
    })
    .catch(() => {
      currentAccount.value = null;
      return null;
    });
  return sessionPromise;
}

/** Forces the next `ensureSession` to re-check the server. */
export function resetSession(): void {
  sessionPromise = undefined;
  currentAccount.value = null;
}

export const accountApi = {
  async login(
    identifier: string,
    password: string,
    totp = "",
    remember = false,
  ): Promise<LoginResult> {
    return request<LoginResult>("POST", `${AUTH}/login`, {
      identifier,
      password,
      totp,
      remember,
    });
  },

  async logout(): Promise<void> {
    await request("POST", `${AUTH}/logout`, {});
    resetSession();
  },

  async requestMagicLink(identifier: string): Promise<{ magic_link?: string }> {
    return request("POST", `${AUTH}/magic-link`, { identifier });
  },

  async consumeMagicLink(token: string): Promise<LoginResult> {
    return request<LoginResult>("POST", `${AUTH}/magic-link/consume`, {
      token,
    });
  },

  async requestPasswordReset(
    identifier: string,
  ): Promise<{ reset_link?: string }> {
    return request("POST", `${AUTH}/password-reset`, { identifier });
  },

  async resetPassword(token: string, password: string): Promise<void> {
    await request("POST", `${AUTH}/password-reset/consume`, {
      token,
      password,
    });
  },

  async confirmEmail(token: string): Promise<void> {
    await request("POST", `${AUTH}/confirm-email`, { token });
  },

  async getAccount(): Promise<AccountPayload> {
    return request<AccountPayload>("GET", ACCOUNT);
  },

  async changePassword(
    currentPassword: string,
    newPassword: string,
  ): Promise<void> {
    await request("POST", `${ACCOUNT}/password`, {
      current_password: currentPassword,
      new_password: newPassword,
    });
    resetSession();
  },

  async updateProfile(
    name: string,
    description: string,
  ): Promise<AccountActor> {
    const res = await request<{ actor: AccountActor }>(
      "POST",
      `${ACCOUNT}/profile`,
      { name, description },
    );
    currentAccount.value = res.actor;
    return res.actor;
  },

  async addEmail(email: string): Promise<{ confirm_link?: string }> {
    return request("POST", `${ACCOUNT}/emails`, { email });
  },

  async deleteEmail(id: number): Promise<void> {
    await request("DELETE", `${ACCOUNT}/emails/${id}`);
  },

  async confirmEmailById(id: number): Promise<{ confirm_link?: string }> {
    return request("POST", `${ACCOUNT}/emails/${id}/confirm`, {});
  },

  async enrollTOTP(): Promise<{ secret: string; otpauth_url: string }> {
    return request("POST", `${ACCOUNT}/totp/enroll`, {});
  },

  async activateTOTP(code: string): Promise<{ recovery_codes: string[] }> {
    return request("POST", `${ACCOUNT}/totp/activate`, { code });
  },

  async disableTOTP(password: string): Promise<void> {
    await request("POST", `${ACCOUNT}/totp/disable`, { password });
  },

  async rotateRecoveryCodes(
    password: string,
  ): Promise<{ recovery_codes: string[] }> {
    return request("POST", `${ACCOUNT}/totp/recovery-codes`, { password });
  },

  async createToken(
    name: string,
    expiresInDays = 0,
  ): Promise<{ token: string; pat: AccountToken }> {
    return request("POST", `${ACCOUNT}/tokens`, {
      name,
      expires_in_days: expiresInDays,
    });
  },

  async revokeToken(id: number): Promise<void> {
    await request("DELETE", `${ACCOUNT}/tokens/${id}`);
  },

  async revokeSession(id: number): Promise<void> {
    await request("DELETE", `${ACCOUNT}/sessions/${id}`);
  },

  async revokeOtherSessions(): Promise<void> {
    await request("POST", `${ACCOUNT}/sessions/revoke-others`, {});
  },

  // --- Federation -----------------------------------------------------------

  async listLoginProviders(): Promise<{
    providers: LoginProvider[];
    ldap_providers: LoginProvider[];
  }> {
    return request("GET", `${AUTH}/providers`);
  },

  /** URL that starts an OIDC/OAuth2 login (or link) flow in the browser. */
  providerStartUrl(slug: string, mode = "login", redirect = "/tasks"): string {
    const params = new URLSearchParams({ mode, redirect });
    return `${AUTH}/providers/${slug}/start?${params.toString()}`;
  },

  async ldapLogin(
    provider: string,
    username: string,
    password: string,
    totp = "",
  ): Promise<LoginResult> {
    return request<LoginResult>("POST", `${AUTH}/ldap/login`, {
      provider,
      username,
      password,
      totp,
    });
  },

  async completeSecondFactor(
    pendingToken: string,
    totp: string,
  ): Promise<LoginResult> {
    const result = await request<LoginResult>("POST", `${AUTH}/2fa/complete`, {
      pending_token: pendingToken,
      totp,
    });
    resetSession();
    return result;
  },

  async getIdentities(): Promise<IdentitiesPayload> {
    return request<IdentitiesPayload>("GET", `${ACCOUNT}/identities`);
  },

  async unlinkIdentity(id: number): Promise<void> {
    await request("DELETE", `${ACCOUNT}/identities/${id}`);
  },

  async unlinkLDAPIdentity(id: number): Promise<void> {
    await request("DELETE", `${ACCOUNT}/ldap-identities/${id}`);
  },

  async linkLDAPIdentity(
    provider: string,
    username: string,
    password: string,
  ): Promise<{ identity: LDAPIdentity }> {
    return request("POST", `${ACCOUNT}/identities/ldap`, {
      provider,
      username,
      password,
    });
  },

  // --- Admin ----------------------------------------------------------------

  async listProviders(): Promise<{ providers: IdentityProvider[] }> {
    return request("GET", `${ACCOUNT}/providers`);
  },

  async createProvider(
    input: Record<string, unknown>,
  ): Promise<{ provider: IdentityProvider }> {
    return request("POST", `${ACCOUNT}/providers`, input);
  },

  async updateProvider(
    id: number,
    input: Record<string, unknown>,
  ): Promise<{ provider: IdentityProvider }> {
    return request("PUT", `${ACCOUNT}/providers/${id}`, input);
  },

  async deleteProvider(id: number): Promise<void> {
    await request("DELETE", `${ACCOUNT}/providers/${id}`);
  },

  async listLDAPProviders(): Promise<{ ldap_providers: LDAPProvider[] }> {
    return request("GET", `${ACCOUNT}/ldap-providers`);
  },

  async createLDAPProvider(
    input: Record<string, unknown>,
  ): Promise<{ ldap_provider: LDAPProvider }> {
    return request("POST", `${ACCOUNT}/ldap-providers`, input);
  },

  async updateLDAPProvider(
    id: number,
    input: Record<string, unknown>,
  ): Promise<{ ldap_provider: LDAPProvider }> {
    return request("PUT", `${ACCOUNT}/ldap-providers/${id}`, input);
  },

  async deleteLDAPProvider(id: number): Promise<void> {
    await request("DELETE", `${ACCOUNT}/ldap-providers/${id}`);
  },

  async getAuthSettings(): Promise<AuthSettings> {
    return request("GET", `${ACCOUNT}/auth-settings`);
  },

  async updateAuthSettings(settings: AuthSettings): Promise<AuthSettings> {
    return request("PUT", `${ACCOUNT}/auth-settings`, settings);
  },

  // --- User directory -------------------------------------------------------

  /** The agency's users, readable by any signed-in member. */
  async listDirectoryUsers(): Promise<{ users: AccountActor[] }> {
    return request("GET", `${ACCOUNT}/directory/users`);
  },

  /** One user's public profile plus their teams, readable by any member. */
  async getDirectoryUser(username: string): Promise<DirectoryProfile> {
    return request(
      "GET",
      `${ACCOUNT}/directory/users/${encodeURIComponent(username)}`,
    );
  },

  // --- Admin users ----------------------------------------------------------

  async listUsers(): Promise<{ users: AdminUser[] }> {
    return request("GET", `${ACCOUNT}/users`);
  },

  async createUser(
    username: string,
    password: string,
    role = "member",
  ): Promise<{ user: AdminUser }> {
    return request("POST", `${ACCOUNT}/users`, { username, password, role });
  },

  async updateUser(
    id: number,
    patch: { role?: string; status?: string; name?: string },
  ): Promise<{ user: AdminUser }> {
    return request("PATCH", `${ACCOUNT}/users/${id}`, patch);
  },

  async deleteUser(id: number): Promise<void> {
    await request("DELETE", `${ACCOUNT}/users/${id}`);
  },

  async setUserPassword(id: number, password: string): Promise<void> {
    await request("POST", `${ACCOUNT}/users/${id}/password`, { password });
  },

  // --- Invitations ----------------------------------------------------------

  async listInvitations(): Promise<{ invitations: Invitation[] }> {
    return request("GET", `${ACCOUNT}/invitations`);
  },

  async createInvitation(input: {
    kind: InviteKind;
    username: string;
    email?: string;
    provider_id?: number;
    external_username?: string;
    role?: string;
  }): Promise<{
    invitation: Invitation;
    password?: string;
    link?: string;
    delivered?: boolean;
    delivery_error?: string;
  }> {
    return request("POST", `${ACCOUNT}/invitations`, input);
  },

  async revokeInvitation(id: number): Promise<void> {
    await request("DELETE", `${ACCOUNT}/invitations/${id}`);
  },

  async resendInvitation(id: number): Promise<{
    invitation: Invitation;
    link?: string;
    delivered?: boolean;
    delivery_error?: string;
  }> {
    return request("POST", `${ACCOUNT}/invitations/${id}/resend`, {});
  },

  async previewInvitation(token: string): Promise<{
    username: string;
    email: string;
    kind: InviteKind;
    role: string;
    expires_at: string;
  }> {
    const params = new URLSearchParams({ token });
    return request("GET", `${AUTH}/invitations/preview?${params.toString()}`);
  },

  async acceptInvitation(
    token: string,
    password: string,
  ): Promise<LoginResult> {
    const result = await request<LoginResult>(
      "POST",
      `${AUTH}/invitations/accept`,
      { token, password },
    );
    resetSession();
    return result;
  },

  async getConfig(): Promise<AuthConfig> {
    return request<AuthConfig>("GET", `${AUTH}/config`);
  },

  // --- SMTP ----------------------------------------------------------------

  async getSMTPSettings(): Promise<SMTPSettings> {
    return request("GET", `${ACCOUNT}/smtp-settings`);
  },

  async updateSMTPSettings(
    settings: Partial<SMTPSettings> & { password?: string },
  ): Promise<SMTPSettings> {
    return request("PUT", `${ACCOUNT}/smtp-settings`, settings);
  },

  async testSMTP(to: string): Promise<void> {
    await request("POST", `${ACCOUNT}/smtp-settings/test`, { to });
  },

  /** The signed-in user's per-user settings. */
  async getPreferences(): Promise<UserPreferences> {
    return request<UserPreferences>("GET", `${ACCOUNT}/preferences`);
  },

  /** Persist one or more per-user settings. */
  async updatePreferences(
    preferences: Partial<UserPreferences>,
  ): Promise<UserPreferences> {
    return request<UserPreferences>(
      "POST",
      `${ACCOUNT}/preferences`,
      preferences,
    );
  },
};

/** Per-user preferences. */
export interface UserPreferences {
  /** Show the first-time archive notice (a session/workspace cleanup warning). */
  notify_first_archive: boolean;
}
