"use client";
import { useState, type FormEvent } from "react";
import Link from "next/link";
import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { ArrowRight, Check, LoaderCircle } from "lucide-react";
import { Wordmark } from "@/components/brand/wordmark";
import { api } from "@/lib/api";

export function AuthForm({ register = false }: { register?: boolean }) {
  const router = useRouter();
  const client = useQueryClient();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (busy) return;
    const data = new FormData(event.currentTarget);
    setBusy(true);
    setError("");
    try {
      await api(`auth/${register ? "register" : "login"}`, {
        method: "POST",
        body: JSON.stringify({
          email: data.get("email"),
          password: data.get("password"),
          ...(register ? { full_name: data.get("name"), role: data.get("role") } : {}),
        }),
      });
      client.clear();
      router.replace("/workspace");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Please try again.");
    } finally {
      setBusy(false);
    }
  }
  return (
    <main id="main-content" className="grid min-h-dvh lg:grid-cols-2">
      <section className="flex flex-col justify-between border-b border-border bg-surface px-6 py-8 sm:px-12 lg:border-b-0 lg:border-r lg:p-16">
        <Wordmark href="/" />
        <div className="my-12 max-w-lg lg:my-20">
          <p className="text-sm font-semibold uppercase tracking-widest text-primary">
            Creator campaign operations
          </p>
          <h1 className="mt-5 text-3xl font-semibold tracking-tight text-foreground-strong sm:text-4xl">
            Good content.
            <br />
            Clear operations.
          </h1>
          <p className="mt-6 text-base leading-7 text-foreground-secondary">
            One place to manage campaign briefs, review submissions, and understand your content performance.
          </p>
          <ul className="mt-8 space-y-4 text-sm text-foreground-secondary">
            {[
              "Campaigns with clear requirements",
              "Submission reviews with a decision trail",
              "Analytics scoped to your workspace",
            ].map((item) => (
              <li key={item} className="flex items-center gap-3">
                <Check size={18} aria-hidden="true" className="text-primary" />
                {item}
              </li>
            ))}
          </ul>
        </div>
        <p className="text-xs leading-5 text-foreground-muted">
          KlipForge MVP / Social performance metrics are simulated.
        </p>
      </section>
      <section className="flex items-center justify-center p-6 py-12 sm:p-12">
        <div className="w-full max-w-md">
          <p className="text-sm font-semibold text-primary">Your workspace awaits</p>
          <h2 className="mt-3 text-2xl font-semibold text-foreground-strong">
            {register ? "Create your account" : "Welcome back"}
          </h2>
          <p className="mt-3 text-sm leading-6 text-foreground-secondary">
            {register
              ? "Choose how you will use KlipForge. You can start as a brand or a clipper."
              : "Sign in to continue managing your campaigns and content."}
          </p>
          <form
            method="post"
            action={register ? "/api/backend/auth/register" : "/api/backend/auth/login"}
            onSubmit={submit}
            className="mt-8 space-y-5"
            aria-busy={busy}
          >
            {register && (
              <label className="block text-sm font-medium">
                Full name
                <input name="name" autoComplete="name" required maxLength={120} className="kf-input mt-2" />
              </label>
            )}
            <label className="block text-sm font-medium">
              Email address
              <input
                name="email"
                type="email"
                autoComplete="email"
                required
                maxLength={254}
                className="kf-input mt-2"
              />
            </label>
            <label className="block text-sm font-medium">
              Password
              <input
                name="password"
                type="password"
                autoComplete={register ? "new-password" : "current-password"}
                required
                minLength={register ? 8 : undefined}
                maxLength={72}
                className="kf-input mt-2"
              />
              {register && (
                <span className="mt-2 block text-xs text-foreground-muted">
                  Use at least 8 characters, with uppercase, lowercase, and a number.
                </span>
              )}
            </label>
            {register && (
              <label className="block text-sm font-medium">
                I am joining as
                <select name="role" className="kf-input mt-2">
                  <option value="BRAND">Brand - manage campaigns</option>
                  <option value="CLIPPER">Clipper - create content</option>
                </select>
              </label>
            )}
            {error && (
              <p
                role="alert"
                className="rounded-xl border border-danger-border bg-danger-subtle p-4 text-sm text-danger-on-subtle"
              >
                {error}
              </p>
            )}
            <button disabled={busy} className="kf-button w-full">
              {busy ? (
                <LoaderCircle size={18} className="animate-spin" aria-hidden="true" />
              ) : (
                <ArrowRight size={18} aria-hidden="true" />
              )}
              {busy ? "Please wait..." : register ? "Create account" : "Sign in"}
            </button>
          </form>
          <p className="mt-6 text-center text-sm text-foreground-secondary">
            {register ? "Already have an account?" : "New to KlipForge?"}{" "}
            <Link
              href={register ? "/login" : "/register"}
              className="font-semibold text-primary underline underline-offset-4"
            >
              {register ? "Sign in" : "Create an account"}
            </Link>
          </p>
          <Link href="/" className="mt-8 block text-center text-sm text-foreground-muted hover:text-primary">
            Back to KlipForge
          </Link>
        </div>
      </section>
    </main>
  );
}
