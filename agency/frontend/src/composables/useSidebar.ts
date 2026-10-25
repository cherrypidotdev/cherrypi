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

import { ref } from "vue";

const STORAGE_KEY = "cherrypi.sidebar.collapsed";

const NARROW_WIDTH = 1024;

function load(): boolean {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw === "1") return true;
    if (raw === "0") return false;
  } catch {}
  return typeof window !== "undefined" && window.innerWidth < NARROW_WIDTH;
}

const collapsed = ref(load());

function persist() {
  try {
    localStorage.setItem(STORAGE_KEY, collapsed.value ? "1" : "0");
  } catch {}
}

export function toggleSidebar() {
  collapsed.value = !collapsed.value;
  persist();
}

export function setSidebarCollapsed(value: boolean) {
  collapsed.value = value;
  persist();
}

export type SidebarSection = "projects" | "agents" | "teams" | "users";

const SECTIONS_KEY = "cherrypi.sidebar.sections";

const DEFAULT_SECTIONS: Record<SidebarSection, boolean> = {
  projects: true,
  agents: false,
  teams: false,
  users: false,
};

function loadSections(): Record<SidebarSection, boolean> {
  try {
    const raw = sessionStorage.getItem(SECTIONS_KEY);
    if (raw) {
      const parsed = JSON.parse(raw) as Partial<
        Record<SidebarSection, boolean>
      >;
      return { ...DEFAULT_SECTIONS, ...parsed };
    }
  } catch {}
  return { ...DEFAULT_SECTIONS };
}

const sections = ref(loadSections());

function persistSections() {
  try {
    sessionStorage.setItem(SECTIONS_KEY, JSON.stringify(sections.value));
  } catch {}
}

export function isSectionOpen(section: SidebarSection): boolean {
  return sections.value[section];
}

export function setSection(section: SidebarSection, value: boolean) {
  sections.value = { ...sections.value, [section]: value };
  persistSections();
}

export function toggleSection(section: SidebarSection) {
  setSection(section, !sections.value[section]);
}

export function useSidebar() {
  return {
    collapsed,
    toggle: toggleSidebar,
    setCollapsed: setSidebarCollapsed,
    sections,
    isSectionOpen,
    setSection,
    toggleSection,
  };
}
