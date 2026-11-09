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
.bg-background.text-foreground.flex.h-screen
  aside.bg-sidebar.text-sidebar-foreground.flex.w-64.shrink-0.flex-col.border-r
    header.flex.items-center.gap-2.border-b.px-3.py-3
      PaletteIcon.text-primary.size-4
      h1.min-w-0.truncate.text-sm.font-semibold Design studio
      span.d-badge.d-badge-xs.d-badge-soft.ml-auto DEV

      button.d-btn.d-btn-ghost.d-btn-xs.d-btn-square(
        type="button"
        :title="isDark ? 'Switch to light theme' : 'Switch to dark theme'"
        :aria-label="isDark ? 'Switch to light theme' : 'Switch to dark theme'"
        @click="toggleDark()"
        data-test="studio-theme"
      )
        MoonIcon(v-if="isDark" :size="14")
        SunIcon(v-else :size="14")

    nav.min-h-0.flex-1.overflow-auto.p-2
      .d-menu.d-menu-sm.w-full(v-for="comp in studioComponents" :key="comp.id")
        li.d-menu-title {{ comp.name }}
        ul
          li(v-for="entry in comp.scenarios" :key="entry.id")
            button(
              type="button"
              :class="{ 'd-menu-active': isActive(comp, entry) }"
              @click="select(comp, entry)"
            )
              span.truncate {{ entry.name }}
              CheckIcon.size-3(v-if="isActive(comp, entry)")

    footer.flex.flex-col.gap-1.border-t.p-3
      p.text-muted-foreground.text-xs
        | Isolated components, real code. Development builds only.
      RouterLink.text-xs.underline(:to="{ name: 'agency-tasks' }")
        | Back to the app

  main.flex.min-w-0.flex-1.flex-col
    header.flex.flex-wrap.items-center.gap-x-4.gap-y-2.border-b.px-4.py-2
      .flex.min-w-0.flex-1.flex-col
        p.text-muted-foreground.text-xs {{ item.blurb }}
        h2.flex.items-center.gap-1.text-sm.font-semibold
          span {{ item.name }}
          span.text-muted-foreground /
          span {{ scenario.name }}
        p.text-muted-foreground.font-mono.text-xs {{ item.source }}

      .flex.flex-wrap.items-center.gap-3
        span.text-muted-foreground.text-xs Preview
        .d-join
          button.d-btn.d-btn-xs.d-join-item(
            v-for="option in themes"
            :key="option.id"
            type="button"
            :title="`Preview with the ${option.label.toLowerCase()} theme`"
            :class="{ 'd-btn-primary': theme === option.id }"
            @click="theme = option.id"
          )
            component(:is="option.icon" :size="12")
            span {{ option.label }}

        .d-join
          button.d-btn.d-btn-xs.d-join-item(
            v-for="option in widths"
            :key="option"
            type="button"
            :title="`Preview at ${option}`"
            :class="{ 'd-btn-primary': width === option }"
            @click="width = option"
          )
            | {{ option === "full" ? "Full" : option }}

        button.d-btn.d-btn-sm.d-btn-ghost(
          type="button"
          title="Remount the state with fresh props and no interactions"
          @click="restart"
        )
          RefreshCwIcon(:size="14")
          span Restart

    section.bg-base-200.flex-1.overflow-auto.p-6
      .mx-auto.flex.w-full.flex-col.gap-3(:style="{ maxWidth: frameWidth }")
        .flex.items-start.justify-between.gap-3
          p.text-muted-foreground.max-w-prose.text-xs
            | {{ scenario.description }}
          span.text-muted-foreground.shrink-0.font-mono.text-xs
            | {{ widthLabel }}

        .d-card.bg-base-100.border(v-if="controls.length" data-test="controls")
          .d-card-body.gap-2.p-3
            h3.font-mono.text-xs.font-semibold Props
            .flex.flex-wrap.items-end.gap-4
              template(v-for="control in controls" :key="control.prop")
                label.flex.cursor-pointer.items-center.gap-2.text-xs(
                  v-if="control.type === 'boolean'"
                )
                  input.d-checkbox.d-checkbox-xs(
                    type="checkbox"
                    :checked="checkedValue(control)"
                    @change="setControl(control, $event)"
                  )
                  span {{ control.label }}
                  code.text-muted-foreground.font-mono {{ control.prop }}

                label.flex.flex-col.gap-1.text-xs(v-else)
                  span.flex.items-center.gap-2
                    | {{ control.label }}
                    code.text-muted-foreground.font-mono {{ control.prop }}
                  input.d-input.d-input-xs.w-64(
                    type="text"
                    :value="textValue(control)"
                    @input="setControl(control, $event)"
                  )

        section.bg-background.rounded-xl.border.p-4(
          :class="{ dark: theme === 'dark', light: theme === 'light' }"
          data-test="stage"
          :data-theme="theme === 'app' ? undefined : theme"
        )
          component(
            :key="`${scenario.id}-${run}`"
            v-on="listeners"
            v-bind="stageProps"
            :is="item.component"
          )

    footer.flex.shrink-0.flex-col.border-t(data-test="events")
      .flex.items-center.gap-2.px-4.py-2
        ActivityIcon(:size="14")
        h2.text-xs.font-semibold Events
        span.d-badge.d-badge-xs.d-badge-soft(v-if="events.length") {{ events.length }}
        p.text-muted-foreground.text-xs(v-if="!events.length")
          | Interactions with the component are recorded here.
        button.d-btn.d-btn-ghost.d-btn-xs.ml-auto(
          type="button"
          @click="events = []"
          :disabled="!events.length"
        ) Clear

      .max-h-48.overflow-auto.border-t(v-if="events.length")
        .flex.items-start.gap-3.border-b.px-4.py-2.text-xs(
          v-for="entry in events"
          :key="entry.id"
        )
          span.text-muted-foreground.shrink-0.font-mono {{ entry.at }}
          span.text-primary.shrink-0.font-semibold {{ entry.name }}
          pre.m-0.min-w-0.font-mono.leading-relaxed.break-all.whitespace-pre-wrap
            | {{ entry.payload }}
</template>

<script setup lang="ts">
import { useTheme } from "@/composables/useTheme";
import { studioComponents } from "@/dev-studio/registry";
import type {
  StudioComponent,
  StudioControl,
  StudioScenario,
} from "@/dev-studio/types";
import {
  ActivityIcon,
  CheckIcon,
  MonitorIcon,
  MoonIcon,
  PaletteIcon,
  RefreshCwIcon,
  SunIcon,
} from "@lucide/vue";
import { type Component, computed, ref, shallowRef, watch } from "vue";
import { RouterLink } from "vue-router";

const { isDark, toggleDark } = useTheme();

type PreviewTheme = "app" | "light" | "dark";

const themes: { id: PreviewTheme; label: string; icon: Component }[] = [
  { id: "app", label: "App", icon: MonitorIcon },
  { id: "light", label: "Light", icon: SunIcon },
  { id: "dark", label: "Dark", icon: MoonIcon },
];

const widths = [360, 480, 640, "full"] as const;

const first = studioComponents[0]!;
const item = shallowRef<StudioComponent>(first);
const scenario = shallowRef<StudioScenario>(first.scenarios[0]!);
const theme = ref<PreviewTheme>("app");
const width = ref<number | "full">(scenario.value.width ?? 480);

const run = ref(0);

function isActive(comp: StudioComponent, entry: StudioScenario): boolean {
  return item.value === comp && scenario.value === entry;
}

function select(comp: StudioComponent, entry: StudioScenario) {
  if (isActive(comp, entry)) {
    return;
  }
  item.value = comp;
  scenario.value = entry;
}

const controlValues = ref(new Map<string, boolean | string>());
const controls = computed(() => scenario.value.controls ?? []);

function controlKey(control: StudioControl): string {
  return `${item.value.id}/${scenario.value.id}/${control.prop}`;
}

function controlValue(control: StudioControl): boolean | string {
  return controlValues.value.get(controlKey(control)) ?? control.initial;
}

function checkedValue(control: StudioControl): boolean {
  return controlValue(control) === true;
}

function textValue(control: StudioControl): string {
  const value = controlValue(control);
  return typeof value === "string" ? value : "";
}

function setControl(control: StudioControl, event: Event) {
  const target = event.target as HTMLInputElement;
  controlValues.value.set(
    controlKey(control),
    control.type === "boolean" ? target.checked : target.value,
  );
}

const stageProps = computed(() => {
  const props: Record<string, unknown> = { ...scenario.value.props };
  for (const control of controls.value) {
    props[control.prop] = controlValue(control);
  }
  return props;
});

const frameWidth = computed(() =>
  width.value === "full" ? "100%" : `${width.value}px`,
);
const widthLabel = computed(() =>
  width.value === "full" ? "full width" : `${width.value}px`,
);

interface StudioEvent {
  id: number;
  at: string;
  name: string;
  payload: string;
}

const events = ref<StudioEvent[]>([]);
let eventSeq = 0;

function formatPayload(args: unknown[]): string {
  if (!args.length) {
    return "(no payload)";
  }
  return args
    .map((arg) => {
      const json = JSON.stringify(arg, null, 2);
      return json === undefined ? String(arg) : json;
    })
    .join("\n");
}

const listeners = computed(() => {
  const map: Record<string, (...args: unknown[]) => void> = {};
  for (const name of scenario.value.events ?? []) {
    map[name] = (...args: unknown[]) => {
      eventSeq += 1;
      events.value = [
        {
          id: eventSeq,
          at: new Date().toLocaleTimeString(),
          name,
          payload: formatPayload(args),
        },
        ...events.value,
      ].slice(0, 20);
    };
  }
  return map;
});

watch(scenario, (next) => {
  width.value = next.width ?? 480;
  events.value = [];
});

function restart() {
  const prefix = `${item.value.id}/${scenario.value.id}/`;
  for (const key of [...controlValues.value.keys()]) {
    if (key.startsWith(prefix)) {
      controlValues.value.delete(key);
    }
  }
  run.value += 1;
}
</script>
