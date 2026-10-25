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

<template>
  <div class="mx-auto flex w-full max-w-6xl flex-col gap-6 p-6">
    <Card>
      <CardHeader class="flex-row items-center justify-between">
        <div class="flex flex-col gap-1">
          <CardTitle>Inbox</CardTitle>
          <CardDescription>
            Attention signals and invitations, newest first.
            <RouterLink
              class="text-foreground ml-1 underline underline-offset-2"
              :to="{ name: 'account', query: { tab: 'notifications' } }"
            >
              Notification settings
            </RouterLink>
          </CardDescription>
        </div>
        <div class="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            :disabled="!unread"
            @click="markAll"
          >
            <CheckCheckIcon data-icon="inline-start" />
            <span>Mark all read</span>
          </Button>
          <Dialog v-model:open="deleteOpen">
            <DialogTrigger as-child>
              <Button
                variant="outline"
                size="sm"
                :disabled="!notifications.length"
              >
                <Trash2Icon data-icon="inline-start" />
                <span>Delete all</span>
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Delete all notifications?</DialogTitle>
                <DialogDescription>
                  This permanently removes every inbox entry and cannot be
                  undone.
                </DialogDescription>
              </DialogHeader>
              <DialogFooter>
                <Button variant="outline" @click="deleteOpen = false">
                  Cancel
                </Button>
                <Button
                  variant="destructive"
                  :disabled="deleting"
                  @click="confirmDeleteAll"
                >
                  <Spinner v-if="deleting" data-icon="inline-start" />
                  <Trash2Icon v-else data-icon="inline-start" />
                  <span>Delete all</span>
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
      </CardHeader>
      <CardContent>
        <p v-if="error" class="text-destructive text-sm">{{ error }}</p>
        <div class="flex flex-col gap-2">
          <Skeleton
            v-if="loading && !notifications.length"
            class="h-16 w-full"
          />
          <div
            v-for="item in notifications"
            :key="item.id.toString()"
            class="relative flex items-start justify-between gap-3 rounded-md border p-3"
            :class="[
              !item.readAt ? 'bg-muted/40' : '',
              item.taskId || item.kind === 'team_invitation'
                ? 'transition-colors hover:bg-muted/60'
                : '',
            ]"
          >
            <div class="flex items-start gap-3">
              <span
                class="mt-1 size-2 shrink-0 rounded-full"
                :class="dotClass(item.severity)"
              />
              <div class="flex flex-col gap-0.5">
                <div class="flex items-center gap-2">
                  <span class="text-sm font-medium">{{ item.title }}</span>
                  <Badge v-if="!item.readAt" variant="secondary">new</Badge>
                </div>
                <span v-if="item.body" class="text-muted-foreground text-sm">{{
                  item.body
                }}</span>
                <span class="text-muted-foreground text-xs">
                  <template v-if="item.taskId">
                    {{ item.agentId.slice(0, 8) }} · task
                    {{ item.taskId.slice(0, 8) }}
                  </template>
                  <template v-else-if="item.kind === 'team_invitation'">
                    Team invitation
                  </template>
                  <template v-else>{{ item.agentId.slice(0, 8) }}</template>
                  · {{ formatTimestamp(item.createdAt) }}
                </span>
              </div>
            </div>
            <div class="relative z-10 flex shrink-0 gap-1">
              <Button
                v-if="!item.readAt"
                variant="ghost"
                size="icon-sm"
                title="Mark read"
                @click.stop="markRead(Number(item.id))"
              >
                <CheckIcon />
              </Button>
              <Button
                variant="ghost"
                size="icon-sm"
                title="Delete"
                @click.stop="remove(Number(item.id))"
              >
                <Trash2Icon />
              </Button>
            </div>

            <RouterLink
              v-if="isLinked(item)"
              class="absolute inset-0 rounded-md"
              :to="linkFor(item)"
              :aria-label="
                item.taskId
                  ? `Open task for: ${item.title}`
                  : `Open teams for: ${item.title}`
              "
              :title="
                item.taskId
                  ? `Open task ${item.taskId.slice(0, 8)}`
                  : 'Open Teams'
              "
              @click="openNotification(item)"
            />
          </div>
          <p
            v-if="!notifications.length && !loading"
            class="text-muted-foreground text-sm"
          >
            No notifications.
          </p>
        </div>
      </CardContent>
    </Card>
  </div>
</template>

<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { useNotifications } from "@/composables/useNotifications";
import { formatTimestamp } from "@/lib/format";
import type { AgencyNotification } from "@cherrypi/sdk";
import { CheckCheckIcon, CheckIcon, Trash2Icon } from "@lucide/vue";
import { onMounted, ref } from "vue";
import { RouterLink, type RouteLocationRaw } from "vue-router";
import { toast } from "vue3-toastify";

const {
  notifications,
  unread,
  loading,
  error,
  refresh,
  markRead,
  markAllRead,
  remove,
  deleteAll,
} = useNotifications();

const deleteOpen = ref(false);
const deleting = ref(false);

function dotClass(severity: string): string {
  switch (severity) {
    case "error":
      return "bg-destructive";
    case "warning":
      return "bg-amber-500";
    default:
      return "bg-muted-foreground/50";
  }
}

function markAll() {
  void markAllRead();
}

async function confirmDeleteAll() {
  deleting.value = true;
  try {
    await deleteAll();
    toast.success("Inbox cleared");
    deleteOpen.value = false;
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    deleting.value = false;
  }
}

function openNotification(item: AgencyNotification) {
  if (isLinked(item) && !item.readAt) void markRead(Number(item.id));
}

function isLinked(item: AgencyNotification): boolean {
  return Boolean(item.taskId) || item.kind === "team_invitation";
}

function linkFor(item: AgencyNotification): RouteLocationRaw {
  if (item.taskId) {
    return {
      name: "task",
      params: { id: item.taskId },
      query: { from: "inbox" },
    };
  }
  if (item.kind === "team_invitation") return { name: "teams" };
  return { name: "inbox" };
}

onMounted(() => void refresh());
</script>
