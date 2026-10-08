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

## 8. Detailed OWASP ZAP Findings and Triage

| ID | Finding | Risk | Affected URL | Disposition | Reason |
| --- | --- | --- | --- | --- | --- |
| 90004 | Cross-Origin-Resource-Policy Header Missing or Invalid | Low | `/health` | FIX | Added `Cross-Origin-Resource-Policy: same-origin` through HTTP middleware. The header is present in the current HTTP response, and the finding is absent from `zap-after.json`. |
| 10049 | Storable and Cacheable Content | Informational | `/`, `/robots.txt`, root URL | ACCEPT | No sensitive content was identified in the affected responses. Re-evaluate by 2027-01-08. |
| 10049 | Non-Storable Content | Informational | `/health`, `/notes` | ACCEPT | The `Cache-Control: no-store` policy is intentional. Re-evaluate by 2027-01-08. |

### Verification

The running application returned HTTP 200 for `/health` with the following security headers:

- `Cache-Control: no-store`
- `Cross-Origin-Resource-Policy: same-origin`
- `X-Content-Type-Options: nosniff`

The ZAP reports are stored in `submissions/lab9/`.

## 9. Design Questions

### Task 1: Trivy

**a) What factors matter beyond CVE severity when triaging vulnerabilities?**

Severity alone does not determine the actual risk. We must consider whether the vulnerable component is reachable, whether a working exploit exists, whether the application is exposed to the internet, and what privileges an attacker could gain. A HIGH vulnerability in an unused component may be less urgent than a MEDIUM vulnerability in a publicly accessible endpoint.

**b) Why are minimal container images an effective security control?**

Minimal images contain fewer packages, libraries, and utilities, reducing the attack surface and the number of potential vulnerabilities. They also reduce the amount of software that needs to be monitored and patched. However, minimal images still require regular updates and vulnerability scanning.

**c) When is using `.trivyignore` appropriate, and when is it security theater?**

Ignoring a finding is appropriate when it has been investigated and confirmed as a false positive or when the risk has been formally accepted with a documented justification and review date. Using `.trivyignore` simply to make security checks pass without addressing the underlying risk is security theater.

**d) What future problem does an SBOM solve?**

An SBOM provides an inventory of software components and their versions. When a new vulnerability such as Log4Shell is disclosed, the team can quickly determine whether the affected component exists in the application instead of manually inspecting every dependency.

### Task 2: OWASP ZAP

**e) Why use middleware instead of setting headers in individual handlers?**

Middleware applies security headers consistently across all routes. This avoids duplicated code and prevents developers from accidentally forgetting security headers when adding new endpoints. It also makes the security policy easier to maintain and test.

**f) What does `Content-Security-Policy: default-src 'none'` break, and why is it suitable for an API?**

This policy blocks loading scripts, stylesheets, images, fonts, and other resources by default. It can break a traditional website that depends on these resources. QuickNotes is primarily a JSON API, so it does not need browser-rendered scripts or stylesheets for its API responses. However, the policy would need adjustment if a web interface such as Swagger UI were added.

**g) What is the danger of accepting all informational ZAP findings without reviewing them?**

Informational findings can reveal insecure configurations or weaknesses that become exploitable when combined with other issues. Automatically accepting every finding may hide real risks and create a false sense of security. Each finding should be evaluated individually, with a documented reason for accepting, fixing, or suppressing it.


## 10. CycloneDX SBOM Evidence

The following excerpt contains the first 30 lines of the generated CycloneDX SBOM:

```json
{
  "$schema": "http://cyclonedx.org/schema/bom-1.7.schema.json",
  "bomFormat": "CycloneDX",
  "specVersion": "1.7",
  "serialNumber": "urn:uuid:e824bc6b-6fbb-4013-aa04-6f8242da3489",
  "version": 1,
  "metadata": {
    "timestamp": "2026-10-08T18:50:03+00:00",
    "tools": {
      "components": [
        {
          "type": "application",
          "manufacturer": {
            "name": "Aqua Security Software Ltd."
          },
          "group": "aquasecurity",
          "name": "trivy",
          "version": "0.75.0"
        }
      ]
    },
    "component": {
      "bom-ref": "pkg:oci/quicknotes@sha256:4af8c461660bf9c447e6a1777e6ea20f39853198ee909cd7ad6ba55ce46172bc?arch=arm64&repository_url=index.docker.io%2Flibrary%2Fquicknotes",
      "type": "container",
      "name": "quicknotes:lab6",
      "purl": "pkg:oci/quicknotes@sha256:4af8c461660bf9c447e6a1777e6ea20f39853198ee909cd7ad6ba55ce46172bc?arch=arm64&repository_url=index.docker.io%2Flibrary%2Fquicknotes",
      "properties": [
        {
          "name": "aquasecurity:trivy:DiffID",
          "value": "sha256:114dde0fefebbca13165d0da9c500a66190e497a82a53dcaabc3172d630be1e9"
```
