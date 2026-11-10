"use client";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
type Rank = {
  rank: number;
  participant_id: string;
  display_name: string;
  views: number;
  approved_clips: number;
};
export function Leaderboard({ campaignID }: { campaignID: string }) {
  const query = useQuery({
    queryKey: ["leaderboard", campaignID],
    queryFn: () => api<{ leaderboard: Rank[] }>(`campaigns/${campaignID}/leaderboard`),
  });
  return (
    <section className="kf-panel mt-6">
      <h2 className="text-lg font-semibold">Campaign leaderboard</h2>
      <p className="mt-2 text-sm text-foreground-secondary">
        Ranked by the latest approved-clip views. Simulated metrics; up to 100 participants.
      </p>
      {query.isPending ? (
        <p role="status" className="mt-5 text-sm">
          Loading rankings...
        </p>
      ) : query.error ? (
        <p role="alert" className="mt-5 text-sm text-danger">
          {query.error instanceof Error ? query.error.message : "Rankings are unavailable."}
        </p>
      ) : !query.data?.leaderboard.length ? (
        <p className="mt-5 text-sm text-foreground-secondary">No approved content has been ranked yet.</p>
      ) : (
        <div className="mt-5 overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr>
                {["Rank", "Creator", "Views", "Approved clips"].map((s) => (
                  <th key={s} className="p-3 font-semibold">
                    {s}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {query.data.leaderboard.map((p) => (
                <tr key={p.participant_id} className="border-t border-border-subtle">
                  <td className="p-3 tabular-nums">{p.rank}</td>
                  <td className="p-3">{p.display_name}</td>
                  <td className="p-3 tabular-nums">{p.views.toLocaleString()}</td>
                  <td className="p-3 tabular-nums">{p.approved_clips}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
