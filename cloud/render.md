
# Lab 10 — Render Deployment

## Deployment Option

**Selected option:** Option A — Render Free Web Service.

Render was selected because it provides a public HTTPS endpoint, supports deployment of prebuilt Docker images from GHCR, and allows automatic redeployment through deploy hooks.

Hugging Face Docker Spaces required a paid subscription, so it was not used.

## Service Configuration

| Setting | Value |
|---|---|
| Service name | quicknotes-lab10 |
| Platform | Render |
| Service type | Web Service |
| Instance type | Free ($0/month) |
| Region | Frankfurt (EU Central) |
| Deployment source | Existing Docker image |
| Initial image | ghcr.io/ariiagaf/devops-intro/quicknotes:v0.1.0 |
| Latest tested release | v0.1.2 |
| PORT | 8080 |
| ADDR | :8080 |
| Health check path | /health |

**Public URL:** https://quicknotes-lab10-r2gw.onrender.com

The existing Docker image was selected to reuse the artifact built and published by GitHub Actions, rather than rebuilding the application separately on Render.

## Endpoint Verification

The public endpoints were tested using curl.

`GET /health` returned HTTP 200:

```json
{"notes":0,"status":"ok"}
```

`GET /notes` returned HTTP 200:

```json
[]
```

The Render deployment completed successfully, and the application was reachable through HTTPS.

## Automatic Deployment from CI

The GitHub Actions release workflow is located at:

`.github/workflows/release.yml`

The workflow:

1. Runs when a version tag matching `v*` is pushed.
2. Builds the Docker image for `linux/amd64`.
3. Publishes versioned and `latest` tags to GHCR.
4. Calls the Render deploy hook after publishing the image.
5. Passes the released image tag using the `imgURL` query parameter.

The deploy hook URL is stored in the GitHub Actions repository secret `RENDER_DEPLOY_HOOK`. It is not committed to the repository.

The `v0.1.2` release workflow completed successfully, including the Render deploy-hook step.

## Cold and Warm Latency

Five consecutive warm requests to `/health` produced:

| Request | Latency (seconds) |
|---|---:|
| 1 | 0.633470 |
| 2 | 0.340963 |
| 3 | 0.421134 |
| 4 | 0.404863 |
| 5 | 0.414965 |

**Warm p50: 0.414965 seconds.**

After separate idle periods of at least 20 minutes, the following cold-start measurements were recorded:

| Measurement | Latency (seconds) |
|---|---:|
| Cold #1 | 12.850384 |
| Cold #2 | 13.730062 |
| Cold #3 | 13.065439 |

**Cold median: 13.065439 seconds.**

The cold-start latency was substantially higher than the warm-request latency because the free service had to resume after being idle.

## Note Persistence After Sleep

A note was created using `POST /notes` and returned HTTP 201.

The created note had the title `Lab 10 persistence test` and ID `1`.

After the service was left idle for more than 20 minutes, `GET /notes` returned:

```json
[]
```

The note was no longer present.

This demonstrates that the current deployment does not provide durable note storage across the observed spin-down and restart cycle. The application would require persistent external storage for reliable data retention.

## Teardown

The service can be suspended or deleted through the Render dashboard under its settings when no longer needed.

