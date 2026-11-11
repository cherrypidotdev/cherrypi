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
.flex.flex-col.items-center.gap-1.pt-2(ref="root")
  Button(
    v-if="hasMore"
    variant="outline"
    size="sm"
    @click="manual"
    :disabled="loading"
  )
    Spinner(v-if="loading" data-icon="inline-start")
    span {{ loading ? "Loading…" : "Load more" }}
  p.text-muted-foreground.text-xs(v-else-if="loaded") {{ $t("pluginLog.end") }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { onBeforeUnmount, onMounted, ref, watch } from "vue";

const props = defineProps<{
  hasMore: boolean;
  loading: boolean;
  auto: boolean;
  loaded: boolean;
}>();

const emit = defineEmits<{ load: [manual: boolean] }>();

const root = ref<HTMLElement>();
let observer: IntersectionObserver | undefined;

function maybeAutoLoad() {
  if (!props.auto || !props.hasMore || props.loading) return;
  emit("load", false);
}

onMounted(() => {
  if (typeof IntersectionObserver === "undefined") return;
  observer = new IntersectionObserver(
    (entries) => {
      if (entries.some((entry) => entry.isIntersecting)) maybeAutoLoad();
    },
    { rootMargin: "160px" },
  );
  observe();
});

watch(root, observe);

watch(
  () => [props.loading, props.hasMore, props.auto],
  () => {
    if (!props.loading) observe();
  },
);

onBeforeUnmount(() => observer?.disconnect());

function observe() {
  if (!observer) return;
  observer.disconnect();
  if (root.value) observer.observe(root.value);
}

function manual() {
  emit("load", true);
}
</script>

<i18n lang="yaml">
en:
  pluginLog:
    end: "End of log"
ru:
  pluginLog:
    end: "Конец журнала"
</i18n>
