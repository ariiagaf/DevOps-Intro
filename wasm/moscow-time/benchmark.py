
import math
import platform
import statistics
import subprocess
import time
import urllib.request

SPIN_URL = "http://127.0.0.1:3000/time"
DOCKER_URL = "http://127.0.0.1:8080/health"

COLD_RUNS = 5
WARMUP_REQUESTS = 20
MEASURED_REQUESTS = 200


def request(url):
    start = time.perf_counter()

    with urllib.request.urlopen(url, timeout=5) as response:
        if response.status != 200:
            raise RuntimeError(f"HTTP {response.status}")
        response.read()

    return (time.perf_counter() - start) * 1000


def wait_until_ready(url, start, timeout=30):
    while time.perf_counter() - start < timeout:
        try:
            request(url)
            return (time.perf_counter() - start) * 1000
        except Exception:
            time.sleep(0.01)

    raise RuntimeError(f"Service did not start: {url}")


def percentile(samples, percent):
    values = sorted(samples)
    index = math.ceil(len(values) * percent / 100) - 1
    return values[index]


def stop_process(process):
    if process.poll() is None:
        process.terminate()

        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()


def benchmark(name, command, url, container_name=None):
    print(f"\n===== {name} =====", flush=True)

    cold_samples = []
    warm_samples = []

    for run in range(1, COLD_RUNS + 1):
        print(f"Cold run {run}/{COLD_RUNS}", flush=True)

        if container_name:
            subprocess.run(
                ["docker", "rm", "-f", container_name],
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
                check=False,
            )

        start = time.perf_counter()

        process = subprocess.Popen(
            command,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.STDOUT,
        )

        try:
            cold_ms = wait_until_ready(url, start)
            cold_samples.append(cold_ms)
            print(f"  Cold start: {cold_ms:.2f} ms", flush=True)

            if run == COLD_RUNS:
                for _ in range(WARMUP_REQUESTS):
                    request(url)

                warm_samples = [
                    request(url)
                    for _ in range(MEASURED_REQUESTS)
                ]

        finally:
            if container_name:
                subprocess.run(
                    ["docker", "rm", "-f", container_name],
                    stdout=subprocess.DEVNULL,
                    stderr=subprocess.DEVNULL,
                    check=False,
                )

            stop_process(process)
            time.sleep(0.2)

    print(f"\n===== {name} RESULTS =====")
    print(f"Cold start samples (ms): {[round(x, 2) for x in cold_samples]}")
    print(f"Cold start median: {statistics.median(cold_samples):.2f} ms")
    print(f"Warm p50: {statistics.median(warm_samples):.2f} ms")
    print(f"Warm p95: {percentile(warm_samples, 95):.2f} ms")


print("===== Benchmark environment =====")
print(f"OS: {platform.platform()}")
print(f"Architecture: {platform.machine()}")
print(f"CPU: {platform.processor() or 'not reported'}")

benchmark(
    "Spin WebAssembly",
    [
        "./spin", "up",
        "--listen", "127.0.0.1:3000",
        "-f", "wasm/moscow-time/spin.toml",
    ],
    SPIN_URL,
)

benchmark(
    "Docker QuickNotes",
    [
        "docker", "run",
        "--name", "quicknotes-lab12-bench",
        "--rm",
        "-p", "8080:8080",
        "quicknotes:lab12",
    ],
    DOCKER_URL,
    container_name="quicknotes-lab12-bench",
)
