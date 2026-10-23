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

import type { VariantProps } from "class-variance-authority";
import { cva } from "class-variance-authority";

export const fieldVariants = cva(
  "data-[invalid=true]:text-destructive gap-2 group/field flex w-full",
  {
    variants: {
      orientation: {
        vertical: "flex-col *:w-full [&>.sr-only]:w-auto",
        horizontal:
          "flex-row items-center has-[>[data-slot=field-content]]:items-start *:data-[slot=field-label]:flex-auto has-[>[data-slot=field-content]]:[&>[role=checkbox],[role=radio]]:mt-px",
        responsive:
          "flex-col *:w-full @md/field-group:flex-row @md/field-group:items-center @md/field-group:*:w-auto @md/field-group:has-[>[data-slot=field-content]]:items-start @md/field-group:*:data-[slot=field-label]:flex-auto [&>.sr-only]:w-auto @md/field-group:has-[>[data-slot=field-content]]:[&>[role=checkbox],[role=radio]]:mt-px",
      },
    },
    defaultVariants: {
      orientation: "vertical",
    },
  },
);

export type FieldVariants = VariantProps<typeof fieldVariants>;

export { default as Field } from "./Field.vue";
export { default as FieldContent } from "./FieldContent.vue";
export { default as FieldDescription } from "./FieldDescription.vue";
export { default as FieldError } from "./FieldError.vue";
export { default as FieldGroup } from "./FieldGroup.vue";
export { default as FieldLabel } from "./FieldLabel.vue";
export { default as FieldLegend } from "./FieldLegend.vue";
export { default as FieldSeparator } from "./FieldSeparator.vue";
export { default as FieldSet } from "./FieldSet.vue";
export { default as FieldTitle } from "./FieldTitle.vue";
