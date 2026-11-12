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
  p.text-destructive.text-sm(v-if="error") {{ error }}

  Tabs(v-model="tab")
    TabsList
      TabsTrigger(value="general") {{ $t("settings.tabs.general") }}
      TabsTrigger(value="tasks") {{ $t("nav.tasks") }}
      TabsTrigger(value="templates") {{ $t("projectSettings.tabs.templates") }}
      TabsTrigger(value="plugins") {{ $t("account.tabs.plugins") }}
      TabsTrigger(value="access") {{ $t("domainLog.kind.access") }}
      TabsTrigger(value="danger") {{ $t("projectSettings.dangerZone") }}

    TabsContent(value="general")
      Card
        CardHeader
          CardTitle {{ $t("settings.tabs.general") }}
          CardDescription {{ $t("projectSettings.general.description") }}
        CardContent.flex.flex-col.gap-6
          form.flex.flex-col.gap-4(@submit.prevent="saveGeneral")
            FieldGroup
              Field
                FieldLabel {{ $t("projectSettings.general.picture") }}
                PictureUpload(
                  :picture="project?.picture"
                  :project-id="project?.id"
                  :fallback="logo"
                  @changed="onPictureChanged"
                )
              Field
                FieldLabel(for="project-name") {{ $t("common.name") }}
                Input#project-name(v-model="name")
              Field
                FieldLabel(for="project-description") {{ $t("common.description") }}
                Textarea#project-description(v-model="description")
              Field
                FieldLabel(for="project-slug") {{ $t("settings.auth.slug") }}
                Input#project-slug(disabled :model-value="project?.slug ?? ''")
                FieldDescription
                  | {{ $t("projectSettings.general.slugHint") }}
            .flex.justify-end
              Button(type="submit" :disabled="saving || !name.trim()")
                Spinner(v-if="saving" data-icon="inline-start")
                CheckIcon(v-else data-icon="inline-start")
                span {{ $t("common.save") }}

        Card
          CardHeader
            CardTitle {{ $t("domainLog.kind.hierarchy") }}
            CardDescription
              | {{ $t("projectSettings.hierarchy.description") }}
          CardContent.flex.flex-col.gap-5
            template(v-if="hierarchy")
              .flex.flex-col.gap-1(v-if="hierarchy.parent")
                span.text-sm.font-medium {{ $t("projectSettings.hierarchy.parent") }}
                .flex.items-center.gap-2
                  RouterLink.text-primary.font-medium(
                    :to="{ name: 'project', params: { slug: hierarchy.parent.slug } }"
                  ) {{ hierarchy.parent.name }}
                  Button(
                    v-if="hierarchy.can_detach"
                    variant="outline"
                    size="sm"
                    @click="detachParentProject"
                    :disabled="hierarchyBusy"
                  ) {{ $t("projectSettings.hierarchy.detach") }}
              p.text-muted-foreground.text-sm(v-else)
                | {{ $t("projectSettings.hierarchy.topLevel") }}

              .flex.flex-wrap.items-center.justify-between.gap-2.rounded-md.border.p-3(
                v-if="hierarchy.incoming_invitation"
              )
                .flex.flex-col
                  span.text-sm.font-medium
                    | {{ $t("projectSettings.invitationFrom", { project: hierarchy.incoming_invitation.parent_project_name }) }}
                  span.text-muted-foreground.text-xs(
                    v-if="hierarchy.incoming_invitation.invited_by_username"
                  )
                    | {{ $t("projectSettings.invitedBy", { username: hierarchy.incoming_invitation.invited_by_username }) }}
                .flex.gap-2(v-if="hierarchy.can_respond")
                  Button(
                    size="sm"
                    @click="respondParentInvitation(hierarchy.incoming_invitation, true)"
                    :disabled="hierarchyBusy"
                  ) {{ $t("projectSettings.hierarchy.accept") }}
                  Button(
                    variant="outline"
                    size="sm"
                    @click="respondParentInvitation(hierarchy.incoming_invitation, false)"
                    :disabled="hierarchyBusy"
                  ) {{ $t("extensionUi.decline") }}

              .flex.flex-col.gap-2(v-if="hierarchy.parent_options.length")
                span.text-sm.font-medium {{ $t("projectSettings.hierarchy.attach") }}
                .flex.flex-wrap.gap-2
                  Combobox.w-64(
                    v-model="newParentId"
                    :placeholder="$t('projectSettings.hierarchy.chooseParent')"
                    empty-text="No matching project."
                    :aria-label="$t('projectSettings.hierarchy.parent')"
                    :options="parentOptions"
                  )
                  Button(
                    @click="attachToParent"
                    :disabled="!newParentId || hierarchyBusy"
                  ) {{ selectedParentOption?.can_link_directly ? "Link" : "Request" }}
                p.text-muted-foreground.text-xs(v-if="selectedParentTip")
                  | {{ selectedParentTip }}

              Separator

              .flex.flex-col.gap-2
                span.text-sm.font-medium {{ $t("projectSettings.hierarchy.subprojects") }}
                p.text-muted-foreground.text-sm(
                  v-if="!hierarchy.children.length"
                )
                  | {{ $t("projectSettings.hierarchy.none") }}
                .flex.items-center.justify-between.gap-2(
                  v-for="child in hierarchy.children"
                  :key="child.id"
                )
                  RouterLink.text-primary.font-medium(
                    :to="{ name: 'project', params: { slug: child.slug } }"
                  ) {{ child.name }}
                  Button(
                    v-if="hierarchy.can_invite_child"
                    variant="outline"
                    size="sm"
                    @click="detachChildProject(child)"
                    :disabled="hierarchyBusy"
                  ) {{ $t("projectSettings.hierarchy.detach") }}

              .flex.flex-col.gap-2(v-if="hierarchy.child_options.length")
                span.text-sm.font-medium {{ $t("projectSettings.hierarchy.addSubproject") }}
                .flex.flex-wrap.gap-2
                  Combobox.w-64(
                    v-model="newChildId"
                    :placeholder="$t('projectSettings.hierarchy.chooseProject')"
                    empty-text="No matching project."
                    :aria-label="$t('projectSettings.hierarchy.subprojectAria')"
                    :options="childOptions"
                  )
                  Button(
                    @click="addChildProject"
                    :disabled="!newChildId || hierarchyBusy"
                  ) {{ selectedChildOption?.can_link_directly ? "Link" : "Request" }}
                p.text-muted-foreground.text-xs(v-if="selectedChildTip")
                  | {{ selectedChildTip }}

              .flex.flex-col.gap-2(v-if="pendingParentInvitations.length")
                span.text-sm.font-medium {{ $t("projectSettings.hierarchy.pendingInvitations") }}
                .flex.items-center.justify-between.gap-2(
                  v-for="invitation in pendingParentInvitations"
                  :key="invitation.id"
                )
                  span.text-sm
                    | {{ invitation.child_project_name }} → {{ invitation.parent_project_name }}
                  Button(
                    variant="ghost"
                    size="sm"
                    @click="revokeParentInvitation(invitation)"
                    :disabled="hierarchyBusy"
                  ) {{ $t("common.revoke") }}

    TabsContent(value="tasks")
      .flex.flex-col.gap-6
        Card
          CardHeader
            CardTitle {{ $t("projectSettings.taskIds.title") }}
            CardDescription
              | {{ $t("projectSettings.taskIds.description2", { prefix: prefix.trim() || "MYPROJECT" }) }}
          CardContent
            form.flex.items-end.gap-2(@submit.prevent="savePrefix")
              Field.flex-1
                FieldLabel(for="project-prefix") {{ $t("projectSettings.taskIds.prefix") }}
                Input#project-prefix(
                  v-model="prefix"
                  :placeholder="$t('projectSettings.taskIds.placeholder')"
                  autocapitalize="characters"
                )
              Button(type="submit" :disabled="savingPrefix || !prefix.trim()")
                Spinner(v-if="savingPrefix" data-icon="inline-start")
                CheckIcon(v-else data-icon="inline-start")
                span {{ $t("projectSettings.taskIds.save") }}

        Card
          CardHeader
            CardTitle {{ $t("projectSettings.archive.title") }}
            CardDescription
              | {{ $t("projectSettings.archive.description") }}
          CardContent
            form.flex.flex-col.gap-3(@submit.prevent="saveArchivePolicy")
              .flex.flex-wrap.items-end.gap-3
                Field.w-72
                  FieldLabel(for="project-archive-mode") {{ $t("projectSettings.archive.mode") }}
                  Select(v-model="archiveMode" :disabled="!canManageProject")
                    SelectTrigger#project-archive-mode.w-full
                      SelectValue
                    SelectContent
                      SelectItem(value="inherit") {{ inheritLabel }}
                      SelectItem(value="custom") {{ $t("projectSettings.archive.custom") }}
                      SelectItem(value="never") {{ $t("projectSettings.archive.never") }}
                Field.w-36(v-if="archiveMode === 'custom'")
                  FieldLabel(for="project-archive-days") {{ $t("projectSettings.archive.days") }}
                  Input#project-archive-days(
                    v-model.number="archiveCustomDays"
                    type="number"
                    min="1"
                    :max="MAX_ARCHIVE_DAYS"
                    :disabled="!canManageProject"
                  )
                Button(
                  type="submit"
                  :disabled="!canManageProject || savingArchive || !archivePolicyValid"
                )
                  Spinner(v-if="savingArchive" data-icon="inline-start")
                  CheckIcon(v-else data-icon="inline-start")
                  span {{ $t("common.save") }}
              p.text-muted-foreground.text-xs {{ archiveModeHint }}
              p.text-muted-foreground.text-xs(v-if="!canManageProject")
                | {{ $t("projectSettings.archive.ownersOnly") }}

        Card
          CardHeader
            CardTitle {{ $t("projectSettings.prompt.title") }}
            CardDescription
              | {{ $t("projectSettings.prompt.description") }}
          CardContent
            PromptEditor(
              :placeholder="$t('projectSettings.prompt.placeholder')"
              :scope="PromptScope.PROJECT"
              :project-id="project?.id"
            )

        Card
          CardHeader
            CardTitle {{ $t("domainLog.kind.tags") }}
            CardDescription {{ $t("projectSettings.tags.description") }}
          CardContent
            TagsPanel(:project-id="project?.id ?? ''")

    TabsContent(value="templates")
      Card
        CardHeader
          CardTitle {{ $t("projectSettings.tabs.templates") }}
          CardDescription
            | {{ $t("projectSettings.templates.description") }}
        CardContent
          TaskTemplatesPanel(:project-id="project?.id ?? ''")

    TabsContent(value="plugins")
      ProjectPluginsPanel(v-if="project" :project-id="project.id")

    TabsContent(value="access")
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
                  TableRow(
                    v-for="grant in directMembers"
                    :key="grant.actor_id"
                  )
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
                          SelectItem(
                            v-for="role in ROLES"
                            :key="role"
                            :value="role"
                          ) {{ $t(PROJECT_ROLE_KEY[role]) }}
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
                Button.h-auto.px-1(
                  variant="link"
                  size="xs"
                  @click="tab = 'plugins'"
                ) {{ $t("projectSettings.access.managePlugins") }}
              Table
                TableHeader
                  TableRow
                    TableHead {{ $t("taskBadges.plugin") }}
                    TableHead {{ $t("domainLog.kind.access") }}
                TableBody
                  TableRow(
                    v-for="grant in pluginMembers"
                    :key="grant.actor_id"
                  )
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
                  TableRow(
                    v-for="link in access?.teams ?? []"
                    :key="link.team_id"
                  )
                    TableCell
                      .flex.flex-col
                        span.font-medium {{ link.team_name }}
                        span.text-muted-foreground.text-xs(
                          v-if="link.inherited_role"
                        )
                          | {{ $t("projectSettings.alsoInherited", { role: $t(PROJECT_ROLE_KEY[link.inherited_role]) }) }}
                          RouterLink.text-primary.ml-1(
                            :to="{ name: 'project', params: { slug: link.inherited_from_project_slug }, query: { tab: 'settings', section: 'access' } }"
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
                Button(
                  @click="inviteTeam"
                  :disabled="!newTeamId || accessBusy"
                )
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
                          :to="{ name: 'project', params: { slug: link.inherited_from_project_slug }, query: { tab: 'settings', section: 'access' } }"
                        ) {{ link.inherited_from_project_name }}

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

    TabsContent(value="danger")
      Card.border-destructive
        CardHeader
          CardTitle {{ $t("projectSettings.dangerZone") }}
          CardDescription
            | {{ $t("projectSettings.danger.description") }}
        CardContent
          .flex.items-center.justify-between.gap-4
            .flex.flex-col
              span.text-sm.font-medium {{ project?.archivedAt ? "Archived" : "Archive this project" }}
              span.text-muted-foreground.text-xs
                | {{ project?.archivedAt ? "Un-archive to show it in the sidebar and task lists again." : "The project keeps its data and can be restored at any time." }}
            Button(
              v-if="project && !project.archivedAt"
              variant="destructive"
              @click="archive"
            )
              ArchiveIcon(data-icon="inline-start")
              span {{ $t("tags.archive") }}
            Button(v-else-if="project" variant="outline" @click="unarchive")
              ArchiveRestoreIcon(data-icon="inline-start")
              span {{ $t("tasksTable.unarchive") }}
</template>

<script setup lang="ts">
import ActorCombobox from "@/components/ActorCombobox.vue";
import PictureUpload from "@/components/attachments/PictureUpload.vue";
import PromptEditor from "@/components/PromptEditor.vue";
import ProjectPluginsPanel from "@/components/settings/ProjectPluginsPanel.vue";
import TagsPanel from "@/components/settings/TagsPanel.vue";
import TaskTemplatesPanel from "@/components/templates/TaskTemplatesPanel.vue";
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
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { withProgress } from "@/composables/useProgress";
import { useProjects } from "@/composables/useProjects";
import { useSettings } from "@/composables/useSettings";
import {
  PROJECT_ROLE_KEY,
  projectAccessApi,
  type ParentInvitation,
  type ParentOption,
  type ProjectAccess,
  type ProjectHierarchy,
  type ProjectMemberGrant,
  type ProjectRole,
  type ProjectTeamLink,
} from "@/lib/projectAccess";
import { agency } from "@/lib/sdk";
import { teamApi, type Team } from "@/lib/team";
import { PromptScope, type Attachment, type Project } from "@cherrypi/sdk";
import {
  ArchiveIcon,
  ArchiveRestoreIcon,
  CheckIcon,
  PlugIcon,
  PlusIcon,
  Trash2Icon,
} from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, useRoute, useRouter } from "vue-router";
import { toast } from "vue3-toastify";

const { t } = useI18n();

const props = defineProps<{ slug: string; tab?: string }>();
const emit = defineEmits<{ changed: [Project] }>();
const route = useRoute();
const router = useRouter();
const { refresh: refreshProjects } = useProjects();

const TABS = ["general", "tasks", "templates", "plugins", "access", "danger"];
const tab = ref(
  typeof props.tab === "string" && TABS.includes(props.tab)
    ? props.tab
    : typeof route.query.section === "string" &&
        TABS.includes(route.query.section)
      ? route.query.section
      : "general",
);
const project = ref<Project>();
const access = ref<ProjectAccess>();
const hierarchy = ref<ProjectHierarchy>();
const newParentId = ref("");
const newChildId = ref("");
const hierarchyBusy = ref(false);
const accessBusy = ref(false);
const allTeams = ref<Team[]>([]);
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
const logo = computed(() => toSvg(project.value?.id ?? "project", 48));

const name = ref("");
const prefix = ref("");
const description = ref("");
const saving = ref(false);
const savingPrefix = ref(false);
const savingArchive = ref(false);
const error = ref("");

const MAX_ARCHIVE_DAYS = 3650;
const archiveMode = ref<"inherit" | "custom" | "never">("inherit");
const archiveCustomDays = ref(30);
const { settings: orgSettings, loadSettings } = useSettings();
const canManageProject = computed(
  () => access.value?.can_manage_access ?? false,
);
const orgArchiveDays = computed(
  () => orgSettings.value?.taskAutoArchive?.days || 30,
);
const orgArchiveEnabled = computed(
  () => orgSettings.value?.taskAutoArchive?.enabled ?? true,
);
const inheritLabel = computed(() =>
  orgArchiveEnabled.value
    ? t("projectSettings.archive.inheritDays", { days: orgArchiveDays.value })
    : t("projectSettings.archive.inheritOff"),
);
const archiveCustomDaysValid = computed(() => {
  const days = Number(archiveCustomDays.value);
  return Number.isInteger(days) && days >= 1 && days <= MAX_ARCHIVE_DAYS;
});
const archivePolicyValid = computed(
  () => archiveMode.value !== "custom" || archiveCustomDaysValid.value,
);
const archiveModeHint = computed(() => {
  switch (archiveMode.value) {
    case "custom":
      return t("projectSettings.archive.customHint");
    case "never":
      return t("projectSettings.archive.neverHint");
    default:
      return t("projectSettings.archive.inheritHint");
  }
});

watch(archiveMode, (mode) => {
  if (mode === "custom" && !archiveCustomDaysValid.value) {
    archiveCustomDays.value = orgArchiveDays.value;
  }
});

const directMembers = computed(() =>
  (access.value?.members ?? []).filter((grant) => grant.kind !== "plugin"),
);
const pluginMembers = computed(() =>
  (access.value?.members ?? []).filter((grant) => grant.kind === "plugin"),
);

// Teams that are neither attached nor already invited, so the picker never

const invitableTeams = computed(() => {
  const linked = new Set((access.value?.teams ?? []).map((t) => t.team_id));
  const pending = new Set(
    (access.value?.invitations ?? [])
      .filter((i) => i.status === "pending")
      .map((i) => i.team_id),
  );
  return allTeams.value.filter((t) => !linked.has(t.id) && !pending.has(t.id));
});

const parentOptions = computed<ComboboxOption[]>(() =>
  (hierarchy.value?.parent_options ?? []).map((option) => ({
    value: option.project.id,
    label: option.project.name,
    description: option.project.prefix
      ? `#${option.project.prefix}`
      : undefined,
  })),
);

const childOptions = computed<ComboboxOption[]>(() =>
  (hierarchy.value?.child_options ?? []).map((option) => ({
    value: option.project.id,
    label: option.project.name,
    description: option.project.prefix
      ? `#${option.project.prefix}`
      : undefined,
  })),
);

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

const pendingParentInvitations = computed(() =>
  (hierarchy.value?.outgoing_invitations ?? []).filter(
    (invitation) => invitation.status === "pending",
  ),
);

const selectedParentOption = computed(() =>
  hierarchy.value?.parent_options.find(
    (option) => option.project.id === newParentId.value,
  ),
);
const selectedChildOption = computed(() =>
  hierarchy.value?.child_options.find(
    (option) => option.project.id === newChildId.value,
  ),
);
const selectedParentTip = computed(() =>
  linkTip(
    selectedParentOption.value,
    selectedParentOption.value?.project.name ?? "the parent",
    project.value?.name ?? "this project",
  ),
);
const selectedChildTip = computed(() =>
  linkTip(
    selectedChildOption.value,
    project.value?.name ?? "this project",
    selectedChildOption.value?.project.name ?? "the child",
  ),
);

function linkTip(
  option: ParentOption | undefined,
  parentName: string,
  childName: string,
): string {
  if (!option) return "";
  if (option.shared_owner_teams.length) {
    const teams = option.shared_owner_teams
      .map((team) => team.team_name)
      .join(", ");
    return `You're a member of ${teams}, which owns both ${parentName} and ${childName}, so you may link them directly.`;
  }
  if (option.can_invite)
    return `The ${childName} owners will need to accept this invitation.`;
  return "";
}

function onPictureChanged(picture: Attachment | undefined) {
  if (project.value) {
    project.value.picture = picture;
    emit("changed", project.value);
  }
}

async function load() {
  try {
    const found = await agency.getProject(props.slug);
    project.value = found;
    name.value = found.name;
    prefix.value = found.prefix ?? "";
    description.value = found.description;
    applyArchivePolicy(found);
    error.value = "";
    await Promise.all([loadAccess(), loadHierarchy()]);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

async function loadAccess() {
  if (!project.value) return;
  try {
    const [payload, teamList] = await Promise.all([
      projectAccessApi.access(project.value.id),
      teamApi.list(),
    ]);
    access.value = payload;
    allTeams.value = teamList.teams;
  } catch {
    access.value = undefined;
  }
}

async function loadHierarchy() {
  if (!project.value) return;
  try {
    hierarchy.value = await projectAccessApi.hierarchy(project.value.id);
  } catch {
    hierarchy.value = undefined;
  }
}

async function attachToParent() {
  if (!project.value || !newParentId.value) return;
  hierarchyBusy.value = true;
  try {
    const result = await projectAccessApi.inviteParent(
      project.value.id,
      newParentId.value,
    );
    toast.success(
      result.direct
        ? t("projectSettings.toasts.linked")
        : t("projectSettings.toasts.invitationSent"),
    );
    newParentId.value = "";
    await Promise.all([load(), refreshProjects()]);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    hierarchyBusy.value = false;
  }
}

async function addChildProject() {
  if (!project.value || !newChildId.value) return;
  hierarchyBusy.value = true;
  try {
    const result = await projectAccessApi.inviteParent(
      newChildId.value,
      project.value.id,
    );
    toast.success(
      result.direct
        ? t("projectSettings.toasts.linked")
        : t("projectSettings.toasts.invitationSent"),
    );
    newChildId.value = "";
    await Promise.all([load(), refreshProjects()]);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    hierarchyBusy.value = false;
  }
}

async function detachParentProject() {
  if (!project.value) return;
  if (!window.confirm(t("projectSettings.toasts.detachQuestion"))) return;
  hierarchyBusy.value = true;
  try {
    await projectAccessApi.detachParent(project.value.id);
    toast.success(t("projectSettings.toasts.detached"));
    await Promise.all([load(), refreshProjects()]);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    hierarchyBusy.value = false;
  }
}

async function detachChildProject(child: { id: string; name: string }) {
  if (!window.confirm(`Detach ${child.name} from this project?`)) return;
  hierarchyBusy.value = true;
  try {
    await projectAccessApi.detachParent(child.id);
    toast.success(t("projectSettings.toasts.detached"));
    await Promise.all([load(), refreshProjects()]);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    hierarchyBusy.value = false;
  }
}

async function respondParentInvitation(
  invitation: ParentInvitation,
  approve: boolean,
) {
  hierarchyBusy.value = true;
  try {
    await projectAccessApi.respondParentInvitation(invitation.id, approve);
    toast.success(
      approve
        ? t("projectSettings.toasts.linked")
        : t("projectSettings.toasts.invitationDeclined"),
    );
    await Promise.all([load(), refreshProjects()]);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    hierarchyBusy.value = false;
  }
}

async function revokeParentInvitation(invitation: ParentInvitation) {
  hierarchyBusy.value = true;
  try {
    await projectAccessApi.revokeParentInvitation(invitation.id);
    await loadHierarchy();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    hierarchyBusy.value = false;
  }
}

async function addMemberGrant() {
  if (!project.value || !newMemberUsername.value.trim()) return;
  accessBusy.value = true;
  try {
    await projectAccessApi.addMember(
      project.value.id,
      newMemberUsername.value.trim(),
      newMemberRole.value,
    );
    newMemberUsername.value = "";
    toast.success(t("projectSettings.toasts.grantAdded"));
    await loadAccess();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    accessBusy.value = false;
  }
}

async function changeMemberRole(grant: ProjectMemberGrant, role: unknown) {
  if (
    !project.value ||
    typeof role !== "string" ||
    !ROLES.includes(role as ProjectRole)
  )
    return;
  accessBusy.value = true;
  try {
    await projectAccessApi.setMemberRole(
      project.value.id,
      grant.actor_id,
      role as ProjectRole,
    );
    await loadAccess();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    accessBusy.value = false;
  }
}

async function removeMemberGrant(grant: ProjectMemberGrant) {
  if (!project.value) return;
  if (!window.confirm(`Remove ${grant.username} from this project?`)) return;
  accessBusy.value = true;
  try {
    await projectAccessApi.removeMember(project.value.id, grant.actor_id);
    toast.success(t("projectSettings.toasts.grantRemoved"));
    await loadAccess();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    accessBusy.value = false;
  }
}

async function inviteTeam() {
  if (!project.value || !newTeamId.value) return;
  accessBusy.value = true;
  try {
    const result = await projectAccessApi.inviteTeam(
      project.value.id,
      Number(newTeamId.value),
      newTeamRole.value,
    );
    newTeamId.value = "";
    toast.success(
      result.team
        ? t("projectSettings.toasts.teamAttached")
        : t("projectSettings.toasts.teamInvited"),
    );
    await loadAccess();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    accessBusy.value = false;
  }
}

async function changeTeamRole(link: ProjectTeamLink, role: unknown) {
  if (
    !project.value ||
    typeof role !== "string" ||
    !ROLES.includes(role as ProjectRole)
  )
    return;
  accessBusy.value = true;
  try {
    await projectAccessApi.setTeamRole(
      project.value.id,
      link.team_id,
      role as ProjectRole,
    );
    await loadAccess();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    accessBusy.value = false;
  }
}

async function removeTeamLink(link: ProjectTeamLink) {
  if (!project.value) return;
  if (!window.confirm(`Detach ${link.team_name} from this project?`)) return;
  accessBusy.value = true;
  try {
    await projectAccessApi.removeTeam(project.value.id, link.team_id);
    toast.success(t("projectSettings.toasts.teamDetached"));
    await loadAccess();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    accessBusy.value = false;
  }
}

async function revokeInvite(invitation: { id: number }) {
  if (!project.value) return;
  accessBusy.value = true;
  try {
    await projectAccessApi.revokeInvitation(project.value.id, invitation.id);
    await loadAccess();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    accessBusy.value = false;
  }
}

async function saveGeneral() {
  if (!project.value) return;
  saving.value = true;
  try {
    project.value = await withProgress(() =>
      agency.updateProject(project.value!.id, {
        name: name.value.trim(),
        description: description.value.trim(),
        prefix: prefix.value.trim(),
      }),
    );
    prefix.value = project.value.prefix ?? "";
    emit("changed", project.value);
    toast.success(t("projectSettings.toasts.updated"));
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}

function applyArchivePolicy(found: Project) {
  const days = found.archiveAfterDays;
  if (days === undefined) {
    archiveMode.value = "inherit";
  } else if (days === 0) {
    archiveMode.value = "never";
  } else {
    archiveMode.value = "custom";
    archiveCustomDays.value = days;
  }
}

async function saveArchivePolicy() {
  if (!project.value) return;
  const days =
    archiveMode.value === "inherit"
      ? null
      : archiveMode.value === "never"
        ? 0
        : Number(archiveCustomDays.value);
  if (days !== null && days !== 0 && !archiveCustomDaysValid.value) return;
  savingArchive.value = true;
  try {
    project.value = await withProgress(() =>
      agency.setProjectArchiveAfterDays(project.value!.id, days),
    );
    applyArchivePolicy(project.value);
    emit("changed", project.value);
    toast.success(t("projectSettings.archive.saved"));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    savingArchive.value = false;
  }
}

async function savePrefix() {
  if (!project.value) return;
  savingPrefix.value = true;
  try {
    project.value = await withProgress(() =>
      agency.updateProject(project.value!.id, {
        name: name.value.trim(),
        description: description.value.trim(),
        prefix: prefix.value.trim(),
      }),
    );
    prefix.value = project.value.prefix ?? "";
    emit("changed", project.value);
    toast.success(t("projectSettings.toasts.prefixUpdated"));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    savingPrefix.value = false;
  }
}

async function archive() {
  if (!project.value) return;
  try {
    const { offlineAgents } = await withProgress(() =>
      agency.archiveProject(project.value!.id),
    );
    project.value = await agency.getProject(project.value.id);
    emit("changed", project.value);
    if (offlineAgents.length) {
      toast.warning(
        `Project archived, but ${offlineAgents.length} offline agent(s) were skipped`,
      );
    } else {
      toast.success(t("admin.toasts.projectArchived"));
    }
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function unarchive() {
  if (!project.value) return;
  try {
    project.value = await withProgress(() =>
      agency.unarchiveProject(project.value!.id),
    );
    emit("changed", project.value);
    toast.success(t("admin.toasts.projectUnarchived"));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

watch(tab, (value) => {
  const next = { ...route.query };
  if (value === "general") delete next.section;
  else next.section = value;
  void router.replace({ query: next });
});

onMounted(() => {
  void loadSettings();
  void load();
});
</script>

<i18n lang="yaml">
en:
  projectSettings:
    dangerZone: "Danger zone"
    general:
      description: "The project's picture, name, description, and slug."
      picture: "Picture"
      slugHint: "The slug is set at creation and used in the project's URL."
    hierarchy:
      description: "Projects form a tree. Anyone who can view a parent may also view every descendant, and tasks may reference tasks in descendant or sibling projects. Root projects are not siblings and cannot reference each other's tasks."
      parent: "Parent project"
      detach: "Detach"
      topLevel: "This is a top-level project."
      accept: "Accept"
      attach: "Attach to a parent project"
      chooseParent: "Choose a parent"
      subprojects: "Subprojects"
      none: "None."
      addSubproject: "Add a subproject"
      chooseProject: "Choose a project"
      subprojectAria: "Subproject"
      pendingInvitations: "Pending invitations"
    taskIds:
      title: "Task ids"
      prefix: "Task prefix"
      placeholder: "MYPROJECT"
      save: "Save prefix"
      description2: "Task ids read {prefix}-1. Changing the prefix renames this project's existing tasks."
    archive:
      title: "Automatic archive"
      description: "Completed and cancelled tasks leave the active lists after a while, and their agent-side sessions are cleaned up. The organization sets the default policy; a project owner may override it here."
      mode: "When to archive"
      inheritDays: "Organization default ({days} days)"
      inheritOff: "Organization default (off)"
      inheritHint: "Follows the organization's policy, including when it is switched off."
      custom: "After a custom delay"
      customHint: "This project's own delay, regardless of the organization's policy."
      never: "Never"
      neverHint: "Closed tasks stay in the active lists even when the organization archives its own."
      days: "Days after close-out"
      ownersOnly: "Only project owners can change this policy."
      saved: "Archive policy saved"
    prompt:
      title: "Project prompt"
      description: "Injected into each new task's system prompt, after the agency and agent prompts. Edits affect future tasks only."
      placeholder: "Instructions for tasks in this project…"
    tags:
      description: "The tags task creators can pick inside this project."
    templates:
      description: "Reusable Jinja prompts saved per project. A task created from a template records the version and the argument values it used."
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
    danger:
      description: "Archiving hides the project and its tasks from the active lists. Un-archiving brings them back. Nothing is deleted."
    toasts:
      linked: "Linked"
      invitationSent: "Invitation sent"
      detachQuestion: "Detach this project from its parent?"
      detached: "Detached"
      invitationDeclined: "Invitation declined"
      grantAdded: "Grant added"
      grantRemoved: "Grant removed"
      teamAttached: "Team attached"
      teamInvited: "Team invited; its owner must approve"
      teamDetached: "Team detached"
      updated: "Project updated"
      prefixUpdated: "Task prefix updated"
    tabs:
      templates: "Templates"
    invitationFrom: "Invitation from {project}"
    invitedBy: "Invited by {'@'}{username}"
    yourRole: "Your role: {role}"
    pluginGrant: "plugin · {'@'}{username}"
    alsoInherited: "Also inherited as {role} from"
    teamAlreadyInherited: "This team already has inherited {role} access from {project}. Adding it directly"
ru:
  projectSettings:
    dangerZone: "Опасная зона"
    general:
      description: "Изображение, название, описание и слаг проекта."
      picture: "Изображение"
      slugHint: "Слаг задаётся при создании и используется в URL проекта."
    hierarchy:
      description: "Проекты образуют дерево. Тот, кто видит родительский проект, видит и все дочерние, а задачи могут ссылаться на задачи в дочерних или соседних проектах. Корневые проекты не являются соседними и не могут ссылаться на задачи друг друга."
      parent: "Родительский проект"
      detach: "Отсоединить"
      topLevel: "Это проект верхнего уровня."
      accept: "Принять"
      attach: "Присоединить к родительскому проекту"
      chooseParent: "Выберите родителя"
      subprojects: "Дочерние проекты"
      none: "Нет."
      addSubproject: "Добавить дочерний проект"
      chooseProject: "Выберите проект"
      subprojectAria: "Дочерний проект"
      pendingInvitations: "Ожидающие приглашения"
    taskIds:
      title: "Идентификаторы задач"
      prefix: "Префикс задач"
      placeholder: "MYPROJECT"
      save: "Сохранить префикс"
      description2: "Идентификаторы задач выглядят как {prefix}-1. Смена префикса переименовывает существующие задачи проекта."
    archive:
      title: "Автоматический архив"
      description: "Завершённые и отменённые задачи со временем покидают активные списки, а их сессии на агентах очищаются. По умолчанию действует правило агентства; владелец проекта может переопределить его здесь."
      mode: "Когда архивировать"
      inheritDays: "Как в агентстве ({days} дн.)"
      inheritOff: "Как в агентстве (выключено)"
      inheritHint: "Следует правилу агентства, в том числе когда оно выключено."
      custom: "Через свой срок"
      customHint: "Собственный срок проекта, независимо от правила агентства."
      never: "Никогда"
      neverHint: "Закрытые задачи остаются в активных списках, даже если агентство архивирует свои."
      days: "Дней после закрытия"
      ownersOnly: "Это правило могут менять только владельцы проекта."
      saved: "Правило архивации сохранено"
    prompt:
      title: "Промпт проекта"
      description: "Добавляется в системный промпт каждой новой задачи после промптов агентства и агента. Изменения влияют только на будущие задачи."
      placeholder: "Инструкции для задач этого проекта…"
    tags:
      description: "Теги, которые создатели задач могут выбрать в этом проекте."
    templates:
      description: "Многоразовые Jinja-промпты, сохранённые для проекта. Созданная из шаблона задача запоминает версию и значения аргументов."
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
    danger:
      description: "Архивация скрывает проект и его задачи из активных списков. Разархивация возвращает их. Ничего не удаляется."
    toasts:
      linked: "Связано"
      invitationSent: "Приглашение отправлено"
      detachQuestion: "Отсоединить проект от родительского?"
      detached: "Отсоединено"
      invitationDeclined: "Приглашение отклонено"
      grantAdded: "Разрешение добавлено"
      grantRemoved: "Разрешение удалено"
      teamAttached: "Команда подключена"
      teamInvited: "Команда приглашена; требуется одобрение её владельца"
      teamDetached: "Команда отключена"
      updated: "Проект обновлён"
      prefixUpdated: "Префикс задач обновлён"
    tabs:
      templates: "Шаблоны"
    invitationFrom: "Приглашение от {project}"
    invitedBy: "Приглашение от {'@'}{username}"
    yourRole: "Ваша роль: {role}"
    pluginGrant: "плагин · {'@'}{username}"
    alsoInherited: "Также унаследовано как {role} из"
    teamAlreadyInherited: "У этой команды уже есть унаследованный доступ {role} из {project}. Прямое подключение"
</i18n>
