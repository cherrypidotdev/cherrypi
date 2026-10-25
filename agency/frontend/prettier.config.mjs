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

const config = {
  tabWidth: 2,
  useTabs: false,
  pugAttributeSeparator: "none",
  pugEmptyAttributes: "none",
  pugPreserveWhitespace: false,
  plugins: [
    "@prettier/plugin-pug",
    "prettier-plugin-tailwindcss",
    "prettier-plugin-organize-imports",
  ],
  pugSortAttributesBeginning: [
    "^cols$",
    "^v-else$",
    "^v-for$",
    "^:key$",
    "^v-if$",
    "^v-else-if$",
    "^v-on$",
    "^v-bind$",
    "^ref$",
    "^v-model",
    "^name$",
    "^:?type$",
    "^:value$",
    "^v-text$",
    "^:?label$",
    "^:headers$",
    "^:items$",
    "^:?item-text$",
    "^:?item-value$",
    "^:?item-disabled$",
    "^:?placeholder$",
    "^:?src$",
    "^:?color$",
    "^:?text-color$",
    "^:?icon$",
    "^:?small$",
  ],
  pugSortAttributesEnd: [
    "^:?hint$",
    "^:?persistent-hint$",
    "^prepend-",
    "^@click:prepend",
    "^append-",
    "^@click:append",
    "^:to$",
    "^exact$",
    "^:(?!(width|height|loading|disabled|data-))",
    "^target$",
    "^:?width$",
    "^:?height$",
    "^@click",
    "^@",
    "^:loading$",
    "^:disabled$",
    "^:?data-",
  ],
};

export default config;
