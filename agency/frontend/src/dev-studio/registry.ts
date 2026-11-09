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

import QuestionnaireWizard from "@/components/tasks/QuestionnaireWizard.vue";
import {
  emptyAnswer,
  type QuestionnaireAnswer,
  type QuestionnaireQuestion,
} from "@/lib/questionnaire";
import type { TaskModelSelection } from "@/lib/task";
import { create } from "@bufbuild/protobuf";
import { TaskConfigSchema, TaskModelSchema } from "@cherrypi/sdk";
import type { StudioComponent } from "./types";

const questions: QuestionnaireQuestion[] = [
  {
    question: "Which database should this service use?",
    header: "Database",
    multiSelect: false,
    options: [
      {
        label: "PostgreSQL",
        description:
          "Already runs in the cluster, with migrations and backups in place.",
        recommended: true,
      },
      {
        label: "SQLite",
        description: "One file, zero operations. Fine until writes contend.",
      },
    ],
  },
  {
    question: "Which test layers should I cover?",
    header: "Testing",
    multiSelect: true,
    options: [
      { label: "Unit", description: "Fast, no I/O." },
      { label: "Integration", description: "Against a real database." },
      {
        label: "End-to-end",
        description: "Drives the UI through the tunnel.",
      },
    ],
  },
];

const topics = [
  "Hosting",
  "Auth",
  "Observability",
  "Rollout",
  "Rollback",
  "Docs",
];
const longQuestions: QuestionnaireQuestion[] = topics.map((topic, index) => ({
  question: `What should the ${topic.toLowerCase()} step do?`,
  header: topic,
  multiSelect: index % 2 === 1,
  options: [
    {
      label: "Ship the default",
      description: `Use the ${topic.toLowerCase()} setup this repo already documents.`,
      recommended: true,
    },
    {
      label: "Ask me first",
      description: "Stop before touching anything and wait for a decision.",
    },
  ],
}));

const answered: QuestionnaireAnswer[] = [
  {
    selected: [0],
    customSelected: false,
    custom: "",
    notes: "**Postgres** it is — the cluster already backs it up.",
  },
  { selected: [0, 1], customSelected: false, custom: "", notes: "" },
];

const custom: QuestionnaireAnswer[] = [
  {
    selected: [],
    customSelected: true,
    custom: "**DuckDB** for the analytics sidecar.",
    notes: "",
  },
  {
    selected: [2],
    customSelected: false,
    custom: "",
    notes: "E2E only for the tunnel path.",
  },
];

const noAnswers = () => questions.map(() => emptyAnswer());

const modelConfig = create(TaskConfigSchema, {
  modelProvider: "anthropic",
  modelId: "claude-sonnet-4-5",
  modelName: "Claude Sonnet 4.5",
  models: [
    create(TaskModelSchema, {
      provider: "anthropic",
      id: "claude-sonnet-4-5",
      name: "Claude Sonnet 4.5",
      reasoning: true,
      thinkingLevels: ["off", "low", "medium", "high"],
    }),
    create(TaskModelSchema, {
      provider: "anthropic",
      id: "claude-haiku-4-5",
      name: "Claude Haiku 4.5",
      reasoning: true,
      thinkingLevels: ["off", "low"],
    }),
    create(TaskModelSchema, {
      provider: "openai",
      id: "gpt-5",
      name: "GPT-5",
      reasoning: true,
      thinkingLevels: ["off", "low", "medium", "high"],
    }),
  ],
  thinkingLevel: "medium",
  thinkingLevels: ["off", "low", "medium", "high"],
});

const modelSelection: TaskModelSelection = {
  modelProvider: "anthropic",
  modelId: "claude-sonnet-4-5",
  modelName: "Claude Sonnet 4.5",
  thinkingLevel: "medium",
};

const agentId = "agent-vlad-desk";

export const studioComponents: StudioComponent[] = [
  {
    id: "questionnaire",
    name: "Questionnaire",
    source: "@/components/tasks/QuestionnaireWizard.vue",
    blurb: "The ask_user_question wizard that ends a task console turn.",
    component: QuestionnaireWizard,
    scenarios: [
      {
        id: "live",
        name: "Live",
        description:
          "The waiting wizard with its model controls. Answer both questions, then submit — the emission is recorded below.",
        props: {
          questions,
          agentId,
          modelConfig,
          modelSelection,
        },
        events: ["submit", "decline", "modelChange"],
        controls: [
          {
            prop: "submitting",
            label: "Submitting",
            type: "boolean",
            initial: false,
          },
          {
            prop: "modelDisabled",
            label: "Model controls disabled",
            type: "boolean",
            initial: false,
          },
          { prop: "errors", label: "Error message", type: "text", initial: "" },
        ],
        width: 640,
      },
      {
        id: "single",
        name: "Single question",
        description: "One question needs no tab strip at all.",
        props: { questions: [questions[0]!] },
        events: ["submit", "decline"],
        width: 480,
      },
      {
        id: "long",
        name: "Many questions",
        description:
          "Six questions: the tab strip scrolls sideways and the check marks track progress.",
        props: { questions: longQuestions },
        events: ["submit", "decline"],
        width: 560,
      },
      {
        id: "submitting",
        name: "Submitting",
        description:
          "While the answers are on their way nothing is editable, model controls included.",
        props: {
          questions,
          agentId,
          modelConfig,
          modelSelection,
          submitting: true,
        },
        width: 640,
      },
      {
        id: "invalid",
        name: "Validation error",
        description: "An error from the host sits under the questions.",
        props: {
          questions,
          errors: "Pick an option for every question before submitting.",
        },
        width: 480,
      },
      {
        id: "answered",
        name: "Answered",
        description:
          "Read-only review of an answered questionnaire, attributed to its author.",
        props: {
          questions,
          submitting: false,
          readonly: true,
          reviewAnswers: answered,
          reviewFinalNote: "Keep the docs in step with the schema.",
          answeredBy: {
            id: "4",
            username: "vladfaust",
            name: "Vlad Faust",
          },
        },
        width: 520,
      },
      {
        id: "custom",
        name: "Custom answers",
        description:
          "The free-text escape chosen on one question and notes kept on the other.",
        props: {
          questions,
          submitting: false,
          readonly: true,
          reviewAnswers: custom,
          answeredBy: { id: "4", username: "vladfaust", name: "Vlad Faust" },
        },
        width: 520,
      },
      {
        id: "declined",
        name: "Declined",
        description: "The user declined, so the answers stay empty.",
        props: {
          questions,
          submitting: false,
          readonly: true,
          cancelled: true,
          reviewAnswers: noAnswers(),
        },
        width: 520,
      },
      {
        id: "unavailable",
        name: "Not shown",
        description:
          "The host had no interactive client, so the questionnaire never appeared.",
        props: {
          questions,
          submitting: false,
          readonly: true,
          unavailable: true,
          reviewAnswers: noAnswers(),
        },
        width: 520,
      },
    ],
  },
];
