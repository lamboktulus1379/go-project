# Sharing Feature Roadmap & Backlog

This document tracks completed work, near-term next steps, and longer‑term backlog items for the social sharing subsystem (Facebook + future platforms) spanning backend (Go) and frontend (Angular).

## ✅ Completed (Baseline)
- Facebook OAuth (user token + page token upgrade; manual page linking by ID or URL).
- Share tracking tables + server_post job queue (video_share_records, share_jobs, audits).
- Manual job trigger endpoint `/api/share/process-jobs` + background ticker.
- Force re-share flag (`force=true`) enabling re-queuing even after success.
- External reference capture (Facebook post ID) stored as `external_ref`.
- Auto schema ensure for `external_ref` columns if missing at startup.
- Enriched Facebook post content: title, description (truncated), hashtags (extracted or fallback), YouTube link.
- Hashtag extraction + default fallback `#alkitab #ayatalkitab #grateful`.
- Frontend: disable already-shared platforms unless user toggles Force; structured share UI.
- JWT issuer claim fix & 24h expiry extension.

## 🚀 Near-Term (Promoted Next Tasks)
1. Configurable Hashtags
   - Add config (env / config.json) `share.default_hashtags` and optional `share.max_auto_hashtags`.
   - Override fallback list; enforce cap.
2. Share Status Preload in UI
   - On video details load, call `/api/youtube/videos/:id/share-status` to pre-populate `sharedPlatforms` before user interacts.
   - Display last shared timestamp & external_ref tooltip.
3. Delete Facebook Post Endpoint
   - `DELETE /api/share/facebook/:recordId` -> use stored `external_ref` (post ID) to call Graph API delete; update audit & mark record status (maybe `deleted`).
   - Add soft-delete or separate `share_deletions` audit row.
4. Retry / Backoff Logic
   - Add columns: `next_attempt_at`, `attempts` already present; schedule exponential backoff (e.g., 1m, 5m, 15m, 1h, stop at 5 attempts) for transient failures.
   - Distinguish transient vs terminal errors (HTTP 5xx vs permission errors).
5. Template Customization
   - Configurable message template per platform: e.g. `{{title}}\n\n{{description}}\n\n{{hashtags}}\n\n{{url}}`.
   - Allow enabling/disabling description or hashtags.

## 📊 Medium-Term Backlog
- Analytics Fetch Endpoint
  - Given `external_ref`, fetch engagement stats (likes, shares, comments) if permissions allow; store snapshot table `share_metrics`.
- Post Health Checker
  - Scheduled job verifying that `external_ref` posts still exist; update status = `removed` if missing.
- Multi-Platform Expansion
  - Stub adapters for Twitter (API v2 / X), WhatsApp share link (client only), Instagram (limitations), LinkedIn.
- Role-Based Force Policy
  - Only admins can Force re-share; regular users see re-share cooldown.
- Share Attempt Classification
  - Add `error_code` field for structured error analytics.
- Observability / Metrics
  - Expose Prometheus metrics (jobs processed, failures, avg latency, retries, deletions).
- Bulk Share Endpoint
  - Share multiple videos in one request for batch operations (respect individual job results).

## 🧪 Testing & Quality Enhancements
- Add integration tests: share job happy path (success), missing token, no page token, retry scenario.
- Mock Facebook Graph API client interface for deterministic tests.
- Frontend e2e: share flow including Force toggle and status preload.

## 🔐 Security & Compliance
- Encrypt stored OAuth access tokens at rest.
- Add per-user rate limits for share requests.
- Audit log expansion: include IP/user-agent for share & delete actions.

## 🗄️ Database / Schema Roadmap
| Change | Status | Notes |
|--------|--------|-------|
| external_ref columns | Done | Added via EnsureShareSchema |
| next_attempt_at column (share_jobs) | Planned | Needed for backoff |
| error_code column (share_jobs, records) | Planned | For structured retry logic |
| share_metrics table | Backlog | Analytics snapshots |
| share_templates table (optional) | Backlog | Overrides per platform/user |

## 🛠 Implementation Notes (Guidance)
- Prefer adding a thin platform adapter layer: `PlatformPoster` interface with methods `Post`, `Delete`, `FetchMetrics`.
- Message templating: use `text/template` with safe variable whitelist.
- Retry scheduler: run in same ticker loop; select pending jobs where `status='pending' AND (next_attempt_at IS NULL OR next_attempt_at <= now())`.
- Deletion: keep original record; add `deleted_at` nullable timestamp or new status `deleted`.

## 🧾 Acceptance Criteria for Promoted Tasks
1. Configurable Hashtags
   - Given custom env list, fallback hashtags replaced; extraction still de-dupes case-insensitively.
2. Share Status Preload
   - Visiting video page immediately reflects shared state before any manual sharing.
3. Delete Endpoint
   - Successful deletion removes Facebook post, records external_ref in deletion audit, returns 200 JSON `{deleted:true}`.
4. Retry/Backoff
   - Transient network failure triggers reschedule; permission error marks failed permanently.
5. Template Customization
   - Changing template in config & restarting updates subsequent share messages (not retroactive) without code changes.

## 🧭 Suggested Order of Execution
1. Configurable hashtags & template foundation (unblocks message customization).
2. Share-status preload (small FE win).
3. Delete endpoint (unlocks lifecycle completeness).
4. Retry/backoff (resilience).
5. Metrics & analytics (observability & insights).

## 📝 Quick Commands (Reference)
(Only if manual migration needed; runtime auto-migration already covers external_ref.)
```sql
ALTER TABLE video_share_records ADD COLUMN IF NOT EXISTS external_ref TEXT;
ALTER TABLE share_jobs ADD COLUMN IF NOT EXISTS external_ref TEXT;
```

---
Keep this file updated after each feature merge (append Completed section & move items from Near-Term to Completed).
