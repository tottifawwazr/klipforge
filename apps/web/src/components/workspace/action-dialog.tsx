"use client";
import { useEffect, useId, useRef, useState, type FormEvent } from "react";

export type Decision = {
  title: string;
  description: string;
  confirmLabel: string;
  field?: { label: string; kind: "reason" | "reference" };
  run: (value: string) => Promise<string | undefined>;
};
export function ActionDialog({ decision, onClose }: { decision: Decision; onClose: () => void }) {
  const dialog = useRef<HTMLDialogElement>(null);
  const id = useId();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    const element = dialog.current;
    element?.showModal();
    return () => element?.close();
  }, []);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending) return;
    const value = String(new FormData(event.currentTarget).get("detail") ?? "").trim();
    if (decision.field && !value) {
      setError(`Enter ${decision.field.kind === "reason" ? "a reason" : "the external payment reference"}.`);
      return;
    }
    setPending(true);
    setError("");
    try {
      const message = await decision.run(value);
      if (message) setError(message);
      else onClose();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The action could not be completed. Try again.");
    } finally {
      setPending(false);
    }
  }
  return (
    <dialog
      ref={dialog}
      aria-labelledby={`${id}-title`}
      aria-describedby={`${id}-description`}
      className="kf-dialog m-auto w-11/12 max-w-lg rounded-2xl border border-border bg-surface p-6 text-foreground shadow-raised"
      onCancel={(event) => {
        event.preventDefault();
        if (!pending) onClose();
      }}
    >
      <form onSubmit={(event) => void submit(event)} aria-busy={pending}>
        <h2 id={`${id}-title`} className="text-xl font-semibold">
          {decision.title}
        </h2>
        <p id={`${id}-description`} className="mt-3 break-words text-sm leading-6 text-foreground-secondary">
          {decision.description}
        </p>
        {decision.field && (
          <div className="mt-5">
            <label htmlFor={`${id}-detail`} className="text-sm font-medium">
              {decision.field.label} (required)
            </label>
            {decision.field.kind === "reason" ? (
              <textarea
                id={`${id}-detail`}
                name="detail"
                required
                maxLength={1000}
                rows={4}
                className="kf-input mt-2"
              />
            ) : (
              <input
                id={`${id}-detail`}
                name="detail"
                required
                maxLength={180}
                className="kf-input mt-2"
                autoComplete="off"
              />
            )}
            <p className="mt-2 text-xs leading-5 text-foreground-muted">
              {decision.field.kind === "reason"
                ? "Explain the decision clearly. The reason is included in the review or payment record."
                : "Use a receipt or transaction reference for a payment already completed outside KlipForge."}
            </p>
          </div>
        )}
        {error && (
          <p
            role="alert"
            className="mt-4 rounded-xl border border-danger-border bg-danger-subtle p-3 text-sm text-danger-on-subtle"
          >
            {error}
          </p>
        )}
        <div className="mt-6 flex flex-wrap justify-end gap-3">
          <button
            type="button"
            onClick={onClose}
            disabled={pending}
            className="kf-button kf-button-secondary"
          >
            Cancel
          </button>
          <button disabled={pending} className="kf-button">
            {pending ? "Saving..." : decision.confirmLabel}
          </button>
        </div>
      </form>
    </dialog>
  );
}
