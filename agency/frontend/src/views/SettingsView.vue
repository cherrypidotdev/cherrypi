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
.mx-auto.flex.w-full.max-w-6xl.flex-col.gap-6.p-6
  .flex.flex-col.gap-1
    h1.text-lg.font-semibold Organization settings
    p.text-muted-foreground.text-sm Agency-wide configuration. Only administrators can open this page.

  Card(v-if="!isAdmin")
    CardHeader
      CardTitle Administrators only
      CardDescription
        | Ask an administrator if you need an agency-wide setting changed.
  template(v-else)
    Tabs(v-model="tab")
      TabsList
        TabsTrigger(value="general") General
        TabsTrigger(value="auth") Auth
        TabsTrigger(value="integrations") Integrations
        TabsTrigger(value="admin") Admin
        TabsTrigger(value="logs") Logs

      TabsContent(value="general")
        .flex.flex-col.gap-6
          Card
            CardHeader
              CardTitle General
              CardDescription The agency's identity, task columns, and organization prompt.
            CardContent.flex.flex-col.gap-6
              form.flex.flex-col.gap-4(@submit.prevent="save")
                FieldGroup
                  Field
                    FieldLabel(for="agency-name") Agency name
                    Input#agency-name(
                      v-model="name"
                      placeholder="CherryPi"
                      maxlength="100"
                    )
                  Field
                    FieldLabel(for="primary-color") Primary color
                    .flex.items-center.gap-2
                      input#primary-color.border-input.h-8.w-12.cursor-pointer.rounded-md.border.bg-transparent(
                        type="color"
                        :value="primary || DEFAULT_PRIMARY"
                        @input="onPick"
                      )
                      Input.max-w-40.font-mono.text-sm(
                        v-model="hexDraft"
                        placeholder="#171717"
                        @blur="commitHex"
                        @keydown.enter.prevent="commitHex"
                        @keydown.esc.prevent="cancelHex"
                      )
                      Button(
                        type="button"
                        variant="outline"
                        size="sm"
                        @click="resetColor"
                      )
                        RotateCcwIcon(data-icon="inline-start")
                        span Reset
                    FieldDescription Empty uses the built-in default; text contrast adapts automatically.
                  Field
                    FieldLabel Logo
                    PictureUpload(
                      agency-kind="logo"
                      fallback-image="/logo.png"
                      shape="square"
                      :picture="settings?.logo"
                      @changed="onLogoChanged"
                    )
                    FieldDescription
                      | Shown in the sidebar and used as the default favicon. Empty
                      | uses the bundled cherry artwork.
                  Field
                    FieldLabel Favicon
                    PictureUpload(
                      agency-kind="favicon"
                      shape="square"
                      :fallback-image="settings?.logo?.url || '/logo.png'"
                      :picture="settings?.favicon"
                      @changed="onFaviconChanged"
                    )
                    FieldDescription Defaults to the logo when empty.
                p.text-destructive.text-sm(v-if="error") {{ error }}
                .flex.justify-end
                  Button(type="submit" :disabled="saving")
                    Spinner(v-if="saving" data-icon="inline-start")
                    CheckIcon(v-else data-icon="inline-start")
                    span Save

              Separator

              .flex.flex-col.gap-2
                .flex.flex-col
                  span.text-sm.font-medium Organization prompt
                  span.text-muted-foreground.text-xs
                    | Injected into every new task's system prompt, before the agent,
                    | project, and agent-project prompts. Edits affect future tasks
                    | only.
                PromptEditor(
                  placeholder="Organization-wide instructions…"
                  :scope="PromptScope.ORG"
                )

              Separator

              TaskColumnsPanel

      TabsContent(value="auth")
        .flex.flex-col.gap-6
          AuthProvidersPanel
          SMTPSettingsPanel

      TabsContent(value="integrations")
        .flex.flex-col.gap-6
          Alert
            AlertTitle Webhooks
            AlertDescription
              | Org-wide webhooks are being simplified: they will fire on every
              | agency event and no longer offer an event picker. That change is
              | tracked in CHPI-176.
          WebhooksPanel

      TabsContent(value="admin")
        AdminPanel

      TabsContent(value="logs")
        ProcessLogsPanel
</template>

<script setup lang="ts">
import PictureUpload from "@/components/attachments/PictureUpload.vue";
import PromptEditor from "@/components/PromptEditor.vue";
import AdminPanel from "@/components/settings/AdminPanel.vue";
import AuthProvidersPanel from "@/components/settings/AuthProvidersPanel.vue";
import ProcessLogsPanel from "@/components/settings/ProcessLogsPanel.vue";
import SMTPSettingsPanel from "@/components/settings/SMTPSettingsPanel.vue";
import TaskColumnsPanel from "@/components/settings/TaskColumnsPanel.vue";
import WebhooksPanel from "@/components/settings/WebhooksPanel.vue";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { Spinner } from "@/components/ui/spinner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { withProgress } from "@/composables/useProgress";
import { useSettings } from "@/composables/useSettings";
import { currentAccount } from "@/lib/account";
import { PromptScope, type Attachment } from "@cherrypi/sdk";
import { CheckIcon, RotateCcwIcon } from "@lucide/vue";
import { computed, onMounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { toast } from "vue3-toastify";

const DEFAULT_PRIMARY = "#171717";

const { settings, error, loadSettings, saveSettings } = useSettings();
const isAdmin = computed(() => currentAccount.value?.is_admin ?? false);
const route = useRoute();

const tab = ref(
  typeof route.query.tab === "string" ? route.query.tab : "general",
);

watch(
  () => route.query.tab,
  (value) => {
    if (typeof value === "string") tab.value = value;
  },
);
const name = ref("");
const primary = ref("");
const hexDraft = ref("");
const saving = ref(false);

watch(primary, (value) => {
  hexDraft.value = value;
});

onMounted(async () => {
  if (!isAdmin.value) return;
  await loadSettings(true);
  name.value = settings.value?.agencyName ?? "";
  primary.value = settings.value?.primaryColor ?? "";
});

function onPick(event: Event) {
  primary.value = (event.target as HTMLInputElement).value;
}

function onLogoChanged(picture: Attachment | undefined) {
  if (settings.value) settings.value.logo = picture;
}

function onFaviconChanged(picture: Attachment | undefined) {
  if (settings.value) settings.value.favicon = picture;
}

function commitHex() {
  const raw = hexDraft.value.trim();
  if (raw === "") {
    primary.value = "";
    return;
  }

  const hex = (raw.startsWith("#") ? raw : `#${raw}`).toLowerCase();
  if (/^#[0-9a-f]{6}$/.test(hex)) {
    primary.value = hex;
  } else {
    hexDraft.value = primary.value;
  }
}

function cancelHex() {
  hexDraft.value = primary.value;
}

function resetColor() {
  primary.value = "";
  hexDraft.value = "";
}

async function save() {
  saving.value = true;
  try {
    await withProgress(() =>
      saveSettings({
        agencyName: name.value.trim(),
        primaryColor: primary.value,
      }),
    );
    toast.success("Settings saved");
  } catch (err) {
    toast.error(err instanceof Error ? err.message : String(err));
  } finally {
    saving.value = false;
  }
}
</script>
