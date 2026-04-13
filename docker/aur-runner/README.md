# Self-Hosted AUR Runner

A Docker-based self-hosted GitHub Actions runner dedicated to publishing
`tpack-bin` to AUR. Runs on any Linux host with Docker installed.

## Why

The `aur-publish` job in `.github/workflows/release.yml` used to run on
GitHub-hosted runners via `KSXGitHub/github-actions-deploy-aur`. Those
pushes were rejected by AUR. Running the push from a host we control
sidesteps the issue.

## Prerequisites

- Docker and Docker Compose v2 on the host.
- A fine-grained GitHub PAT scoped to `tmuxpack/tpack` with
  `Administration: Read & Write` (needed to mint runner registration
  tokens). See
  <https://docs.github.com/en/rest/actions/self-hosted-runners#create-a-registration-token-for-a-repository>.
- The AUR SSH private key as a file on the host, mode `600`, registered
  with the `tmuxpack` AUR account.

## Setup

```bash
cd docker/aur-runner
cp .env.example .env
# Edit .env: fill GITHUB_PAT, AUR_SSH_KEY_PATH, optionally RUNNER_NAME
chmod 600 .env
docker compose up -d
docker compose logs -f aur-runner
```

The runner will appear in GitHub → Settings → Actions → Runners with the
label `aur-publisher` within about 30 seconds.

## Operating

- **Logs:** `docker compose logs -f aur-runner`
- **Restart:** `docker compose restart aur-runner`
- **Stop:** `docker compose down` (this triggers the entrypoint's
  deregistration trap)
- **Update runner version:** bump `RUNNER_VERSION` build arg in the
  Dockerfile, then `docker compose build --no-cache && docker compose up -d`

## Rotation

**PAT rotation:** replace `GITHUB_PAT` in `.env`, then
`docker compose up -d --force-recreate`. The entrypoint fetches a fresh
registration token on each start.

**AUR SSH key rotation:** replace the key file at `AUR_SSH_KEY_PATH` on
the host. Container picks up the new key on next job run (mount is live,
no restart needed).

## Decommission

```bash
docker compose down
```

Then, in GitHub Settings → Actions → Runners, delete the runner entry if
it wasn't cleaned up by the trap (e.g., after `docker kill`).

## Troubleshooting

- **Container restart loop:** likely a bad `GITHUB_PAT`. Check
  `docker compose logs aur-runner` for `Bad credentials`.
- **Runner shows offline:** container may be stopped. `docker compose ps`
  and `docker compose up -d`.
- **AUR push fails with permission denied:** verify the bind-mounted key
  path is correct and the key is registered with your AUR account.
