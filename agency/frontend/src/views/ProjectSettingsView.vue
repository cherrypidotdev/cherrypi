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
      TabsTrigger(value="general") General
      TabsTrigger(value="tasks") Tasks
      TabsTrigger(value="access") Access
      TabsTrigger(value="danger") Danger zone

    TabsContent(value="general")
      Card
        CardHeader
          CardTitle General
          CardDescription The project's picture, name, description, and slug.
        CardContent.flex.flex-col.gap-6
          form.flex.flex-col.gap-4(@submit.prevent="saveGeneral")
            FieldGroup
              Field
                FieldLabel Picture
                PictureUpload(
                  :picture="project?.picture"
                  :project-id="project?.id"
                  :fallback="logo"
                  @changed="onPictureChanged"
                )
              Field
                FieldLabel(for="project-name") Name
                Input#project-name(v-model="name")
              Field
                FieldLabel(for="project-description") Description
                Textarea#project-description(v-model="description")
              Field
                FieldLabel(for="project-slug") Slug
                Input#project-slug(disabled :model-value="project?.slug ?? ''")
                FieldDescription
                  | The slug is set at creation and used in the project's URL.
            .flex.justify-end
              Button(type="submit" :disabled="saving || !name.trim()")
                Spinner(v-if="saving" data-icon="inline-start")
                CheckIcon(v-else data-icon="inline-start")
                span Save

        Card
          CardHeader
            CardTitle Project hierarchy
            CardDescription
              | Projects form a tree. Anyone who can view a parent may also view
              | every descendant, and tasks may reference tasks in descendant or
              | sibling projects.
          CardContent.flex.flex-col.gap-5
            template(v-if="hierarchy")
              .flex.flex-col.gap-1(v-if="hierarchy.parent")
                span.text-sm.font-medium Parent project
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
                  ) Detach
              p.text-muted-foreground.text-sm(v-else)
                | This is a top-level project.

              .flex.flex-wrap.items-center.justify-between.gap-2.rounded-md.border.p-3(
                v-if="hierarchy.incoming_invitation"
              )
                .flex.flex-col
                  span.text-sm.font-medium
                    | Invitation from {{ hierarchy.incoming_invitation.parent_project_name }}
                  span.text-muted-foreground.text-xs(
                    v-if="hierarchy.incoming_invitation.invited_by_username"
                  )
                    | Invited by @{{ hierarchy.incoming_invitation.invited_by_username }}
                .flex.gap-2(v-if="hierarchy.can_respond")
                  Button(
                    size="sm"
                    @click="respondParentInvitation(hierarchy.incoming_invitation, true)"
                    :disabled="hierarchyBusy"
                  ) Accept
                  Button(
                    variant="outline"
                    size="sm"
                    @click="respondParentInvitation(hierarchy.incoming_invitation, false)"
                    :disabled="hierarchyBusy"
                  ) Decline

              .flex.flex-col.gap-2(v-if="hierarchy.parent_options.length")
                span.text-sm.font-medium Attach to a parent project
                .flex.flex-wrap.gap-2
                  Select(v-model="newParentId")
                    SelectTrigger.w-64
                      SelectValue(placeholder="Choose a parent")
                    SelectContent
                      SelectGroup
                        SelectItem(
                          v-for="option in hierarchy.parent_options"
                          :key="option.project.id"
                          :value="option.project.id"
                        ) {{ option.project.name }}
                  Button(
                    @click="attachToParent"
                    :disabled="!newParentId || hierarchyBusy"
                  ) {{ selectedParentOption?.can_link_directly ? "Link" : "Request" }}
                p.text-muted-foreground.text-xs(v-if="selectedParentTip")
                  | {{ selectedParentTip }}

              Separator

              .flex.flex-col.gap-2
                span.text-sm.font-medium Subprojects
                p.text-muted-foreground.text-sm(
                  v-if="!hierarchy.children.length"
                )
                  | None.
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
                  ) Detach

              .flex.flex-col.gap-2(v-if="hierarchy.child_options.length")
                span.text-sm.font-medium Add a subproject
                .flex.flex-wrap.gap-2
                  Select(v-model="newChildId")
                    SelectTrigger.w-64
                      SelectValue(placeholder="Choose a project")
                    SelectContent
                      SelectGroup
                        SelectItem(
                          v-for="option in hierarchy.child_options"
                          :key="option.project.id"
                          :value="option.project.id"
                        ) {{ option.project.name }}
                  Button(
                    @click="addChildProject"
                    :disabled="!newChildId || hierarchyBusy"
                  ) {{ selectedChildOption?.can_link_directly ? "Link" : "Request" }}
                p.text-muted-foreground.text-xs(v-if="selectedChildTip")
                  | {{ selectedChildTip }}

              .flex.flex-col.gap-2(v-if="pendingParentInvitations.length")
                span.text-sm.font-medium Pending invitations
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
                  ) Revoke

    TabsContent(value="tasks")
      .flex.flex-col.gap-6
        Card
          CardHeader
            CardTitle Task ids
            CardDescription
              | Task ids read {{ prefix.trim() || "MYPROJECT" }}-1. Changing the
              | prefix renames this project's existing tasks.
          CardContent
            form.flex.items-end.gap-2(@submit.prevent="savePrefix")
              Field.flex-1
                FieldLabel(for="project-prefix") Task prefix
                Input#project-prefix(
                  v-model="prefix"
                  placeholder="MYPROJECT"
                  autocapitalize="characters"
                )
              Button(type="submit" :disabled="savingPrefix || !prefix.trim()")
                Spinner(v-if="savingPrefix" data-icon="inline-start")
                CheckIcon(v-else data-icon="inline-start")
                span Save prefix

        Card
          CardHeader
            CardTitle Project prompt
            CardDescription
              | Injected into each new task's system prompt, after the agency and
              | agent prompts. Edits affect future tasks only.
          CardContent
            PromptEditor(
              placeholder="Instructions for tasks in this project…"
              :scope="PromptScope.PROJECT"
              :project-id="project?.id"
            )

        Card
          CardHeader
            CardTitle Tags
            CardDescription The tags task creators can pick inside this project.
          CardContent
            TagsPanel(:project-id="project?.id ?? ''")

    TabsContent(value="access")
      .flex.flex-col.gap-6
        Card
          CardHeader
            CardTitle Access
            CardDescription
              | Who can reach this project. The effective role is the strongest of a
              | direct grant and the caller's teams; a viewer may view and comment,
              | a member may prompt and edit, an owner may also manage access.
          CardContent.flex.flex-col.gap-6
            .flex.items-center.gap-2
              Badge(variant="secondary")
                | Your role: {{ PROJECT_ROLE_LABEL[access?.role ?? "viewer"] }}
              span.text-muted-foreground.text-xs(
                v-if="access && !access.can_manage_access"
              ) Only project owners can change access.

            Separator

            .flex.flex-col.gap-3
              span.text-sm.font-medium Direct access
              Table
                TableHeader
                  TableRow
                    TableHead Actor
                    TableHead Role
                    TableHead
                TableBody
                  TableRow(
                    v-for="grant in access?.members ?? []"
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
                          ) {{ PROJECT_ROLE_LABEL[role] }}
                      span(v-else) {{ PROJECT_ROLE_LABEL[grant.role] }}
                    TableCell
                      .flex.justify-end(v-if="access?.can_manage_access")
                        Button.text-destructive(
                          variant="outline"
                          size="sm"
                          @click="removeMemberGrant(grant)"
                        )
                          Trash2Icon(data-icon="inline-start")
                          span Remove
                  TableRow(v-if="!(access?.members ?? []).length")
                    TableCell.text-muted-foreground(:colspan="3")
                      | No direct access.
              .flex.items-center.gap-2(v-if="access?.can_manage_access")
                Input.w-56(
                  v-model="newMemberUsername"
                  placeholder="username"
                  autocomplete="off"
                )
                Select(v-model="newMemberRole")
                  SelectTrigger.w-36
                    SelectValue
                  SelectContent
                    SelectItem(v-for="role in ROLES" :key="role" :value="role") {{ PROJECT_ROLE_LABEL[role] }}
                Button(
                  @click="addMemberGrant"
                  :disabled="!newMemberUsername.trim() || accessBusy"
                )
                  PlusIcon(data-icon="inline-start")
                  span Add

            Separator

            .flex.flex-col.gap-3
              span.text-sm.font-medium Teams
              Table
                TableHeader
                  TableRow
                    TableHead Team
                    TableHead Members
                    TableHead Role
                    TableHead
                TableBody
                  TableRow(
                    v-for="link in access?.teams ?? []"
                    :key="link.team_id"
                  )
                    TableCell.font-medium {{ link.team_name }}
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
                            v-for="role in ROLES"
                            :key="role"
                            :value="role"
                          ) {{ PROJECT_ROLE_LABEL[role] }}
                      span(v-else) {{ PROJECT_ROLE_LABEL[link.role] }}
                    TableCell
                      .flex.justify-end(v-if="access?.can_manage_access")
                        Button.text-destructive(
                          variant="outline"
                          size="sm"
                          @click="removeTeamLink(link)"
                        )
                          Trash2Icon(data-icon="inline-start")
                          span Remove
                  TableRow(v-if="!(access?.teams ?? []).length")
                    TableCell.text-muted-foreground(:colspan="4")
                      | No teams attached.
              .flex.items-center.gap-2(v-if="access?.can_manage_access")
                Select(v-model="newTeamId")
                  SelectTrigger.w-56
                    SelectValue(placeholder="Invite a team…")
                  SelectContent
                    SelectGroup
                      SelectItem(
                        v-for="team in invitableTeams"
                        :key="team.id"
                        :value="String(team.id)"
                      ) {{ team.name }}
                Select(v-model="newTeamRole")
                  SelectTrigger.w-36
                    SelectValue
                  SelectContent
                    SelectItem(v-for="role in ROLES" :key="role" :value="role") {{ PROJECT_ROLE_LABEL[role] }}
                Button(
                  @click="inviteTeam"
                  :disabled="!newTeamId || accessBusy"
                )
                  PlusIcon(data-icon="inline-start")
                  span Invite
              p.text-muted-foreground.text-xs(v-if="!invitableTeams.length")
                | No other teams available to invite.

            Separator

            .flex.flex-col.gap-3
              span.text-sm.font-medium Invitations
              Table
                TableHeader
                  TableRow
                    TableHead Team
                    TableHead Role
                    TableHead Status
                    TableHead
                TableBody
                  TableRow(
                    v-for="invitation in access?.invitations ?? []"
                    :key="invitation.id"
                  )
                    TableCell.font-medium {{ invitation.team_name }}
                    TableCell {{ PROJECT_ROLE_LABEL[invitation.role] }}
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
                          span Revoke
                  TableRow(v-if="!(access?.invitations ?? []).length")
                    TableCell.text-muted-foreground(:colspan="4")
                      | No invitations.

    TabsContent(value="danger")
      Card.border-destructive
        CardHeader
          CardTitle Danger zone
          CardDescription
            | Archiving hides the project and its tasks from the active lists.
            | Un-archiving brings them back. Nothing is deleted.
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
              span Archive
            Button(v-else-if="project" variant="outline" @click="unarchive")
              ArchiveRestoreIcon(data-icon="inline-start")
              span Unarchive
</template>

<script setup lang="ts">
import PictureUpload from "@/components/attachments/PictureUpload.vue";
import PromptEditor from "@/components/PromptEditor.vue";
import TagsPanel from "@/components/settings/TagsPanel.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
  SelectGroup,
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
import {
  PROJECT_ROLE_LABEL,
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
  PlusIcon,
  Trash2Icon,
} from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed, onMounted, ref, watch } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";
import { toast } from "vue3-toastify";

const props = defineProps<{ slug: string; tab?: string }>();
const emit = defineEmits<{ changed: [Project] }>();
const route = useRoute();
const router = useRouter();
const { refresh: refreshProjects } = useProjects();

const TABS = ["general", "tasks", "access", "danger"];
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
const logo = computed(() => toSvg(project.value?.id ?? "project", 48));

const name = ref("");
const prefix = ref("");
const description = ref("");
const saving = ref(false);
const savingPrefix = ref(false);
const error = ref("");

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
    toast.success(result.direct ? "Linked" : "Invitation sent");
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
    toast.success(result.direct ? "Linked" : "Invitation sent");
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
  if (!window.confirm("Detach this project from its parent?")) return;
  hierarchyBusy.value = true;
  try {
    await projectAccessApi.detachParent(project.value.id);
    toast.success("Detached");
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
    toast.success("Detached");
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
    toast.success(approve ? "Linked" : "Invitation declined");
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
    toast.success("Grant added");
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
    toast.success("Grant removed");
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
    await projectAccessApi.inviteTeam(
      project.value.id,
      Number(newTeamId.value),
      newTeamRole.value,
    );
    newTeamId.value = "";
    toast.success("Team invited; its owner must approve");
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
    toast.success("Team detached");
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
    toast.success("Project updated");
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
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
    toast.success("Task prefix updated");
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
      toast.success("Project archived");
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
    toast.success("Project unarchived");
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

onMounted(load);
</script>
