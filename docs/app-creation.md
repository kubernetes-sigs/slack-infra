# Creating Slack Apps

The tools in slack-infra are designed to serve one Slack team per app instance, and so need to
each have their own Slack app. This document guides you through creating a suitable Slack app.

### Step 1: Create the app

Go to https://api.slack.com/apps and click "Create New App" in the top right:

![screenshot of the page at api.slack.com/apps, with the green "create new app" button in the top right](./images/create-new-app.png)

You will be prompted for an app name and a development slack workspace. The app name can be anything
you like, but keep in mind it will appear in the Slack UI when the app takes actions. The
Development Slack Workspace _must_ be the workspace you intend to use the app with.

![screenshot of the create a slack app prompt](./images/create-a-slack-app.png)

### Step 2: Grant OAuth permissions

To grant the app the OAuth permissions it needs, select "OAuth & Permissions" from the left sidebar.
In the "**Scopes**" section, add the scopes required by the app, which will be listed in the README
for the tool you are deploying. For example:

![screenshot of scopes list with two example scopes added](./images/scopes.png)

Be sure to hit the "Save Changes" button in the "Scopes" box.

Note that you do _not_ need to create any Redirect URLs or IP address ranges.

### Step 3: Install the app to your workspace

Still on the "OAuth & Permissions" page, click the "Install App to Workspace" button. If it's still
disabled, make sure you clicked Save Changes in the Scopes section.

Once you click this button, you will be taken to an authorization screen. If your app can post to
a single channel (i.e. you requested the `incoming-webhook` scope), you will need to select the
channel it sends messages to now.

![screenshot of Slack authorization prompt with a channel requested](./images/authorize.png)

### Step 4: Add Collaborators

In order to prevent loss of control over an app, please add multiple Collaborators
who have admin rights in each app.  This should include at least four of the
people in [the `k8s-infra-rbac-slack-infra` group](https://github.com/kubernetes/k8s.io/blob/main/groups/sig-contributor-experience/groups.yaml),
or ideally all of them.

![screenshot of collaborators for an app](./images/collaborators.png)

### Step 5: Create your config.json

You now have all the information you need to create the `config.json` for most tools.

`config.json` should look similar to this:

```json
{
  "signingSecret": "some_slack_signing_secret",
  "accessToken": "xoxp-some-slack-access-token-these-are-very-long-and-start-with-xo",
  "webhook": "https://hooks.slack.com/services/Tsomething/Banotherthing/somerandomsecret"
}
```

Almost all apps require the `signingSecret` and the `accessToken`.  Depending on 
the app, it may require other settings; see the docs for each individual app
to find out what.

#### `accessToken`

The value for `accessToken` is the "OAuth Access Token" given under "Tokens for Your Workspace"
on the "OAuth & Permissions" page once you've installed the app.

#### `signingSecret`

The value for `signingSecret` can be found on the "Basic Information" page, under App Credentials.
It is hidden by default: click "show" to see it. Be sure not to confuse it with "Client Secret".
We don't use anything  other than "Signing Secret" from this area.

#### `webhook`

If you did not previously select a channel, this field can be omitted. If it is necessary, the
required URL can be found at the bottom of the "Incoming Webhooks" page, under "Webhook URLs for
Your Workspace".

If you have ended up with multiple URLs here, pick the one corresponding to the channel you want to
send messages to.

### Step 6: Deploy your service

We aren't done here, but Slack expects the app to be up and responsive before some configuration
can be completed. Refer to the relevant README for more details.

### Step 7: Set up any event subscriptions

If the tool you are deploying needs event subscriptions, set these up now.

On the "Event Subscriptions"
page, set the toggle to "On", and then enter the event endpoint for your service under "Enable Events"
(tip: probably the URL you deployed it to, plus `/webhook`). Slack will immediately send a challenge
request to that URL and expect a successful response.

![screenshot of Enable Events section with URL successfully entered](./images/events-url.png)

If Slack fails validation, ensure that your service is up and you gave the correct value for
`signingSecret` in your `config.json` (see also step 4).

Once events are enabled, add the events listed in your service's README under the "Subscribe to
Workplace Events" section.

![screenshot of Subscribe to Events](./images/subscibe-to-events.png)

### Step 8: Set up any interactive components

If the tool you are deploying has interactive components, set these up now. 

On the "Interactive Commands" page, set the toggle switch to On, and then enter the interactivity
endpoint for your service in "Request URL" under "Interactivity". Check the 
[ingress configuration](https://github.com/kubernetes/k8s.io/blob/main/apps/slack-infra/resources/ingress.yaml)
to find out what this URL should be. The URL may need `/webhook/` added.
Unlike event subscriptions, Slack will not try to verify the URL.

Once interactive commands are enabled, add the actions specified in the tool's README. While the
value for "Callback ID" must be exactly what is specified in the tool's README, the Action Name and
Description can be anything you like. A preview is shown on the left.

![screenshot of Create Action prompt](./images/create-action.png)

Be sure to click "Save Changes" in the bottom right corner of the Interactive Commands page.

### Step 9: Optionally, customise the app

If you want to change your app's avatar from the default, you can change it under "Display
Information". Note, annoyingly, that all the fields are mandatory despite defaulting to be unset.

### Done!

Tada!

## Additional Notes

Slack has two different sites for managing apps, and limited navigation between
the two sites. `https://api.slack.com/apps/` is the actual app creation and 
configuration site for apps you own or collaborate on, and `https://app.slack.com/apps-manage`,
which lists all the apps for our slack, but has no clear navigation to changing
the actual setup for an app.  Since the latter is the "Manage Apps" link from
the Slack Admin interface, follow the other URL from these docs.
