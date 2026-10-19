# Skills catalog and server execution

MCPBox remains Apache-2.0. Downloaded third-party packages retain their own licenses and attribution. This feature does not convert or delete existing SkillBox data.

## User workflow

1. Open **Skills** in the navigation. Search the catalog by name, author or description and filter by category. A bundled snapshot provides an initial catalog; refreshing downloads `https://mcpbox.sh/skills.json`. The URL is configurable and JSON can also be imported locally. Failed refreshes preserve the last successful snapshot. Catalog updates never update installed packages or project bindings automatically.
2. Download a catalog item, import a public GitHub repository plus its Skill directory, or upload a ZIP. ZIP imports require `SKILL.md` at the root, or an explicit package directory. No `npx` install commands or package hooks are executed during import.
3. Inspect package files, origin, commit, requirements and inspection findings. Packages contain `SKILL.md` and optional scripts, references, assets and other resources. Inspection is not proof of safety. Missing compatibility metadata means requirements are unknown.
4. In a project, open **Skills**, press **+**, and connect a downloaded version. One version can be connected to several projects; newer versions remain separate. Duplicating a project copies its bindings, never its job files.
5. For scripts, open the package in the library and configure an execution profile as administrator. Select Python, Node.js or Bash, an image, exact allowed scripts, timeout and optional runtime network access. Save and **Prepare environment**. Preparation downloads the image and, when selected, installs dependencies in a build container. This is an explicit operation which can execute dependency build code and access the network. Jobs do not install dependencies.
6. Create a project job, upload input files, select an approved script and provide a JSON array of arguments, e.g. `["input.csv", "output.txt"]`. Scripts execute on the server with `/work` as the working directory. Download outputs from the same job. A job accepts inputs only before its single execution. Create another job to rerun.

## Runtime

Docker must be installed and running on the MCPBox server. No host Python or Node.js is required. Instructions and package management work without Docker.

Suggested base images are `python:3.12-slim`, `node:22-bookworm-slim` and `debian:bookworm-slim` (Bash). Preparation records the resulting immutable Docker image ID. Profiles can instead use an administrator-built image containing system tools, browser binaries or additional libraries.

Python dependency setup accepts a package requirements `.txt` file; Node.js accepts root `package-lock.json` and runs `npm ci --ignore-scripts`. Packages requiring lifecycle scripts or other installation steps need a prebuilt image. Changing a profile clears its prepared state; prepare it again. Enabling a shared profile authorizes execution for every project connected to that version.

Each job uses an unprivileged container, a read-only root filesystem, dropped capabilities, no-new-privileges, CPU/memory/process/file-size limits and no network by default. The original package is mounted read-only at `/package`; `/work` is the job's own writable directory. Dependency images also contain the package at `/skill`, where scripts run to preserve module resolution. No host project directories, Docker socket or host environment secrets are mounted/passed into the job. Arguments are passed directly to the interpreter, never through a generated shell command. These controls do not make arbitrary third-party code trusted.

Limits: 32 MiB per file, 128 MiB per package/job, 2,000 files, 100 retained jobs per project, 1–300 seconds per execution, 1 MiB each for stdout/stderr, 512 MiB RAM, 1 CPU and 128 processes. Job tree quotas are monitored during execution; this is not a filesystem quota and can transiently overshoot. Delete old jobs to release space. Package traversal, symlinks, special files and duplicate archive paths are rejected. Package hashes are verified before preparation and execution.

On timeout/disconnect MCPBox forcibly removes the named container. If termination cannot be confirmed, the job stays locked as running and its files cannot be downloaded. Restarting MCPBox with Docker available recovers unfinished jobs. Docker is started/managed by the operator, not by MCPBox.

## Storage

`skill_store/` is under MCPBox's data root (beside the database in a normal installation), independent of launch CWD. `packages/<installed-version-id>/` holds complete packages; `jobs/<project-id>/<job-id>/` holds uploaded inputs and outputs. SQLite contains the catalog cache, immutable version metadata, bindings, execution profiles and job history. Back up the database and `skill_store` together. Runtime data is ignored by Git.

Uninstall is refused while a version is connected. Deleting a project removes its bindings and job index records but intentionally retains its job files for operator recovery; remove orphaned directories only after checking backups. Running jobs block project deletion. Existing SkillBox databases are not auto-migrated. Import its portable ZIP packages explicitly and assign them to MCPBox projects.

Disconnecting a Skill from a project is refused while that project retains jobs for the Skill, including unstarted jobs. Wait for running jobs, download any files you need, delete the jobs explicitly, then disconnect. Jobs left by an older version can still be deleted from the project job list.

## MCP tools

Tools are exposed through the existing project endpoint when it has connected Skills. A Skills-only project is a valid MCP connection. Existing project token, bearer/OAuth authorization and pause behavior apply.

| Tool | Purpose |
| --- | --- |
| `list_skills` | Search names/descriptions of this project's connected versions |
| `get_skill` | Load complete `SKILL.md`, requirements, profile and file list |
| `read_skill_file` | Read a package text resource (up to 1 MiB) |
| `create_skill_job` | Create a server job for a connected Skill |
| `upload_skill_file` | Upload base64 input bytes (up to 1 MiB through MCP) |
| `run_skill_script` | Execute an allowed script in its prepared server environment |
| `get_skill_job` | Read status, logs and file manifest |
| `read_skill_job_file` | Download base64 chunks, up to 256 KiB, using `offset`/`limit` |

The agent must have the actual input bytes: a client-local path is not available on the MCPBox server. Larger inputs can be uploaded in the admin UI (32 MiB/file). A chat client must implement its own attachments/upload integration; MCP does not automatically turn a file on the user's computer into server bytes. Clients also decide when to call discovery tools; attaching a Skill does not guarantee that every client will activate it.

## Tool discovery and connection refresh

Skills tools are included in `tools/list` only while at least one Skill version is connected to the project. The initialization hint about `list_skills` is also omitted for projects without Skills. A downloaded package alone does not expose these tools to a project.

Successful binding changes send `notifications/tools/list_changed` to the project's active MCP SSE sessions; initialization advertises `tools.listChanged`. Re-saving an unchanged binding does not send a notification. Clients that support dynamic discovery can fetch the updated catalog. Clients that import or cache tool definitions without listening for these notifications need their connection/tool catalog refreshed or reconnected. This does not require calling `list_skills` on every user message.

If Skills appear in MCPBox but are missing in a chat, compare a fresh `tools/list` response from that exact project endpoint with the client's imported tool list. Confirm the endpoint, listener/port and running binary first; another instance or a saved client catalog can differ from the UI. Server-side tests cannot confirm a third-party connector has refreshed its imported tools.

## Admin API

Read endpoints use `pro:read`; package/catalog/profile mutations require `pro:admin`. Project job creation, uploads and runs use `pro:write`; deletion uses `pro:admin`, following existing shared admin authorization. Runtime enablement is always controlled by the administrator, never a model argument.

- `GET /api/skills`, `/api/skills/catalog`, `/api/skills/runtime`
- `POST /api/skills/catalog/sync` with `{ "url": "https://…" }` or `{ "content": { "schema_version": 1, "items": […] } }`
- `POST /api/skills/install` with `catalog_id`, or `source: { repository, path, ref?, commit? }`
- `POST /api/skills/import`: multipart `file` (ZIP) and optional `path`
- `GET /api/skills/{id}/file?path=SKILL.md`
- `PUT /api/skills/{id}/profile`, `POST /api/skills/{id}/prepare`, `DELETE /api/skills/{id}`
- `GET /api/projects/{id}/skills`; `PUT`/`DELETE /api/projects/{id}/skills/{skill-id}`
- `GET`/`POST /api/projects/{id}/skill-jobs` (creation body: `skill_id`)
- `GET`/`DELETE /api/projects/{id}/skill-jobs/{job-id}`
- `POST …/skill-jobs/{job-id}/files`: multipart `file`
- `POST …/skill-jobs/{job-id}/run`: `script` and `args` array
- `GET …/skill-jobs/{job-id}/file?path=output.txt`: authenticated download

## Central catalog builder

```sh
GOWORK=off go run ./cmd/skills-catalog --out skills.json
```

Discovery follows directory/category/detail links from `https://mcpservers.org/ru/agent-skills` and obtains source repositories without executing their suggested install commands. Repositories are resolved to a commit; the builder reads package metadata from GitHub ZIPs. `--max-pages` limits discovery and causes a failure rather than silently publishing a truncated catalog. Errors never replace an existing output.

The directory may require Cloudflare/browser verification. Use an approved HTML export with `--html directory.html`, or explicit repositories:

```sh
GOWORK=off go run ./cmd/skills-catalog \
  --repositories https://github.com/anthropics/skills,https://github.com/openai/skills,https://github.com/lllllllama/rigorpilot-skills \
  --out skills.json
```

`GITHUB_TOKEN` is optional for API rate limits. Review the generated file, then publish it as `https://mcpbox.sh/skills.json` using the site's normal deployment. This checkout does not deploy that site. The bundled initial snapshot is `internal/skills/catalog_seed.json` (1174 packages from 29 repositories). Regenerate that file to update the offline snapshot in the next binary build.

The builder indexes valid Skill packages from each discovered repository. It does not promise that every package is compatible with MCPBox server execution. Review metadata and prepare an environment per installed version.

## Verification

```sh
GOWORK=off go test ./internal/skills ./internal/httpapi ./internal/storage ./cmd/skills-catalog
MCPBOX_TEST_DOCKER=1 GOWORK=off go test ./internal/skills -run TestRealDockerInputOutput -v
(cd html && npm run build)
```

The matching site asset is prepared in `../MCPSite/public/skills.json`; publishing MCPSite will expose it at `/skills.json`. Repository-level LICENSE/NOTICE files are preserved under `.mcpbox-origin/` when importing a nested GitHub Skill.

Repeatable parser updates, saved HTML input, caching and reports: [parser guide](../cmd/skills-catalog/README.md). The editable source list is `catalog/skills-sources.txt`. Git resolves commits by default; use `--resolve-with api` to opt into GitHub API resolution.
