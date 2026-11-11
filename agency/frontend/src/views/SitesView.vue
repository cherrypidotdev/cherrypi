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
    .flex.items-center.gap-3.px-4.py-3
      GlobeIcon.text-muted-foreground.size-5
      .flex.min-w-0.flex-1.flex-col
        h1.text-sm.font-semibold {{ $t("nav.sites") }}
        p.text-muted-foreground.text-xs
          | {{ $t("sitesView.description") }}
  .min-h-0.flex-1.overflow-auto.p-4
    .mb-3.flex.items-center.gap-2
      Input.w-64(
        v-model="search"
        :placeholder="$t('sitesView.filterPlaceholder')"
        :aria-label="$t('sitesView.filterAria')"
      )
      span.text-muted-foreground.text-xs
        | {{ filtered.length }} {{ filtered.length === 1 ? "site" : "sites" }}
    .grid.gap-3(class="sm:grid-cols-2 xl:grid-cols-3")
      RouterLink.block.rounded-lg.border.p-4.transition-colors(
        v-for="site in filtered"
        :key="site.id"
        class="hover:bg-muted/40"
        :to="{ name: 'site', params: { id: site.id } }"
      )
        .flex.items-start.gap-3
          span.text-primary.inline-flex.size-9.shrink-0.items-center.justify-center.rounded-md(
            class="bg-primary/10"
          )
            GlobeIcon.size-4
          .flex.min-w-0.flex-1.flex-col(class="gap-0.5")
            .flex.items-center.gap-2
              span.truncate.text-sm.font-semibold {{ site.title || site.id }}
              span.inline-flex.items-center.rounded-full.border.text-xs(
                v-if="site.agencyWide"
                class="py-0.5"
              ) {{ $t("sitesView.agencyWide") }}
            p.text-muted-foreground.line-clamp-2.text-xs(
              v-if="site.description"
            ) {{ site.description }}
            p.text-muted-foreground.font-mono.text-xs(v-else) /sites/{{ site.id }}
        .mt-3.flex.items-center.gap-2.text-xs
          span.text-muted-foreground.truncate(v-if="site.originTaskId") {{ $t("sitesView.fromTask", { id: site.originTaskId }) }}
          span.text-muted-foreground.ml-auto.shrink-0
            | {{ formatTimestamp(site.updatedAt) }}
    .flex.flex-col.items-center.justify-center.p-10.text-center(
      v-if="!loading && !filtered.length"
    )
      GlobeIcon.text-muted-foreground.size-8
      p.mt-2.text-sm.font-medium(v-if="search") {{ $t("sitesView.noMatch", { search }) }}
      p.mt-2.text-sm.font-medium(v-else) {{ $t("sitesView.empty") }}
      p.text-muted-foreground.mt-1.max-w-md.text-xs
        | {{ $t("sitesView.emptyHint") }}
</template>

<script setup lang="ts">
import { Input } from "@/components/ui/input";
import { useRealtime } from "@/composables/useRealtime";
import { useSites } from "@/composables/useSites";
import { formatTimestamp } from "@/lib/format";
import { GlobeIcon } from "@lucide/vue";
import { computed, onMounted, ref } from "vue";

const { sites, loading, load } = useSites();
const search = ref("");

const filtered = computed(() => {
  const query = search.value.trim().toLowerCase();
  if (!query) return sites.value;
  return sites.value.filter((site) =>
    `${site.title} ${site.description} ${site.id} ${site.originTaskId}`
      .toLowerCase()
      .includes(query),
  );
});

onMounted(() => void load());
useRealtime(["sites"], () => void load());
</script>

<i18n lang="yaml">
en:
  sitesView:
    agencyWide: "Agency-wide"
    description: "Static sites published by tasks. Every signed-in member can open them; owners and admins manage settings."
    empty: "No sites yet."
    emptyHint: "A task publishes one with the publishSite tool; the site then appears here and as a tab in its origin task."
    filterPlaceholder: "Filter sites…"
    filterAria: "Filter sites"
    fromTask: "From {id}"
    noMatch: "No sites match “{search}”."
ru:
  sitesView:
    agencyWide: "Для всего агентства"
    description: "Статические сайты, опубликованные задачами. Открыть их может любой вошедший участник; настройками управляют владельцы и администраторы."
    empty: "Сайтов пока нет."
    emptyHint: "Задача публикует сайт инструментом publishSite; затем сайт появляется здесь и вкладкой в исходной задаче."
    filterPlaceholder: "Фильтр сайтов…"
    filterAria: "Фильтр сайтов"
    fromTask: "Из задачи {id}"
    noMatch: "Нет сайтов, подходящих под «{search}»."
</i18n>
