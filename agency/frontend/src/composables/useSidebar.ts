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

// The sidebar collapse is a per-browser view preference, so it lives in
// localStorage next to the attention preferences and needs no agency column.
const STORAGE_KEY = "cherrypi.sidebar.collapsed";

// Below this width the rail starts collapsed so it does not eat the content on
// a narrow laptop or tablet. The user's explicit choice always wins after that.
const NARROW_WIDTH = 1024;

function load(): boolean {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (raw === "1") return true;
    if (raw === "0") return false;
  } catch {
    // Unavailable storage: fall through to the viewport default.
  }
  return typeof window !== "undefined" && window.innerWidth < NARROW_WIDTH;
}

// Module-scoped so the shell (and any future consumer) shares one value.
const collapsed = ref(load());

function persist() {
  try {
    localStorage.setItem(STORAGE_KEY, collapsed.value ? "1" : "0");
  } catch {
    // Best effort; a blocked localStorage must never break the shell.
  }
}

/** Collapse or expand the agency sidebar and remember the choice. */
export function toggleSidebar() {
  collapsed.value = !collapsed.value;
  persist();
}

/** Force the sidebar into an explicit state and persist it. */
export function setSidebarCollapsed(value: boolean) {
  collapsed.value = value;
  persist();
}

// The sidebar's directories are collapsible. Unlike the rail width (a durable
// preference), which directories are open is a per-tab working state, so it
// lives in sessionStorage and resets with the tab.
export type SidebarSection = "projects" | "agents" | "teams" | "users";

const SECTIONS_KEY = "cherrypi.sidebar.sections";

// Projects is the most-used list, so it starts open; users and agents start
// collapsed to keep the rail short.
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
  } catch {
    // Unavailable storage: fall through to the defaults.
  }
  return { ...DEFAULT_SECTIONS };
}

const sections = ref(loadSections());

function persistSections() {
  try {
    sessionStorage.setItem(SECTIONS_KEY, JSON.stringify(sections.value));
  } catch {
    // Best effort; a blocked sessionStorage must never break the shell.
  }
}

/** Report whether a sidebar directory is expanded. */
export function isSectionOpen(section: SidebarSection): boolean {
  return sections.value[section];
}

/** Force a sidebar directory into an explicit state and persist it. */
export function setSection(section: SidebarSection, value: boolean) {
  sections.value = { ...sections.value, [section]: value };
  persistSections();
}

/** Collapse or expand a sidebar directory and remember the choice. */
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
