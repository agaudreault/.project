# Maintainer automation

This repo's `maintainers.yaml` is the authoritative source for Argo maintainer
membership and groups. The **org team sync**
([scripts/cmd/sync-org-teams](../scripts/cmd/sync-org-teams), run from
[.github/workflows/sync-maintainers.yaml](../.github/workflows/sync-maintainers.yaml))
reconciles the GitHub org teams (`argocd-approvers`, `argocd-maintainers`,
`argo-workflows-*`, `argo-events-*`, `argo-rollouts-*`, `helm-approvers`, ...)
to match the **managed** teams in `maintainers.yaml`. It **creates** any team
that does not exist yet and sets each team's description to `Managed by
automation on .project/maintainers.yaml. DO NOT EDIT manually.` so the GitHub UI
signals that membership is automated. Teams marked `managed: false` (e.g.
`emeritus`) are skipped.

The tool lives in the Go module under [scripts/](../scripts) and uses a
[maintainers.yaml loader](../scripts/internal/maintainers/maintainers.go) that
parses the roster with a real YAML parser rather than ad-hoc text scraping.

> The `argoproj/argoproj` `MAINTAINERS.md` file is **not** generated. It is
> maintained by hand and points back to this `maintainers.yaml` as the source
> of truth for membership; update it in the same change that edits the roster.

## Behaviour

- On a **pull request** touching `maintainers.yaml` or `scripts/**`, or on a
  manual **workflow_dispatch**: the team sync runs in **dry-run** (prints the
  add/remove/description plan). No changes are made.
- On **push to `main`** touching `maintainers.yaml`: the sync runs with
  `--apply`, creating any missing teams and reconciling membership/descriptions.

In dry-run a team that does not exist yet is shown as `[CREATE TEAM]`; with
`--apply` it is created (privacy "closed") before its members are added.

## One-time setup

### 1. Create a GitHub App (recommended) or a fine-grained PAT

Create a GitHub App owned by the `argoproj` org with these permissions:

- **Organization > Members**: Read and write (team membership + description sync).
- **Repository > Metadata**: Read (required by GitHub).

Install the App on the `argoproj` org. Generate an installation token in the
workflow (e.g. via `actions/create-github-app-token`) or store a PAT.

### 2. Configure repository secrets (this repo)

| Secret | Purpose |
|--------|---------|
| `ORG_TEAM_TOKEN` | Token with org members read+write for team sync. |

If the secret is absent, the sync job logs a notice and is skipped, so the
automation fails safe.

### 3. Grant teams repo access

The automation **creates** the teams referenced by `maintainers.yaml`, but it
does not grant them repository access. For the subproject `CODEOWNERS` reviews
to work, an org owner must grant each team the appropriate access to its
repositories (one-time, and for any newly created team). Preview what the sync
will create/change with a dry-run:

```bash
cd scripts && GITHUB_TOKEN=... go run ./cmd/sync-org-teams --maintainers-yaml ../maintainers.yaml
```

`[CREATE TEAM]` lines are teams that will be created on the next `--apply` run.

## Safety / operations

- Least privilege: the App only needs the org **Members** permission; it does
  not need access to any repository contents.
- Dry-run first: keep `--apply` off until a dry-run looks correct.
- Key rotation: rotate the App private key / PAT periodically and update the
  secret.

## Local usage

Run from the `scripts/` directory (the Go module root):

```bash
cd scripts

# Preview org team changes (no writes; add --apply to write):
GITHUB_TOKEN=... go run ./cmd/sync-org-teams --maintainers-yaml ../maintainers.yaml
```
