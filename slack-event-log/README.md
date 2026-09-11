# slack-event-log

slack-event-log sends a log of interesting Slack events to a Slack channel (and stdout) to make it
clearer to moderators when interesting things are happening. It also doubles as an audit log.

In the name of simplicity, slack-event-log currently only provides the information that can be
directly extracted from the webhook.

## Example output

* Channel #unknown-channel was **deleted**
* Channel [#channel-name](#) was **archived** by [@Some User](#)
* Channel [#channel-name](#) was **unarchived** by [@Some User](#)
* A user was deactivated: [@Some User](#)
* The **Slack team was renamed** to "New Team Name"
* The **Slack team moved** to [https://new-team-name.slack.com](#)
* A **new emoji was added**: `:shipit:` :shipit:
* A **new emoji alias was added**: `:eyeroll:`. It's an alias for `:face_with_rolling_eyes:`. :eyeroll:
* An **emoji was deleted**. It had several names: `:oops:`, `:facepalm:`.

## Configuration

slack-event-log requires a configuration file, by default called `config.json` in the working
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

slack-event-log requires the following OAuth scopes on its Slack app:

- `channels:read`
- `incoming-webhook`
- `emoji:read`
- `usergroups:read`
- `users:read`
- `team:read`

Additionally, slack-event-log also requires the following event subscriptions:

- `channel_archive`
- `channel_created`
- `channel_deleted`
- `channel_rename`
- `channel_unarchive`
- `emoji_changed`
- `subteam_created`
- `subteam_updated`
- `team_domain_change`
- `team_join`
- `team_rename`
- `team_change`

slack-event-log does not require any interactive components.

The [slack app creation guide][app-creation] explains what to do with these values.

## Deployment

Kubernetes runs slack-event-log in a Kubernetes cluster; check out the [config](../cluster/slack-event-log).

slack-event-log can also run on Google App Engine. To do this, create a `config.json` file in this
directory as described above and then run `gcloud app deploy`, using a Google Cloud Platform project
that has [App Engine](https://console.cloud.google.com/appengine) enabled. For most Slack teams,
slack-event-log should fit in the free quota.


[app-creation]: ../docs/app-creation.md
