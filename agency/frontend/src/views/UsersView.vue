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
    h1.text-lg.font-semibold {{ $t("nav.users") }}
    p.text-muted-foreground.text-sm
      | {{ $t("usersView.description") }}
  p.text-destructive.text-sm(v-if="error") {{ error }}
  Card
    CardHeader
      CardTitle {{ $t("usersView.title") }}
      CardDescription {{ $t("usersView.hint") }}
    CardContent
      p.text-muted-foreground.text-sm(v-if="loading && !users.length")
        | {{ $t("common.loading") }}
      Empty(v-else-if="!users.length")
        EmptyTitle {{ $t("usersView.empty") }}
        EmptyDescription {{ $t("usersView.emptyHint") }}
      .grid.gap-3(v-else class="sm:grid-cols-2 lg:grid-cols-3")
        RouterLink.flex.items-center.gap-3.rounded-lg.border.p-3.transition-colors(
          v-for="person in users"
          :key="person.id"
          class="hover:bg-muted/50"
          :to="{ name: 'user', params: { username: person.username } }"
        )
          img.size-10.shrink-0.rounded-full.object-cover(
            v-if="person.picture"
            :src="person.picture.url"
            :alt="person.name || person.username"
          )
          span.size-10.shrink-0.rounded-full(
            v-else
            :innerHTML="avatar(person)"
          )
          .flex.min-w-0.flex-col
            .flex.items-center(class="gap-1.5")
              span.truncate.font-medium {{ person.name || person.username }}
              Badge(v-if="person.is_admin" variant="secondary") {{ $t("admin.roleAdmin") }}
            span.text-muted-foreground.truncate.text-xs @{{ person.username }}
            span.text-muted-foreground.truncate.text-xs(
              v-if="person.description"
            ) {{ person.description }}
</template>

<script setup lang="ts">
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Empty, EmptyDescription, EmptyTitle } from "@/components/ui/empty";
import { useUsers } from "@/composables/useUsers";
import type { AccountActor } from "@/lib/account";
import { toSvg } from "jdenticon";
import { onMounted } from "vue";
import { RouterLink } from "vue-router";

const { users, loading, error, loadUsers } = useUsers();

onMounted(() => void loadUsers());

function avatar(person: AccountActor): string {
  return toSvg(person.username || String(person.id), 40);
}
</script>

<i18n lang="yaml">
en:
  usersView:
    description: "People in this agency. Open a profile to see their teams, projects, and agents."
    title: "People"
    hint: "Everyone with an account in this agency."
    empty: "No users yet"
    emptyHint: "New members appear here once they join."
ru:
  usersView:
    description: "Люди в этом агентстве. Откройте профиль, чтобы увидеть команды, проекты и агентов."
    title: "Люди"
    hint: "Все, у кого есть аккаунт в этом агентстве."
    empty: "Пользователей пока нет"
    emptyHint: "Новые участники появятся здесь после присоединения."
</i18n>
