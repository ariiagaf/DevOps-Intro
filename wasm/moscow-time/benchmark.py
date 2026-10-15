import statistics
import subprocess
import time
import urllib.request

SPIN_URL = "http://127.0.0.1:3000/time"
DOCKER_URL = "http://127.0.0.1:8080/health"


def request(url):
    start = time.perf_counter()
    with urllib.request.urlopen(url, timeout=5) as response:
        if response.status != 200:
            raise RuntimeError(f"HTTP {response.status}")
        response.read()
    return (time.perf_counter() - start) * 1000


def wait_until_ready(url, timeout=30):
    start = time.perf_counter()

    while time.perf_counter() - start < timeout:
        try:
            request(url)
            return (time.perf_counter() - start) * 1000
        except Exception:
            time.sleep(0.05)

    raise RuntimeError(f"Service did not start: {url}")


def benchmark(name, command, url):
    print(f"\n===== {name} =====", flush=True)

    start = time.perf_counter()
    process = subprocess.Popen(
        command,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.STDOUT,
    )

    try:
        cold_start = wait_until_ready(url)
        print(f"Cold start: {cold_start:.2f} ms")

        # Warm-up requests
        for _ in range(20):
            request(url)

        samples = [request(url) for _ in range(200)]
        samples.sort()

        p50 = statistics.median(samples)
        p95 = samples[189]

        print(f"Warm p50: {p50:.2f} ms")
        print(f"Warm p95: {p95:.2f} ms")

    finally:
        process.terminate()
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            process.wait()


benchmark(
    "Spin WebAssembly",
    ["./spin", "up", "--listen", "127.0.0.1:3000", "-f", "wasm/moscow-time/spin.toml"],
    SPIN_URL,
)

benchmark(
    "Docker QuickNotes",
    ["docker", "run", "--rm", "-p", "8080:8080", "quicknotes:lab12"],
    DOCKER_URL,
)