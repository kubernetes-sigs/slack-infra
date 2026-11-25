# slack-welcomer

slack-welcomer does two things:

- Sends a configurable welcome message as a direct message to every member who joins the Slack
  team.
- Deletes posts in a configured list of workflow-only channels and tells the poster to use the
  channel's workflow instead. See [Workflow-only channels](#workflow-only-channels).

The second job is unrelated to welcoming new members. It runs here so it can use the existing
Slack app instead of a separate one.

## Configuration

slack-welcomer reads a JSON configuration file, `config.json` in the working directory by default
(set with `--config-path`):

```json
{
  "signingSecret": "some_slack_signing_secret",
  "accessToken": "xoxb-some-bot-access-token-starting-with-xoxb",
  "adminToken": "xoxp-some-user-access-token-starting-with-xoxp",
  "guardedChannels": ["C0123456789"]
}
```

- `signingSecret` and `accessToken` come from Slack when you create and install the app. See the
  [slack app creation guide][app-creation].
- `adminToken` is a user token belonging to a workspace admin or owner. slack-welcomer uses it to
  delete other members' posts.
- `guardedChannels` is a list of channel IDs (not names) to treat as workflow-only.

All four fields are required. slack-welcomer exits at startup if `adminToken` is empty or
`guardedChannels` has no entries.

The welcome message is a separate file, `welcome.md` in the working directory by default (set with
`--message-path`). Write it in [mrkdwn][mrkdwn], Slack's own markup, which is not Markdown.
slack-welcomer reads the file on every join, so edits take effect without a restart.

The server listens on `$PORT` (default `8080`) and serves:

- `/healthz` for readiness checks.
- `$PATH_PREFIX/webhook` for Slack events. `PATH_PREFIX` defaults to empty.

## Workflow-only channels

When a member posts a new top-level message in a channel listed in `guardedChannels`,
slack-welcomer:

1. Looks the member up with `users.info`. Posts by workspace admins and owners are left alone.
2. Deletes the message with `chat.delete`, using `adminToken`.
3. Sends the member a direct message saying the channel only accepts posts submitted through the
   official workflow, and to ask in #slack-admins with questions. The text is set in
   `handler.go` and mentions job listings and resumes.

Bot messages and thread replies are ignored. If the `users.info` lookup fails, the post is left in
place.

The bot user must be a member of each guarded channel; Slack does not send it message events for
channels it is not in.

## Slack setup

Bot token scopes:

- `chat:write`
- `im:write`
- `users:read`

The Kubernetes workspace runs slack-welcomer as a classic Slack app with the `bot` and `users:read`
scopes, which cover the same calls.

`adminToken` needs the `chat:write` user scope.

Event subscriptions (bot events):

- `team_join`, for the welcome message.
- `message.channels`, for workflow-only channels.

Set the Request URL to `https://<your-host>$PATH_PREFIX/webhook`.

slack-welcomer does not use any interactive components.

The [slack app creation guide][app-creation] explains where to enter these values.

## Deployment

The Kubernetes community's deployment lives in the
[kubernetes/k8s.io](https://github.com/kubernetes/k8s.io) repository:

- [`apps/slack-infra/resources/slack-welcomer/`](https://github.com/kubernetes/k8s.io/tree/main/apps/slack-infra/resources/slack-welcomer)
  holds the Deployment, Service, and ExternalSecret.
  - `deployment.yaml` sets the image tag and `PATH_PREFIX=/infra/welcomer`.
  - `message.yaml` is the ConfigMap holding the welcome message. Edit the welcome text here.
  - `externalsecrets.yaml` loads `config.json` from the Google Secret Manager secret
    `slack-welcomer-config` in the `kubernetes-public` GCP project. The tokens and
    `guardedChannels` live in that secret.
- [`apps/slack-infra/resources/ingress.yaml`](https://github.com/kubernetes/k8s.io/blob/main/apps/slack-infra/resources/ingress.yaml)
  routes `/infra/welcomer/*` to the service.
- New images are promoted in
  [`registry.k8s.io/images/k8s-staging-slack-infra/images.yaml`](https://github.com/kubernetes/k8s.io/blob/main/registry.k8s.io/images/k8s-staging-slack-infra/images.yaml)
  and then referenced from `deployment.yaml`.

[app-creation]: ../docs/app-creation.md
[mrkdwn]: https://api.slack.com/reference/surfaces/formatting
