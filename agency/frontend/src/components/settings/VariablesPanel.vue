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
.flex.flex-col.gap-4
  Card
    CardHeader
      CardTitle {{ $t("variables.title") }}
      CardAction
        Button(size="sm" @click="openCreate" :disabled="!canManage")
          PlusIcon(data-icon="inline-start")
          span {{ $t("variables.add") }}
      CardDescription
        | {{ $t("variables.description") }}
    CardContent.flex.flex-col.gap-4
      p.text-destructive.text-sm(v-if="error") {{ error }}
      p.text-muted-foreground.text-sm(v-if="!canManage")
        | {{ $t("variables.readOnly") }}
      Alert(v-if="!secretsEnabled" variant="destructive")
        AlertTitle {{ $t("variables.noKeyTitle") }}
        AlertDescription
          | {{ $t("variables.noKeyHint") }}

      Table
        TableHeader
          TableRow
            TableHead {{ $t("variables.columnName") }}
            TableHead {{ $t("variables.columnValue") }}
            TableHead {{ $t("variables.columnNote") }}
            TableHead.w-40 {{ $t("variables.columnUpdated") }}
            TableHead.w-48
        TableBody
          TableRowsSkeleton(
            v-if="loading && !items.length"
            :widths="['w-32', 'w-24', 'w-40', 'w-24', 'w-32']"
          )
          TableRow(v-for="variable in items" :key="variable.name")
            TableCell
              .flex.items-center.gap-2
                span.font-mono.text-sm.font-medium {{ variable.name }}
                Badge(v-if="variable.secret" variant="secondary")
                  LockIcon.size-3
                  span.ml-1 {{ $t("variables.secret") }}
            TableCell
              span.text-muted-foreground.font-mono.text-xs(
                v-if="variable.secret"
                :title="$t('variables.secretValueHint')"
              ) ••••••••
              span.font-mono.text-xs.break-all(v-else) {{ variable.value }}
            TableCell.text-muted-foreground.text-sm
              | {{ variable.note || "—" }}
            TableCell.text-muted-foreground.text-xs
              .flex.flex-col
                span {{ updatedBy(variable) }}
                span {{ formatWhen(variable.updatedAt) }}
            TableCell
              .flex.justify-end.gap-2
                Button(
                  variant="outline"
                  size="sm"
                  @click="openEdit(variable)"
                  :disabled="!canManage"
                )
                  PencilIcon(data-icon="inline-start")
                  span {{ $t("common.edit") }}
                Button(
                  variant="outline"
                  size="sm"
                  :title="variable.secret ? $t('variables.alreadySecret') : $t('variables.promoteHint')"
                  @click="promote(variable)"
                  :disabled="!canManage || variable.secret"
                )
                  LockIcon(data-icon="inline-start")
                  span {{ $t("variables.promote") }}
                Button(
                  variant="outline"
                  size="sm"
                  @click="remove(variable)"
                  :disabled="!canManage"
                )
                  Trash2Icon(data-icon="inline-start")
                  span {{ $t("common.delete") }}
          TableRow(v-if="!loading && !items.length")
            TableCell.text-muted-foreground(colspan="5")
              | {{ $t("variables.empty") }}

  Card(v-if="showEffective")
    CardHeader
      CardTitle {{ $t("variables.effectiveTitle") }}
      CardDescription
        | {{ $t("variables.effectiveDescription") }}
      CardAction
        Select(v-model="effectiveProject")
          SelectTrigger.w-56
            SelectValue(:placeholder="$t('variables.effectiveAllProjects')")
          SelectContent
            SelectItem(:value="NO_PROJECT")
              | {{ $t("variables.effectiveAllProjects") }}
            SelectItem(
              v-for="project in projectOptions"
              :key="project.id"
              :value="project.id"
            )
              | {{ project.name }}
    CardContent.flex.flex-col.gap-3
      p.text-destructive.text-sm(v-if="effectiveError") {{ effectiveError }}
      Table
        TableHeader
          TableRow
            TableHead {{ $t("variables.columnName") }}
            TableHead {{ $t("variables.columnValue") }}
            TableHead {{ $t("variables.effectiveSource") }}
            TableHead {{ $t("variables.effectiveShadowed") }}
        TableBody
          TableRowsSkeleton(
            v-if="effectiveLoading && !effective.length"
            :widths="['w-32', 'w-24', 'w-24', 'w-32']"
          )
          TableRow(v-for="variable in effective" :key="variable.name")
            TableCell
              .flex.items-center.gap-2
                span.font-mono.text-sm.font-medium {{ variable.name }}
                Badge(v-if="variable.secret" variant="secondary")
                  LockIcon.size-3
                  span.ml-1 {{ $t("variables.secret") }}
            TableCell
              span.text-muted-foreground.font-mono.text-xs(
                v-if="variable.secret"
                :title="$t('variables.secretValueHint')"
              )
                | ••••••••
              span.text-muted-foreground.font-mono.text-xs(
                v-else-if="!variable.value"
                :title="$t('variables.inheritedValueHint')"
              )
                | ••••••••
              span.font-mono.text-xs.break-all(v-else) {{ variable.value }}
            TableCell
              Badge(variant="outline") {{ scopeLabel(variable.source) }}
            TableCell.text-muted-foreground.text-xs
              span(v-if="!variable.shadowed.length") —
              span(v-else)
                | {{ variable.shadowed.map(scopeLabel).join(", ") }}
          TableRow(v-if="!effectiveLoading && !effective.length")
            TableCell.text-muted-foreground(colspan="4")
              | {{ $t("variables.effectiveEmpty") }}

  VariableDialog(
    v-model:open="dialogOpen"
    :variable="editing"
    :target="target"
    :secrets-enabled="secretsEnabled"
    :scope-label="scopeLabelText"
    @saved="refresh"
  )
</template>

<script setup lang="ts">
import VariableDialog from "@/components/settings/VariableDialog.vue";
import TableRowsSkeleton from "@/components/TableRowsSkeleton.vue";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { withProgress } from "@/composables/useProgress";
import { useProjects } from "@/composables/useProjects";
import {
  useEffectiveVariables,
  useVariables,
} from "@/composables/useVariables";
import type { Project, Variable, VariableTargetInput } from "@cherrypi/sdk";
import { LockIcon, PencilIcon, PlusIcon, Trash2Icon } from "@lucide/vue";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const { t, locale } = useI18n();

const props = defineProps<{
  target: VariableTargetInput;

  canManage: boolean;

  agentId?: string;
  projects?: Project[];
}>();

const {
  items,
  loading,
  error,
  secretsEnabled,
  refresh,
  loadCapabilities,
  promoteVariable,
  deleteVariable,
} = useVariables(() => props.target);

const showEffective = computed(() => Boolean(props.agentId));

const { projects: allProjects, loadProjects } = useProjects();
const projectOptions = computed(() => props.projects ?? allProjects.value);

if (props.agentId) {
  void loadProjects();
}

const NO_PROJECT = "__none__";
const effectiveProject = ref(NO_PROJECT);
const {
  items: effective,
  loading: effectiveLoading,
  error: effectiveError,
} = useEffectiveVariables(
  () => props.agentId,
  () => (effectiveProject.value === NO_PROJECT ? "" : effectiveProject.value),
);

const dialogOpen = ref(false);
const editing = ref<Variable | null>(null);

const scopeLabelText = computed(() => scopeLabel(props.target));

void loadCapabilities();

function openCreate() {
  editing.value = null;
  dialogOpen.value = true;
}

function openEdit(variable: Variable) {
  editing.value = variable;
  dialogOpen.value = true;
}

async function promote(variable: Variable) {
  if (!window.confirm(t("variables.promoteConfirm", { name: variable.name }))) {
    return;
  }
  try {
    await withProgress(() => promoteVariable(variable.name));
    toast.success(t("variables.toasts.promoted", { name: variable.name }));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

async function remove(variable: Variable) {
  if (!window.confirm(t("variables.deleteConfirm", { name: variable.name }))) {
    return;
  }
  try {
    await withProgress(() => deleteVariable(variable.name));
    toast.success(t("variables.toasts.deleted", { name: variable.name }));
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  }
}

function scopeLabel(scope?: VariableTargetInput): string {
  if (!scope) return "";
  switch (scope.scope) {
    case 1:
      return t("variables.scopes.org");
    case 2:
      return t("variables.scopes.project");
    case 3:
      return t("variables.scopes.team");
    case 4:
      return t("variables.scopes.group");
    case 5:
      return t("variables.scopes.agent");
    default:
      return "";
  }
}

function updatedBy(variable: Variable): string {
  return variable.updatedByName || variable.updatedByUsername || "";
}

function formatWhen(value?: { seconds: bigint }): string {
  if (!value) return "";
  const date = new Date(Number(value.seconds) * 1000);
  return date.toLocaleString(locale.value, {
    dateStyle: "medium",
    timeStyle: "short",
  });
}
</script>

<i18n lang="yaml">
en:
  variables:
    title: Variables
    description: Environment values this task's shell receives. A secret's value is sealed and never shown again; a plain variable is readable here.
    add: Add variable
    readOnly: Only this scope's owners can change its variables.
    empty: No variables defined at this scope yet.
    columnName: Name
    columnValue: Value
    columnNote: Note
    columnUpdated: Updated
    secret: secret
    secretValueHint: The value is write-only and cannot be read back.
    alreadySecret: Already a secret.
    promote: Make secret
    promoteHint: Seal this value so it is never readable again.
    noKeyTitle: Secrets are unavailable
    noKeyHint: Set CHERRYPI_SECRET_KEY to store secret values. Plain variables still work.
    deleteConfirm: Delete the variable {name}? A running task keeps the value it already has.
    promoteConfirm: Make {name} a secret? Its value is sealed and can never be read back, and a secret cannot be made plain again.
    inheritedValueHint: The value belongs to a scope you do not manage, so only its name and source are shown.
    effectiveTitle: Effective variables
    effectiveDescription: What this agent's tasks receive, merged across every scope that applies. The most specific scope wins.
    effectiveAllProjects: No project
    effectiveSource: From
    effectiveShadowed: Overridden
    effectiveEmpty: This agent receives no variables yet.
    scopes:
      org: Organization
      project: Project
      team: Team
      group: Group
      agent: Agent
    toasts:
      promoted: "{name} is now a secret"
      deleted: Deleted {name}
ru:
  variables:
    title: Переменные
    description: Значения окружения, которые получает оболочка задачи. Значение секрета шифруется и больше не показывается; обычная переменная видна здесь.
    add: Добавить переменную
    readOnly: Менять переменные этого уровня могут только его владельцы.
    empty: На этом уровне переменных пока нет.
    columnName: Имя
    columnValue: Значение
    columnNote: Заметка
    columnUpdated: Изменено
    secret: секрет
    secretValueHint: Значение доступно только на запись и не читается обратно.
    alreadySecret: Уже секрет.
    promote: Сделать секретом
    promoteHint: Зашифровать значение, чтобы его нельзя было прочитать.
    noKeyTitle: Секреты недоступны
    noKeyHint: Задайте CHERRYPI_SECRET_KEY, чтобы хранить секреты. Обычные переменные работают и без него.
    deleteConfirm: Удалить переменную {name}? Запущенная задача сохранит уже полученное значение.
    promoteConfirm: Сделать {name} секретом? Значение будет зашифровано и больше не прочитается, а вернуть секрет в обычные переменные нельзя.
    inheritedValueHint: Значение принадлежит уровню, которым вы не управляете, поэтому показаны только имя и источник.
    effectiveTitle: Действующие переменные
    effectiveDescription: Что получают задачи этого агента после слияния всех применимых уровней. Побеждает самый конкретный уровень.
    effectiveAllProjects: Без проекта
    effectiveSource: Источник
    effectiveShadowed: Перекрыто
    effectiveEmpty: Этот агент пока не получает переменных.
    scopes:
      org: Организация
      project: Проект
      team: Команда
      group: Группа
      agent: Агент
    toasts:
      promoted: "{name} теперь секрет"
      deleted: "{name} удалена"
</i18n>
