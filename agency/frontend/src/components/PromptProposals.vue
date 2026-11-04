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
.flex.flex-col.gap-3(v-if="canPropose || canApprove")
  .flex.items-center.justify-between.gap-2
    .flex.items-center.gap-2
      span.text-sm.font-medium Pending proposals
      Badge(v-if="pending.length" variant="secondary") {{ pending.length }}
    Button(
      v-if="canPropose && !canEdit"
      type="button"
      variant="outline"
      size="sm"
      @click="openComposer"
    )
      LightbulbIcon(data-icon="inline-start")
      span Suggest change
  p.text-muted-foreground.text-xs(v-if="canPropose && !canEdit")
    | Owners edit this prompt directly. You can propose a complete replacement
    | and an owner will review it.
  p.text-muted-foreground.text-sm(v-if="!pending.length && !loading")
    | No pending proposals.
  Skeleton.h-16.w-full(v-if="loading && !pending.length")
  p.text-destructive.text-sm(v-if="error") {{ error }}
  .flex.flex-col.gap-2.rounded-lg.border.p-3(
    v-for="proposal in pending"
    :key="proposal.id.toString()"
  )
    .flex.flex-wrap.items-center.justify-between.gap-2
      .flex.items-center.gap-2
        ActorAvatar(
          :name="proposal.author?.name"
          :username="proposal.author?.username"
          :size="18"
        )
        span.text-sm.font-medium {{ actorLabel(proposal.author) }}
        span.text-muted-foreground.text-xs proposed
        span.text-muted-foreground.text-xs ·
        span.text-muted-foreground.text-xs {{ formatTimestamp(proposal.createdAt) }}
      .flex.flex-wrap.items-center.gap-1
        Badge(v-if="proposal.stale" variant="destructive") stale
        Badge(v-if="proposal.autoApproved" variant="outline") auto-approved
    p.text-sm(v-if="proposal.note") {{ proposal.note }}
    p.text-muted-foreground.text-xs(v-if="proposal.stale")
      | The prompt changed after this proposal was written, so it cannot be
      | approved. The author must propose again.
    details.rounded-md.border
      summary.text-muted-foreground.cursor-pointer.px-2.py-1.text-xs Show diff
      .p-2
        PromptDiff(
          :label="`Proposal vs version ${currentVersion}`"
          :before="currentContent"
          :after="proposal.content"
        )
    .flex.flex-wrap.justify-end.gap-2
      Button(
        v-if="canApprove && !proposal.stale"
        type="button"
        size="sm"
        @click="openDecision(proposal, true)"
      )
        CheckIcon(data-icon="inline-start")
        span Approve
      Button(
        v-if="canApprove"
        type="button"
        size="sm"
        variant="outline"
        @click="openDecision(proposal, false)"
      )
        XIcon(data-icon="inline-start")
        span Reject
      Button(
        v-if="proposal.isMine"
        type="button"
        size="sm"
        variant="ghost"
        @click="withdraw(proposal)"
      )
        Undo2Icon(data-icon="inline-start")
        span Withdraw

  Dialog(v-model:open="composerOpen")
    DialogScrollContent.max-w-3xl
      DialogHeader
        DialogTitle Suggest a prompt change
        DialogDescription
          | The owner sees a diff against the current version. A proposal is a
          | complete replacement, not a patch.
      .flex.flex-col.gap-3
        Textarea.min-h-64.font-mono.text-sm(
          v-model="proposalContent"
          placeholder="The complete prompt content…"
        )
        Input(v-model="proposalNote" placeholder="Why this change? (optional)")
        p.text-destructive.text-sm(v-if="composerError") {{ composerError }}
      DialogFooter
        Button(variant="outline" @click="composerOpen = false") Cancel
        Button(
          @click="submitProposal"
          :disabled="saving || !proposalContent.trim()"
        )
          Spinner(v-if="saving" data-icon="inline-start")
          SendIcon(v-else data-icon="inline-start")
          span Submit proposal

  Dialog(v-model:open="decisionOpen")
    DialogContent
      DialogHeader
        DialogTitle {{ decisionApprove ? "Approve proposal?" : "Reject proposal?" }}
        DialogDescription
          | {{ decisionApprove ? "This appends the proposed content as a new prompt version." : "The proposal stays in history as rejected." }}
      Textarea(v-model="decisionNote" placeholder="Decision note (optional)")
      p.text-destructive.text-sm(v-if="decisionError") {{ decisionError }}
      DialogFooter
        Button(variant="outline" @click="decisionOpen = false") Cancel
        Button(
          :variant="decisionApprove ? 'default' : 'destructive'"
          @click="confirmDecision"
          :disabled="deciding"
        )
          Spinner(v-if="deciding" data-icon="inline-start")
          CheckIcon(v-else-if="decisionApprove" data-icon="inline-start")
          XIcon(v-else data-icon="inline-start")
          span {{ decisionApprove ? "Approve" : "Reject" }}
</template>

<script setup lang="ts">
import ActorAvatar from "@/components/ActorAvatar.vue";
import PromptDiff from "@/components/PromptDiff.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogScrollContent,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { Textarea } from "@/components/ui/textarea";
import { formatTimestamp } from "@/lib/format";
import { agency } from "@/lib/sdk";
import {
  PromptProposalStatus,
  type PromptActor,
  type PromptProposal,
  type PromptScope,
  type PromptTarget,
} from "@cherrypi/sdk";
import {
  CheckIcon,
  LightbulbIcon,
  SendIcon,
  Undo2Icon,
  XIcon,
} from "@lucide/vue";
import { computed, ref, watch } from "vue";
import { toast } from "vue3-toastify";

const props = defineProps<{
  scope: PromptScope;
  target: PromptTarget;
  currentContent: string;
  currentVersion: bigint;
  canEdit: boolean;
  canPropose: boolean;
  canApprove: boolean;
}>();

const emit = defineEmits<{ changed: [] }>();

const proposals = ref<PromptProposal[]>([]);
const loading = ref(false);
const error = ref("");
const composerOpen = ref(false);
const proposalContent = ref("");
const proposalNote = ref("");
const composerError = ref("");
const saving = ref(false);
const decisionOpen = ref(false);
const decisionApprove = ref(true);
const decisionTarget = ref<PromptProposal>();
const decisionNote = ref("");
const decisionError = ref("");
const deciding = ref(false);

const pending = computed(() =>
  proposals.value.filter(
    (proposal) => proposal.status === PromptProposalStatus.PENDING,
  ),
);

function actorLabel(actor?: PromptActor): string {
  if (!actor) return "Unknown";
  return actor.name || actor.username || "Unknown";
}

async function load() {
  if (!props.canPropose && !props.canApprove) return;
  loading.value = true;
  try {
    proposals.value = await agency.listPromptProposals({
      target: props.target,
      scope: props.scope,
      status: PromptProposalStatus.PENDING,
    });
    error.value = "";
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  } finally {
    loading.value = false;
  }
}

function openComposer() {
  proposalContent.value = props.currentContent;
  proposalNote.value = "";
  composerError.value = "";
  composerOpen.value = true;
}

async function submitProposal() {
  saving.value = true;
  try {
    const proposal = await agency.createPromptProposal({
      scope: props.scope,
      target: props.target,
      content: proposalContent.value,
      note: proposalNote.value,
    });
    composerOpen.value = false;
    if (proposal.status === PromptProposalStatus.APPROVED) {
      toast.success("Prompt change auto-approved");
    } else {
      toast.success("Prompt change proposed");
    }
    await load();
    emit("changed");
  } catch (err) {
    composerError.value = err instanceof Error ? err.message : String(err);
  } finally {
    saving.value = false;
  }
}

function openDecision(proposal: PromptProposal, approve: boolean) {
  decisionTarget.value = proposal;
  decisionApprove.value = approve;
  decisionNote.value = "";
  decisionError.value = "";
  decisionOpen.value = true;
}

async function confirmDecision() {
  if (!decisionTarget.value) return;
  deciding.value = true;
  try {
    if (decisionApprove.value) {
      await agency.approvePromptProposal(
        decisionTarget.value.id,
        decisionNote.value,
      );
      toast.success("Prompt proposal approved");
    } else {
      await agency.rejectPromptProposal(
        decisionTarget.value.id,
        decisionNote.value,
      );
      toast.success("Prompt proposal rejected");
    }
    decisionOpen.value = false;
    await load();
    emit("changed");
  } catch (err) {
    decisionError.value = err instanceof Error ? err.message : String(err);
  } finally {
    deciding.value = false;
  }
}

async function withdraw(proposal: PromptProposal) {
  try {
    await agency.withdrawPromptProposal(proposal.id);
    toast.success("Proposal withdrawn");
    await load();
    emit("changed");
  } catch (err) {
    error.value = err instanceof Error ? err.message : String(err);
  }
}

watch(
  () => [
    props.scope,
    props.target.agentId,
    props.target.teamId,
    props.target.projectId,
    props.canPropose,
    props.canApprove,
  ],
  () => void load(),
  { immediate: true },
);
</script>
