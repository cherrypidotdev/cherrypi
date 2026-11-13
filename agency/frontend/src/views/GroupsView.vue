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
.mx-auto.flex.w-full.max-w-6xl.flex-col.gap-6.p-6
  .flex.items-start.justify-between.gap-4
    .flex.min-w-0.flex-col
      h1.text-lg.font-semibold {{ $t("groups.title") }}
      p.text-muted-foreground.text-sm {{ $t("groups.description") }}
    Button(v-if="canCreate" @click="createOpen = !createOpen")
      PlusIcon(data-icon="inline-start")
      span {{ $t("groups.create") }}

  Card(v-if="createOpen && canCreate")
    CardHeader
      CardTitle {{ $t("groups.create") }}
      CardDescription {{ $t("groups.createHint") }}
    CardContent
      form.flex.flex-col.gap-4(@submit.prevent="create")
        FieldGroup
          .grid.grid-cols-1.gap-3(class="sm:grid-cols-2")
            Field
              FieldLabel(for="group-name") {{ $t("common.name") }}
              Input#group-name(v-model="name" maxlength="100" required)
            Field
              FieldLabel(for="group-slug") {{ $t("groups.slug") }}
              Input#group-slug(v-model="slug" maxlength="100")
          Field
            FieldLabel(for="group-description") {{ $t("common.description") }}
            Input#group-description(v-model="description")
        p.text-destructive.text-sm(v-if="createError") {{ createError }}
        .flex.justify-end
          Button(type="submit" :disabled="creating || !name.trim()")
            Spinner(v-if="creating" data-icon="inline-start")
            CheckIcon(v-else data-icon="inline-start")
            span {{ $t("common.create") }}

  p.text-destructive.text-sm(v-if="error") {{ error }}

  Card
    CardHeader
      CardTitle {{ $t("groups.title") }}
      CardDescription {{ $t("groups.listHint") }}
    CardContent
      Empty(v-if="!loading && !groups.length")
        EmptyTitle {{ $t("groups.empty") }}
        EmptyDescription {{ $t("groups.emptyHint") }}
      Table(v-else)
        TableHeader
          TableRow
            TableHead {{ $t("common.name") }}
            TableHead {{ $t("groups.memberColumn") }}
            TableHead {{ $t("common.status") }}
            TableHead
        TableBody
          TableRowsSkeleton(
            v-if="loading && !groups.length"
            :widths="['w-40', 'w-16', 'w-20', 'w-16']"
          )
          TableRow(v-for="group in sortedGroups" :key="String(group.id)")
            TableCell
              RouterLink.font-medium(
                class="hover:text-primary"
                :to="{ name: 'group-overview', params: { slug: group.slug } }"
              ) {{ group.name }}
              p.text-muted-foreground.text-xs(v-if="group.description")
                | {{ group.description }}
            TableCell.text-muted-foreground {{ group.memberCount }}
            TableCell
              Badge(v-if="group.archivedAt" variant="outline")
                | {{ $t("groups.archived") }}
              Badge(v-else variant="secondary") {{ $t("groups.active") }}
            TableCell
              .flex.justify-end
                RouterLink.text-muted-foreground.text-xs(
                  class="hover:text-foreground"
                  :to="{ name: 'group-overview', params: { slug: group.slug } }"
                ) {{ $t("common.open") }}
</template>

<script setup lang="ts">
import TableRowsSkeleton from "@/components/TableRowsSkeleton.vue";
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
import { Spinner } from "@/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAgentGroups } from "@/composables/useAgentGroups";
import { currentAccount } from "@/lib/account";
import { CheckIcon, PlusIcon } from "@lucide/vue";
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { toast } from "vue3-toastify";

const { t } = useI18n();
const router = useRouter();
const { groups, loading, error, loadAgentGroups, createGroup } =
  useAgentGroups();

const createOpen = ref(false);
const name = ref("");
const slug = ref("");
const description = ref("");
const creating = ref(false);
const createError = ref("");

const canCreate = computed(() => currentAccount.value?.kind !== "agent");

const sortedGroups = computed(() =>
  [...groups.value].sort((a, b) => a.name.localeCompare(b.name)),
);

onMounted(() => {
  void loadAgentGroups(true);
});

async function create() {
  creating.value = true;
  createError.value = "";
  try {
    const group = await createGroup({
      name: name.value.trim(),
      slug: slug.value.trim() || undefined,
      description: description.value.trim(),
    });
    toast.success(t("groups.toasts.created", { name: group.name }));
    await router.push({
      name: "group-overview",
      params: { slug: group.slug },
    });
  } catch (err) {
    createError.value = err instanceof Error ? err.message : String(err);
  } finally {
    creating.value = false;
  }
}
</script>

<i18n lang="yaml">
en:
  groups:
    title: Agent groups
    description: A group gives its member machines a shared set of versioned prompts. It never changes assignment, scheduling or project access.
    listHint: Every group is visible to everyone; only group owners may edit its prompts or membership.
    create: Create a group
    createHint: You become the group's first owner. Its slug is the page address and can be changed later.
    slug: Slug
    memberColumn: Members
    archived: Archived
    active: Active
    empty: No agent groups yet
    emptyHint: Create a group to give identical machines a shared prompt.
    toasts:
      created: "Created the {name} group"
ru:
  groups:
    title: Группы агентов
    description: Группа даёт машинам-участникам общий набор версионируемых промптов. Она не меняет назначение, планирование или доступ к проектам.
    listHint: Все группы видны всем; редактировать промпты и состав может только владелец группы.
    create: Создать группу
    createHint: Вы становитесь первым владельцем группы. Её слаг — это адрес страницы, его можно изменить позже.
    slug: Слаг
    memberColumn: Участники
    archived: В архиве
    active: Активна
    empty: Групп агентов пока нет
    emptyHint: Создайте группу, чтобы дать одинаковым машинам общий промпт.
    toasts:
      created: "Группа {name} создана"
</i18n>
