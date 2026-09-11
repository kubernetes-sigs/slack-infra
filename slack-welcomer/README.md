# slack-welcomer

slack-welcomer sends a configurable message to every member who joins the Slack team.

## Configuration

slack-welcomer requires a configuration file, by default called `config.json` in the working
directory. It must look like this:

```json
{
  "signingSecret": "some_slack_signing_secret",
  "accessToken": "xoxb-some-bot-access-token-starting-with-xoxb",
}
```

`signingSecret` and `accessToken` are values provided by Slack when creating and
installing the app. Check out the [slack app creation guide][app-creation] for more details.

In addition, slack-welcomer requires a welcome message, written in `mrkdwn`, Slack's thing that is
not really Markdown at all. No good documentation of mrkdwn exists, but the best can be found
[on Slack's developer site](https://api.slack.com/docs/message-formatting).

By default, the welcome message is expected to be found in `welcome.md` in the working directory.

### Slack setup

slack-welcomer requires the following OAuth scopes:

- `bot`
- `chat:write:bot`
- `users:read`

Additionally, `slack-event-log` also requires the following event subscriptions:

- `team_join`

slack-welcomer does not require any interactive components.

The [slack app creation guide][app-creation] explains what to do with these values. Additionally,
you will want to create a bot user, using "Bot Users" in the left sidebar of the Slack app creation
page.

## Deployment

Kubernetes runs slack-welcomer in a Kubernetes cluster; check out the [config](../cluster/slack-welcomer).

slack-welcomer can run on Google App Engine. To do this, create `config.json` and `welcome.md` files in this
directory as described above and then run `gcloud app deploy`, using a Google Cloud Platform project
that has [App Engine](https://console.cloud.google.com/appengine) enabled. For most Slack teams,
slack-welcomer should fit in the free quota.

[app-creation]: ../docs/app-creation.md
