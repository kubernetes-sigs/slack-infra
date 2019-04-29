# slack-report-message

slack-report-message adds a "Report message" button to every message and sends the resulting
reports to a predetermined channel. The reported messages include the message content, even when
sent from a direct message. Message reports from channels or groups can be sent anonymously.

![screenshot of a blank report message dialog](./screenshot.png)

## Configuration

slack-report-message requires a configuration file, by default called `config.json` in the working
directory. It must look like this:

```json
{
  "signingSecret": "some_slack_signing_secret",
  "accessToken": "xoxp-some-slack-access-token-these-are-very-long-and-start-with-xoxp",
  "webhook": "https://hooks.slack.com/services/Tsomething/Banotherthing/somerandomsecret"
}
```

`signingSecret`, `accessToken`, and `webhook` are all values provided by Slack when creating and
installing the app. Check out the [slack app creation guide][app-creation] for more details.

### Slack setup

slack-report-message requires the following OAuth scopes:

- `commands`
- `incoming-webhook`
- `users:read`

slack-report-message requires the following interactive components:

- Callback ID: `report_message`. Recommended action name: "Report message"

slack-report-message does not require any event subscriptions.

The [slack app creation guide][app-creation] explains what to do with these values.

## Deployment

Kubernetes does not run slack-report-message. Instead, we run [slack-moderator](../slack-moderator),
which provides a superset of the functionality, but is consequently more invasive.

slack-report-message can run on Google App Engine. To do this, create a `config.json` file in this
directory as described above and then run `gcloud app deploy`, using a Google Cloud Platform project
that has [App Engine](https://console.cloud.google.com/appengine) enabled. For most Slack teams,
slack-report-message should fit in the free quota.

slack-report-message can also run on Kubernetes, but we do not currently have any published Docker
images.

<!-- TODO(Katharine): publish some Docker images. -->

[app-creation]: ../docs/app-creation.md
