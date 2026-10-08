# Lab 9: Security Scanning and Hardening

## 1. Overview

This lab introduces automated security checks for the QuickNotes application. The work covers container and repository scanning, static code analysis, application hardening, dynamic security testing, SBOM generation, and integration into GitHub Actions.

## 2. Trivy Security Scanning

### 2.1 Container Image Scanning

The QuickNotes Docker image was scanned using Trivy with HIGH and CRITICAL severity filtering.

Command:

`trivy image --severity HIGH,CRITICAL --exit-code 1 quicknotes:lab6`

The final scan reported zero HIGH or CRITICAL vulnerabilities in the Debian runtime, healthcheck binary, and QuickNotes binary.

The Go builder image was updated to `golang:1.26.6` to address vulnerabilities detected in the previous Go toolchain.

### 2.2 Filesystem Scanning

Command:

`trivy fs --severity HIGH,CRITICAL .`

The local scan detected two HIGH-severity secret findings associated with private keys:

- `.vagrant/machines/default/virtualbox/private_key`
- `app/localhost-key.pem`

These are local, untracked files and were not included in the Git commit. The private keys must remain outside the repository.

### 2.3 Configuration Scanning

Command:

`trivy config .`

The configuration scan reported zero HIGH or CRITICAL findings and one LOW-severity Dockerfile healthcheck finding.

The application has a healthcheck configured through Docker Compose.

### 2.4 Software Bill of Materials

A CycloneDX SBOM was generated for the application image.

Evidence:

`submissions/lab9/sbom.cdx.json`

The CI workflow also generates and uploads an SBOM as a GitHub Actions artifact.

## 3. Static Application Security Testing

The Go application was analyzed using gosec.

The final local gosec scan reported zero issues.

The following hardening changes were implemented:

- Restricted file and directory permissions.
- Improved handling of application data files using Go filesystem APIs.
- Added security-related HTTP response headers.
- Added automated tests for the response headers.

The `/health` endpoint now returns:

- `X-Content-Type-Options: nosniff`
- `Cache-Control: no-store`
- `Cross-Origin-Resource-Policy: same-origin`

The Go unit tests passed successfully.

## 4. OWASP ZAP Baseline Testing

OWASP ZAP baseline scans were performed against the running QuickNotes application before and after HTTP response hardening.

### 4.1 Before Hardening

The initial `/health` scan reported two warnings related to caching behavior and the missing Cross-Origin-Resource-Policy header.

### 4.2 After Hardening

Two security improvements were implemented:

1. Added `Cross-Origin-Resource-Policy: same-origin` to restrict cross-origin resource usage.
2. Added `Cache-Control: no-store` to prevent caching of health endpoint responses.

The subsequent ZAP scan reported one remaining warning, "Non-Storable Content", associated with the intentional `no-store` policy.

The missing CORP header finding was resolved.

### 4.3 Scan Limitations

The ZAP baseline scan is a passive security assessment and does not replace comprehensive penetration testing.

The `/` endpoint returns HTTP 404, and the baseline spider did not exercise all application operations, including POST and DELETE requests.

### 4.4 ZAP Evidence

Reports are available at:

- `submissions/lab9/zap-before.html`
- `submissions/lab9/zap-before.json`
- `submissions/lab9/zap-after.html`
- `submissions/lab9/zap-after.json`
- `submissions/lab9/zap-health.html`
- `submissions/lab9/zap-health.json`
- `submissions/lab9/zap-notes.html`
- `submissions/lab9/zap-notes.json`

## 5. GitHub Actions Security Pipeline

The security workflow is defined in:

`.github/workflows/security.yml`

It runs automatically on pushes to the configured branches and on pull requests.

The pipeline includes:

1. Checkout and Go environment setup.
2. Go unit tests.
3. gosec static analysis.
4. Docker image build.
5. Trivy container image scan.
6. Trivy repository filesystem scan.
7. Trivy configuration scan.
8. CycloneDX SBOM generation.
9. Upload of security reports and SBOM artifacts.

The Trivy scans are configured to fail when HIGH or CRITICAL findings are detected.

Trivy version `v0.75.0` is explicitly configured for the scan steps.

The final Security Checks workflow completed successfully on GitHub Actions.

## 6. Security Findings and Triage

| Finding | Severity | Resolution |
| --- | --- | --- |
| Vulnerabilities in previous Go builder | HIGH | Updated Go builder and rebuilt the image |
| Local private-key findings | HIGH | Kept untracked private keys out of the repository |
| Missing Cross-Origin-Resource-Policy | ZAP warning | Added `same-origin` header |
| Cacheable health endpoint response | ZAP warning | Added `Cache-Control: no-store` |
| Dockerfile HEALTHCHECK missing | LOW | Compose healthcheck exists; finding documented |
| ZAP Non-Storable Content | ZAP warning | Accepted as an intentional consequence of `no-store` |

## 7. Conclusion

Lab 9 successfully integrated automated security testing into the QuickNotes development workflow.

The final container image had no detected HIGH or CRITICAL vulnerabilities, the final local gosec scan reported zero issues, and the GitHub Actions security pipeline passed.

The ZAP before-and-after comparison demonstrated improvements in HTTP response security, while the remaining warnings and scanning limitations were documented.

Security reports and the CycloneDX SBOM are preserved as evidence for review.
