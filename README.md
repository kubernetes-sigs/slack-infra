# Kubernetes Slack Infra

This repo contains tooling for the [Kubernetes slack instance](http://kubernetes.slack.com/).

We have the following tools:

- [slack-event-log](./slack-event-log): Event logging for global Kubernetes events. Watch users join, create/remove emoji, and so forth.
- [slack-report-message](./slack-report-message): Enables users to optionally anonymously report messages, and sends those reports to some Slack channel
- [slack-moderator](./slack-moderator): Like `slack-report-message`, but if a Slack Admin or Owner uses the "report" button, instead lets them remove users and/or their content.
- [slack-welcomer](./slack-welcomer): Sends a welcome message to every user who joins Slack.
- [Tempelis](./tempelis): Control your Slack setup with yaml config files. Combine with a CI system to implement Slack GitOps.

## Talk to us

Discussion of slack-infra is on the Kubernetes slack in [#slack-infra](https://kubernetes.slack.com/messages/slack-infra).
Need an invite? Grab one from [slack.k8s.io](https://slack.k8s.io)!

## Deployment

For the purposes of the Kubernetes slack, we run these tools in a Kubernetes cluster (of course).
You can see our configuration for that in the [`cluster`](./cluster) directory.

For a lower-effort deployment, we also support deployment to Google App Engine, with more deployment
options coming soon. The READMEs for each tool discuss deployment of each of them.

## Code of conduct

Participation in the Kubernetes community is governed by the [Kubernetes Code of Conduct](code-of-conduct.md).
