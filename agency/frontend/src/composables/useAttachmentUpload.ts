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

import { attachmentFilename } from "@/lib/attachments";
import { agency } from "@/lib/sdk";
import type { Attachment } from "@cherrypi/sdk";
import { ref } from "vue";
import { toast } from "vue3-toastify";

/** The owner an attachment is uploaded against (exactly one is required). */
export interface AttachmentOwner {
  taskId?: string;
  agentId?: string;
  projectId?: string;
}

/**
 * Pick attachments up from any source — the file picker, a clipboard paste, or
 * a drag&drop — and upload them against a fixed owner. One failed file stops
 * that batch but keeps the attachments already uploaded, so a partial success
 * still stages what made it.
 */
export function useAttachmentUpload(owner: () => AttachmentOwner) {
  const uploading = ref(false);

  async function upload(files: Iterable<File>): Promise<Attachment[]> {
    const list = Array.from(files);
    if (!list.length) return [];

    uploading.value = true;
    const uploaded: Attachment[] = [];
    try {
      for (const file of list) {
        uploaded.push(
          await agency.uploadAttachment({
            file,
            filename: attachmentFilename(file),
            ...owner(),
          }),
        );
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      uploading.value = false;
    }
    return uploaded;
  }

  return { uploading, upload };
}
