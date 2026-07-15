# slack-prune

`slack-prune` reduces clutter in the Kubernetes Slack by acting on inactive
accounts. Slack has no per-channel "last read" signal, so inactivity is measured
by a user's **workspace-wide last activity** as reported by
[`team.accessLogs`][accesslogs] (most recently, a login or session). A user who
has not been active for longer than a configurable threshold is a prune
candidate.

Two eventual actions share that one activity signal, differing only in threshold
and in what they do:

- **channel-kick** — remove long-inactive members from high-traffic, capacity-
  constrained channels (starting with `#kubernetes-users`) via
  [`conversations.kick`][kick]. Reversible: a kicked user can rejoin freely.
- **deactivate** — deactivate long-inactive accounts workspace-wide via the
  admin `users.admin.setInactive` endpoint (the same mechanism
  [`slack-moderator`](../slack-moderator) already uses).

## Modes

`slack-prune` selects behavior with `--mode`:

- `report` — non-destructive. Walk `team.accessLogs`, build the per-user
  last-active map, and report the volume and runtime of that walk. This exists to
  confirm the walk is tractable (and how long a weekly job takes) before acting
  on the results — the Kubernetes Slack is large, and `team.accessLogs` returns
  one entry per (user, IP, user-agent) combination.
- `channel-kick` — remove long-inactive members from the configured channels via
  [`conversations.kick`][kick]. **Defaults to a dry run.**
- `deactivate` — deactivate long-inactive accounts workspace-wide via the admin
  `users.admin.setInactive` endpoint. **Defaults to a dry run.**

## Usage

Report:

```
slack-prune --mode report --config config.json --cutoff 17520h
```

Channel-kick, dry run (the default):

```
slack-prune --mode channel-kick --config config.json --channels kubernetes-users --cutoff 17520h
```

Channel-kick for real:

```
slack-prune --mode channel-kick --config config.json --channels kubernetes-users --dry-run=false
```

Deactivate, dry run:

```
slack-prune --mode deactivate --config config.json --cutoff 17520h
```

Deactivate for real:

```
slack-prune --mode deactivate --config config.json --dry-run=false
```

### Flags

Common:

- `--mode`: `report`, `channel-kick`, or `deactivate` (default `report`).
- `--config`: path to the Slack auth config file (default `config.json`).
- `--cutoff`: inactivity threshold; a user with no activity for longer than this is a prune candidate (default `17520h`, i.e. 2 years).
- `--page-size`: access-log entries per API call, max 1000 (default 1000).
- `--max-api-calls`: safety cap on API calls **per invocation**, `0` = unlimited (default 5000). With `--checkpoint` set, hitting it pauses resumably; without it the walk bails.
- `--progress-every`: log an access-log progress line every N API calls (default 25).
- `--checkpoint`: path to a checkpoint file. When set, the walk saves progress periodically and resumes from it on restart — see [Measuring at scale](#measuring-at-scale). Omit for a one-shot in-memory walk.
- `--checkpoint-every`: save the checkpoint every N API calls (default 200).

Report mode:

- `--dump-active`: also print every user found active since the cutoff.

Channel-kick mode:

- `--channels`: comma-separated channel names to prune (default `kubernetes-users`).
- `--max-kicks`: safety cap on kicks performed per run (default 500). If hit, re-run to continue.

Deactivate mode:

- `--max-deactivations`: safety cap on deactivations performed per run (default 1000). If hit, re-run to continue.

Shared by channel-kick and deactivate:

- `--dry-run`: if true (the default), only report what would be done.
- `--allow-users`: comma-separated user IDs or usernames to never act on.
- `--store`: local path or `gs://` URL of the durable [activity store](#activity-store); makes runs incremental. Empty means a full walk every run.

### Safety

Both acting modes only proceed when activity coverage extends over the **entire
cutoff window** — if it doesn't (the walk hit the API-call cap, or the store /
log retention doesn't reach back far enough), inactivity can't be confirmed and
the run aborts rather than guessing. Both skip admins, owners, bots/apps,
restricted/guest accounts, and already-deactivated users; cap the number of
actions per run; and honor `--allow-users`. Channel-kick additionally honors
`guardedChannels` from the config and never touches `#general`. Kicked users can
rejoin the channel freely; deactivation is more disruptive to reverse, which is
why it runs on its own, slower cadence with a longer threshold.

## Activity store

Re-walking years of `team.accessLogs` on every run is impractical at scale, so
the acting modes keep a durable **activity store** (`--store`) — a small JSON
document mapping each user ID to their last-active time, plus a high-water mark:

```json
{ "version": 1, "coversSince": 1657000000, "highWater": 1752624000, "users": { "U123": 1710000000 } }
```

- **Backfill (first run, empty store):** walks the access logs all the way back
  to the cutoff, records how far back coverage reaches (`coversSince`), and saves
  the store. This is the one expensive walk; pair it with `--checkpoint` so it
  can resume if interrupted (see [Measuring at scale](#measuring-at-scale)).
- **Incremental (every later run):** fetches only the logs newer than
  `highWater`, merges them in (keeping the newest time per user), advances
  `highWater`, and saves. This is the short, cheap path a weekly job runs.

A run refuses to act unless `coversSince` reaches back to the cutoff, so a store
that was never backfilled far enough (or a workspace whose log retention is
shorter than the cutoff) fails safe instead of over-pruning.

### Where the store lives

`--store` accepts a local filesystem path or a GCS URL (`gs://bucket/object`);
the backend is chosen by the value. GCS access authenticates with the token from
the GKE metadata server (Workload Identity) and pulls in no GCP SDK.

The deployed CronJob keeps the store on a **PersistentVolume** in the
`slack-infra` namespace (see [`cluster/slack-prune`](../cluster/slack-prune)). The
volume is seeded once by a one-off backfill Job that mounts the same claim with
`--store /state/activity.json --checkpoint /state/backfill.json`; the weekly
CronJob then updates it incrementally.

Because the store is login metadata (who was active when), keep whatever holds it
private.

## Measuring at scale

`team.accessLogs` returns one entry per (user, IP, user-agent) combination, so on
a large, active workspace a full multi-year walk can be very large — potentially
hours of rate-limited API calls. Before relying on it, measure it with `--mode
report`, and use `--checkpoint` so the walk can actually finish.

With a checkpoint, the walk periodically persists both its results and its exact
cursor position. If the process is killed (crash, eviction, rate-limit timeout)
or hits the per-invocation `--max-api-calls` cap, just run the same command again
— it resumes where it left off instead of starting over:

```
# Run it; if it prints "PAUSED" or dies, run the exact same line again to continue.
slack-prune --mode report --config config.json --cutoff 17520h \
    --checkpoint /var/lib/slack-prune/activity.json --max-api-calls 0
```

Run this **off the production cluster**. When the walk finishes it reports
`REACHED CUTOFF` (the window is fully walkable — note the API-call count and
elapsed time) or `LOGS EXHAUSTED` (Slack's retention is shorter than the cutoff).
A completed checkpoint is reported as-is on subsequent runs without re-walking, so
you can re-print the numbers cheaply. The same `--checkpoint` also makes the
one-time [activity store](#activity-store) backfill resumable.

Resuming requires the same `--cutoff` and `--page-size`; changing either is
rejected (delete the checkpoint file to start over).

## Config

```json
{
  "accessToken": "xoxp-an-admin-legacy-token"
}
```

A single token is used for everything.

### Getting a token

`slack-prune` relies on the `admin` scope (`team.accessLogs`, and for
`deactivate` the undocumented `users.admin.setInactive`). `admin` is a **legacy
scope**: modern OAuth apps (the flow in [app creation docs](../docs/app-creation.md))
cannot request it.

**Preferred: an existing admin legacy token.** Set `accessToken` to a legacy
`xoxp-…` token belonging to an Admin or Owner. Slack no longer issues new legacy
tokens, but the maintainers already hold one suitable for this; that is what the
deployed job uses.

### Optional: browser session cookie (local testing)

If you don't have the admin token — e.g. for local testing — you can instead
authenticate with a **browser session token** from your own logged-in Admin or
Owner session:

1. Sign in to the workspace in a browser as an admin/owner.
2. **`accessToken` (`xoxc-…`)** — in the DevTools console run
   `JSON.parse(localStorage.localConfig_v2).teams` and copy the `token` for the
   workspace.
3. **`cookie` (`xoxd-…`)** — DevTools → Application → Cookies →
   `https://app.slack.com` → copy the value of the `d` cookie verbatim.

```json
{
  "accessToken": "xoxc-a-browser-session-token",
  "cookie": "xoxd-the-matching-d-cookie-value"
}
```

An `xoxc` token is only accepted together with its `d` cookie, which is why both
are configured; `cookie` is otherwise omitted. These are **session credentials
tied to your account**: treat them like a password, keep them out of git
(`config.json` is gitignored), and note they stop working when that browser
session signs out. Prefer the admin token for anything beyond local testing.

`team.accessLogs` is only available on **paid** workspaces (free workspaces
return `paid_only`). The Kubernetes Slack is a single Business+ workspace, so
this method — rather than the Enterprise-Grid-only Audit Logs API — is the
activity source.

## Deployment

Unlike the webhook-driven tools in this repo, `slack-prune` is a one-shot
command (like [Tempelis](../tempelis)): it runs, reports/acts, and exits. It is
intended to run on a weekly schedule as a Kubernetes `CronJob`. Logs go to
stdout, which the cluster ships to Cloud Logging — that is the audit trail for
what was pruned.

**Token:** the deployed job uses the maintainers' admin legacy token, which does
not expire with a browser session. The optional browser session cookie is for
local testing only — it is tied to a person's login and stops working when that
session ends, so it is not suitable for the unattended weekly job.

[accesslogs]: https://docs.slack.dev/reference/methods/team.accessLogs
[kick]: https://docs.slack.dev/reference/methods/conversations.kick
