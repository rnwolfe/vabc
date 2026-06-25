# Security Policy

## Supported versions

`vabc` is pre-1.0; only the latest release is supported. Please upgrade before reporting.

| Version | Supported |
| ------- | --------- |
| latest  | ✅        |
| older   | ❌        |

## Reporting a vulnerability

Please report security issues **privately** via GitHub's
[Private Vulnerability Reporting](https://github.com/rnwolfe/vabc/security/advisories/new)
(Security → Report a vulnerability). Do not open a public issue for a vulnerability.

- **Response SLA:** acknowledgement within ~48 hours.
- **Disclosure:** coordinated. We'll agree on a disclosure timeline and credit you unless you
  prefer otherwise. Good-faith research is welcome (safe harbor); don't run high-volume or
  disruptive tests against Virginia ABC's servers.

When reporting, include the `vabc --version`, your OS, exact commands, and redacted output.

## Threat model & handling notes

`vabc` is deliberately small in attack surface, but a few things are worth stating explicitly:

- **No credentials, ever.** Virginia ABC's endpoints are public and unauthenticated. `vabc`
  stores no tokens, reads no secret files, and sends nothing in `Authorization` headers. There is
  no keyring usage and no secret material to leak via logs, `argv`, env, or temp files.
- **Read-only.** Every command is a read. There are no state-changing operations against Virginia
  ABC. The mutation gate (`--allow-mutations`) is present for contract uniformity but gates
  nothing.
- **Untrusted target text is fenced.** Free text returned from the target (limited-availability
  event titles) is wrapped as untrusted (`⟦UNTRUSTED⟧ … ⟦/UNTRUSTED⟧`) by default so a downstream
  agent does not execute instructions embedded in fetched content. Disable with
  `--no-wrap-untrusted`.
- **Local state.** The only file `vabc` writes is a small throttle/circuit-breaker state file
  under `os.UserCacheDir()/vabc` (or `$VABC_STATE_DIR`). It contains timestamps only — no secrets.
- **Backend etiquette.** `vabc` self-throttles (persistent, cross-process) and contains **no
  evasion** — no User-Agent spoofing, proxy rotation, or challenge solving. It identifies itself
  honestly and treats a persistent block as a stop signal.
