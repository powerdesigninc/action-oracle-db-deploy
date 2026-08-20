# action-oracle-db-deploy

Oracle database CI/CD for GitHub Actions, running SQL through **SQLcl** (`sql`).

The image (`ghcr.io/powerdesigninc/action-oracle-db-deploy`) ships the
`oracle-db-deploy` CLI plus SQLcl, `openconnect` for the VPN step, and git.

## Usage

Call the reusable workflow from your repo — it checks out the caller, brings up
the VPN, and runs the deployment inside the image:

```yaml
name: Deploy Lab02

on:
  workflow_dispatch:
  push:
    branches: [main]

jobs:
  lab02:
    uses: powerdesigninc/action-oracle-db-deploy/.github/workflows/deploy.yml@main
    with:
      DB_HOST: ${{ vars.DB_HOST_LAB02 }}
      DB_USER: ${{ vars.DB_USERNAME }}
      DB_APP_ID: ${{ vars.DB_APP_ID }}
      INSTALL_PATH: ./00_installs/dev
      FLOWS: installs
    secrets:
      VPN_USERNAME: ${{ secrets.VPN_USERNAME }}
      VPN_PASSWORD: ${{ secrets.VPN_PASSWORD }}
      DB_PASSWORD: ${{ secrets.DB_PASSWORD_NON_PROD }}
```

> Replace Lab02 to Lab03 or Prod

### Inputs

| Input | Required | Description |
| --- | --- | --- |
| `DB_HOST` | yes | EZConnect string, e.g. `db.example.com:1521/ORCL` |
| `DB_USER` | yes | Database user / schema |
| `INSTALL_PATH` | yes | Folder holding the install `.sql` files |
| `DB_APP_ID` | only when files are uploaded | APEX application id that receives the static files |
| `FLOWS` | no, default `installs` | Comma separated: `installs`, `files2` |
| `CONTINUE_ON_ERROR` | no, default `true` | Keep executing the remaining install files after a failure |

### Secrets

| Secret | Description |
| --- | --- |
| `VPN_USERNAME` | FortiGate VPN user |
| `VPN_PASSWORD` | FortiGate VPN password |
| `DB_PASSWORD` | Database password for `DB_USER` |

`vars.VPN_HOST` must be set in the calling repo (or its org/environment) — the
workflow reads it directly rather than taking it as an input.

## Flows

**`installs`** — executes every `.sql` file under `INSTALL_PATH` whose hash
changed since the last run. The hash covers the file *and* every sql file it
pulls in with `@"..."` / `@@"..."`, so a change to an included file re-runs its
parent. Files are then scanned for `@includeFile(<path>)` markers; if any are
found, the file upload runs afterwards.

**`files2`** — scans the same install files for `@includeFile(<path>)` markers
and runs the file upload, without executing any sql.

## File upload

Triggered by `@includeFile(...)` markers. Only the marked paths are uploaded.

- `files/scripts` — `npm install && npm run build`, then `files/scripts/dist/*`
  is uploaded under `scripts/`
- `files/styles` — `npx sass --style compressed index.s[ac]ss application.css`,
  then uploaded under `styles/`
- `files/statics` — uploaded as-is

Uploads go through `wwv_flow_api.create_app_static_file` and are hash-tracked in
`apps.xxcicd_histories` under the `file` flow.

## CLI

The reusable workflow is a thin wrapper around this; use it directly when
running outside the workflow.

```
oracle-db-deploy \
  --host <host:1521/service> \
  --user <schema> \
  --password <password> \
  --install-path <folder with the install sql files> \
  [--flows installs] \
  [--app-id <apex application id>] \
  [--continue-on-error]
```

| Flag | Required | Description |
| --- | --- | --- |
| `--host` | yes | EZConnect string passed to SQLcl, e.g. `db.example.com:1521/ORCL` |
| `--user` | yes | Database user / schema |
| `--password` | yes | Database password |
| `--install-path` | yes | Folder holding the install `.sql` files (one layer, `00_template.sql` excluded) |
| `--flows` | no, default `installs` | Comma separated: `installs`, `files2` |
| `--app-id` | only when files are uploaded | APEX application id that receives the static files |
| `--continue-on-error` | no | Keep executing the remaining install files after a failure |

`$GITHUB_REPOSITORY` must be set — the repo name (the part after the slash) is
the key used in the `apps.xxcicd_histories` table.
