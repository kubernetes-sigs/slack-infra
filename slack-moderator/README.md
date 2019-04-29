# slack-moderator

slack-moderator provides the same message reporting functionality as
[slack-report-message](../slack-report-message), but overrides the behaviour for Slack Admins and
Slack Owners to instead function as a user and content removal tool:

![screenshot of Moderate User prompt](./screenshot.png)

This vastly improves the experience of removing troll users who have been spamming undesirable
content. The user themselves will be deactivated (without going through the Slack user deactivation
mess) and all their content from some time span will be removed.

**Note**: slack-moderator uses an undocumented API to deactivate users. This API is also only
available on paid Slack teams. Content removal uses documented APIs and should work on all Slack
teams.

## Configuration

slack-moderator requires a configuration file, by default called `config.json` in the working
directory. It must look like this:

```json
{
  "signingSecret": "some_slack_signing_secret",
  "accessToken": "xoxp-some-slack-access-token-these-are-very-long-and-start-with-xoxp",
  "adminToken": "xoxp-another-slack-access-token-that-also-starts-with-xoxp",
  "webhook": "https://hooks.slack.com/services/Tsomething/Banotherthing/somerandomsecret"
}
```

`signingSecret`, `accessToken`, and `webhook` are all values provided by Slack when creating and
installing the app. Check out the [slack app creation guide][app-creation] for more details.
Because slack-moderator uses an undocumented API it also needs a legacy token, which is provided
as `adminToken`. A legacy token can be found on
[Slack's Legacy Token page](https://api.slack.com/custom-integrations/legacy-tokens), under
"Legacy token generator".

### Slack setup

The slack-moderator app must be created by a user with Admin or Owner powers. It requires the
following OAuth scopes:

- `chat:write:user`
- `incoming-webhook`
- `files:write:user`
- `commands`
- `search:read`
- `users:read`

slack-moderator also requires the following interactive components:
                     
- Callback ID: `report_message`. Recommended action name: "Report message"
 
slack-moderator does not require any event subscriptions.
 
The [slack app creation guide][app-creation] explains what to do with these values.

## Deployment

Kubernetes runs slack-moderator in a Kubernetes cluster; check out the [config](../cluster/slack-moderator).

slack-moderator can also run on Google App Engine. To do this, create a `config.json` file in this
directory as described above and then run `gcloud app deploy`, using a Google Cloud Platform project
that has [App Engine](https://console.cloud.google.com/appengine) enabled. For most Slack teams,
slack-moderator should fit in the free quota.

[app-creation]: ../docs/app-creation.md
