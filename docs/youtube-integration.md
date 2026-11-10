# YouTube public metrics

## Current state

The backend integration and Analytics panel are implemented. No YouTube API key was supplied for this session, so the local UI deliberately shows **Not connected**. Provider behavior is tested against a local HTTP test server, not a live YouTube account.

This first integration reads **public lifetime video counters** for an already approved YouTube submission. It does not verify channel ownership, calculate campaign-attributable view growth, or change the existing simulated metrics/payout ledger. `used_for_payouts` is always false.

## Enable it locally

1. Create/select a Google Cloud project, enable YouTube Data API v3, and create an API key restricted appropriately for your server. Follow the [official authorization setup](https://developers.google.com/youtube/v3/guides/authentication).
2. Edit the ignored repository-root `.env` file and add one line. Preserve any existing variables:

```dotenv
YOUTUBE_API_KEY=your_server_api_key
```

Never use `NEXT_PUBLIC_` for this value and do not paste the key into chat, logs, screenshots, or Git.

3. Restart the local applications so the API receives the key:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\start-local.ps1 -RestartApps
```

The helper imports only `YOUTUBE_API_KEY` from the root `.env`; it never executes dotenv contents. Generated database/JWT settings remain in the ignored runtime configuration. Existing environment values take precedence. RestartApps stops only application processes whose command line identifies this checkout and preserves PostgreSQL/Redis data. Without RestartApps, an already-running API does not reload a changed key.

For Docker, the API service receives `YOUTUBE_API_KEY` through Compose. Recreate/rebuild that service after changing configuration. Docker was not available for testing on this host.

4. Sign in, open Analytics, and find **YouTube public metrics**. A configured key is labelled **API key configured**; this does not claim the key has already been authorized by Google.
5. Select one of up to 20 recent approved YouTube clips visible to your account and choose **Load public counters**. A successful response shows its source, retrieval time and cache expiry. Admin can request **Refresh from YouTube**.

If no clip is available, publish a campaign supporting YouTube, join as a Clipper, submit your direct video URL, and have the Brand/Admin approve it. Campaign creation and content publishing remain user actions.

## Data contract and freshness

The provider calls the official `videos.list` endpoint with `part=statistics`, a validated 11-character video ID, and a restricted fields selection. See [videos.list](https://developers.google.com/youtube/v3/docs/videos/list) and the [statistics resource](https://developers.google.com/youtube/v3/docs/videos#statistics).

Views are required; likes/comments omitted by the provider remain null and display **Not provided**. A reported zero remains zero. Shares, saves, and private dislike counters are not collected. Values are public lifetime counters, not proof of creator ownership or campaign-only performance.

Results live only in a dedicated Redis cache with a 15-minute TTL; they are not inserted into `metric_snapshots` or `payouts`. Requests to the same video are guarded for one minute, including failures. A separate application cap allows at most 1,000 external requests per UTC day, shared by this app's Redis database; Google can apply its own quota independently. Server/API responses to the browser use no-store.

## Authorization and errors

| Route                                                           | Purpose                                                                        |
| --------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| `GET /api/v1/integrations/youtube?campaign_id=...`              | Active user: key-configured flag and actor-scoped approved YouTube submissions |
| `GET /api/v1/integrations/youtube/submissions/{submissionID}`   | Owner Brand, owner Clipper, or Admin: read public counter snapshot             |
| `POST /api/v1/admin/youtube/submissions/{submissionID}/refresh` | Admin only: bypass valid cached data while retaining rate/quota guards         |

The browser calls the existing same-origin `/api/backend/` bridge. Authentication, current account/session checks and CSRF protection remain in place. Service-level authorization executes before any cache read or provider request. Other users receive an enumeration-resistant 404. Pending clips receive 409; no key gives 503; missing video statistics gives 422; rate/quota limits give 429.

User-submitted URLs are parsed only to extract an allow-listed YouTube video ID; they are never fetched. The HTTP target is fixed to Google, redirects are not followed, requests time out, and response size is capped. Provider error messages/request URLs are not returned or logged, so the server key cannot appear in client errors.

## Before using live counters for payouts

Define and verify creator/channel ownership and the intended campaign measurement period, then design an explicit live-data settlement basis with audit/provenance. This integration intentionally leaves the existing simulation-based payout behavior unchanged. Do not present a public lifetime count as creator ownership verification or campaign-attributable earnings.
