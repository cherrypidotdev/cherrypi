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
      CardTitle {{ $t("settings.tabs.general") }}
      CardDescription {{ $t("projectSettings.general.description") }}
    CardContent.flex.flex-col.gap-6
      form.flex.flex-col.gap-4(@submit.prevent="saveGeneral")
        FieldGroup
          Field
            FieldLabel {{ $t("projectSettings.general.picture") }}
            PictureUpload(
              :picture="page.project?.picture"
              :project-id="page.project?.id"
              :fallback="logo"
              @changed="onPictureChanged"
            )
          Field
            FieldLabel {{ $t("actor.emoji.label") }}
            ActorEmojiField(v-model="emoji")
            FieldDescription {{ $t("actor.emoji.hint") }}
          Field
            FieldLabel(for="project-name") {{ $t("common.name") }}
            Input#project-name(v-model="name")
          Field
            FieldLabel(for="project-description") {{ $t("common.description") }}
            Textarea#project-description(v-model="description")
          Field
            FieldLabel(for="project-slug") {{ $t("projectSettings.general.slug") }}
            Input#project-slug(
              v-model="slug"
              autocapitalize="none"
              autocomplete="off"
            )
            FieldDescription
              | {{ $t("projectSettings.general.slugHint", { prefix: prefixPreview }) }}
          Field
            FieldLabel {{ $t("actor.links.title") }}
            ActorLinksField(v-model="links")
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
      template(v-if="settings.hierarchy")
        .flex.flex-col.gap-1(v-if="settings.hierarchy.parent")
          span.text-sm.font-medium {{ $t("projectSettings.hierarchy.parent") }}
          .flex.items-center.gap-2
            RouterLink.text-primary.font-medium(
              :to="{ name: 'project', params: { slug: settings.hierarchy.parent.slug } }"
            ) {{ settings.hierarchy.parent.name }}
            Button(
              v-if="settings.hierarchy.can_detach"
              variant="outline"
              size="sm"
              @click="detachParentProject"
              :disabled="hierarchyBusy"
            ) {{ $t("projectSettings.hierarchy.detach") }}
        p.text-muted-foreground.text-sm(v-else)
          | {{ $t("projectSettings.hierarchy.topLevel") }}

        .flex.flex-wrap.items-center.justify-between.gap-2.rounded-md.border.p-3(
          v-if="settings.hierarchy.incoming_invitation"
        )
          .flex.flex-col
            span.text-sm.font-medium
              | {{ $t("projectSettings.invitationFrom", { project: settings.hierarchy.incoming_invitation.parent_project_name }) }}
            span.text-muted-foreground.text-xs(
              v-if="settings.hierarchy.incoming_invitation.invited_by_username"
            )
              | {{ $t("projectSettings.invitedBy", { username: settings.hierarchy.incoming_invitation.invited_by_username }) }}
          .flex.gap-2(v-if="settings.hierarchy.can_respond")
            Button(
              size="sm"
              @click="respondParentInvitation(settings.hierarchy.incoming_invitation, true)"
              :disabled="hierarchyBusy"
            ) {{ $t("projectSettings.hierarchy.accept") }}
            Button(
              variant="outline"
              size="sm"
              @click="respondParentInvitation(settings.hierarchy.incoming_invitation, false)"
              :disabled="hierarchyBusy"
            ) {{ $t("extensionUi.decline") }}

        .flex.flex-col.gap-2(v-if="settings.hierarchy.parent_options.length")
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
            v-if="!settings.hierarchy.children.length"
          )
            | {{ $t("projectSettings.hierarchy.none") }}
          .flex.items-center.justify-between.gap-2(
            v-for="child in settings.hierarchy.children"
            :key="child.id"
          )
            RouterLink.text-primary.font-medium(
              :to="{ name: 'project', params: { slug: child.slug } }"
            ) {{ child.name }}
            Button(
              v-if="settings.hierarchy.can_invite_child"
              variant="outline"
              size="sm"
              @click="detachChildProject(child)"
              :disabled="hierarchyBusy"
            ) {{ $t("projectSettings.hierarchy.detach") }}

        .flex.flex-col.gap-2(v-if="settings.hierarchy.child_options.length")
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
</template>

<script setup lang="ts">
import ActorEmojiField from "@/components/actor/ActorEmojiField.vue";
import ActorLinksField from "@/components/actor/ActorLinksField.vue";
import PictureUpload from "@/components/attachments/PictureUpload.vue";
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
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Textarea } from "@/components/ui/textarea";
import { withProgress } from "@/composables/useProgress";
import { useProjects } from "@/composables/useProjects";
import { normalizeTaskPrefix } from "@/lib/project";
import {
  projectAccessApi,
  type ParentInvitation,
  type ParentOption,
} from "@/lib/projectAccess";
import type { ProjectPageState, ProjectSettingsState } from "@/lib/projectPage";
import type { ActorLinkInput, Attachment } from "@cherrypi/sdk";
import { CheckIcon } from "@lucide/vue";
import { toSvg } from "jdenticon";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { RouterLink, useRouter } from "vue-router";
import { toast } from "vue3-toastify";

const props = defineProps<{
  page: ProjectPageState;
  settings: ProjectSettingsState;
}>();

const { t } = useI18n();
const router = useRouter();
const { refresh: refreshProjects, updateProject } = useProjects();

const logo = computed(() => toSvg(props.page.project?.id ?? "project", 48));

const name = ref("");
const description = ref("");
const slug = ref("");
const emoji = ref("");
const links = ref<ActorLinkInput[]>([]);
const saving = ref(false);
const hierarchyBusy = ref(false);
const newParentId = ref("");
const newChildId = ref("");

const prefixPreview = computed(
  () => normalizeTaskPrefix(slug.value.trim()) || "MYPROJECT",
);

watch(
  () => props.page.project,
  (project) => {
    if (!project) return;
    name.value = project.name;
    description.value = project.description ?? "";
    slug.value = project.slug ?? "";
    emoji.value = project.emoji ?? "";
    links.value = [...(project.links ?? [])];
  },
  { immediate: true },
);

const parentOptions = computed<ComboboxOption[]>(() =>
  (props.settings.hierarchy?.parent_options ?? []).map((option) => ({
    value: option.project.id,
    label: option.project.name,
    description: option.project.prefix
      ? `#${option.project.prefix}`
      : undefined,
  })),
);

const childOptions = computed<ComboboxOption[]>(() =>
  (props.settings.hierarchy?.child_options ?? []).map((option) => ({
    value: option.project.id,
    label: option.project.name,
    description: option.project.prefix
      ? `#${option.project.prefix}`
      : undefined,
  })),
);

const pendingParentInvitations = computed(() =>
  (props.settings.hierarchy?.outgoing_invitations ?? []).filter(
    (invitation) => invitation.status === "pending",
  ),
);

const selectedParentOption = computed(() =>
  props.settings.hierarchy?.parent_options.find(
    (option) => option.project.id === newParentId.value,
  ),
);
const selectedChildOption = computed(() =>
  props.settings.hierarchy?.child_options.find(
    (option) => option.project.id === newChildId.value,
  ),
);
const selectedParentTip = computed(() =>
  linkTip(
    selectedParentOption.value,
    selectedParentOption.value?.project.name ?? "the parent",
    props.page.project?.name ?? "this project",
  ),
);
const selectedChildTip = computed(() =>
  linkTip(
    selectedChildOption.value,
    props.page.project?.name ?? "this project",
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
  const project = props.page.project;
  if (!project) return;
  project.picture = picture;
  if (picture) {
    project.emoji = "";
    emoji.value = "";
  }
  props.page.setProject(project);
}

async function saveGeneral() {
  const project = props.page.project;
  if (!project) return;
  const previousSlug = project.slug;
  saving.value = true;
  try {
    const updated = await withProgress(() =>
      updateProject(project.id, {
        name: name.value.trim(),
        description: description.value.trim(),
        slug: slug.value.trim(),
        emoji: emoji.value,
        links: links.value,
      }),
    );
    props.page.setProject(updated);
    slug.value = updated.slug ?? "";
    emoji.value = updated.emoji ?? "";
    links.value = [...(updated.links ?? [])];

    if (updated.slug && updated.slug !== previousSlug) {
      await router.replace({
        name: "project-settings",
        params: { slug: updated.slug },
      });
    }
    toast.success(t("projectSettings.toasts.updated"));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.value = false;
  }
}

async function attachToParent() {
  if (!props.page.project || !newParentId.value) return;
  hierarchyBusy.value = true;
  try {
    const result = await projectAccessApi.inviteParent(
      props.page.project.id,
      newParentId.value,
    );
    toast.success(
      result.direct
        ? t("projectSettings.toasts.linked")
        : t("projectSettings.toasts.invitationSent"),
    );
    newParentId.value = "";
    await Promise.all([
      props.page.reload(),
      props.settings.reloadHierarchy(),
      refreshProjects(),
    ]);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    hierarchyBusy.value = false;
  }
}

async function addChildProject() {
  if (!props.page.project || !newChildId.value) return;
  hierarchyBusy.value = true;
  try {
    const result = await projectAccessApi.inviteParent(
      newChildId.value,
      props.page.project.id,
    );
    toast.success(
      result.direct
        ? t("projectSettings.toasts.linked")
        : t("projectSettings.toasts.invitationSent"),
    );
    newChildId.value = "";
    await Promise.all([props.settings.reloadHierarchy(), refreshProjects()]);
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    hierarchyBusy.value = false;
  }
}

async function detachParentProject() {
  if (!props.page.project) return;
  if (!window.confirm(t("projectSettings.toasts.detachQuestion"))) return;
  hierarchyBusy.value = true;
  try {
    await projectAccessApi.detachParent(props.page.project.id);
    toast.success(t("projectSettings.toasts.detached"));
    await Promise.all([props.settings.reloadHierarchy(), refreshProjects()]);
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
    await Promise.all([props.settings.reloadHierarchy(), refreshProjects()]);
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
    await Promise.all([
      props.page.reload(),
      props.settings.reloadHierarchy(),
      refreshProjects(),
    ]);
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
    await props.settings.reloadHierarchy();
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    hierarchyBusy.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  projectSettings:
    general:
      description: "The project's picture or emoji, name, description, links, and slug."
      picture: "Picture"
      slug: "Slug and task prefix"
      slugHint: "Lowercase in URLs, uppercase in task ids ({prefix}-1). Changing it renames this project's existing tasks."
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
    toasts:
      linked: "Linked"
      invitationSent: "Invitation sent"
      detachQuestion: "Detach this project from its parent?"
      detached: "Detached"
      invitationDeclined: "Invitation declined"
      updated: "Project updated"
    invitationFrom: "Invitation from {project}"
    invitedBy: "Invited by {'@'}{username}"
ru:
  projectSettings:
    general:
      description: "Изображение или эмодзи, название, описание, ссылки и слаг проекта."
      picture: "Изображение"
      slug: "Слаг и префикс задач"
      slugHint: "В нижнем регистре в URL, в верхнем в идентификаторах задач ({prefix}-1). Смена переименовывает существующие задачи проекта."
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
    toasts:
      linked: "Связано"
      invitationSent: "Приглашение отправлено"
      detachQuestion: "Отсоединить проект от родительского?"
      detached: "Отсоединено"
      invitationDeclined: "Приглашение отклонено"
      updated: "Проект обновлён"
    invitationFrom: "Приглашение от {project}"
    invitedBy: "Приглашение от {'@'}{username}"
</i18n>
