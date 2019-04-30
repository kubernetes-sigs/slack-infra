# Tempelis

Tempelis updates the state of Slack to match the provided yaml configuration files. Combined with a
CI system, it can be used to implement GitOps for Slack.

## Features

- Creating and archiving channels to match a list in a yaml file.
- Creating, archiving, and modifying usergroups to match a list in a yaml file.
- Restricting what can be defined where in a tree of files (which is useful in combination with an
  OWNERS-type system)

## Usage

* `--auth /path/to/auth`: path to slack auth config file.
* `--config /path/to/config`: path to the Tempelis config file, or the root of the config directory.
* `--dry-run`: does nothing if true, which is the default. Use `--dry-run=false` to run for real.
* `--restrictions`: optional: path to a config file that gives restrictions on what other config
  files can contain.

## Config

### Authentication

Tempelis expects a config file in the location given by `--auth` that looks like this:

```
{
  "accessToken": "xoxp-some-slack-access-token-these-are-very-long-and-start-with-xoxp",
}
```

`accessToken` is a value provided by Slack when creating and installing the app.
Check out the [slack app creation guide][app-creation] for more details, but
note that you can ignore references to `signingSecret` and `webhook`.

**Note**: I strongly suggest creating Tempelis on a dedicated user account - whoever makes the
app will find themselves joined to random channels. Whatever account creates the app must be a
Slack Admin or Owner.

#### OAuth scopes

Tempelis requires the following OAuth scopes:

- `channels:read`
- `channels:write`
- `chat:write:bot`
- `pins:read`
- `pins:write`
- `usergroups:read`
- `usergroups:write`

To run in dry-run mode, only the `read` permissions are required.

Tempelis does not require event subscriptions or interactive components.

### Slack config

The Slack config can live either in a single file or in a directory tree (in which case all files
matching `*.yaml` are assumed to be config files). Tempelis will look for a file or directory tree
in the location given by `--config`.

Except for `restrictions` and `template`, all Tempelis config can be split across multiple files.
The results will be merged, but any duplicates will be considered an error.

#### Restrictions

`restrictions` defines a list of file globs, each of which has a config specifying what files
matching that glob can contain. The list is searched top to bottom and terminates at the first
matching entry. If nothing is matched (or `restrictions` is not specified), the file is assumed
to be fully permissive. An entry for `**/*` can be used as the last entry to match all files not
previously matched. Any property of a retriction for a path not specified is assumed to prevent
including the property in the matching files (there is no fallthrough).

If restrictions are used, pass the file they are defined to `--restrictions` to ensure they are
parsed before the rest of the config and so consistently in effect. 

```yaml
restrictions:
- path: glob
  users: boolean    # true: allow defining user mappings in this file, false: don't
  template: boolean # true: allow defining the channel template in this file, false: don't
  channels:
  - regex list      # list of regexes matching permitted channels. remember to use $ and ^ 
  usergroups:
  - regex list      # list of regexes matching permitted usergroups. remember to use $ and ^ 
```

Check out [Kubernetes' config](https://github.com/kubernetes/community/blob/master/communication/slack-config/restrictions.yaml)
for a practical example.

#### Channels

Tempelis expects a complete list of public channels to be provided. If a public channel exists on
Slack that is not in Tempelis' channel list, it will error out. Tempelis does not, however, care
about private channels at all.

A channel list with a single fully-specified channel looks like this:


```yaml
channels:
- name: slack-admins # mandatory
  id: C4M06S5HS      # optional except when renaming
  archived: false    # optional for unarchived channels
```

To rename a channel, set its `id` property to its current Slack ID, then change
the name. To archive a channel, set `archived` to true. To unarchive it, set `archived` to false
or remove it entirely.

##### Channel templates

Tempelis supports a channel template when creating a channel. This must be defined no more than once:

```yaml
channel_template:
  topic: The initial topic for the channel.
  purpose: The initial purpose for the channel.
  pins:
    - This channel abides to the Kubernetes Code of Conduct.
    - Hey look, another pin.
```

All fields of the template are optional, as is defining one at all.
If pins are specified, Tempelis will send messages with the given content to the channel and immediately
pin them.

#### Users

There is no stable, safe, human readable way to refer to a Slack user. To avoid config files full of
Slack IDs, a `users` block is expected to contain a mapping from human-readable strings to Slack IDs.
Tempelis does not care what the human-readable strings are or place any restrictions on their format,
but we recommend using GitHub usernames. In the future, this mechanism may be supplanted by an
automatic GitHub mapping system. This mapping can be spread across multiple files if necessary.
Duplicate human-readable names will be considered an error.

```yaml
users:
  jeefy: U5MCFK468
  katharine: UBTBNJ6GL
  mrbobbytables: U511ZSKHD
```

#### Usergroups

Usergroups are pingable Slack groups. All members of a usergroup can be
automatically added to certain channels. A usergroup must have at least one
member. 

Tempelis expects a complete list of usergroups. If a usergroup exists but is not defined in Tempelis'
config, it will archive it (notably, this is not the same way it handles unexpected channels).
Consequently, there is no `archived` flag on usergroups - just delete it from the config.

If you have usergroups managed by other tools, you can add them to the config and mark them as
`external`, in which case Tempelis will ignore them.

```yaml
usergroups:
- name: test-infra-oncall
  external: true
- name: slack-admins               # mandatory, the pingable handle
  long_name: Slack Admins          # mandatory, the human-readable name
  description: Slack Admin Group   # mandatory, a description
  channels:                        # optional, a list of channels for members to auto-join
    - slack-admins
  members:                         # mandatory, a list of at least one member.
    - castrojo                     # member names must be listed in the users object.
    - katharine
    - jeefy
    - mrbobbytables
    - idealhack
```

## Deployment

Unlike other tools in slack-infra, Tempelis is structured as a one-shot tool: it reads its config,
performs some action, and then exits.

In Kubernetes, we have it set up as a CI presubmit and postsubmit. The presubmit runs using a
read-only slack app in dry-run mode, and the postsubmit uses a read/write slack app with `--dry-run=false`.

You can take a look at [the presubmit](https://github.com/kubernetes/test-infra/blob/120245b29a7f174f91369da541ff6f82dffcb1f8/config/jobs/kubernetes/community/community-presubmit.yaml#L20-L41)
and [the postsubmit](https://github.com/kubernetes/test-infra/blob/120245b29a7f174f91369da541ff6f82dffcb1f8/config/jobs/kubernetes/test-infra/test-infra-trusted.yaml#L480-L504).

[app-creation]: ../docs/app-creation.md
