<!--
CherryPi: A multi-agent-multi-human self-hosted agency
Copyright (C) 2026 CherryPi authors <cherrypi.dev>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
-->

<template lang="pug">
.flex.flex-col.gap-6
  Card
    CardHeader
      CardTitle Federated identity providers
      CardDescription
        | Generic OIDC discovery and the generic OAuth2 shim. Users link an
        | identity from their account after signing in; logins are invite-only.
      .flex.justify-end
        Button(size="sm" variant="outline" @click="startAddProvider")
          PlusIcon(data-icon="inline-start")
          span Add provider
    CardContent.flex.flex-col.gap-4
      .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
        v-for="provider in providers"
        :key="provider.id"
      )
        .min-w-0
          .flex.items-center.gap-2
            p.truncate.text-sm.font-medium {{ provider.name }}
            Badge(variant="secondary") {{ provider.kind }}
            Badge(v-if="!provider.enabled" variant="outline") disabled
          p.text-muted-foreground.truncate.text-xs
            | {{ provider.slug }} · {{ provider.issuer || provider.authorization_endpoint }}
        .flex.shrink-0.gap-2
          Button(
            variant="outline"
            size="sm"
            @click="startEditProvider(provider)"
          ) Edit
          Button(
            variant="destructive"
            size="sm"
            @click="removeProvider(provider)"
          ) Delete
      Empty(v-if="!providers.length")
        EmptyTitle No providers
        EmptyDescription Add an OIDC or OAuth2 provider to enable federated sign-in.

      form.flex.flex-col.gap-4.rounded-md.border.p-4(
        v-if="providerFormOpen"
        @submit.prevent="submitProvider"
      )
        p.text-sm.font-medium {{ editingProviderId ? "Edit provider" : "New provider" }}
        FieldGroup
          .grid.grid-cols-2.gap-3
            Field
              FieldLabel Kind
              Select(
                :model-value="providerForm.kind"
                @update:model-value="providerForm.kind = String($event)"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(value="oidc") OIDC (discovery)
                  SelectItem(value="oauth2") OAuth2 (shim)
            Field
              FieldLabel Slug
              Input(v-model="providerForm.slug" placeholder="dex")
            Field
              FieldLabel Name
              Input(v-model="providerForm.name" placeholder="Company SSO")
            Field
              FieldLabel Client ID
              Input(v-model="providerForm.client_id")
            Field
              FieldLabel(for="provider-secret") Client secret
              Input#provider-secret(
                v-model="providerForm.client_secret"
                type="password"
                :placeholder="editingProviderId ? 'Leave blank to keep' : ''"
              )
            Field
              FieldLabel Scopes
              Input(
                v-model="providerForm.scopes"
                placeholder="openid email profile"
              )
          template(v-if="providerForm.kind === 'oidc'")
            Field
              FieldLabel Issuer
              Input(
                v-model="providerForm.issuer"
                placeholder="https://id.example.com"
              )
            .grid.grid-cols-3.gap-3
              Field
                FieldLabel Authorization URL
                Input(
                  v-model="providerForm.authorization_endpoint"
                  placeholder="discovered"
                )
              Field
                FieldLabel Token URL
                Input(
                  v-model="providerForm.token_endpoint"
                  placeholder="discovered"
                )
              Field
                FieldLabel Userinfo URL
                Input(
                  v-model="providerForm.userinfo_endpoint"
                  placeholder="discovered"
                )
          template(v-else)
            Field
              FieldLabel Authorization URL
              Input(
                v-model="providerForm.authorization_endpoint"
                placeholder="https://id.example.com/authorize"
              )
            Field
              FieldLabel Token URL
              Input(
                v-model="providerForm.token_endpoint"
                placeholder="https://id.example.com/token"
              )
            Field
              FieldLabel Userinfo URL
              Input(
                v-model="providerForm.userinfo_endpoint"
                placeholder="https://id.example.com/userinfo"
              )
            .grid.grid-cols-3.gap-3
              Field
                FieldLabel Subject path
                Input(v-model="providerForm.subject_path" placeholder="sub")
              Field
                FieldLabel E-mail path
                Input(v-model="providerForm.email_path" placeholder="email")
              Field
                FieldLabel Name path
                Input(v-model="providerForm.name_path" placeholder="name")
          Field.flex-row.items-center.gap-2
            Switch(
              :model-value="providerForm.enabled"
              @update:model-value="providerForm.enabled = Boolean($event)"
            )
            FieldLabel Enabled
        p.text-destructive.text-sm(v-if="error") {{ error }}
        .flex.justify-end.gap-2
          Button(
            type="button"
            variant="outline"
            @click="providerFormOpen = false"
          ) Cancel
          Button(type="submit" :disabled="saving") Save provider

  Card
    CardHeader
      CardTitle LDAP directories
      CardDescription
        | Multiple providers; the sign-in page lists them. Users link a
        | directory username from their account.
      .flex.justify-end
        Button(size="sm" variant="outline" @click="startAddLDAP")
          PlusIcon(data-icon="inline-start")
          span Add directory
    CardContent.flex.flex-col.gap-4
      .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
        v-for="provider in ldapProviders"
        :key="provider.id"
      )
        .min-w-0
          .flex.items-center.gap-2
            p.truncate.text-sm.font-medium {{ provider.name }}
            Badge(variant="secondary") {{ provider.encryption }}
            Badge(v-if="!provider.enabled" variant="outline") disabled
          p.text-muted-foreground.truncate.text-xs
            | {{ provider.slug }} · {{ provider.host }}:{{ provider.port }} · {{ provider.base_dn }}
        .flex.shrink-0.gap-2
          Button(variant="outline" size="sm" @click="startEditLDAP(provider)") Edit
          Button(variant="destructive" size="sm" @click="removeLDAP(provider)") Delete
      Empty(v-if="!ldapProviders.length")
        EmptyTitle No directories
        EmptyDescription Add an LDAP directory to enable directory sign-in.

      form.flex.flex-col.gap-4.rounded-md.border.p-4(
        v-if="ldapFormOpen"
        @submit.prevent="submitLDAP"
      )
        p.text-sm.font-medium {{ editingLDAPId ? "Edit directory" : "New directory" }}
        FieldGroup
          .grid.grid-cols-3.gap-3
            Field
              FieldLabel Slug
              Input(v-model="ldapForm.slug" placeholder="glauth")
            Field
              FieldLabel Name
              Input(v-model="ldapForm.name" placeholder="Company directory")
            Field
              FieldLabel Encryption
              Select(
                :model-value="ldapForm.encryption"
                @update:model-value="ldapForm.encryption = String($event)"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(value="none") None
                  SelectItem(value="starttls") STARTTLS
                  SelectItem(value="ldaps") LDAPS
          .grid.grid-cols-3.gap-3
            Field
              FieldLabel Host
              Input(v-model="ldapForm.host" placeholder="ldap.example.com")
            Field
              FieldLabel Port
              Input(v-model.number="ldapForm.port" type="number")
            Field.flex-row.items-center.gap-2
              Switch(
                :model-value="ldapForm.insecure"
                @update:model-value="ldapForm.insecure = Boolean($event)"
              )
              FieldLabel Skip TLS verification
          Field
            FieldLabel Base DN
            Input(v-model="ldapForm.base_dn" placeholder="dc=example,dc=org")
          .grid.grid-cols-2.gap-3
            Field
              FieldLabel Bind DN / template
              Input(
                v-model="ldapForm.bind_dn"
                placeholder="uid={username},ou=people,dc=example,dc=org"
              )
            Field
              FieldLabel Bind password
              Input(
                v-model="ldapForm.bind_password"
                type="password"
                :placeholder="editingLDAPId ? 'Leave blank to keep' : ''"
              )
          .grid.grid-cols-2.gap-3
            Field
              FieldLabel User filter
              Input(
                v-model="ldapForm.user_filter"
                placeholder="(uid={username})"
              )
            Field
              FieldLabel Username attribute
              Input(v-model="ldapForm.username_attr" placeholder="uid")
          .grid.grid-cols-2.gap-3
            Field
              FieldLabel E-mail attribute
              Input(v-model="ldapForm.email_attr" placeholder="mail")
            Field
              FieldLabel Name attribute
              Input(v-model="ldapForm.name_attr" placeholder="cn")
          Field.flex-row.items-center.gap-2
            Switch(
              :model-value="ldapForm.enabled"
              @update:model-value="ldapForm.enabled = Boolean($event)"
            )
            FieldLabel Enabled
        p.text-destructive.text-sm(v-if="error") {{ error }}
        .flex.justify-end.gap-2
          Button(type="button" variant="outline" @click="ldapFormOpen = false") Cancel
          Button(type="submit" :disabled="saving") Save directory

  Card
    CardHeader
      CardTitle Two-factor enforcement
      CardDescription
        | Whether a method must present TOTP when the account has it enabled.
        | Turning a method off lets an upstream provider supply the second
        | factor.
    CardContent.flex.flex-col.gap-3
      .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
        v-for="toggle in totpToggles"
        :key="toggle.key"
      )
        .min-w-0
          p.text-sm.font-medium {{ toggle.label }}
          p.text-muted-foreground.text-xs {{ toggle.description }}
        Switch(
          :model-value="Boolean(totpSettings[toggle.key])"
          @update:model-value="totpSettings[toggle.key] = Boolean($event)"
        )
      .flex.justify-end
        Button(@click="saveSettings" :disabled="saving") Save

  Card
    CardHeader
      CardTitle Sign-in methods
      CardDescription
        | Org-wide toggles for the credential-based sign-in methods.
        | Administrators always keep the username/password escape hatch.
    CardContent.flex.flex-col.gap-3
      .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
        v-for="toggle in signInToggles"
        :key="toggle.key"
      )
        .min-w-0
          p.text-sm.font-medium {{ toggle.label }}
          p.text-muted-foreground.text-xs {{ toggle.description }}
        Switch(
          :model-value="Boolean(totpSettings[toggle.key])"
          @update:model-value="totpSettings[toggle.key] = Boolean($event)"
        )
      .flex.justify-end
        Button(@click="saveSettings" :disabled="saving") Save

  Card
    CardHeader
      CardTitle Membership & governance
      CardDescription
        | What plain members may do, and whether destructive object actions
        | are reserved for administrators.
    CardContent.flex.flex-col.gap-3
      .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
        v-for="toggle in governanceToggles"
        :key="toggle.key"
      )
        .min-w-0
          p.text-sm.font-medium {{ toggle.label }}
          p.text-muted-foreground.text-xs {{ toggle.description }}
        Switch(
          :model-value="Boolean(totpSettings[toggle.key])"
          @update:model-value="totpSettings[toggle.key] = Boolean($event)"
        )
      .flex.justify-end
        Button(@click="saveSettings" :disabled="saving") Save
</template>

<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Empty, EmptyDescription, EmptyTitle } from "@/components/ui/empty";
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import {
  accountApi,
  type AuthSettings,
  type IdentityProvider,
  type LDAPProvider,
} from "@/lib/account";
import { PlusIcon } from "@lucide/vue";
import { onMounted, reactive, ref } from "vue";
import { toast } from "vue3-toastify";

const providers = ref<IdentityProvider[]>([]);
const ldapProviders = ref<LDAPProvider[]>([]);
const saving = ref(false);
const error = ref("");

const providerFormOpen = ref(false);
const editingProviderId = ref<number | null>(null);
const providerForm = reactive(defaultProviderForm());

const ldapFormOpen = ref(false);
const editingLDAPId = ref<number | null>(null);
const ldapForm = reactive(defaultLDAPForm());

const totpSettings = reactive({
  totp_password: true,
  totp_magic_link: true,
  totp_oidc: true,
  totp_oauth2: true,
  totp_ldap: true,
  password_login: true,
  magic_link_login: true,
  email_confirmation: false,
  session_ttl_hours: 168,
  remember_ttl_hours: 720,
  members_can_invite_users: false,
  members_can_create_agents: false,
  members_can_create_teams: false,
  project_deletion_admin_only: true,
  team_deletion_admin_only: true,
});

const signInToggles = [
  {
    key: "password_login" as const,
    label: "Password sign-in",
    description:
      "Username and password login; disabling also disables username invites.",
  },
  {
    key: "magic_link_login" as const,
    label: "Magic-link sign-in",
    description: "Password-less sign-in via a confirmed e-mail address.",
  },
  {
    key: "email_confirmation" as const,
    label: "Require e-mail confirmation",
    description: "Users must confirm an address before they can sign in.",
  },
];

const governanceToggles = [
  {
    key: "members_can_invite_users" as const,
    label: "Members may invite users",
    description: "Plain members may send invitations from this page.",
  },
  {
    key: "members_can_create_agents" as const,
    label: "Members may create agents",
    description: "Plain members may register new agents (phase 4).",
  },
  {
    key: "members_can_create_teams" as const,
    label: "Members may create teams",
    description: "Plain members may create teams (phase 4).",
  },
  {
    key: "project_deletion_admin_only" as const,
    label: "Project deletion is admin-only",
    description: "Only administrators may delete a project (phase 5).",
  },
  {
    key: "team_deletion_admin_only" as const,
    label: "Team deletion is admin-only",
    description: "Only administrators may delete a team (phase 4).",
  },
];

const totpToggles = [
  {
    key: "totp_password" as const,
    label: "Password",
    description: "Username/password sign-in.",
  },
  {
    key: "totp_magic_link" as const,
    label: "Magic link",
    description: "Password-less e-mail sign-in.",
  },
  {
    key: "totp_oidc" as const,
    label: "OIDC",
    description: "OIDC discovery providers.",
  },
  {
    key: "totp_oauth2" as const,
    label: "OAuth2",
    description: "The generic OAuth2 shim.",
  },
  {
    key: "totp_ldap" as const,
    label: "LDAP",
    description: "Directory sign-in.",
  },
];

onMounted(load);

function defaultProviderForm() {
  return {
    kind: "oidc",
    slug: "",
    name: "",
    enabled: true,
    issuer: "",
    authorization_endpoint: "",
    token_endpoint: "",
    userinfo_endpoint: "",
    jwks_uri: "",
    client_id: "",
    client_secret: "",
    scopes: "",
    subject_path: "",
    email_path: "",
    name_path: "",
  };
}

function defaultLDAPForm() {
  return {
    slug: "",
    name: "",
    enabled: true,
    host: "",
    port: 389,
    encryption: "none",
    insecure: false,
    base_dn: "",
    bind_dn: "",
    bind_password: "",
    user_filter: "(uid={username})",
    username_attr: "uid",
    email_attr: "mail",
    name_attr: "cn",
  };
}

async function load(): Promise<void> {
  try {
    const [ids, ldaps, settings] = await Promise.all([
      accountApi.listProviders(),
      accountApi.listLDAPProviders(),
      accountApi.getAuthSettings(),
    ]);
    providers.value = ids.providers;
    ldapProviders.value = ldaps.ldap_providers;
    applySettings(settings);
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Failed to load";
  }
}

function applySettings(settings: AuthSettings): void {
  Object.assign(totpSettings, settings);
}

function startAddProvider(): void {
  error.value = "";
  editingProviderId.value = null;
  Object.assign(providerForm, defaultProviderForm());
  providerFormOpen.value = true;
}

function startEditProvider(provider: IdentityProvider): void {
  error.value = "";
  editingProviderId.value = provider.id;
  Object.assign(providerForm, {
    kind: provider.kind,
    slug: provider.slug,
    name: provider.name,
    enabled: provider.enabled,
    issuer: provider.issuer,
    authorization_endpoint: provider.authorization_endpoint,
    token_endpoint: provider.token_endpoint,
    userinfo_endpoint: provider.userinfo_endpoint,
    jwks_uri: provider.jwks_uri,
    client_id: provider.client_id,
    client_secret: "",
    scopes: provider.scopes,
    subject_path: provider.subject_path,
    email_path: provider.email_path,
    name_path: provider.name_path,
  });
  providerFormOpen.value = true;
}

async function submitProvider(): Promise<void> {
  saving.value = true;
  error.value = "";
  try {
    const payload = { ...providerForm };
    if (editingProviderId.value) {
      await accountApi.updateProvider(editingProviderId.value, payload);
    } else {
      await accountApi.createProvider(payload);
    }
    providerFormOpen.value = false;
    await load();
    toast.success("Provider saved");
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Save failed";
  } finally {
    saving.value = false;
  }
}

async function removeProvider(provider: IdentityProvider): Promise<void> {
  try {
    await accountApi.deleteProvider(provider.id);
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Delete failed");
  }
}

function startAddLDAP(): void {
  error.value = "";
  editingLDAPId.value = null;
  Object.assign(ldapForm, defaultLDAPForm());
  ldapFormOpen.value = true;
}

function startEditLDAP(provider: LDAPProvider): void {
  error.value = "";
  editingLDAPId.value = provider.id;
  Object.assign(ldapForm, {
    slug: provider.slug,
    name: provider.name,
    enabled: provider.enabled,
    host: provider.host,
    port: provider.port,
    encryption: provider.encryption,
    insecure: provider.insecure,
    base_dn: provider.base_dn,
    bind_dn: provider.bind_dn,
    bind_password: "",
    user_filter: provider.user_filter,
    username_attr: provider.username_attr,
    email_attr: provider.email_attr,
    name_attr: provider.name_attr,
  });
  ldapFormOpen.value = true;
}

async function submitLDAP(): Promise<void> {
  saving.value = true;
  error.value = "";
  try {
    const payload = { ...ldapForm };
    if (editingLDAPId.value) {
      await accountApi.updateLDAPProvider(editingLDAPId.value, payload);
    } else {
      await accountApi.createLDAPProvider(payload);
    }
    ldapFormOpen.value = false;
    await load();
    toast.success("Directory saved");
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Save failed";
  } finally {
    saving.value = false;
  }
}

async function removeLDAP(provider: LDAPProvider): Promise<void> {
  try {
    await accountApi.deleteLDAPProvider(provider.id);
    await load();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : "Delete failed");
  }
}

async function saveSettings(): Promise<void> {
  saving.value = true;
  error.value = "";
  try {
    const settings = await accountApi.updateAuthSettings({
      totp_password: totpSettings.totp_password,
      totp_magic_link: totpSettings.totp_magic_link,
      totp_oidc: totpSettings.totp_oidc,
      totp_oauth2: totpSettings.totp_oauth2,
      totp_ldap: totpSettings.totp_ldap,
      password_login: totpSettings.password_login,
      magic_link_login: totpSettings.magic_link_login,
      email_confirmation: totpSettings.email_confirmation,
      session_ttl_hours: totpSettings.session_ttl_hours,
      remember_ttl_hours: totpSettings.remember_ttl_hours,
      members_can_invite_users: totpSettings.members_can_invite_users,
      members_can_create_agents: totpSettings.members_can_create_agents,
      members_can_create_teams: totpSettings.members_can_create_teams,
      project_deletion_admin_only: totpSettings.project_deletion_admin_only,
      team_deletion_admin_only: totpSettings.team_deletion_admin_only,
    });
    applySettings(settings);
    toast.success("Authentication settings saved");
  } catch (err) {
    error.value = err instanceof Error ? err.message : "Save failed";
  } finally {
    saving.value = false;
  }
}
</script>
