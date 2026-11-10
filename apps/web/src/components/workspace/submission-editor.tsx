"use client";
import { useState, type FormEvent } from "react";
import { api } from "@/lib/api";
type Clip = { id: string; platform: string; content_url: string; caption: string };
export function SubmissionEditor({
  clip,
  platforms,
  onSaved,
  onCancel,
}: {
  clip: Clip;
  platforms: string[];
  onSaved: () => void;
  onCancel: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (busy) return;
    const f = new FormData(e.currentTarget);
    setBusy(true);
    setError("");
    try {
      await api(`clipper/submissions/${clip.id}`, {
        method: "PATCH",
        body: JSON.stringify({
          platform: f.get("platform"),
          content_url: f.get("url"),
          caption: f.get("caption"),
        }),
      });
      onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : "The submission could not be saved.");
    } finally {
      setBusy(false);
    }
  }
  return (
    <form id="submission-editor" onSubmit={submit} className="kf-panel mb-6 space-y-4">
      <h2 className="font-semibold">Edit pending clip</h2>
      <p className="text-sm text-foreground-secondary">
        Only pending submissions can be edited. Moderation fields are managed by the reviewer.
      </p>
      <label className="block text-sm font-medium">
        Platform
        <select name="platform" defaultValue={clip.platform} className="kf-input mt-2">
          {platforms.map((p) => (
            <option key={p} value={p}>
              {p}
            </option>
          ))}
        </select>
      </label>
      <label className="block text-sm font-medium">
        Published clip URL
        <input name="url" type="url" defaultValue={clip.content_url} required className="kf-input mt-2" />
      </label>
      <div>
        <label htmlFor="edit-clip-text-1" className="block text-sm font-medium">
          Caption
        </label>
        <textarea
          id="edit-clip-text-1"
          name="caption"
          defaultValue={clip.caption}
          maxLength={2200}
          required
          className="kf-input mt-2"
          rows={3}
        />
      </div>
      {error && (
        <p
          role="alert"
          className="rounded-xl border border-danger-border bg-danger-subtle p-4 text-sm text-danger-on-subtle"
        >
          {error}
        </p>
      )}
      <div className="flex flex-wrap gap-3">
        <button className="kf-button" disabled={busy}>
          {busy ? "Saving..." : "Save submission"}
        </button>
        <button
          type="button"
          className="kf-button kf-button-secondary"
          onClick={() => {
            if (confirm("Discard your unsaved submission changes?")) onCancel();
          }}
        >
          Cancel
        </button>
      </div>
    </form>
  );
}
