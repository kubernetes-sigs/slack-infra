# hack

Use the `Makefile` at the repo root for local builds and checks. Run `make` to
list the targets.

The scripts here are kept only because Prow presubmit jobs call them by path.
Each one runs the matching `make` target:

| Script | Prow job | Runs |
|---|---|---|
| `verify-all.sh` | `pull-slack-infra-verify` | `make verify` |
| `verify-build.sh` | `pull-slack-infra-build` | `make build` |

The jobs are defined in
[kubernetes/test-infra](https://github.com/kubernetes/test-infra/blob/master/config/jobs/kubernetes-sigs/slack-infra/slack-infra-config.yaml).
Renaming or deleting a script here breaks its job. To remove a script, first
change its job to call the `make` target directly, then delete the script once
that change merges.
