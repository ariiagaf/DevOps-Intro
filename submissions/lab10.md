
# Lab 10 — Cloud Computing: QuickNotes Deployment

## Overview

This lab demonstrates a complete cloud deployment pipeline for QuickNotes using GitHub Actions, GitHub Container Registry (GHCR), and Render.

The application was packaged as a Docker image, published through a tag-triggered CI workflow, and deployed to a publicly accessible Render Free Web Service.

**Repository:** https://github.com/ariiagaf/DevOps-Intro

**Branch:** `feature/lab10`

**Public service:** https://quicknotes-lab10-r2gw.onrender.com

**Deployment option:** Option A — Render

---

## Task 1 — Release Pipeline and GHCR (6 points)

### 1.1 Docker Image

The Dockerfile is located at `app/Dockerfile`.

It uses a multi-stage build:

- Build stage: Go 1.26.6.
- Runtime stage: `gcr.io/distroless/static-debian12:nonroot`.
- Static Go binary compiled with `CGO_ENABLED=0`.
- Dedicated healthcheck binary.
- Non-root runtime user `65532:65532`.
- Application port `8080`.

The image is built for `linux/amd64` to ensure compatibility with the cloud deployment environment.

### 1.2 GitHub Actions Release Workflow

The release workflow is located at:

`.github/workflows/release.yml`

It is triggered when a version tag matching `v*` is pushed.

The workflow performs the following operations:

1. Checks out the repository.
2. Authenticates to GHCR using `GITHUB_TOKEN`.
3. Generates Docker image metadata and tags.
4. Builds and publishes the Docker image.
5. Calls the Render deploy hook to deploy the released version.

All third-party GitHub Actions are pinned to full 40-character commit SHAs.

The workflow uses the following permissions:

```yaml
permissions:
  contents: read
  packages: write
```

The image is published with both a version-specific tag and `latest`.

### 1.3 Published Container Image

**Registry:** GitHub Container Registry

**Versioned image:**

`ghcr.io/ariiagaf/devops-intro/quicknotes:v0.1.0`

**Latest tested release:**

`ghcr.io/ariiagaf/devops-intro/quicknotes:v0.1.2`

**Moving tag:**

`ghcr.io/ariiagaf/devops-intro/quicknotes:latest`

The GHCR package visibility was confirmed as Public in the GitHub package settings.

The initial release was created using a signed annotated Git tag:

```bash
git tag -a -s v0.1.0 -m "Lab 10 release"
git push origin v0.1.0
```

The tag object contains an SSH signature.

### 1.4 CI Evidence

Successful initial release:

https://github.com/ariiagaf/DevOps-Intro/actions/runs/37844430165/job/113541633059

Release workflow:

https://github.com/ariiagaf/DevOps-Intro/actions/workflows/release.yml

The initial release successfully built and published the image.

The `v0.1.2` release completed successfully, including the automatic Render deployment step.

**Anonymous GHCR pull verification:**

https://github.com/ariiagaf/DevOps-Intro/actions/runs/37857360027/job/113584648315

The verification workflow successfully executed on a clean GitHub-hosted runner without `docker login`. It pulled and inspected `ghcr.io/ariiagaf/devops-intro/quicknotes:v0.1.2`, confirming that the image is publicly pullable.

### 1.5 Design Questions

**a) OIDC versus GITHUB_TOKEN**

`GITHUB_TOKEN` is a short-lived token automatically provided to GitHub Actions. It can authenticate to GHCR without storing a long-lived personal access token.

OIDC allows GitHub Actions to request short-lived credentials from external cloud providers through a trust relationship, avoiding static cloud credentials.

For this lab, `GITHUB_TOKEN` was sufficient because the image was published directly to GHCR. OIDC would be more appropriate when authenticating to a cloud provider that supports federated identity.

**b) Why publish both `latest` and immutable version tags?**

A version-specific tag, such as `v0.1.0`, identifies a particular release and makes deployments easier to reproduce and roll back.

The `latest` tag is a convenient moving reference to the most recently published release, but it should not be relied upon for reproducible deployments.

Version tags are intended to remain stable, while `latest` changes whenever a new release is published. Image digests provide the strongest immutable reference.

**c) How does least privilege apply to the workflow?**

The workflow explicitly grants only `contents: read` and `packages: write`.

`contents: read` allows the workflow to access repository contents, while `packages: write` allows publishing images to GHCR.

The workflow does not require administrative repository permissions or a personal access token. The Render deploy hook is stored as a GitHub Actions secret rather than committed to the repository.

---

## Task 2 — Render Cloud Deployment (4 points)

### 2.1 Deployment Option and Configuration

**Selected option:** Option A — Render Free Web Service.

Hugging Face Docker Spaces required a paid subscription, so Render was selected according to the updated lab instructions.

The service was created using the existing public GHCR image rather than rebuilding the application from source.

Configuration:

| Setting | Value |
|---|---|
| Service name | quicknotes-lab10 |
| Provider | Render |
| Instance type | Free ($0/month) |
| Region | Frankfurt (EU Central) |
| Source | Existing Docker image from GHCR |
| Initial image | `ghcr.io/ariiagaf/devops-intro/quicknotes:v0.1.0` |
| Environment variable `PORT` | `8080` |
| Environment variable `ADDR` | `:8080` |
| Health check path | `/health` |

The configuration is documented in `cloud/render.md`.

The matching port configuration prevented the unnecessary restart caused by a mismatch between Render's expected port and the application's listening port.

**Render deployment logs:**

```text
==> Starting service...
==> Setting WEB_CONCURRENCY=1 by default, based on available CPUs in the instance
2026/10/08 21:41:25 quicknotes listening on :8080 (notes loaded: 0)
==> Your service is live 🎉
==> Available at your primary URL https://quicknotes-lab10-r2gw.onrender.com
```

The deployment logs confirm that QuickNotes started successfully on port 8080 and that Render made the service publicly accessible. The application listening port matches the configured `PORT=8080` and `ADDR=:8080` environment variables.

### 2.2 Public Endpoint Verification

**Service URL:**

https://quicknotes-lab10-r2gw.onrender.com

The following requests were performed:

```bash
curl -i https://quicknotes-lab10-r2gw.onrender.com/health
curl -i https://quicknotes-lab10-r2gw.onrender.com/notes
```

**GET /health**

HTTP status: `200`

Response:

```json
{"notes":0,"status":"ok"}
```

**GET /notes**

HTTP status: `200`

Response:

```json
[]
```

The HTTP response headers included `x-render-origin-server: Render`, confirming that the requests were served through the Render deployment.


**Verbose HTTPS verification:**

Command:

```bash
curl -v https://quicknotes-lab10-r2gw.onrender.com/health 2>&1
```

Relevant output:

```text
* SSL certificate verify ok.
* using HTTP/2
> GET /health HTTP/2
> Host: quicknotes-lab10-r2gw.onrender.com
< HTTP/2 200
< content-type: application/json
< server: cloudflare
< x-render-origin-server: Render

{"notes":0,"status":"ok"}
```

The request confirmed a valid HTTPS connection, a successful HTTP 200 response, and the expected QuickNotes health JSON.

### 2.3 Automatic Deployment from CI

The release workflow contains a deployment step that calls the Render deploy hook after successfully publishing the Docker image.

The hook URL is stored in the GitHub Actions repository secret:

`RENDER_DEPLOY_HOOK`

The released image URL is supplied through the `imgURL` query parameter.

The deploy hook is invoked using `curl` with URL globbing disabled and the image URL encoded through `--data-urlencode`.

An initial deploy-hook attempt failed due to URL handling. The workflow was corrected, and the `v0.1.2` release subsequently completed successfully.

This demonstrates the automated release pipeline:

**Git tag → GitHub Actions → GHCR → Render deploy hook**

The deploy hook URL is not stored in the repository.

### 2.4 Warm Latency Measurements

Five consecutive requests were made to `/health` using `curl` and its `time_total` metric.

| Request | Total time (seconds) |
|---|---:|
| 1 | 0.633470 |
| 2 | 0.340963 |
| 3 | 0.421134 |
| 4 | 0.404863 |
| 5 | 0.414965 |

Sorted measurements:

`0.340963, 0.404863, 0.414965, 0.421134, 0.633470`

**Warm p50: 0.414965 seconds (approximately 415 ms).**

### 2.5 Cold-Start Measurements

The Render Free service was left idle for at least 20 minutes between cold-start measurements.

Each cold start was measured using a single HTTP request after the idle period.

| Measurement | Total time (seconds) |
|---|---:|
| Cold #1 | 12.850384 |
| Cold #2 | 13.730062 |
| Cold #3 | 13.065439 |

**Cold median: 13.065439 seconds.**

Cold-start latency was substantially higher than warm latency because the service needed to resume after being idle.

### 2.6 Note Persistence After Sleep

A note was created using:

```bash
curl -i -X POST \
  -H "Content-Type: application/json" \
  -d '{"title":"Lab 10 persistence test","body":"This note tests persistence after Render sleep."}' \
  https://quicknotes-lab10-r2gw.onrender.com/notes
```

The response returned HTTP `201 Created` with the following note information:

```json
{
  "id": 1,
  "title": "Lab 10 persistence test",
  "body": "This note tests persistence after Render sleep."
}
```

After the service was left idle for more than 20 minutes, a subsequent `GET /notes` request returned:

```json
[]
```

**Result:** The note was no longer present after the idle and restart cycle.

This demonstrates that the deployment does not provide durable storage for notes across the observed restart. Persistent external storage would be needed to guarantee data retention.

### 2.7 Design Questions

**d) Render spin-down versus Cloud Run scale-to-zero**

Both Render Free and Google Cloud Run reduce resource usage when services are idle.

Render Free services can spin down after inactivity. Waking a service can involve restarting its container and application, producing noticeable cold-start latency.

Cloud Run is designed for production serverless workloads and can scale instances down to zero while providing automated request-driven scaling. Its startup performance depends on the container, application initialization, configuration, and available infrastructure.

Render Free prioritizes accessible, low-cost hosting, while Cloud Run prioritizes scalable, production-oriented infrastructure.

In this experiment, the Render cold-start measurements were approximately 12.85–13.73 seconds, compared with a warm p50 of approximately 0.415 seconds.

**e) Why does Render inject `PORT` instead of reading Docker `EXPOSE`?**

Docker's `EXPOSE` instruction documents the port a container is expected to use, but it does not force the application to listen on that port.

Render provides a `PORT` environment variable so its routing infrastructure can determine which port the service should use.

QuickNotes listens according to the `ADDR` environment variable, so both settings were explicitly configured:

- `PORT=8080`
- `ADDR=:8080`

If the configured port and the actual listening port differ, Render may detect the application's real port and restart the deployment. This introduces unnecessary startup delay and complicates deployment diagnostics.

Setting both values consistently avoids this mismatch.

**f) Existing Docker image versus building from the repository**

Using an existing GHCR image provides consistency between CI and deployment. Render runs the artifact that was already built and published by GitHub Actions, rather than independently rebuilding the source code.

This improves reproducibility and allows the same image to be reused across environments. It also supports reusing an artifact previously checked by security tooling.

Building directly from the repository can simplify the workflow and allow the deployment provider to use its own build caching, but it introduces another build environment and potentially different outputs.

For this lab, the existing GHCR image was selected to keep the release artifact consistent.

The note disappeared after the service's idle and restart cycle because the current application deployment does not use durable external storage. Data stored only in the running container or ephemeral filesystem cannot be relied upon to survive a restart.

---

## Conclusion

Lab 10 successfully demonstrated an automated cloud release and deployment pipeline.

The QuickNotes application was packaged into a Docker image, published to the public GHCR registry through a signed tag-triggered GitHub Actions workflow, and deployed to a Render Free Web Service.

The release workflow also triggered automatic deployment through a protected Render deploy hook.

The experiment confirmed a significant difference between warm request latency and cold-start latency. It also demonstrated the limitations of ephemeral application storage, as the test note disappeared after the service's idle and restart cycle.

The deployment is suitable for demonstrating cloud delivery, CI/CD, container registries, and scale-to-zero behavior. A production deployment would additionally require durable storage and more comprehensive operational monitoring.

## Submission Artifacts

- `.github/workflows/release.yml`
- `app/Dockerfile`
- `app/cmd/healthcheck/main.go`
- `cloud/render.md`
- `submissions/lab10.md`
- Signed release tags `v0.1.0`, `v0.1.1`, and `v0.1.2`

**Bonus Task:** Not attempted.

