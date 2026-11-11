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
.relative.flex.h-full.min-h-0.w-full.flex-col
  .flex.h-full.items-center.justify-center(v-if="!ready && !frameError")
    Spinner.size-5
  p.text-muted-foreground.max-w-md.p-6.text-center.text-sm(
    v-else-if="frameError"
  )
    | {{ frameError }}
  iframe.h-full.w-full.border-0(
    v-else
    :key="frameKey"
    ref="frame"
    :src="contentSrc"
    sandbox="allow-scripts allow-forms allow-popups allow-modals allow-downloads"
    referrerpolicy="no-referrer"
    :name="frameName"
    :title="site.title || site.id"
    data-testid="site-frame"
  )

  Dialog(:open="consentOpen" @update:open="onConsentOpenChange")
    DialogContent(class="sm:max-w-md")
      DialogHeader
        .mb-1.flex.items-center.gap-2
          span.text-primary.inline-flex.size-8.items-center.justify-center.rounded-full(
            class="bg-primary/10"
          )
            ShieldCheckIcon.size-4
          span.text-muted-foreground.text-xs.font-medium.tracking-wide.uppercase {{ $t("siteFrame.agency") }}
        DialogTitle {{ $t("siteFrame.consentTitle") }}
        DialogDescription
          | {{ $t("siteFrame.consentSite") }}
          strong.mx-1 {{ site.title || site.id }}
          | {{ $t("siteFrame.consentBody") }}
      .flex.flex-col.gap-1.rounded-md.border.p-3.text-xs(class="bg-muted/40")
        span.font-medium {{ $t("siteFrame.consentLabel") }}
        span.font-mono.break-all {{ site.id }}
        span.text-muted-foreground.mt-1(v-if="site.originTaskId") {{ $t("siteFrame.publishedFrom", { id: site.originTaskId }) }}
      p.text-destructive.text-xs(v-if="consentError") {{ consentError }}
      DialogFooter
        Button(variant="outline" @click="denyConsent" :disabled="consentBusy") {{ $t("siteFrame.deny") }}
        Button(@click="allowConsent" :disabled="consentBusy")
          Spinner(v-if="consentBusy" data-icon="inline-start")
          ShieldCheckIcon(v-else data-icon="inline-start")
          span {{ $t("siteFrame.allow") }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Spinner } from "@/components/ui/spinner";
import { t } from "@/i18n/translate";
import { agency } from "@/lib/sdk";
import type { Site } from "@cherrypi/sdk";
import { ShieldCheckIcon } from "@lucide/vue";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";

const props = defineProps<{ site: Site }>();

const frame = ref<HTMLIFrameElement>();
const contentSrc = ref("");
const nonce = `cherrypi-site-${crypto.randomUUID()}`;
const frameKey = computed(
  () =>
    `${props.site.id}:${props.site.updatedAt?.seconds ?? 0}:${props.site.updatedAt?.nanos ?? 0}:${contentSrc.value}`,
);

const frameName = computed(() => {
  void contentSrc.value;
  return JSON.stringify({
    n: nonce,
    s: {
      local: readSiteStorage("local", props.site.id),
      session: readSiteStorage("session", props.site.id),
    },
  });
});

const consentOpen = ref(false);
const consentBusy = ref(false);
const consentError = ref("");
const ready = ref(false);
const frameError = ref("");
let pendingConsent: { source: Window; id: string } | null = null;

function tokenKey(siteId: string) {
  return `cherrypi.siteToken.${siteId}`;
}

function storageKey(siteId: string) {
  return `cherrypi.siteStorage.${siteId}`;
}

function readSiteStorage(
  kind: "local" | "session",
  siteId: string,
): Record<string, string> {
  try {
    const store = kind === "local" ? localStorage : sessionStorage;
    const raw = store.getItem(storageKey(siteId));
    if (!raw) return {};
    const parsed = JSON.parse(raw) as unknown;
    if (!parsed || typeof parsed !== "object") return {};
    const values: Record<string, string> = {};
    for (const [key, value] of Object.entries(parsed)) {
      if (typeof value === "string") values[key] = value;
    }
    return values;
  } catch {
    return {};
  }
}

function applyStorageMessage(data: {
  kind?: string;
  op?: string;
  key?: string;
  value?: unknown;
}) {
  const kind = data.kind === "session" ? "session" : "local";
  const store = kind === "session" ? sessionStorage : localStorage;
  const values = readSiteStorage(kind, props.site.id);
  switch (data.op) {
    case "set":
      if (typeof data.key !== "string") return;
      values[data.key] =
        typeof data.value === "string" ? data.value : String(data.value ?? "");
      break;
    case "remove":
      if (typeof data.key !== "string") return;
      delete values[data.key];
      break;
    case "clear":
      for (const key of Object.keys(values)) delete values[key];
      break;
    default:
      return;
  }
  try {
    store.setItem(storageKey(props.site.id), JSON.stringify(values));
  } catch {
    // The shell's own quota is the last line of defense; the site already
  }
}

function storedToken(): string {
  try {
    return localStorage.getItem(tokenKey(props.site.id)) ?? "";
  } catch {
    return "";
  }
}

function storeToken(token: string) {
  try {
    if (token) localStorage.setItem(tokenKey(props.site.id), token);
    else localStorage.removeItem(tokenKey(props.site.id));
  } catch {}
}

function reply(
  source: Window,
  id: string,
  ok: boolean,
  result?: unknown,
  error?: string,
) {
  source.postMessage(
    {
      __cherrypi: true,
      type: "cherrypi-site-reply",
      nonce,
      id,
      ok,
      result,
      error,
    },
    "*",
  );
}

function onMessage(event: MessageEvent) {
  const target = frame.value?.contentWindow;
  if (!target || event.source !== target) return;
  const data = event.data as {
    __cherrypi?: boolean;
    type?: string;
    nonce?: string;
    id?: string;
    kind?: string;
    op?: string;
    key?: string;
    value?: unknown;
  };
  if (!data || data.__cherrypi !== true || !data.type) return;
  if (data.nonce && data.nonce !== nonce) return;
  if (data.type === "cherrypi-site-storage") {
    applyStorageMessage(data);
    return;
  }
  if (!data.id) return;

  switch (data.type) {
    case "cherrypi-site-auth-request": {
      const token = storedToken();
      if (token) {
        reply(event.source as Window, data.id, true, {
          token,
          site: siteInfo(),
        });
        return;
      }
      if (pendingConsent) {
        reply(
          event.source as Window,
          data.id,
          false,
          undefined,
          t("siteFrame.alreadyOpen"),
        );
        return;
      }
      pendingConsent = { source: event.source as Window, id: data.id };
      consentError.value = "";
      consentOpen.value = true;
      return;
    }
    case "cherrypi-site-token-request": {
      const token = storedToken();
      reply(event.source as Window, data.id, true, token ? { token } : null);
      return;
    }
    case "cherrypi-site-logout": {
      storeToken("");
      reply(event.source as Window, data.id, true, {});
      return;
    }
    case "cherrypi-site-info": {
      reply(event.source as Window, data.id, true, siteInfo());
      return;
    }
    default:
      return;
  }
}

function siteInfo() {
  return {
    id: props.site.id,
    title: props.site.title,
    description: props.site.description,
    url: props.site.url,
  };
}

function onConsentOpenChange(open: boolean) {
  if (!open && pendingConsent) denyConsent();
}

function denyConsent() {
  if (pendingConsent) {
    reply(
      pendingConsent.source,
      pendingConsent.id,
      false,
      undefined,
      t("siteFrame.declined"),
    );
  }
  pendingConsent = null;
  consentOpen.value = false;
  consentBusy.value = false;
}

async function allowConsent() {
  if (!pendingConsent || consentBusy.value) return;
  consentBusy.value = true;
  consentError.value = "";
  try {
    const token = await agency.createSiteToken(props.site.id);
    if (!token) throw new Error(t("siteFrame.emptyToken"));
    storeToken(token);
    reply(pendingConsent.source, pendingConsent.id, true, {
      token,
      site: siteInfo(),
    });
    pendingConsent = null;
    consentOpen.value = false;
  } catch (err) {
    consentError.value = err instanceof Error ? err.message : String(err);
  } finally {
    consentBusy.value = false;
  }
}

async function prepare() {
  ready.value = false;
  contentSrc.value = "";
  frameError.value = "";
  try {
    contentSrc.value = await agency.prepareSiteView(props.site.id);
    ready.value = true;
  } catch (err) {
    frameError.value = err instanceof Error ? err.message : String(err);
  }
}

watch(
  () => props.site.id,
  () => {
    denyConsent();
    void prepare();
  },
);

onMounted(() => {
  window.addEventListener("message", onMessage);
  void prepare();
});
onBeforeUnmount(() => {
  window.removeEventListener("message", onMessage);
  denyConsent();
});
</script>

<i18n lang="yaml">
en:
  siteFrame:
    agency: "CherryPi Agency"
    consentTitle: "Allow this site to use your account?"
    consentSite: "The site"
    consentBody: "is asking for access to your CherryPi account. It will receive a personal access token you can revoke any time in Account → Tokens."
    consentLabel: "Site"
    deny: "Deny"
    allow: "Allow access"
    publishedFrom: "Published from task {id}"
    alreadyOpen: "Another access request is already open."
    declined: "The visitor declined site access."
    emptyToken: "The agency returned an empty token."
ru:
  siteFrame:
    agency: "Агентство CherryPi"
    consentTitle: "Разрешить сайту использовать ваш аккаунт?"
    consentSite: "Сайт"
    consentBody: "запрашивает доступ к вашему аккаунту CherryPi. Он получит персональный токен доступа, который можно отозвать в любой момент в разделе «Аккаунт → Токены»."
    consentLabel: "Сайт"
    deny: "Запретить"
    allow: "Разрешить доступ"
    publishedFrom: "Опубликовано из задачи {id}"
    alreadyOpen: "Другой запрос доступа уже открыт."
    declined: "Посетитель отклонил доступ к сайту."
    emptyToken: "Агентство вернуло пустой токен."
</i18n>
