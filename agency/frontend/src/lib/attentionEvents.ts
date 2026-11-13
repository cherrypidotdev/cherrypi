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

export interface AttentionEventOption {
  kind: string;

  labelKey: string;
}

export const ATTENTION_EVENTS: AttentionEventOption[] = [
  { kind: "task_started", labelKey: "attention.event.taskStarted" },
  { kind: "input_requested", labelKey: "attention.event.needsInput" },
  { kind: "task_finished", labelKey: "attention.event.settled" },
  { kind: "failed", labelKey: "attention.event.failed" },
  { kind: "task_completed", labelKey: "attention.event.completed" },
  { kind: "task_cancelled", labelKey: "attention.event.cancelled" },
  { kind: "team_invitation", labelKey: "attention.event.teamInvitation" },
  { kind: "task_comment", labelKey: "attention.event.comment" },

  {
    kind: "prompt_proposal_created",
    labelKey: "attention.event.promptProposal",
  },
  { kind: "task_mention", labelKey: "attention.event.mention" },
  {
    kind: "agent_group_member_requested",
    labelKey: "attention.event.groupMemberRequested",
  },
  {
    kind: "agent_group_member_approved",
    labelKey: "attention.event.groupMemberApproved",
  },
  {
    kind: "agent_group_member_declined",
    labelKey: "attention.event.groupMemberDeclined",
  },
  {
    kind: "agent_group_member_removed",
    labelKey: "attention.event.groupMemberRemoved",
  },
  {
    kind: "agent_group_member_left",
    labelKey: "attention.event.groupMemberLeft",
  },
  {
    kind: "agent_group_access_invited",
    labelKey: "attention.event.groupAccessInvited",
  },
  {
    kind: "agent_group_access_removed",
    labelKey: "attention.event.groupAccessRemoved",
  },
  {
    kind: "agent_group_access_member_added",
    labelKey: "attention.event.groupAccessMemberAdded",
  },
];

export type AttentionChannel = "sound" | "browser" | "toast";

export interface AttentionPreference {
  sound: boolean;
  browser: boolean;

  toast: boolean;
}

export type AttentionPreferences = Record<string, AttentionPreference>;

const ENABLED_BY_DEFAULT = new Set([
  "input_requested",
  "failed",
  "task_finished",
]);

export function defaultAttentionPreferences(): AttentionPreferences {
  const out: AttentionPreferences = {};
  for (const event of ATTENTION_EVENTS) {
    const enabled = ENABLED_BY_DEFAULT.has(event.kind);
    out[event.kind] = { sound: enabled, browser: enabled, toast: enabled };
  }
  return out;
}
