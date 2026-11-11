// CherryPi: A multi-agent-multi-human self-hosted agency
// Copyright (C) 2026 CherryPi authors <cherrypi.dev>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

import {
  toKpiWidgetState,
  toKpiWidgetWire,
  type KpiWidgetState,
} from "@/lib/kpiWidgets";
import { agency } from "@/lib/sdk";
import type { KpiPage, KpiPageLayout } from "@cherrypi/sdk";
import { onMounted, ref, toValue, watch, type MaybeRefOrGetter } from "vue";

export interface KpiLayoutState {
  widgets: KpiWidgetState[];

  inherited: boolean;

  canEdit: boolean;

  fallback: KpiWidgetState[];
}

function pageKey(page: KpiPage | undefined): string {
  if (!page) return "";
  return page.kind === "agency" ? "agency" : `${page.kind}:${page.id}`;
}

function readLayout(layout: KpiPageLayout): KpiLayoutState {
  return {
    widgets: layout.widgets.map(toKpiWidgetState),
    inherited: layout.inherited,
    canEdit: layout.canEdit,
    fallback: layout.fallback.map(toKpiWidgetState),
  };
}

export function useKpiLayout(page: MaybeRefOrGetter<KpiPage | undefined>) {
  const widgets = ref<KpiWidgetState[]>([]);
  const inherited = ref(false);
  const canEdit = ref(false);
  const fallback = ref<KpiWidgetState[]>([]);
  const loading = ref(true);
  const error = ref("");
  const saving = ref(false);
  let generation = 0;

  async function load() {
    const current = toValue(page);
    const gen = ++generation;
    if (!current) {
      widgets.value = [];
      inherited.value = false;
      canEdit.value = false;
      fallback.value = [];
      loading.value = false;
      return;
    }
    loading.value = true;
    try {
      const layout = await agency.getKpiLayout(current);
      if (gen !== generation) return;
      const state = readLayout(layout);
      widgets.value = state.widgets;
      inherited.value = state.inherited;
      canEdit.value = state.canEdit;
      fallback.value = state.fallback;
      error.value = "";
    } catch (err) {
      if (gen !== generation) return;
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      if (gen === generation) loading.value = false;
    }
  }

  async function save(next: KpiWidgetState[]) {
    const current = toValue(page);
    if (!current) return;
    saving.value = true;
    try {
      const layout = await agency.updateKpiLayout(
        current,
        next.map(toKpiWidgetWire),
      );
      const state = readLayout(layout);
      widgets.value = state.widgets;
      inherited.value = state.inherited;
      canEdit.value = state.canEdit;
      fallback.value = state.fallback;
      error.value = "";
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
      throw err;
    } finally {
      saving.value = false;
    }
  }

  async function reset() {
    const current = toValue(page);
    if (!current) return;
    saving.value = true;
    try {
      const layout = await agency.resetKpiLayout(current);
      const state = readLayout(layout);
      widgets.value = state.widgets;
      inherited.value = state.inherited;
      canEdit.value = state.canEdit;
      fallback.value = state.fallback;
      error.value = "";
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
      throw err;
    } finally {
      saving.value = false;
    }
  }

  watch(() => pageKey(toValue(page)), load, { immediate: true });

  return {
    widgets,
    inherited,
    canEdit,
    fallback,
    loading,
    saving,
    error,
    reload: load,
    save,
    reset,
  };
}

export function useKpiLayoutDefaults() {
  const project = ref<KpiWidgetState[]>([]);
  const agent = ref<KpiWidgetState[]>([]);
  const canEdit = ref(false);
  const loading = ref(true);
  const error = ref("");
  const saving = ref(false);

  async function load() {
    loading.value = true;
    try {
      const defaults = await agency.getKpiLayoutDefaults();
      project.value = defaults.project.map(toKpiWidgetState);
      agent.value = defaults.agent.map(toKpiWidgetState);
      canEdit.value = defaults.canEdit;
      error.value = "";
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err);
    } finally {
      loading.value = false;
    }
  }

  async function save(next: {
    project: KpiWidgetState[];
    agent: KpiWidgetState[];
  }) {
    saving.value = true;
    try {
      const defaults = await agency.updateKpiLayoutDefaults({
        project: next.project.map(toKpiWidgetWire),
        agent: next.agent.map(toKpiWidgetWire),
      });
      project.value = defaults.project.map(toKpiWidgetState);
      agent.value = defaults.agent.map(toKpiWidgetState);
      canEdit.value = defaults.canEdit;
      error.value = "";
    } finally {
      saving.value = false;
    }
  }

  onMounted(() => void load());

  return {
    project,
    agent,
    canEdit,
    loading,
    saving,
    error,
    reload: load,
    save,
  };
}

export const kpiPageKey = pageKey;

export function kpiPageLabel(page: KpiPage | undefined): string {
  if (!page) return "this page";
  switch (page.kind) {
    case "agency":
      return "the Tasks page";
    case "project":
      return "this project";
    case "agent":
      return "this agent";
  }
}
