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
.flex.h-full.min-h-0.flex-col
  header.shrink-0.border-b
    .flex.items-center.gap-3.px-4.py-2
      BackButton(:fallback="{ to: { name: 'sites' }, label: 'Sites' }")
      .flex.min-w-0.flex-1.flex-col
        .flex.items-center.gap-2
          GlobeIcon.text-muted-foreground.size-4.shrink-0
          h1.truncate.text-sm.font-semibold {{ site?.title || id }}
          span.inline-flex.items-center.rounded-full.border.px-2.text-xs(
            v-if="site?.agencyWide"
            class="py-0.5"
            :title="$t('siteView.agencyWideTitle')"
          ) {{ $t("sitesView.agencyWide") }}
          span.inline-flex.items-center.rounded-full.border.px-2.text-xs(
            v-if="site?.onlyCreatorMayEdit"
            class="py-0.5"
            :title="$t('siteView.creatorOnlyTitle')"
          ) {{ $t("siteView.creatorOnly") }}
        p.text-muted-foreground.truncate.text-xs(v-if="site?.description")
          | {{ site.description }}
        p.text-muted-foreground.truncate.text-xs(v-else)
          | /sites/{{ id }}
      .flex.shrink-0.items-center.gap-2(v-if="site")
        RouterLink.text-muted-foreground.text-xs(
          v-if="site.originTaskId"
          class="hover:text-foreground"
          :to="{ name: 'task', params: { id: site.originTaskId } }"
          :title="$t('siteView.originTitle')"
        ) {{ site.originTaskId }}
        Button(variant="outline" size="sm" @click="openNewTab")
          ExternalLinkIcon(data-icon="inline-start")
          span {{ $t("siteView.newTab") }}
        Button(
          v-if="site.canEdit"
          variant="outline"
          size="sm"
          @click="settingsOpen = true"
        )
          SettingsIcon(data-icon="inline-start")
          span {{ $t("common.settings") }}
        Button.text-destructive(
          v-if="site.canDelete"
          variant="ghost"
          size="sm"
          @click="remove"
          :disabled="deleting"
        )
          Spinner(v-if="deleting" data-icon="inline-start")
          Trash2Icon(v-else data-icon="inline-start")
          span {{ $t("common.delete") }}
  .flex.min-h-0.flex-1.items-center.justify-center.p-6(v-if="loading && !site")
    Spinner.size-5
  .flex.min-h-0.flex-1.items-center.justify-center.p-6(v-else-if="error")
    p.text-muted-foreground.max-w-md.text-center.text-sm {{ error }}
  .min-h-0.flex-1(v-else-if="site")
    SiteFrame(:site="site")
  SiteSettingsDialog(
    v-if="site"
    v-model:open="settingsOpen"
    :site="site"
    @updated="onUpdated"
  )
</template>

<script setup lang="ts">
import BackButton from "@/components/BackButton.vue";
import SiteFrame from "@/components/sites/SiteFrame.vue";
import SiteSettingsDialog from "@/components/sites/SiteSettingsDialog.vue";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useRealtime } from "@/composables/useRealtime";
import { agency } from "@/lib/sdk";
import type { Site } from "@cherrypi/sdk";
import {
  ExternalLinkIcon,
  GlobeIcon,
  SettingsIcon,
  Trash2Icon,
} from "@lucide/vue";
import { onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";

const props = defineProps<{ id: string }>();
const router = useRouter();

const site = ref<Site>();
const loading = ref(false);
const error = ref("");
const settingsOpen = ref(false);
const deleting = ref(false);

async function load() {
  loading.value = true;
  error.value = "";
  try {
    site.value = await agency.getSite(props.id);
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

function onUpdated(updated: Site) {
  site.value = updated;
}

function openNewTab() {
  if (site.value) window.open(site.value.url, "_blank", "noopener");
}

async function remove() {
  if (!site.value) return;
  if (
    !window.confirm(`Delete the site "${site.value.title || site.value.id}"?`)
  )
    return;
  deleting.value = true;
  try {
    await agency.deleteSite(site.value.id);
    await router.push({ name: "sites" });
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    deleting.value = false;
  }
}

onMounted(load);
watch(() => props.id, load);
useRealtime(["sites"], () => {
  if (site.value) void load();
});
</script>

<i18n lang="yaml">
en:
  siteView:
    creatorOnly: "Creator-only edits"
    newTab: "New tab"
    agencyWideTitle: "Listed under Sites for every signed-in user"
    creatorOnlyTitle: "Only the creating agent may republish or change settings through the tools"
    originTitle: "Open the task this site was first published from"
ru:
  siteView:
    creatorOnly: "Правки только автором"
    newTab: "Новая вкладка"
    agencyWideTitle: "Показывается в разделе «Сайты» всем вошедшим пользователям"
    creatorOnlyTitle: "Только создавший агент может переопубликовать или изменить настройки через инструменты"
    originTitle: "Открыть задачу, из которой сайт впервые опубликован"
</i18n>
