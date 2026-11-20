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
      CardTitle {{ $t("domainLog.kind.access") }}
      CardDescription
        | {{ $t("projectSettings.access.description") }}
    CardContent.flex.flex-col.gap-6
      .flex.items-center.gap-2
        Badge(variant="secondary")
          | {{ $t("projectSettings.yourRole", { role: $t(PROJECT_ROLE_KEY[access?.role ?? "viewer"]) }) }}
        span.text-muted-foreground.text-xs(
          v-if="access && !access.can_manage_access"
        ) {{ $t("projectSettings.access.ownersOnly") }}

      Separator

      .flex.flex-col.gap-3
        span.text-sm.font-medium {{ $t("projectSettings.access.direct") }}
        Table
          TableHeader
            TableRow
              TableHead {{ $t("settings.events.actor") }}
              TableHead {{ $t("admin.role") }}
              TableHead
          TableBody
            TableRow(v-for="grant in directMembers" :key="grant.actor_id")
              TableCell
                .flex.flex-col
                  span.font-medium {{ grant.name || grant.username }}
                  span.text-muted-foreground.text-xs
                    | {{ grant.kind }} · @{{ grant.username }}
              TableCell
                Select(
                  v-if="access?.can_manage_access"
                  :model-value="grant.role"
                  @update:model-value="(value) => changeMemberRole(grant, value)"
                )
                  SelectTrigger.w-32
                    SelectValue
                  SelectContent
                    SelectItem(v-for="role in ROLES" :key="role" :value="role") {{ $t(PROJECT_ROLE_KEY[role]) }}
                span(v-else) {{ $t(PROJECT_ROLE_KEY[grant.role]) }}
              TableCell
                .flex.justify-end(v-if="access?.can_manage_access")
                  Button.text-destructive(
                    variant="outline"
                    size="sm"
                    @click="removeMemberGrant(grant)"
                  )
                    Trash2Icon(data-icon="inline-start")
                    span {{ $t("common.remove") }}
            TableRow(v-if="!directMembers.length")
              TableCell.text-muted-foreground(:colspan="3")
                | {{ $t("projectSettings.access.noDirect") }}
        .flex.items-center.gap-2(v-if="access?.can_manage_access")
          ActorCombobox#new-member-actor.w-56(
            v-model="newMemberUsername"
            :placeholder="$t('projectSettings.access.addPlaceholder')"
            kinds="all"
          )
          Select(v-model="newMemberRole")
            SelectTrigger.w-36
              SelectValue
            SelectContent
              SelectItem(v-for="role in ROLES" :key="role" :value="role") {{ $t(PROJECT_ROLE_KEY[role]) }}
          Button(
            @click="addMemberGrant"
            :disabled="!newMemberUsername.trim() || accessBusy"
          )
            PlusIcon(data-icon="inline-start")
            span {{ $t("common.add") }}

      Separator

      .flex.flex-col.gap-3
        span.text-sm.font-medium {{ $t("projectSettings.access.plugins") }}
        p.text-muted-foreground.text-xs
          | {{ $t("projectSettings.access.pluginsHint") }}
          RouterLink.text-primary.ml-1(
            :to="{ name: 'project-settings-plugins', params: { slug } }"
          ) {{ $t("projectSettings.access.managePlugins") }}
        Table
          TableHeader
            TableRow
              TableHead {{ $t("taskBadges.plugin") }}
              TableHead {{ $t("domainLog.kind.access") }}
          TableBody
            TableRow(v-for="grant in pluginMembers" :key="grant.actor_id")
              TableCell
                .flex.items-center.gap-2
                  PlugIcon.text-muted-foreground.size-4
                  .flex.flex-col
                    span.font-medium {{ grant.name || grant.username }}
                    span.text-muted-foreground.text-xs
                      | {{ $t("projectSettings.pluginGrant", { username: grant.username }) }}
              TableCell
                .flex.items-center.gap-2
                  Badge(variant="secondary")
                    | {{ $t(PROJECT_ROLE_KEY[grant.role]) }}
                  span.text-muted-foreground.text-xs
                    | {{ $t("projectSettings.access.alwaysByInstall") }}
            TableRow(v-if="!pluginMembers.length")
              TableCell.text-muted-foreground(:colspan="2")
                | {{ $t("projectSettings.access.noPlugins") }}

      Separator

      .flex.flex-col.gap-3
        span.text-sm.font-medium {{ $t("nav.teams") }}
        Table
          TableHeader
            TableRow
              TableHead {{ $t("admin.teams.columnTeam") }}
              TableHead {{ $t("admin.teams.columnMembers") }}
              TableHead {{ $t("admin.role") }}
              TableHead
          TableBody
            TableRow(v-for="link in access?.teams ?? []" :key="link.team_id")
              TableCell
                .flex.flex-col
                  span.font-medium {{ link.team_name }}
                  span.text-muted-foreground.text-xs(
                    v-if="link.inherited_role"
                  )
                    | {{ $t("projectSettings.alsoInherited", { role: $t(PROJECT_ROLE_KEY[link.inherited_role]) }) }}
                    RouterLink.text-primary.ml-1(
                      :to="{ name: 'project-settings-access', params: { slug: link.inherited_from_project_slug } }"
                    ) {{ link.inherited_from_project_name }}
              TableCell.text-muted-foreground {{ link.member_count }}
              TableCell
                Select(
                  v-if="access?.can_manage_access"
                  :model-value="link.role"
                  @update:model-value="(value) => changeTeamRole(link, value)"
                )
                  SelectTrigger.w-32
                    SelectValue
                  SelectContent
                    SelectItem(
                      v-for="role in roleOptionsAbove(link.inherited_role)"
                      :key="role"
                      :value="role"
                    ) {{ $t(PROJECT_ROLE_KEY[role]) }}
                span(v-else) {{ $t(PROJECT_ROLE_KEY[link.role]) }}
              TableCell
                .flex.justify-end(v-if="access?.can_manage_access")
                  Button.text-destructive(
                    variant="outline"
                    size="sm"
                    @click="removeTeamLink(link)"
                  )
                    Trash2Icon(data-icon="inline-start")
                    span {{ $t("common.remove") }}
            TableRow(v-if="!(access?.teams ?? []).length")
              TableCell.text-muted-foreground(:colspan="4")
                | {{ $t("projectSettings.teams.noTeams") }}
        .flex.items-center.gap-2(v-if="access?.can_manage_access")
          Combobox.w-56(
            v-model="newTeamId"
            :placeholder="$t('projectSettings.teams.invitePlaceholder')"
            empty-text="No matching team."
            :aria-label="$t('projectSettings.teams.inviteAria')"
            :options="teamOptions"
          )
          Select(v-model="newTeamRole")
            SelectTrigger.w-36
              SelectValue
            SelectContent
              SelectItem(
                v-for="role in newTeamRoleOptions"
                :key="role"
                :value="role"
              ) {{ $t(PROJECT_ROLE_KEY[role]) }}
          Button(@click="inviteTeam" :disabled="!newTeamId || accessBusy")
            PlusIcon(data-icon="inline-start")
            span {{ selectedTeamInheritedRole ? "Add" : "Invite" }}
        p.text-muted-foreground.text-xs(
          v-if="selectedTeamInheritedRole && !selectedTeamUpgrade"
        )
          | {{ $t("projectSettings.teamAlreadyInherited", { role: $t(PROJECT_ROLE_KEY[selectedTeamInheritedRole]), project: selectedTeamInheritedFrom }) }}
          | {{ $t("projectSettings.teams.linksWithoutApproval") }}
        p.text-muted-foreground.text-xs(v-else-if="selectedTeamUpgrade")
          | {{ $t("projectSettings.teams.needsApproval") }}
        p.text-muted-foreground.text-xs(v-if="!invitableTeams.length")
          | {{ $t("projectSettings.teams.noOtherTeams") }}

      template(v-if="(access?.inherited_teams ?? []).length")
        Separator

        .flex.flex-col.gap-3
          span.text-sm.font-medium {{ $t("projectSettings.teams.inherited") }}
          p.text-muted-foreground.text-xs
            | {{ $t("projectSettings.teams.inheritedHint") }}
          Table
            TableHeader
              TableRow
                TableHead {{ $t("admin.teams.columnTeam") }}
                TableHead {{ $t("admin.teams.columnMembers") }}
                TableHead {{ $t("projectSettings.teams.effectiveRole") }}
                TableHead {{ $t("projectSettings.teams.inheritedFrom") }}
            TableBody
              TableRow(
                v-for="link in access?.inherited_teams ?? []"
                :key="link.team_id"
              )
                TableCell.font-medium {{ link.team_name }}
                TableCell.text-muted-foreground {{ link.member_count }}
                TableCell
                  Badge(variant="secondary")
                    | {{ $t(PROJECT_ROLE_KEY[link.role]) }}
                TableCell
                  RouterLink.text-primary(
                    :to="{ name: 'project-settings-access', params: { slug: link.inherited_from_project_slug } }"
                  ) {{ link.inherited_from_project_name }}

      template(v-if="page.project")
        Separator

        ProjectGroupsPanel(
          :project-id="page.project.id"
          :can-manage="access?.can_manage_access === true"
        )

      Separator

      .flex.flex-col.gap-3
        span.text-sm.font-medium {{ $t("users.invitations.title") }}
        Table
          TableHeader
            TableRow
              TableHead {{ $t("admin.teams.columnTeam") }}
              TableHead {{ $t("admin.role") }}
              TableHead {{ $t("common.status") }}
              TableHead
          TableBody
            TableRow(
              v-for="invitation in access?.invitations ?? []"
              :key="invitation.id"
            )
              TableCell.font-medium {{ invitation.team_name }}
              TableCell {{ $t(PROJECT_ROLE_KEY[invitation.role]) }}
              TableCell.text-muted-foreground {{ invitation.status }}
              TableCell
                .flex.justify-end(
                  v-if="access?.can_manage_access && invitation.status === 'pending'"
                )
                  Button.text-destructive(
                    variant="outline"
                    size="sm"
                    @click="revokeInvite(invitation)"
                  )
                    Trash2Icon(data-icon="inline-start")
                    span {{ $t("common.revoke") }}
            TableRow(v-if="!(access?.invitations ?? []).length")
              TableCell.text-muted-foreground(:colspan="4")
                | {{ $t("users.invitations.empty") }}
</template>

<script setup lang="ts">
import ActorCombobox from "@/components/ActorCombobox.vue";
import ProjectGroupsPanel from "@/components/settings/ProjectGroupsPanel.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Combobox, type ComboboxOption } from "@/components/ui/combobox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  PROJECT_ROLE_KEY,
  projectAccessApi,
  type ProjectMemberGrant,
  type ProjectRole,
  type ProjectTeamLink,
} from "@/lib/projectAccess";
import type { ProjectPageState, ProjectSettingsState } from "@/lib/projectPage";
import { teamApi, type Team } from "@/lib/team";
import { PlugIcon, PlusIcon, Trash2Icon } from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, useRoute } from "vue-router";
import { toast } from "vue3-toastify";

const props = defineProps<{
  page: ProjectPageState;
  settings: ProjectSettingsState;
}>();

const { t } = useI18n();
const route = useRoute();

const slug = computed(() => String(route.params.slug ?? ""));
const access = computed(() => props.page.access);
const allTeams = ref<Team[]>([]);
const accessBusy = ref(false);
const newMemberUsername = ref("");
const newMemberRole = ref<ProjectRole>("member");
const newTeamId = ref("");
const newTeamRole = ref<ProjectRole>("member");
const ROLES: ProjectRole[] = ["owner", "member", "viewer"];
const ROLE_WEIGHT: Record<ProjectRole, number> = {
  owner: 3,
  member: 2,
  viewer: 1,
};

const directMembers = computed(() =>
  (access.value?.members ?? []).filter((grant) => grant.kind !== "plugin"),
);
const pluginMembers = computed(() =>
  (access.value?.members ?? []).filter((grant) => grant.kind === "plugin"),
);

// Teams that are neither attached nor already invited, so the picker never

const invitableTeams = computed(() => {
  const linked = new Set(
    (access.value?.teams ?? []).map((item) => item.team_id),
  );
  const pending = new Set(
    (access.value?.invitations ?? [])
      .filter((invitation) => invitation.status === "pending")
      .map((invitation) => invitation.team_id),
  );
  return allTeams.value.filter(
    (team) => !linked.has(team.id) && !pending.has(team.id),
  );
});

const teamOptions = computed<ComboboxOption[]>(() =>
  invitableTeams.value.map((team) => ({
    value: String(team.id),
    label: team.name,
    description:
      team.description ||
      `${team.member_count} ${team.member_count === 1 ? "member" : "members"}`,
  })),
);

const selectedTeamInheritedLink = computed(() =>
  (access.value?.inherited_teams ?? []).find(
    (link) => String(link.team_id) === newTeamId.value,
  ),
);
const selectedTeamInheritedRole = computed<ProjectRole | undefined>(
  () => selectedTeamInheritedLink.value?.role,
);
const selectedTeamInheritedFrom = computed(
  () => selectedTeamInheritedLink.value?.inherited_from_project_name ?? "",
);
const selectedTeamUpgrade = computed(
  () =>
    !!selectedTeamInheritedRole.value &&
    ROLE_WEIGHT[newTeamRole.value] >
      ROLE_WEIGHT[selectedTeamInheritedRole.value],
);

function roleOptionsAbove(inherited?: ProjectRole): ProjectRole[] {
  if (!inherited) return ROLES;
  return ROLES.filter((role) => ROLE_WEIGHT[role] >= ROLE_WEIGHT[inherited]);
}
const newTeamRoleOptions = computed(() =>
  roleOptionsAbove(selectedTeamInheritedRole.value),
);
watch(newTeamId, () => {
  const inherited = selectedTeamInheritedRole.value;
  if (inherited && ROLE_WEIGHT[newTeamRole.value] < ROLE_WEIGHT[inherited]) {
    newTeamRole.value = inherited;
  }
});

async function loadTeams() {
  try {
    allTeams.value = (await teamApi.list()).teams;
  } catch {
    allTeams.value = [];
  }
}

async function addMemberGrant() {
  const project = props.page.project;
  if (!project || !newMemberUsername.value.trim()) return;
  accessBusy.value = true;
  try {
    await projectAccessApi.addMember(
      project.id,
      newMemberUsername.value.trim(),
      newMemberRole.value,
    );
    newMemberUsername.value = "";
    toast.success(t("projectSettings.toasts.grantAdded"));
    await props.page.reloadAccess();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    accessBusy.value = false;
  }
}

async function changeMemberRole(grant: ProjectMemberGrant, role: unknown) {
  const project = props.page.project;
  if (
    !project ||
    typeof role !== "string" ||
    !ROLES.includes(role as ProjectRole)
  )
    return;
  accessBusy.value = true;
  try {
    await projectAccessApi.setMemberRole(
      project.id,
      grant.actor_id,
      role as ProjectRole,
    );
    await props.page.reloadAccess();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    accessBusy.value = false;
  }
}

async function removeMemberGrant(grant: ProjectMemberGrant) {
  const project = props.page.project;
  if (!project) return;
  if (!window.confirm(`Remove ${grant.username} from this project?`)) return;
  accessBusy.value = true;
  try {
    await projectAccessApi.removeMember(project.id, grant.actor_id);
    toast.success(t("projectSettings.toasts.grantRemoved"));
    await props.page.reloadAccess();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    accessBusy.value = false;
  }
}

async function inviteTeam() {
  const project = props.page.project;
  if (!project || !newTeamId.value) return;
  accessBusy.value = true;
  try {
    const result = await projectAccessApi.inviteTeam(
      project.id,
      Number(newTeamId.value),
      newTeamRole.value,
    );
    newTeamId.value = "";
    toast.success(
      result.team
        ? t("projectSettings.toasts.teamAttached")
        : t("projectSettings.toasts.teamInvited"),
    );
    await props.page.reloadAccess();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    accessBusy.value = false;
  }
}

async function changeTeamRole(link: ProjectTeamLink, role: unknown) {
  const project = props.page.project;
  if (
    !project ||
    typeof role !== "string" ||
    !ROLES.includes(role as ProjectRole)
  )
    return;
  accessBusy.value = true;
  try {
    await projectAccessApi.setTeamRole(
      project.id,
      link.team_id,
      role as ProjectRole,
    );
    await props.page.reloadAccess();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    accessBusy.value = false;
  }
}

async function removeTeamLink(link: ProjectTeamLink) {
  const project = props.page.project;
  if (!project) return;
  if (!window.confirm(`Detach ${link.team_name} from this project?`)) return;
  accessBusy.value = true;
  try {
    await projectAccessApi.removeTeam(project.id, link.team_id);
    toast.success(t("projectSettings.toasts.teamDetached"));
    await props.page.reloadAccess();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    accessBusy.value = false;
  }
}

async function revokeInvite(invitation: { id: number }) {
  const project = props.page.project;
  if (!project) return;
  accessBusy.value = true;
  try {
    await projectAccessApi.revokeInvitation(project.id, invitation.id);
    await props.page.reloadAccess();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    accessBusy.value = false;
  }
}

onMounted(() => {
  void loadTeams();
});
</script>

<i18n lang="yaml">
en:
  projectSettings:
    access:
      description: "Who can reach this project. The effective role is the strongest of a direct grant and the caller's teams; a viewer may view and comment, a member may prompt and edit, an owner may also manage access."
      ownersOnly: "Only project owners can change access."
      direct: "Direct access"
      noDirect: "No direct access."
      addPlaceholder: "Add a person or agent…"
      plugins: "Installed plugins"
      pluginsHint: "Installed plugins have direct access to the project, always 'member' level."
      managePlugins: "Manage in Plugins"
      alwaysByInstall: "always · set by install"
      noPlugins: "No plugins have access to this project."
    teams:
      noTeams: "No teams attached."
      invitePlaceholder: "Invite a team…"
      inviteAria: "Team to invite"
      linksWithoutApproval: "links it without another approval."
      needsApproval: "A role above the inherited role needs the team owner's approval."
      noOtherTeams: "No other teams available to invite."
      inherited: "Inherited teams"
      inheritedHint: "These teams reach this project through an ancestor project. Their access is read-only here; attach the team explicitly above to give it a project-specific role."
      effectiveRole: "Effective role"
      inheritedFrom: "Inherited from"
    toasts:
      grantAdded: "Grant added"
      grantRemoved: "Grant removed"
      teamAttached: "Team attached"
      teamInvited: "Team invited; its owner must approve"
      teamDetached: "Team detached"
    yourRole: "Your role: {role}"
    pluginGrant: "plugin · {'@'}{username}"
    alsoInherited: "Also inherited as {role} from"
    teamAlreadyInherited: "This team already has inherited {role} access from {project}. Adding it directly"
ru:
  projectSettings:
    access:
      description: "Кто имеет доступ к проекту. Итоговая роль — сильнейшая из прямого разрешения и ролей в командах; наблюдатель может смотреть и комментировать, участник — отправлять промпты и редактировать, владелец — ещё и управлять доступом."
      ownersOnly: "Изменять доступ могут только владельцы проекта."
      direct: "Прямой доступ"
      noDirect: "Прямого доступа нет."
      addPlaceholder: "Добавить пользователя или агента…"
      plugins: "Установленные плагины"
      pluginsHint: "Установленные плагины имеют прямой доступ к проекту, всегда уровня «участник»."
      managePlugins: "Управлять в плагинах"
      alwaysByInstall: "всегда · задано при установке"
      noPlugins: "Ни один плагин не имеет доступа к этому проекту."
    teams:
      noTeams: "Команды не подключены."
      invitePlaceholder: "Пригласить команду…"
      inviteAria: "Команда для приглашения"
      linksWithoutApproval: "подключает её без дополнительного одобрения."
      needsApproval: "Роль выше унаследованной требует одобрения владельца команды."
      noOtherTeams: "Нет других команд для приглашения."
      inherited: "Унаследованные команды"
      inheritedHint: "Эти команды получают доступ к проекту через родительский проект. Здесь их доступ только для чтения; подключите команду явно выше, чтобы задать роль в проекте."
      effectiveRole: "Итоговая роль"
      inheritedFrom: "Унаследовано от"
    toasts:
      grantAdded: "Разрешение добавлено"
      grantRemoved: "Разрешение удалено"
      teamAttached: "Команда подключена"
      teamInvited: "Команда приглашена; требуется одобрение её владельца"
      teamDetached: "Команда отключена"
    yourRole: "Ваша роль: {role}"
    pluginGrant: "плагин · {'@'}{username}"
    alsoInherited: "Также унаследовано как {role} из"
    teamAlreadyInherited: "У этой команды уже есть унаследованный доступ {role} из {project}. Прямое подключение"
</i18n>
