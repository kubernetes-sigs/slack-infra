# slack-moderator-words

slack-moderator-words provides a moderation when posting some specific words, and will let the user know how to write better messages.

In Kubenetes slack, this app is usually set up as "Kubernetes Moderator Words".

## Configuration

slack-moderator-words requires a configuration file, by default called `config.json` in the working
directory. It must look like this:

```json
{
  "signingSecret": "some_slack_signing_secret",
  "accessToken": "xoxp-some-slack-access-token-these-are-very-long-and-start-with-xo",
}
```

`signingSecret`, `accessToken` are all values provided by Slack when creating and
installing the app. Check out the [slack app creation guide][app-creation] for more details.

Also, requires a filter file, by default called `filters.yaml` in the working
directory. It must look like this:

```yaml
- triggers:
  - guys
  action: chat.postEphemeral
  message: "May I suggest \"all\" instead when addessing a group of people? Thank you. :slightly_smiling_face:"
```

### Slack setup

slack-moderator-words requires the following OAuth scopes on its Slack app:

- `channels:history`
- `channels:join`
- `channels:read`
- `chat:write`
- `chat:write.public`

Additionally, slack-moderator-words also requires the following Workspace event 
subscriptions (Subscribe to events on behalf of users):

- `channel_created`
- `message.channels`

slack-moderator-words does not require any interactive components.

The [slack app creation guide][app-creation] explains what to do with these values.

## Deployment

Kubernetes runs slack-moderator-words in a Kubernetes cluster; check out the [config](../cluster/slack-moderator-words).

slack-moderator-words can also run on Google App Engine. To do this, create a `config.json` file in this
directory as described above and then run `gcloud app deploy`, using a Google Cloud Platform project
that has [App Engine](https://console.cloud.google.com/appengine) enabled. For most Slack teams,
slack-moderator should fit in the free quota.

[app-creation]: ../docs/app-creation.md
