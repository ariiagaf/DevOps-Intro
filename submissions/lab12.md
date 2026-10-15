# Lab 12 — WebAssembly Containers

**Repository:** https://github.com/ariiagaf/DevOps-Intro  
**Branch:** `feature/lab12-clean`  
**Source:** [`wasm/moscow-time/`](https://github.com/ariiagaf/DevOps-Intro/tree/feature/lab12-clean/wasm/moscow-time) · [`wasm-cli/`](https://github.com/ariiagaf/DevOps-Intro/tree/feature/lab12-clean/wasm-cli) · [CI workflow](https://github.com/ariiagaf/DevOps-Intro/blob/feature/lab12-clean/.github/workflows/lab12-scaffold.yml)

## Task 1 — Spin HTTP endpoint

Implemented `GET /time` using `spinhttp.Handle` and the Spin Go SDK v2. The response contains Moscow time (UTC+3) as `unix`, `iso` (RFC3339), and `hour_minute`, with `Content-Type: application/json`. Other HTTP methods return 405. The Spin manifest routes `/time`, disables outbound hosts (`allowed_outbound_hosts = []`), and uses TinyGo 0.41.1 and Spin 3.4.0.

**Files:** [`main.go`](https://github.com/ariiagaf/DevOps-Intro/blob/feature/lab12-clean/wasm/moscow-time/main.go), [`spin.toml`](https://github.com/ariiagaf/DevOps-Intro/blob/feature/lab12-clean/wasm/moscow-time/spin.toml), [`go.mod`](https://github.com/ariiagaf/DevOps-Intro/blob/feature/lab12-clean/wasm/moscow-time/go.mod), [`go.sum`](https://github.com/ariiagaf/DevOps-Intro/blob/feature/lab12-clean/wasm/moscow-time/go.sum).

```bash
cd wasm/moscow-time
spin build
spin up
# In another terminal:
curl -i http://127.0.0.1:3000/time
```

Configured build command:

```bash
tinygo build -target=wasip1 -gc=leaking -buildmode=c-shared -no-debug -o main.wasm main.go
```

**Verified output (CI):**

```text
HTTP/1.1 200 OK
content-type: application/json

{"unix":1792023657,"iso":"2026-10-15T03:20:57+03:00","hour_minute":"03:20"}
```

`POST /time` returned HTTP 405. **WASM size: 442,421 bytes.**

### Design questions (a–d)

**a) Browser vs server WASM.** Browser WASM uses JavaScript/browser APIs, while WASI modules do not have DOM access or unrestricted OS access. WASI provides portable, sandboxed execution through host-granted interfaces.

**b) `-buildmode=c-shared`.** The Spin Go SDK build requires host-callable exports compatible with its HTTP handler ABI rather than a standalone command entry point. The flag produces the expected build form.

**c) Capability security.** `allowed_outbound_hosts = []` denies outbound HTTP access unless a host capability is granted. Docker `--network none` isolates networking at the Linux network-namespace level; Spin's restriction is host-mediated and capability-specific.

**d) TinyGo limitations.** Time-zone database availability and reflection-heavy standard-library functionality may be limited. We used `time.FixedZone` and JSON serialization of a concrete struct. TinyGo 0.43.0 caused a runtime nil-pointer failure in this setup; TinyGo 0.41.1 passed CI.

## Task 2 — Spin vs QuickNotes Docker

**Environment:** GitHub Actions Ubuntu runner, Linux 6.17.0-1022-azure, x86_64; Spin 3.4.0; TinyGo 0.41.1. The QuickNotes image was built from `app/` as `quicknotes:lab12` for comparison with the Lab 6 container implementation.

**Method:** Five independent cold starts per application, timed from process launch to first successful HTTP response. Warm latency: 20 warm-up requests followed by 200 sequential requests. The Python script is [`benchmark.py`](https://github.com/ariiagaf/DevOps-Intro/blob/feature/lab12-clean/wasm/moscow-time/benchmark.py). Spin used `GET /time`; Docker used `GET /health`.

| Metric | Docker QuickNotes | Spin WASM |
|---|---:|---:|
| Artifact size | 14,075,212 B | 442,421 B |
| Cold start p50 (5 runs) | 125.05 ms | 54.19 ms |
| Warm latency p50 | 0.45 ms | 0.48 ms |
| Warm latency p95 | 0.54 ms | 0.55 ms |

**Cold-start samples (ms):** Spin: 61.37, 54.84, 54.10, 54.19, 43.34; Docker: 183.36, 115.16, 125.05, 114.63, 125.47.

Spin's median cold start was approximately **2.31× faster** in this run. Warm latencies were similar. Measurements include startup/readiness overhead, and the endpoints perform different operations.

### Design questions (e–g)

**e) Cold-start costs.** Docker initializes the container, namespaces, mounts, and application process; image extraction adds cost when needed. Spin loads/instantiates WASM and initializes the WASI/HTTP host. Our measurements also include runtime startup and readiness checks.

**f) Workload suitability.** WASM fits small sandboxed functions, plugins, and edge or multi-tenant workloads. Docker fits full applications requiring broad OS/library compatibility, background services, and mature container tooling.

**g) Multi-tenant safety.** WASM guests cannot directly access arbitrary host files, syscalls, or other tenants' memory without permitted host capabilities. For example, an untrusted module cannot simply read `/etc/shadow` in a properly restricted WASI environment. Docker uses kernel namespaces and other isolation controls but shares the host kernel.

## Bonus — Standalone WASI CLI

Implemented a separate [`wasm-cli/main.go`](https://github.com/ariiagaf/DevOps-Intro/blob/feature/lab12-clean/wasm-cli/main.go) without the Spin SDK. It reads `REQUEST_METHOD` and `PATH_INFO`, prints Moscow-time JSON for `GET /time`, and rejects invalid methods/paths.

**Build and run:**

```bash
cd wasm-cli
tinygo build -o main.wasm -target=wasi -no-debug ./main.go
wasmtime run --env REQUEST_METHOD=GET --env PATH_INFO=/time main.wasm
```

**Verified output (Wasmtime 49.0.3):**

```json
{"unix":1792026807,"iso":"2026-10-15T04:13:27+03:00","hour_minute":"04:13"}
```

| Metric | Spin HTTP | WASI CLI |
|---|---:|---:|
| Module size | 442,421 B | 376,721 B |
| Cold-start median (5 runs) | 54.19 ms | 8.66 ms |
| Execution | Persistent HTTP host | One CLI invocation per run |

**CLI cold-start samples (ms):** 8.85, 8.65, 8.66, 8.62, 10.14. CLI timing covers process launch through completion; Spin timing covers startup through the first HTTP response, so the values measure different execution paths.

### Design questions (h–j)

**h) Why not bare `wasmtime run` for Spin?** The Spin component exposes a WASI HTTP handler rather than a standalone command entry point. It requires an HTTP-capable host such as Spin or a compatible `wasmtime serve` configuration.

**i) What does Spin add?** Spin adds the HTTP server, routing, application manifest, component lifecycle management, and host-capability policies on top of the Wasmtime execution engine.

**j) When to use each?** `wasmtime run` suits one-off CLI jobs, scripts, and scheduled tasks. Spin suits continuously served HTTP APIs, webhooks, and edge endpoints.
