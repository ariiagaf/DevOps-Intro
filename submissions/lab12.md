# Lab 12 — WebAssembly Containers: Moscow Time with Spin and TinyGo

**Repository:** https://github.com/ariiagaf/DevOps-Intro  
**Submission branch:** `feature/lab12-clean` (clean branch based on `main`; the original development branch is `feature/lab12`)  
**Spin implementation:** [`wasm/moscow-time/`](https://github.com/ariiagaf/DevOps-Intro/tree/feature/lab12-clean/wasm/moscow-time)  
**Standalone WASI CLI:** [`wasm-cli/`](https://github.com/ariiagaf/DevOps-Intro/tree/feature/lab12-clean/wasm-cli)  
**CI workflow:** [`.github/workflows/lab12-scaffold.yml`](https://github.com/ariiagaf/DevOps-Intro/blob/feature/lab12-clean/.github/workflows/lab12-scaffold.yml)

## Task 1 — HTTP WebAssembly component using Spin (4 pts)

### Implementation and configuration

The application implements a QuickNotes-style single HTTP endpoint using the Spin Go SDK v2. The HTTP handler is registered with `spinhttp.Handle(...)`. A `GET /time` request returns the current Moscow time (fixed UTC+03:00 offset) as JSON; a request using another HTTP method returns `405 Method Not Allowed`. The JSON response contains `unix` (Unix seconds), `iso` (RFC3339 timestamp), and `hour_minute` (`HH:MM`). The HTTP response has `Content-Type: application/json`.

Source code: [`wasm/moscow-time/main.go`](https://github.com/ariiagaf/DevOps-Intro/blob/feature/lab12-clean/wasm/moscow-time/main.go).  
Spin manifest: [`wasm/moscow-time/spin.toml`](https://github.com/ariiagaf/DevOps-Intro/blob/feature/lab12-clean/wasm/moscow-time/spin.toml).  
Dependencies: [`go.mod`](https://github.com/ariiagaf/DevOps-Intro/blob/feature/lab12-clean/wasm/moscow-time/go.mod), [`go.sum`](https://github.com/ariiagaf/DevOps-Intro/blob/feature/lab12-clean/wasm/moscow-time/go.sum).

The project was scaffolded using the Spin `http-go` template and uses the `github.com/spinframework/spin-go-sdk/v2` SDK. The manifest maps the HTTP trigger to `/time`, sets `allowed_outbound_hosts = []`, and builds with the TinyGo WASI Preview 1 target and `-buildmode=c-shared`.

Build command configured in `spin.toml`:

```bash
tinygo build -target=wasip1 -gc=leaking -buildmode=c-shared -no-debug -o main.wasm main.go
```

Build and run from the Spin application directory:

```bash
cd wasm/moscow-time
spin build
spin up
# In another terminal:
curl -i http://127.0.0.1:3000/time
curl -i -X POST http://127.0.0.1:3000/time
```

**Observed CI verification:** `GET /time` returned HTTP 200 with JSON content type; `POST /time` returned HTTP 405. A captured successful response was:

```http
HTTP/1.1 200 OK
content-type: application/json

{"unix":1792023657,"iso":"2026-10-15T03:20:57+03:00","hour_minute":"03:20"}
```

The compiled `main.wasm` was **442,421 bytes** (about 432.1 KiB), as reported by the GitHub Actions artifact-size step.

### Design questions a–d

**a) Browser WASM vs server WASM.** Browser-targeted Go WebAssembly (`GOOS=js GOARCH=wasm`, for example) uses JavaScript/browser integration to access browser-provided facilities such as the DOM and browser APIs. A `wasip1` server module does not automatically have the browser DOM, JavaScript host APIs, or unrestricted OS access. Instead, it uses explicitly provided WASI/host capabilities. This makes it suitable for portable, sandboxed execution outside the browser, with a smaller and more controlled host interface. Note that the `go build -target=js/wasm` syntax in the question is illustrative rather than the standard `go build` flag syntax.

**b) Why `-buildmode=c-shared`?** Spin's Go SDK and generated bindings require the WebAssembly artifact to expose the appropriate host-callable entry points and ABI, rather than only behaving like a standalone command with a `_start` entry point. The TinyGo `-buildmode=c-shared` setting used by the Spin scaffold enables the component-compatible export arrangement expected by this SDK/build pipeline. Omitting it can lead to an incompatible artifact or request handling failure. We retained the scaffold-compatible flag.

**c) Capability security and Docker `--network none`.** `allowed_outbound_hosts = []` grants no outbound HTTP destinations to the Spin component: outbound access is denied unless the host explicitly grants it. This follows a capability-based, least-privilege model. Docker `--network none` disables networking for a container at the network-namespace level. Both reduce network access, but they operate at different layers: Spin grants specific host-mediated capabilities; Docker primarily isolates processes with Linux namespaces and other kernel controls. Neither setting alone guarantees complete application security.

**d) TinyGo standard-library limitations.** TinyGo does not support every part of the upstream Go runtime and standard library identically. Common issues include time-zone database availability (for `time.LoadLocation`) and reflection-heavy JSON operations. This implementation avoids depending on external IANA time-zone data by using `time.FixedZone("UTC+3", 3*60*60)` and serializes a concrete struct rather than a reflection-heavy `map[string]any`. During development, the initial TinyGo 0.43.0 build produced a nil-pointer runtime failure and HTTP 500; switching to TinyGo **0.41.1** resolved it in CI. This observed failure does not by itself prove that time-zone loading or JSON reflection caused the crash.

## Task 2 — Performance comparison with QuickNotes Docker (4 pts)

### Test environment and procedure

Measurements were collected on the same GitHub Actions Ubuntu/Linux runner, not on the local Mac:

```text
OS: Linux-6.17.0-1022-azure-x86_64-with-glibc2.39
Architecture: x86_64
CPU (as reported by Python): x86_64
```

The workflow used Spin CLI **3.4.0**, TinyGo **0.41.1**, and a Docker Engine on the runner. The Docker image was built from the repository's QuickNotes `app/` directory using `docker build -t quicknotes:lab12 ./app`. This reproduces the QuickNotes container baseline from the earlier lab, but the benchmarked tag is `quicknotes:lab12`, **not** an independently verified pre-existing `quicknotes:lab6` image.

Benchmarked endpoints:

- **Spin:** `GET http://127.0.0.1:3000/time`
- **Docker QuickNotes:** `GET http://127.0.0.1:8080/health`

The script [`wasm/moscow-time/benchmark.py`](https://github.com/ariiagaf/DevOps-Intro/blob/feature/lab12-clean/wasm/moscow-time/benchmark.py) launches and stops each service for **five independent cold-start samples**, measuring elapsed time from just before process creation to the first successful HTTP response. For warm latency, it sends **20 warm-up requests** followed by **200 measured sequential requests**, and calculates p50 and p95. This exceeds the assignment's minimum of five warm-ups and 50 measurements; Python's timing/HTTP client is used in place of `hyperfine` or `wrk`.

The runner also reported artifact sizes via `stat` for the WASM module and `docker image inspect` for the image. The Docker cold-start measurements include container launch/readiness overhead; Spin measurements include its server startup and readiness polling.

### Recorded results

| Dimension | QuickNotes Docker | WASM / Spin |
|---|---:|---:|
| Artifact size | **14,075,212 bytes** (13.42 MiB) | **442,421 bytes** (432.1 KiB) |
| Cold start p50 / median (5 runs) | **125.05 ms** | **54.19 ms** |
| Warm HTTP latency p50 (200 requests) | **0.45 ms** | **0.48 ms** |
| Warm HTTP latency p95 (200 requests) | **0.54 ms** | **0.55 ms** |

Cold-start raw samples (milliseconds):

```text
Spin:   61.37, 54.84, 54.10, 54.19, 43.34
Docker: 183.36, 115.16, 125.05, 114.63, 125.47
```

In this run, Spin's median time to first successful response was about **2.31× shorter** than Docker's. The Docker image was about **31.8× larger** than the compiled Spin WASM module. Warm HTTP latencies were very close, with Docker slightly lower in this particular run.

**Limitations:** This is one CI benchmark session on shared runner infrastructure; the two endpoints perform different work (`/time` vs `/health`), and artifact size compares a module file with a complete OCI image. Startup timings include polling and process/container orchestration overhead. These observations should not be generalized into universal runtime-performance claims.

### Design questions e–g

**e) What dominates cold start?** Docker startup may include container creation, Linux namespace/cgroup setup, filesystem/mount initialization, and application process startup; pulling or extracting an image would add overhead if not already cached, but this benchmark used a locally built image. Spin startup includes starting the Spin server, loading/compiling or retrieving compiled WASM code, instantiating the component, setting up WASI/HTTP host interfaces, and becoming ready to serve. The measured figures include more than pure runtime instantiation in both cases.

**f) When is WASM better, and when is Docker appropriate?** WASM is attractive for compact, sandboxed plugins, short-lived functions, edge workloads, and multi-tenant execution where narrow host capabilities and rapid instantiation matter. Docker remains a strong fit for full services with broad OS/library compatibility, mature tooling, background processes, and dependencies that are not easily portable to WASI. The right choice depends on compatibility, operational needs, and real workload measurements.

**g) Multi-tenant safety.** WASM's sandbox makes it harder for untrusted guest code to directly invoke arbitrary host syscalls, inspect another tenant's process memory, or read host files without a granted capability. For example, a malicious plugin trying to open `/etc/shadow` or access another tenant's data cannot simply call unrestricted host OS interfaces from a properly configured WASI sandbox. Docker also provides isolation, but container processes generally use the shared Linux kernel and depend on correct namespace, capability, seccomp, and filesystem policies. WASM isolation is not absolute; host/runtime vulnerabilities remain possible.

## Bonus — Two WebAssembly execution models (2 pts)

### Standalone WASI CLI implementation

The standalone CLI implementation is in [`wasm-cli/main.go`](https://github.com/ariiagaf/DevOps-Intro/blob/feature/lab12-clean/wasm-cli/main.go). It does **not** import the Spin SDK. Instead, it reads the CGI-style `REQUEST_METHOD` and `PATH_INFO` environment variables, accepts `GET /time`, and prints the same Moscow-time JSON structure to standard output. Invalid methods or paths are rejected with an error on standard error and a nonzero exit code. This is a CGI-shaped simulation, not a network listener or a complete CGI HTTP response.

Build and run commands:

```bash
cd wasm-cli
tinygo build -o main.wasm -target=wasi -no-debug ./main.go
wasmtime run --env REQUEST_METHOD=GET --env PATH_INFO=/time main.wasm
```

Observed successful CI output:

```json
{"unix":1792026807,"iso":"2026-10-15T04:13:27+03:00","hour_minute":"04:13"}
```

### Module size and cold-start comparison

The CI workflow built the CLI with the command above, executed it under **Wasmtime 49.0.3**, checked its JSON fields, measured the module size, and timed **five separate `wasmtime run` invocations**.

| Dimension | Spin HTTP component | Standalone WASI CLI |
|---|---:|---:|
| WASM module size | **442,421 bytes** | **376,721 bytes** |
| Cold-start median (5 samples) | **54.19 ms** | **8.66 ms** |
| Execution model | Persistent Spin HTTP host serving requests | One Wasmtime process per invocation |
| Input | HTTP GET `/time` | `REQUEST_METHOD=GET`, `PATH_INFO=/time` |
| Output | HTTP JSON response | JSON on stdout |

CLI cold-start raw samples (milliseconds):

```text
8.85, 8.65, 8.66, 8.62, 10.14
```

The CLI module is **65,700 bytes smaller** (about **14.9%**) than the Spin HTTP module. The CLI's measured process-to-exit median was lower, but **the cold-start numbers are not directly equivalent**: CLI timing includes a fresh process, computation, and exit without an HTTP server; Spin timing measures startup to the first HTTP response, including server/readiness checks. The two modules also expose different interfaces.

### Design questions h–j

**h) Why not `wasmtime run` for the Spin component?** The Spin component is built to expose a WASI HTTP handler interface that a compatible host invokes. Bare `wasmtime run` expects a command-style entry point such as `_start`/the WASI CLI command world; it does not automatically provide an HTTP listener and dispatch requests to the Spin handler. A compatible HTTP host, such as Spin or an appropriately configured `wasmtime serve`, is required for the HTTP component.

**i) What does Spin add over bare Wasmtime?** Spin provides an application manifest and component configuration, an HTTP server and request-routing layer, integration with WASI HTTP and the Spin SDK, host-capability policies such as allowed outbound destinations, and lifecycle/instance-management facilities. Wasmtime is the underlying execution engine; Spin supplies the higher-level application platform around it.

**j) When does each model fit?** Per-invocation `wasmtime run` suits short command-line jobs, scheduled scripts, and event-driven tasks that receive input and exit—for example, converting one file or producing a one-off timestamp. Spin's HTTP host suits continuously available API endpoints, webhooks, and edge services that accept many HTTP requests—for example, the `/time` endpoint in this lab. The choice depends on whether the workload is command/event-oriented or HTTP request/response-oriented.

## Reproducibility and submission

The GitHub Actions workflow builds the Spin component, verifies GET and POST behavior, builds the QuickNotes Docker image, executes the five-run cold-start and 200-request warm-latency benchmark, measures artifact sizes, and builds/runs/benchmarks the standalone WASI CLI.

**Submission checklist:**

- [x] Spin Go SDK handler for `GET /time`, with RFC3339 Moscow-time JSON and content type
- [x] Spin manifest with `/time` routing, `allowed_outbound_hosts = []`, and TinyGo `wasip1` / `-buildmode=c-shared`
- [x] Spin build and HTTP verification evidence; module size
- [x] Design answers **a–d**
- [x] Docker vs Spin table: artifact size, 5 cold starts, warm p50/p95, runner information
- [x] Design answers **e–g**
- [x] Separate `wasm-cli/` implementation with environment-driven input and `wasmtime run`
- [x] CLI build/run commands, module size, 5 cold-start samples, execution-model comparison
- [x] Design answers **h–j**
- [ ] Open the final PR from `feature/lab12-clean` to the **course repository's** `main`
- [ ] Submit the PR URL via Moodle

**Note on branch naming:** The original development branch was `feature/lab12`, but it was based on the still-open Lab 10 branch. To avoid including unrelated Lab 10 changes in this PR, the deliverable was reconstructed on `feature/lab12-clean` based on `main`. If the course requires the PR head branch to be named exactly `feature/lab12`, the branch name must be reconciled before submission; do not force-push or delete the original branch without checking its status.
