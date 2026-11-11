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
.flex.items-center.gap-1(
  v-if="available"
  data-testid="voice-input"
  :data-mode="mode"
)
  .bg-border.w-px(aria-hidden="true" class="mx-0.5 h-3.5")
  Button.size-5(
    type="button"
    variant="ghost"
    size="icon-xs"
    :class="recording ? 'bg-destructive text-white hover:bg-destructive/90' : ''"
    :title="buttonTitle"
    :aria-label="buttonTitle"
    :aria-pressed="recording ? 'true' : 'false'"
    @click="toggle"
    :disabled="disabled || starting || transcribing"
    data-testid="markdown-editor-dictate"
  )
    Spinner.size-3(v-if="transcribing")
    SquareIcon.size-3(v-else-if="recording")
    MicIcon.size-3(v-else)
  .text-destructive.flex.items-center.gap-1.text-xs.font-medium(
    v-if="recording"
    aria-hidden="true"
    data-testid="voice-input-recording"
  )
    span.bg-destructive.animate-pulse.rounded-full(class="size-1.5")
    .flex.items-center.gap-px(class="h-2.5")
      span.bg-destructive.h-1.animate-pulse.rounded-full(class="w-0.5")
      span.bg-destructive.h-2.animate-pulse.rounded-full(
        class="w-0.5 [animation-delay:120ms]"
      )
      span.bg-destructive.animate-pulse.rounded-full(
        class="h-1.5 w-0.5 [animation-delay:240ms]"
      )
    span.tabular-nums {{ elapsedLabel }}
  span.text-muted-foreground.text-xs(v-else-if="transcribing")
    | {{ $t("voiceButton.transcribing") }}
</template>

<script setup lang="ts">
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useSettings } from "@/composables/useSettings";
import { agency } from "@/lib/sdk";
import {
  browserSpeechRecognition,
  isBenignRecognitionError,
  joinTranscript,
  recognitionErrorMessage,
  recordingFilename,
  recordingMimeType,
  type SpeechRecognitionLike,
} from "@/lib/speech";
import { StateEffect, StateField } from "@codemirror/state";
import type { EditorView } from "@codemirror/view";
import { MicIcon, SquareIcon } from "@lucide/vue";
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { toast } from "vue3-toastify";

const props = withDefaults(
  defineProps<{
    view?: EditorView;
    disabled?: boolean;
  }>(),
  { view: undefined, disabled: false },
);

const { settings, loadSettings } = useSettings();
onMounted(() => {
  void loadSettings();
});

const voice = computed(() => settings.value?.voiceInput);
const enabled = computed(() => voice.value?.enabled ?? false);
const serverConfigured = computed(() => voice.value?.serverConfigured ?? false);
const browserRecognizer = browserSpeechRecognition();
const available = computed(
  () =>
    enabled.value &&
    !props.disabled &&
    !!props.view &&
    (serverConfigured.value || browserRecognizer !== null),
);
const mode = computed(() => (serverConfigured.value ? "server" : "browser"));

const recording = ref(false);
const transcribing = ref(false);
const starting = ref(false);
const elapsed = ref(0);
const elapsedLabel = computed(() => {
  const minutes = Math.floor(elapsed.value / 60);
  const seconds = elapsed.value % 60;
  return `${minutes}:${String(seconds).padStart(2, "0")}`;
});
const buttonTitle = computed(() => {
  if (transcribing.value) return t("voiceButton.transcribing");
  if (recording.value) return t("voiceButton.stop");
  return mode.value === "server"
    ? t("voiceButton.dictateServer")
    : t("voiceButton.dictateBrowser");
});

let startedAt = 0;
let timer: ReturnType<typeof setInterval> | undefined;

function startTimer() {
  startedAt = Date.now();
  elapsed.value = 0;
  stopTimer();
  timer = setInterval(() => {
    elapsed.value = Math.floor((Date.now() - startedAt) / 1000);
  }, 250);
}

function stopTimer() {
  if (timer !== undefined) clearInterval(timer);
  timer = undefined;
}

const setDictationRange = StateEffect.define<{
  from: number;
  to: number;
} | null>();
const dictationRange = StateField.define<{ from: number; to: number } | null>({
  create: () => null,
  update(value, transaction) {
    for (const effect of transaction.effects) {
      if (effect.is(setDictationRange)) return effect.value;
    }
    if (!value) return null;
    return {
      from: transaction.changes.mapPos(value.from, -1),
      to: transaction.changes.mapPos(value.to, 1),
    };
  },
});

function ensureRangeField(view: EditorView) {
  if (view.state.field(dictationRange, false)) return;
  view.dispatch({ effects: StateEffect.appendConfig.of(dictationRange) });
}

function clearRange(view: EditorView | undefined) {
  if (!view) return;
  if (!view.state.field(dictationRange, false)) return;
  view.dispatch({ effects: setDictationRange.of(null) });
}

function padded(view: EditorView, from: number, text: string): string {
  if (from <= 0 || !text) return text;
  const before = view.state.sliceDoc(from - 1, from);
  return before !== "" && !/\s/.test(before) ? ` ${text}` : text;
}

function replaceDictationText(view: EditorView, text: string) {
  ensureRangeField(view);
  const range = view.state.field(dictationRange);
  const selection = view.state.selection.main;
  const from = range?.from ?? selection.from;
  const to = range?.to ?? selection.to;
  const insert = padded(view, from, text);
  view.dispatch({
    changes: { from, to, insert },
    selection: { anchor: from + insert.length },
    effects: setDictationRange.of({ from, to: from + insert.length }),
  });
  view.focus();
}

function insertAtSelection(view: EditorView, text: string) {
  const selection = view.state.selection.main;
  const insert = padded(view, selection.from, text);
  view.dispatch({
    changes: { from: selection.from, to: selection.to, insert },
    selection: { anchor: selection.from + insert.length },
  });
  view.focus();
}

let recorder: MediaRecorder | undefined;
let stream: MediaStream | undefined;
let chunks: Blob[] = [];
let disposed = false;

async function startServerRecording() {
  if (!navigator.mediaDevices?.getUserMedia) {
    toast.error(t("voice.errors.recordUnsupported"));
    return;
  }
  starting.value = true;
  try {
    stream = await navigator.mediaDevices.getUserMedia({ audio: true });
    if (disposed) {
      stopStream();
      return;
    }
    const requestedType = recordingMimeType();
    const active = new MediaRecorder(
      stream,
      requestedType ? { mimeType: requestedType } : undefined,
    );
    recorder = active;
    chunks = [];
    active.ondataavailable = (event) => {
      if (event.data.size > 0) chunks.push(event.data);
    };
    active.onstop = () => {
      const type =
        chunks[0]?.type || active.mimeType || requestedType || "audio/webm";
      const blob = new Blob(chunks, { type });
      chunks = [];
      if (!disposed) void transcribe(blob, recordingFilename(type));
    };
    active.start();
    recording.value = true;
    startTimer();
  } catch (error) {
    console.error("microphone unavailable", error);
    toast.error(t("voice.errors.micDenied"));
    stopStream();
  } finally {
    starting.value = false;
  }
}

async function transcribe(blob: Blob, filename: string) {
  const view = props.view;
  if (!view || blob.size === 0) return;
  transcribing.value = true;
  try {
    const text = await agency.transcribeAudio({ file: blob, filename });
    if (!disposed && text) insertAtSelection(view, text);
  } catch (error) {
    toast.error(error instanceof Error ? error.message : String(error));
  } finally {
    transcribing.value = false;
  }
}

function stopServerRecording() {
  if (!recorder && !stream) return;
  stopTimer();
  recording.value = false;
  if (recorder && recorder.state !== "inactive") {
    recorder.stop();
  }
  recorder = undefined;
  stopStream();
}

function stopStream() {
  stream?.getTracks().forEach((track) => track.stop());
  stream = undefined;
}

let recognition: SpeechRecognitionLike | undefined;
let finalTranscript = "";
let interimTranscript = "";
let lastTranscript = "";
let stoppedByUser = false;

function startBrowserRecognition() {
  if (!browserRecognizer) return;
  recognition = new browserRecognizer();
  recognition.continuous = true;
  recognition.interimResults = true;
  recognition.maxAlternatives = 1;
  recognition.lang = navigator.language || "en-US";
  finalTranscript = "";
  interimTranscript = "";
  lastTranscript = "";
  stoppedByUser = false;

  recognition.onresult = (event) => {
    interimTranscript = "";

    for (let index = event.resultIndex; index < event.results.length; index++) {
      const result = event.results[index]!;
      const transcript = result[0]?.transcript ?? "";
      if (result.isFinal) {
        finalTranscript = joinTranscript(finalTranscript, transcript);
      } else {
        interimTranscript = joinTranscript(interimTranscript, transcript);
      }
    }
    const text = joinTranscript(finalTranscript, interimTranscript);
    if (!text || text === lastTranscript) return;
    lastTranscript = text;
    const view = props.view;
    if (view) replaceDictationText(view, text);
  };

  recognition.onerror = (event) => {
    if (!isBenignRecognitionError(event.error)) {
      toast.error(recognitionErrorMessage(event.error));
    }
    finishBrowserRecognition();
  };

  recognition.onend = () => {
    if (recording.value) finishBrowserRecognition();
  };

  try {
    recognition.start();
    ensureRangeField(props.view!);
    const selection = props.view!.state.selection.main;
    props.view!.dispatch({
      effects: setDictationRange.of({
        from: selection.from,
        to: selection.to,
      }),
    });
    recording.value = true;
    startTimer();
  } catch (error) {
    toast.error(
      error instanceof Error ? error.message : t("voice.errors.startFailed"),
    );
    recognition = undefined;
  }
}

function stopBrowserRecognition() {
  if (!recognition) return;
  stoppedByUser = true;
  stopTimer();
  recording.value = false;
  recognition?.stop();
  recognition = undefined;
  clearRange(props.view);
}

function finishBrowserRecognition() {
  stopTimer();
  recording.value = false;
  recognition = undefined;
  if (stoppedByUser) stoppedByUser = false;
  clearRange(props.view);
}

function toggle() {
  if (recording.value) {
    stopServerRecording();
    stopBrowserRecognition();
    return;
  }
  if (mode.value === "server") void startServerRecording();
  else startBrowserRecognition();
}

onBeforeUnmount(() => {
  disposed = true;
  clearRange(props.view);
  if (recorder && recorder.state !== "inactive") {
    recorder.onstop = null;
    recorder.stop();
  }
  recorder = undefined;
  stopStream();
  recognition?.abort();
  recognition = undefined;
  stopTimer();
});

watch(
  () => props.view,
  (view) => {
    if (view) ensureRangeField(view);
  },
  { immediate: true },
);

watch(available, (stillAvailable) => {
  if (stillAvailable) return;
  stopServerRecording();
  stopBrowserRecognition();
});

const { t } = useI18n();
</script>

<i18n lang="yaml">
en:
  voiceButton:
    transcribing: "Transcribing…"
    stop: "Stop dictation"
    dictateServer: "Dictate (transcribed by the agency)"
    dictateBrowser: "Dictate (browser speech recognition)"
  voice:
    errors:
      micDenied: "Microphone access was denied. Allow it and try again."
      recordUnsupported: "This browser cannot record audio."
      startFailed: "Speech recognition could not start."
ru:
  voiceButton:
    transcribing: "Распознавание…"
    stop: "Остановить диктовку"
    dictateServer: "Диктовать (расшифровывает агентство)"
    dictateBrowser: "Диктовать (распознавание браузера)"
  voice:
    errors:
      micDenied: "Доступ к микрофону запрещён. Разрешите его и повторите попытку."
      recordUnsupported: "Этот браузер не умеет записывать звук."
      startFailed: "Не удалось запустить распознавание речи."
</i18n>
