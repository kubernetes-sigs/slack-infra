# slack-post-message

slack-post-message adds a "Post a message" button in the slack shortcuts section. This can be used to post a message in multiple channels. The message will be posted in the channels selected, by the bot on behalf of the user. Only the users part of the *slack usergroups* given in the config file are permitted to use the bot. This app uses [Slack modals](https://api.slack.com/surfaces/modals) rather than dialogs

![screenshot of a blank send message modal](./screenshot.png)

This app is likely to be called "Announcement Bot" in the Slack Configuration.

## Configuration

slack-post-message requires a configuration file, by default called `config.json` in the working
directory. If the location of the `config.json` is different, change the mount path accordingly in [deployment.yaml](../cluster/slack-post-message/deployment.yaml). The configuration file must look like this:

```json
{
  "signingSecret": "some_slack_signing_secret",
  "accessToken": "xoxp-some-slack-access-token-these-are-very-long-and-start-with-xo",
  "userGroups": ["UsergroupID1", "UsergroupID2"]
}
```
`signingSecret`, `accessToken` are all values provided by Slack when creating and
installing the app. Check out the [slack app creation guide][app-creation] for more details.

For `userGroups`, from slack browser view find the ID(s) of the usergroups in the URL placeholder of the [browser](https://stackoverflow.com/questions/40940327/what-is-the-simplest-way-to-find-a-slack-team-id-and-a-channel-id).

In our specific deployment for kubernetes slack, we use the id for `@contributor-comms` group as shown below:
```
"userGroups": ["STZT16J91"]
```

### Slack setup

- `slack-post-message` requires the following OAuth scopes:

  - `usergroups:read`
  - `channels:read`
  - `groups:read`
  - `chat:write`

- `slack-post-message` requires the following interactive components:

- Set the request URL for the interactive component appropriately. The URL is currently configured as `https://slack-infra.k8s.io/$PATH_PREFIX/webhook`. *PATH_PREFIX* is set as environment variable in [deployment.yaml](../cluster/slack-post-message/deployment.yaml)

  - The location required is 'Global'
  - Recommended action name: "Post message"
  - Callback ID: `write_message`

- slack-post-message does not require any event subscriptions

The [slack app creation guide][app-creation] explains what to do with these values.

## Deployment

Kubernetes runs slack-post-message in a Kubernetes cluster; check out the [config](../cluster/slack-post-message).

slack-post-message can also run on Google App Engine. To do this, create a `config.json` file in this
directory as described above and then run `gcloud app deploy`, using a Google Cloud Platform project
that has [App Engine](https://console.cloud.google.com/appengine) enabled. For most Slack teams,
slack-post-message should fit in the free quota.

<!-- TODO: Explore posting message with a custom username and icon -->

[app-creation]: ../docs/app-creation.md
