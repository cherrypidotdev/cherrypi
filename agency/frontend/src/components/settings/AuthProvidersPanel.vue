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
      CardTitle {{ $t("settings.auth.title") }}
      CardDescription
        | {{ $t("settings.auth.description") }}
      .flex.justify-end
        Button(size="sm" variant="outline" @click="startAddProvider")
          PlusIcon(data-icon="inline-start")
          span {{ $t("settings.auth.addProvider") }}
    CardContent.flex.flex-col.gap-4
      .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
        v-for="provider in providers"
        :key="provider.id"
      )
        .min-w-0
          .flex.items-center.gap-2
            p.truncate.text-sm.font-medium {{ provider.name }}
            Badge(variant="secondary") {{ provider.kind }}
            Badge(v-if="provider.auto_provision" variant="outline") {{ $t("settings.auth.autoProvision") }}
            Badge(v-if="!provider.enabled" variant="outline") {{ $t("settings.auth.disabled") }}
          p.text-muted-foreground.truncate.text-xs
            | {{ provider.slug }} · {{ provider.issuer || provider.authorization_endpoint }}
        .flex.shrink-0.gap-2
          Button(
            variant="outline"
            size="sm"
            @click="startEditProvider(provider)"
          ) {{ $t("common.edit") }}
          Button(
            variant="destructive"
            size="sm"
            @click="removeProvider(provider)"
          ) {{ $t("common.delete") }}
      Empty(v-if="!providers.length")
        EmptyTitle {{ $t("settings.auth.noProviders") }}
        EmptyDescription {{ $t("settings.auth.noProvidersHint") }}

      form.flex.flex-col.gap-4.rounded-md.border.p-4(
        v-if="providerFormOpen"
        @submit.prevent="submitProvider"
      )
        p.text-sm.font-medium {{ editingProviderId ? "Edit provider" : "New provider" }}
        FieldGroup
          .grid.grid-cols-2.gap-3
            Field
              FieldLabel {{ $t("common.kind") }}
              Select(
                :model-value="providerForm.kind"
                @update:model-value="providerForm.kind = String($event)"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(value="oidc") {{ $t("settings.auth.kindOidc") }}
                  SelectItem(value="oauth2") {{ $t("settings.auth.kindOauth2") }}
            Field
              FieldLabel {{ $t("settings.auth.slug") }}
              Input(
                v-model="providerForm.slug"
                :placeholder="$t('settings.auth.slugPlaceholder')"
              )
            Field
              FieldLabel {{ $t("common.name") }}
              Input(
                v-model="providerForm.name"
                :placeholder="$t('settings.auth.namePlaceholder')"
              )
            Field
              FieldLabel {{ $t("settings.auth.clientId") }}
              Input(v-model="providerForm.client_id")
            Field
              FieldLabel(for="provider-secret") {{ $t("settings.auth.clientSecret") }}
              Input#provider-secret(
                v-model="providerForm.client_secret"
                type="password"
                :placeholder="editingProviderId ? $t('settings.auth.keepBlank') : ''"
              )
            Field
              FieldLabel {{ $t("settings.auth.scopes") }}
              Input(
                v-model="providerForm.scopes"
                :placeholder="$t('settings.auth.scopesPlaceholder')"
              )
          template(v-if="providerForm.kind === 'oidc'")
            Field
              FieldLabel {{ $t("settings.auth.issuer") }}
              Input(
                v-model="providerForm.issuer"
                :placeholder="$t('settings.auth.issuerPlaceholder')"
              )
            .grid.grid-cols-3.gap-3
              Field
                FieldLabel {{ $t("settings.auth.authorizationUrl") }}
                Input(
                  v-model="providerForm.authorization_endpoint"
                  :placeholder="$t('settings.auth.discovered')"
                )
              Field
                FieldLabel {{ $t("settings.auth.tokenUrl") }}
                Input(
                  v-model="providerForm.token_endpoint"
                  :placeholder="$t('settings.auth.discovered')"
                )
              Field
                FieldLabel {{ $t("settings.auth.userinfoUrl") }}
                Input(
                  v-model="providerForm.userinfo_endpoint"
                  :placeholder="$t('settings.auth.discovered')"
                )
          template(v-else)
            Field
              FieldLabel {{ $t("settings.auth.authorizationUrl") }}
              Input(
                v-model="providerForm.authorization_endpoint"
                :placeholder="$t('settings.auth.authorizationPlaceholder')"
              )
            Field
              FieldLabel {{ $t("settings.auth.tokenUrl") }}
              Input(
                v-model="providerForm.token_endpoint"
                :placeholder="$t('settings.auth.tokenPlaceholder')"
              )
            Field
              FieldLabel {{ $t("settings.auth.userinfoUrl") }}
              Input(
                v-model="providerForm.userinfo_endpoint"
                :placeholder="$t('settings.auth.userinfoPlaceholder')"
              )
            .grid.grid-cols-3.gap-3
              Field
                FieldLabel {{ $t("settings.auth.subjectPath") }}
                Input(
                  v-model="providerForm.subject_path"
                  :placeholder="$t('settings.auth.subjectPlaceholder')"
                )
              Field
                FieldLabel {{ $t("settings.auth.emailPath") }}
                Input(
                  v-model="providerForm.email_path"
                  :placeholder="$t('settings.auth.emailPlaceholder')"
                )
              Field
                FieldLabel {{ $t("settings.auth.namePath") }}
                Input(
                  v-model="providerForm.name_path"
                  :placeholder="$t('settings.auth.namePlaceholderPath')"
                )
          .flex.flex-col.gap-1
            Field.flex-row.items-center.gap-2
              Switch(
                :model-value="providerForm.enabled"
                @update:model-value="providerForm.enabled = Boolean($event)"
              )
              FieldLabel {{ $t("common.enabled") }}
            Field.flex-row.items-center.gap-2
              Switch(
                :model-value="providerForm.auto_provision"
                @update:model-value="providerForm.auto_provision = Boolean($event)"
              )
              FieldLabel {{ $t("settings.auth.createUsers") }}
            p.text-muted-foreground.text-xs
              | {{ $t("settings.auth.createUsersHint") }}
        p.text-destructive.text-sm(v-if="error") {{ error }}
        .flex.justify-end.gap-2
          Button(
            type="button"
            variant="outline"
            @click="providerFormOpen = false"
          ) {{ $t("common.cancel") }}
          Button(type="submit" :disabled="saving") {{ $t("settings.auth.saveProvider") }}

  Card
    CardHeader
      CardTitle {{ $t("settings.auth.ldapTitle") }}
      CardDescription
        | {{ $t("settings.auth.ldapDescription") }}
      .flex.justify-end
        Button(size="sm" variant="outline" @click="startAddLDAP")
          PlusIcon(data-icon="inline-start")
          span {{ $t("settings.auth.addDirectory") }}
    CardContent.flex.flex-col.gap-4
      .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
        v-for="provider in ldapProviders"
        :key="provider.id"
      )
        .min-w-0
          .flex.items-center.gap-2
            p.truncate.text-sm.font-medium {{ provider.name }}
            Badge(variant="secondary") {{ provider.encryption }}
            Badge(v-if="provider.auto_provision" variant="outline") {{ $t("settings.auth.autoProvision") }}
            Badge(v-if="!provider.enabled" variant="outline") {{ $t("settings.auth.disabled") }}
          p.text-muted-foreground.truncate.text-xs
            | {{ provider.slug }} · {{ provider.host }}:{{ provider.port }} · {{ provider.base_dn }}
        .flex.shrink-0.gap-2
          Button(variant="outline" size="sm" @click="startEditLDAP(provider)") {{ $t("common.edit") }}
          Button(variant="destructive" size="sm" @click="removeLDAP(provider)") {{ $t("common.delete") }}
      Empty(v-if="!ldapProviders.length")
        EmptyTitle {{ $t("settings.auth.noDirectories") }}
        EmptyDescription {{ $t("settings.auth.noDirectoriesHint") }}

      form.flex.flex-col.gap-4.rounded-md.border.p-4(
        v-if="ldapFormOpen"
        @submit.prevent="submitLDAP"
      )
        p.text-sm.font-medium {{ editingLDAPId ? "Edit directory" : "New directory" }}
        FieldGroup
          .grid.grid-cols-3.gap-3
            Field
              FieldLabel {{ $t("settings.auth.slug") }}
              Input(
                v-model="ldapForm.slug"
                :placeholder="$t('settings.auth.ldapSlugPlaceholder')"
              )
            Field
              FieldLabel {{ $t("common.name") }}
              Input(
                v-model="ldapForm.name"
                :placeholder="$t('settings.auth.ldapNamePlaceholder')"
              )
            Field
              FieldLabel {{ $t("settings.auth.encryption") }}
              Select(
                :model-value="ldapForm.encryption"
                @update:model-value="ldapForm.encryption = String($event)"
              )
                SelectTrigger
                  SelectValue
                SelectContent
                  SelectItem(value="none") {{ $t("common.none") }}
                  SelectItem(value="starttls") {{ $t("settings.auth.starttls") }}
                  SelectItem(value="ldaps") {{ $t("settings.auth.ldaps") }}
          .grid.grid-cols-3.gap-3
            Field
              FieldLabel {{ $t("settings.auth.host") }}
              Input(
                v-model="ldapForm.host"
                :placeholder="$t('settings.auth.hostPlaceholder')"
              )
            Field
              FieldLabel {{ $t("settings.auth.port") }}
              Input(v-model.number="ldapForm.port" type="number")
            Field.flex-row.items-center.gap-2
              Switch(
                :model-value="ldapForm.insecure"
                @update:model-value="ldapForm.insecure = Boolean($event)"
              )
              FieldLabel {{ $t("settings.auth.skipTls") }}
          Field
            FieldLabel {{ $t("settings.auth.baseDn") }}
            Input(
              v-model="ldapForm.base_dn"
              :placeholder="$t('settings.auth.baseDnPlaceholder')"
            )
          .grid.grid-cols-2.gap-3
            Field
              FieldLabel {{ $t("settings.auth.bindDn") }}
              Input(
                v-model="ldapForm.bind_dn"
                :placeholder="$t('settings.auth.bindDnPlaceholder')"
              )
            Field
              FieldLabel {{ $t("settings.auth.bindPassword") }}
              Input(
                v-model="ldapForm.bind_password"
                type="password"
                :placeholder="editingLDAPId ? $t('settings.auth.keepBlank') : ''"
              )
          .grid.grid-cols-2.gap-3
            Field
              FieldLabel {{ $t("settings.auth.userFilter") }}
              Input(
                v-model="ldapForm.user_filter"
                :placeholder="$t('settings.auth.userFilterPlaceholder')"
              )
            Field
              FieldLabel {{ $t("settings.auth.usernameAttribute") }}
              Input(
                v-model="ldapForm.username_attr"
                :placeholder="$t('settings.auth.usernameAttributePlaceholder')"
              )
          .grid.grid-cols-2.gap-3
            Field
              FieldLabel {{ $t("settings.auth.emailAttribute") }}
              Input(
                v-model="ldapForm.email_attr"
                :placeholder="$t('settings.auth.emailAttributePlaceholder')"
              )
            Field
              FieldLabel {{ $t("settings.auth.nameAttribute") }}
              Input(
                v-model="ldapForm.name_attr"
                :placeholder="$t('settings.auth.nameAttributePlaceholder')"
              )
          .flex.flex-col.gap-1
            Field.flex-row.items-center.gap-2
              Switch(
                :model-value="ldapForm.enabled"
                @update:model-value="ldapForm.enabled = Boolean($event)"
              )
              FieldLabel {{ $t("common.enabled") }}
            Field.flex-row.items-center.gap-2
              Switch(
                :model-value="ldapForm.auto_provision"
                @update:model-value="ldapForm.auto_provision = Boolean($event)"
              )
              FieldLabel {{ $t("settings.auth.createUsers") }}
            p.text-muted-foreground.text-xs
              | {{ $t("settings.auth.ldapCreateUsersHint") }}
        p.text-destructive.text-sm(v-if="error") {{ error }}
        .flex.justify-end.gap-2
          Button(type="button" variant="outline" @click="ldapFormOpen = false") {{ $t("common.cancel") }}
          Button(type="submit" :disabled="saving") {{ $t("settings.auth.saveDirectory") }}

  Card
    CardHeader
      CardTitle {{ $t("settings.auth.twoFactorTitle") }}
      CardDescription
        | {{ $t("settings.auth.twoFactorDescription") }}
    CardContent.flex.flex-col.gap-3
      .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
        v-for="toggle in totpToggles"
        :key="toggle.key"
      )
        .min-w-0
          p.text-sm.font-medium {{ $t(toggle.labelKey) }}
          p.text-muted-foreground.text-xs {{ $t(toggle.descriptionKey) }}
        Switch(
          :model-value="Boolean(totpSettings[toggle.key])"
          @update:model-value="totpSettings[toggle.key] = Boolean($event)"
        )
      .flex.justify-end
        Button(@click="saveSettings" :disabled="saving") {{ $t("common.save") }}

  Card
    CardHeader
      CardTitle {{ $t("settings.auth.methodsTitle") }}
      CardDescription
        | {{ $t("settings.auth.methodsDescription") }}
    CardContent.flex.flex-col.gap-3
      .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
        v-for="toggle in signInToggles"
        :key="toggle.key"
      )
        .min-w-0
          p.text-sm.font-medium {{ $t(toggle.labelKey) }}
          p.text-muted-foreground.text-xs {{ $t(toggle.descriptionKey) }}
        Switch(
          :model-value="Boolean(totpSettings[toggle.key])"
          @update:model-value="totpSettings[toggle.key] = Boolean($event)"
        )
      .flex.justify-end
        Button(@click="saveSettings" :disabled="saving") {{ $t("common.save") }}

  Card
    CardHeader
      CardTitle {{ $t("settings.auth.governanceTitle") }}
      CardDescription
        | {{ $t("settings.auth.governanceDescription") }}
    CardContent.flex.flex-col.gap-3
      .flex.items-center.justify-between.gap-2.rounded-md.border.p-3(
        v-for="toggle in governanceToggles"
        :key="toggle.key"
      )
        .min-w-0
          p.text-sm.font-medium {{ $t(toggle.labelKey) }}
          p.text-muted-foreground.text-xs {{ $t(toggle.descriptionKey) }}
        Switch(
          :model-value="Boolean(totpSettings[toggle.key])"
          @update:model-value="totpSettings[toggle.key] = Boolean($event)"
        )
      .flex.justify-end
        Button(@click="saveSettings" :disabled="saving") {{ $t("common.save") }}
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
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t } = useI18n();

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
    labelKey: "settings.auth.passwordSignIn",
    descriptionKey: "settings.auth.passwordSignInHint",
  },
  {
    key: "magic_link_login" as const,
    labelKey: "settings.auth.magicLinkSignIn",
    descriptionKey: "settings.auth.magicLinkSignInHint",
  },
  {
    key: "email_confirmation" as const,
    labelKey: "settings.auth.requireEmail",
    descriptionKey: "settings.auth.requireEmailHint",
  },
];

const governanceToggles = [
  {
    key: "members_can_invite_users" as const,
    labelKey: "settings.auth.membersInvite",
    descriptionKey: "settings.auth.membersInviteHint",
  },
  {
    key: "members_can_create_agents" as const,
    labelKey: "settings.auth.membersAgents",
    descriptionKey: "settings.auth.membersAgentsHint",
  },
  {
    key: "members_can_create_teams" as const,
    labelKey: "settings.auth.membersTeams",
    descriptionKey: "settings.auth.membersTeamsHint",
  },
  {
    key: "project_deletion_admin_only" as const,
    labelKey: "settings.auth.projectDeleteAdmin",
    descriptionKey: "settings.auth.projectDeleteAdminHint",
  },
  {
    key: "team_deletion_admin_only" as const,
    labelKey: "settings.auth.teamDeleteAdmin",
    descriptionKey: "settings.auth.teamDeleteAdminHint",
  },
];

const totpToggles = [
  {
    key: "totp_password" as const,
    labelKey: "common.password",
    descriptionKey: "settings.auth.methodPassword",
  },
  {
    key: "totp_magic_link" as const,
    labelKey: "settings.auth.methodMagicLink",
    descriptionKey: "settings.auth.methodMagicLinkHint",
  },
  {
    key: "totp_oidc" as const,
    labelKey: "settings.auth.methodOidc",
    descriptionKey: "settings.auth.methodOidcHint",
  },
  {
    key: "totp_oauth2" as const,
    labelKey: "settings.auth.methodOauth2",
    descriptionKey: "settings.auth.methodOauth2Hint",
  },
  {
    key: "totp_ldap" as const,
    labelKey: "settings.auth.methodLdap",
    descriptionKey: "settings.auth.methodLdapHint",
  },
];

onMounted(load);

function defaultProviderForm() {
  return {
    kind: "oidc",
    slug: "",
    name: "",
    enabled: true,
    auto_provision: false,
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
    auto_provision: false,
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
    error.value =
      err instanceof Error ? err.message : t("settings.auth.errors.load");
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
    auto_provision: provider.auto_provision,
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
    toast.success(t("settings.auth.toasts.providerSaved"));
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : t("account.toasts.saveFailed");
  } finally {
    saving.value = false;
  }
}

async function removeProvider(provider: IdentityProvider): Promise<void> {
  try {
    await accountApi.deleteProvider(provider.id);
    await load();
  } catch (err) {
    toast.error(
      err instanceof Error
        ? err.message
        : t("settings.auth.toasts.deleteFailed"),
    );
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
    auto_provision: provider.auto_provision,
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
    toast.success(t("settings.auth.toasts.directorySaved"));
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : t("account.toasts.saveFailed");
  } finally {
    saving.value = false;
  }
}

async function removeLDAP(provider: LDAPProvider): Promise<void> {
  try {
    await accountApi.deleteLDAPProvider(provider.id);
    await load();
  } catch (err) {
    toast.error(
      err instanceof Error
        ? err.message
        : t("settings.auth.toasts.deleteFailed"),
    );
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
    toast.success(t("settings.auth.toasts.saved"));
  } catch (err) {
    error.value =
      err instanceof Error ? err.message : t("account.toasts.saveFailed");
  } finally {
    saving.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  settings:
    auth:
      namePlaceholderPath: "name"
      host: "Host"
      slug: "Slug"
      disabled: "disabled"
      title: "Federated identity providers"
      description: "Generic OIDC discovery and the generic OAuth2 shim. Users link an identity from their account after signing in. Without automatic account creation, logins stay invite-only."
      addProvider: "Add provider"
      autoProvision: "auto-provision"
      noProviders: "No providers"
      noProvidersHint: "Add an OIDC or OAuth2 provider to enable federated sign-in."
      kindOidc: "OIDC (discovery)"
      kindOauth2: "OAuth2 (shim)"
      clientId: "Client ID"
      clientSecret: "Client secret"
      scopes: "Scopes"
      issuer: "Issuer"
      authorizationUrl: "Authorization URL"
      tokenUrl: "Token URL"
      userinfoUrl: "Userinfo URL"
      subjectPath: "Subject path"
      emailPath: "E-mail path"
      namePath: "Name path"
      createUsers: "Create users on first sign-in"
      createUsersHint: "A member account is created and linked on the first successful login. The provider username is used; a numeric suffix is added when it is taken. Leave off to keep invite-only."
      saveProvider: "Save provider"
      ldapTitle: "LDAP directories"
      ldapDescription: "Multiple providers; the sign-in page lists them. Users link a directory username from their account. Without automatic account creation, directory logins stay invite-only."
      addDirectory: "Add directory"
      noDirectories: "No directories"
      noDirectoriesHint: "Add an LDAP directory to enable directory sign-in."
      encryption: "Encryption"
      starttls: "STARTTLS"
      ldaps: "LDAPS"
      port: "Port"
      skipTls: "Skip TLS verification"
      baseDn: "Base DN"
      bindDn: "Bind DN / template"
      bindPassword: "Bind password"
      userFilter: "User filter"
      usernameAttribute: "Username attribute"
      emailAttribute: "E-mail attribute"
      nameAttribute: "Name attribute"
      ldapCreateUsersHint: "A member account is created and linked on the first successful bind. The directory username is used; a numeric suffix is added when it is taken. Leave off to keep invite-only."
      saveDirectory: "Save directory"
      twoFactorTitle: "Two-factor enforcement"
      twoFactorDescription: "Whether a method must present TOTP when the account has it enabled. Turning a method off lets an upstream provider supply the second factor."
      methodsTitle: "Sign-in methods"
      methodsDescription: "Org-wide toggles for the credential-based sign-in methods. Administrators always keep the username/password escape hatch."
      governanceTitle: "Membership & governance"
      governanceDescription: "What plain members may do, and whether destructive object actions are reserved for administrators."
      slugPlaceholder: "dex"
      namePlaceholder: "Company SSO"
      scopesPlaceholder: "openid email profile"
      issuerPlaceholder: "https://id.example.com"
      discovered: "discovered"
      authorizationPlaceholder: "https://id.example.com/authorize"
      tokenPlaceholder: "https://id.example.com/token"
      userinfoPlaceholder: "https://id.example.com/userinfo"
      subjectPlaceholder: "sub"
      emailPlaceholder: "email"
      ldapSlugPlaceholder: "glauth"
      ldapNamePlaceholder: "Company directory"
      hostPlaceholder: "ldap.example.com"
      baseDnPlaceholder: "dc=example,dc=org"
      bindDnPlaceholder: "uid={username},ou=people,dc=example,dc=org"
      userFilterPlaceholder: "(uid={username})"
      usernameAttributePlaceholder: "uid"
      emailAttributePlaceholder: "mail"
      nameAttributePlaceholder: "cn"
      implicitTls: "Implicit TLS"
      keepBlank: "Leave blank to keep"
      passwordSignIn: "Password sign-in"
      passwordSignInHint: "Username and password login; disabling also disables username invites."
      magicLinkSignIn: "Magic-link sign-in"
      magicLinkSignInHint: "Password-less sign-in via a confirmed e-mail address."
      requireEmail: "Require e-mail confirmation"
      requireEmailHint: "Users must confirm an address before they can sign in."
      membersInvite: "Members may invite users"
      membersInviteHint: "Plain members may send invitations from this page."
      membersAgents: "Members may create agents"
      membersAgentsHint: "Plain members may register new agents (phase 4)."
      membersTeams: "Members may create teams"
      membersTeamsHint: "Plain members may create teams (phase 4)."
      projectDeleteAdmin: "Project deletion is admin-only"
      projectDeleteAdminHint: "Only administrators may delete a project (phase 5)."
      teamDeleteAdmin: "Team deletion is admin-only"
      teamDeleteAdminHint: "Only administrators may delete a team (phase 4)."
      methodMagicLink: "Magic link"
      methodOidc: "OIDC"
      methodOauth2: "OAuth2"
      methodLdap: "LDAP"
      errors:
        load: "Failed to load"
      toasts:
        providerSaved: "Provider saved"
        deleteFailed: "Delete failed"
        directorySaved: "Directory saved"
        saved: "Authentication settings saved"
      methodPassword: "Username/password sign-in."
      methodMagicLinkHint: "Password-less e-mail sign-in."
      methodOidcHint: "OIDC discovery providers."
      methodOauth2Hint: "The generic OAuth2 shim."
      methodLdapHint: "Directory sign-in."
ru:
  settings:
    auth:
      namePlaceholderPath: "name"
      host: "Хост"
      slug: "Слаг"
      disabled: "отключён"
      title: "Федеративные провайдеры идентификации"
      description: "Универсальное обнаружение OIDC и общая прослойка OAuth2. Пользователи привязывают учётную запись из своего аккаунта после входа. Без автоматического создания аккаунтов вход остаётся только по приглашениям."
      addProvider: "Добавить провайдера"
      autoProvision: "автосоздание"
      noProviders: "Провайдеров нет"
      noProvidersHint: "Добавьте провайдера OIDC или OAuth2, чтобы включить федеративный вход."
      kindOidc: "OIDC (обнаружение)"
      kindOauth2: "OAuth2 (прослойка)"
      clientId: "Client ID"
      clientSecret: "Client secret"
      scopes: "Области доступа"
      issuer: "Издатель"
      authorizationUrl: "URL авторизации"
      tokenUrl: "URL токена"
      userinfoUrl: "URL профиля"
      subjectPath: "Путь к subject"
      emailPath: "Путь к e-mail"
      namePath: "Путь к имени"
      createUsers: "Создавать пользователей при первом входе"
      createUsersHint: "Аккаунт участника создаётся и связывается при первом успешном входе. Используется имя пользователя из провайдера; при совпадении добавляется числовой суффикс. Выключите, чтобы оставить вход только по приглашениям."
      saveProvider: "Сохранить провайдера"
      ldapTitle: "Каталоги LDAP"
      ldapDescription: "Несколько провайдеров; страница входа показывает их списком. Пользователи привязывают имя пользователя каталога из своего аккаунта. Без автоматического создания аккаунтов вход через каталог остаётся только по приглашениям."
      addDirectory: "Добавить каталог"
      noDirectories: "Каталогов нет"
      noDirectoriesHint: "Добавьте каталог LDAP, чтобы включить вход через каталог."
      encryption: "Шифрование"
      starttls: "STARTTLS"
      ldaps: "LDAPS"
      port: "Порт"
      skipTls: "Пропускать проверку TLS"
      baseDn: "Base DN"
      bindDn: "Bind DN / шаблон"
      bindPassword: "Пароль привязки"
      userFilter: "Фильтр пользователей"
      usernameAttribute: "Атрибут имени пользователя"
      emailAttribute: "Атрибут e-mail"
      nameAttribute: "Атрибут имени"
      ldapCreateUsersHint: "Аккаунт участника создаётся и связывается при первой успешной привязке. Используется имя пользователя каталога; при совпадении добавляется числовой суффикс. Выключите, чтобы оставить вход только по приглашениям."
      saveDirectory: "Сохранить каталог"
      twoFactorTitle: "Обязательность двухфакторной защиты"
      twoFactorDescription: "Должен ли способ входа запрашивать TOTP, если в аккаунте он включён. Если способ выключен, второй фактор может предоставить внешний провайдер."
      methodsTitle: "Способы входа"
      methodsDescription: "Общие для агентства переключатели способов входа по учётным данным. Администраторы всегда могут войти по имени пользователя и паролю."
      governanceTitle: "Участники и управление"
      governanceDescription: "Что могут делать обычные участники и закреплены ли разрушительные действия за администраторами."
      slugPlaceholder: "dex"
      namePlaceholder: "Корпоративный SSO"
      scopesPlaceholder: "openid email profile"
      issuerPlaceholder: "https://id.example.com"
      discovered: "определяется автоматически"
      authorizationPlaceholder: "https://id.example.com/authorize"
      tokenPlaceholder: "https://id.example.com/token"
      userinfoPlaceholder: "https://id.example.com/userinfo"
      subjectPlaceholder: "sub"
      emailPlaceholder: "email"
      ldapSlugPlaceholder: "glauth"
      ldapNamePlaceholder: "Корпоративный каталог"
      hostPlaceholder: "ldap.example.com"
      baseDnPlaceholder: "dc=example,dc=org"
      bindDnPlaceholder: "uid={username},ou=people,dc=example,dc=org"
      userFilterPlaceholder: "(uid={username})"
      usernameAttributePlaceholder: "uid"
      emailAttributePlaceholder: "mail"
      nameAttributePlaceholder: "cn"
      implicitTls: "Неявный TLS"
      keepBlank: "Оставьте пустым, чтобы не менять"
      passwordSignIn: "Вход по паролю"
      passwordSignInHint: "Вход по имени пользователя и паролю; при отключении отключаются и приглашения по имени пользователя."
      magicLinkSignIn: "Вход по ссылке"
      magicLinkSignInHint: "Вход без пароля через подтверждённый адрес e-mail."
      requireEmail: "Требовать подтверждение e-mail"
      requireEmailHint: "Пользователи должны подтвердить адрес, прежде чем войти."
      membersInvite: "Участники могут приглашать пользователей"
      membersInviteHint: "Обычные участники могут отправлять приглашения с этой страницы."
      membersAgents: "Участники могут создавать агентов"
      membersAgentsHint: "Обычные участники могут регистрировать новых агентов (этап 4)."
      membersTeams: "Участники могут создавать команды"
      membersTeamsHint: "Обычные участники могут создавать команды (этап 4)."
      projectDeleteAdmin: "Удалять проекты может только администратор"
      projectDeleteAdminHint: "Удалять проект могут только администраторы (этап 5)."
      teamDeleteAdmin: "Удалять команды может только администратор"
      teamDeleteAdminHint: "Удалять команду могут только администраторы (этап 4)."
      methodMagicLink: "Ссылка для входа"
      methodOidc: "OIDC"
      methodOauth2: "OAuth2"
      methodLdap: "LDAP"
      errors:
        load: "Не удалось загрузить"
      toasts:
        providerSaved: "Провайдер сохранён"
        deleteFailed: "Не удалось удалить"
        directorySaved: "Каталог сохранён"
        saved: "Настройки аутентификации сохранены"
      methodPassword: "Вход по имени пользователя и паролю."
      methodMagicLinkHint: "Вход по e-mail без пароля."
      methodOidcHint: "Провайдеры с обнаружением OIDC."
      methodOauth2Hint: "Универсальная прослойка OAuth2."
      methodLdapHint: "Вход через каталог."
</i18n>
