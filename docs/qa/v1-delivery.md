# v1.0.0 release and installed-delivery evidence

**Published, Homebrew-installed, consumer-qualified and independently reviewed.**
This report records actual release, installation, final fresh-context consumers
and bounded automated review. [PR25](https://github.com/matheus3301/kb/pull/25)
records the required exact successor-head and merged-main integration checks. The [whitelisted evidence](../evidence/v1-delivery.json) retains
identities, checks, timestamps and all failed consumer-attempt hashes without
private paths, session identifiers, prompts, answers or raw tool arguments.

## Separate source and executable identities

[v1.0.0](https://github.com/matheus3301/kb/releases/tag/v1.0.0) was published at **2026-10-26T07:56:16Z** from
`0a6d04d3c25840b2c71c0b98d50478dc6749879a`, the protected PR24 merge. The actual published source archive
contains **72 Go/module files byte-identical** to qualified source
`cb66fd77a120d8c8531f9077a32d2bf51ef7921c`; the editor independently compared every content SHA against the
existing [qualification manifest](../evidence/v1-qualification.json).

| Executable | Observed SHA-256 |
| --- | --- |
| Qualified local candidate.5 | `59d1b477031792cdf539aa3777fafc3942403b80aa8691a3403a8732bf928d46` |
| Published macOS arm64 native binary | `745dfa43a3f33bdbb8b468c7afd59cdd16dab0c05304c378717f1717a245a5a3` |
| Actual Homebrew source-built macOS arm64 installation | `cb3abc103c5e8e1dbc0af6d8e2812695b30e2f10354bef90e5850425aad13878` |

The released and installed binaries identify version **v1.0.0** and full source
commit **0a6d04d3c25840b2c71c0b98d50478dc6749879a**. Different linker identity/build contexts produce
separately retained artifacts; source equality is not binary equality. The
[candidate.5 qualification](v1-qualification.md) keeps its actual long workload,
resource samples, regressions, judgments and failures. None is relabeled as a
new long workload or quiet benchmark of the installed v1 executable.

## Protected integration and actual publication

[PR24](https://github.com/matheus3301/kb/pull/24) final head `dcc7ad90665209886009c8674ad2d10e8c0de4de` passed
all **11** jobs in [CI40287161946](https://github.com/matheus3301/kb/actions/runs/40287161946).
[Exact merged-main CI](https://github.com/matheus3301/kb/actions/runs/40287653868)
passed application jobs and the gate; Dependency Review's intentional push-event
skip is not counted as application validation.

[Release40288191833](https://github.com/matheus3301/kb/actions/runs/40288191833) completed **all five jobs successfully** through
the normal guarded flow. Both native verification arms completed before the tap
job started; the editor independently checked their recorded UTC ordering.

| Actual job | Started UTC | Completed UTC | Result |
| --- | --- | --- | --- |
| [Verify release source](https://github.com/matheus3301/kb/actions/runs/40288191833/job/121456762886) | 2026-10-26T07:49:49Z | 2026-10-26T07:55:07Z | PASS |
| [Publish GitHub release](https://github.com/matheus3301/kb/actions/runs/40288191833/job/121458573866) | 2026-10-26T07:55:09Z | 2026-10-26T07:56:20Z | PASS |
| [Verify published native and source archives (macos-15)](https://github.com/matheus3301/kb/actions/runs/40288191833/job/121458990802) | 2026-10-26T07:56:24Z | 2026-10-26T07:57:15Z | PASS |
| [Verify published native and source archives (ubuntu-24.04)](https://github.com/matheus3301/kb/actions/runs/40288191833/job/121458990816) | 2026-10-26T07:56:23Z | 2026-10-26T07:56:58Z | PASS |
| [Update Homebrew tap](https://github.com/matheus3301/kb/actions/runs/40288191833/job/121459303050) | 2026-10-26T07:57:18Z | 2026-10-26T07:57:25Z | PASS |

Actual anonymous downloads verified **four native-distribution archives plus the
source archive**, checksums, normalized ownership/path privacy and the source
formula identity. Their five archive hashes/member counts and checksum-file SHA
are recorded in the evidence; the editor independently rehashed the downloaded
files. Both **Linux amd64 / ubuntu-24.04** and **macOS arm64 / macos-15** executed
published native binaries and built/executed archived source, including **43 real
documentation requests per artifact** and CLI/MCP backup/restore workflows.
The macOS amd64 and Linux arm64 archives were inspected, not natively executed.

The successful tap update is commit
[`766bb23eae2624c8ed6b7599525c93447a7a85c5`](https://github.com/matheus3301/homebrew-tap/commit/766bb23eae2624c8ed6b7599525c93447a7a85c5).
A local snapshot, cross-build or configured matrix was not substituted for these
actual published native/source job outcomes.

## Actual installed functional checks

An actual **Homebrew v0.4.0 → v1.0.0 upgrade and formula test passed on macOS
arm64**. This is a source build, not a bottle: Go **go1.27.3**, CGO=0,
trimpath, with the observed installed hash above. It is not evidence of a Linux
Homebrew installation.

On disposable synthetic roots, the installed executable passed **14 acceptance
groups**, **43 real documentation requests**, and actual preserved **v0.3.1
eight-check / v0.4.0 nine-check** upgrade runs. The latter consumed an archive
actually produced by the old v0.4.0 binary, checking exact authority/receipts,
replay, continued writes and source immutability; old pending state was recovered
by its prior binary before upgrade. These are separate installed-v1 reports,
not replacements for the historical candidate.5 compatibility receipts.

The six shared installed CLI-skill files are byte-identical to public and
published-source skill files; the editor independently checked all six hashes.
Native workflow/installed scripted CLI and MCP checks passed. Those checks are
**not** completion of the two independent OMP consumer sessions below.

## Final fresh-context installed consumers

Both final runs used **openai-codex/gpt-6.1-sol, high effort**, with separate
native learning and query contexts and no prior-chat carryover. Learning teardown
preceded the fresh query; final teardown and full-root byte purity passed.

| Actual final run | Passed checks | Immutable report SHA-256 |
| --- | ---: | --- |
| Installed CLI + public skill, attempt 3 fresh | 17/17 | `4683e9525d15c7859317bce6309c157b6498259b90763be2e1d9e49b20997fbf` |
| Installed MCP-only, attempt 5 fresh | 20/20 | `15b64628646cb74b054b472e1ce8e7d2cafa9e1f47cde3489862cfa36cc09a8e` |

Both used the same frozen **fresh-v4 harness / structured-v3 oracle**, hashes
`dfeac1c6525354365b8cc62dd97464783c56ac62410fe6c464b153fb481362d7` /
`c3ddc974c8ec69f175d324c6bdaaf57ba3d69082e142cf6588d1999d926259e2`.
This is a finite structured question, not unrestricted prose/NLI evaluation.
Each typed current fact and each citation was checked independently. Both
recorded **30 policy days** and **runtime enforcement not verified**, citing
exact current `source-new.md`, physical **6:6**, revision
`sha256:abcf7bb1f6f8bbb44de5cd014bf5352f2a049ddcdfe6883e41caad84c2f009f3`.
The editor independently hashed captured source bytes and inspected that
positive current-policy/replaces-old-policy line; a regex candidate alone was
not treated as semantic proof. No raw answer is published.

An independent automated focused OMP reviewer accepted both actual results:
fresh contexts/model/tools, positive current citation, byte purity and teardowns.
The CLI read/bash allowed-scope condition is closed **for this run**; the editor
also classified eight skill reads, 30 installed-CLI calls and two temporary
request creation/cleanup calls. MCP-only and retrieval-only follow-up checks
passed. Model identity/task obedience is metadata, not correctness proof.
The structured oracle's 18 meaningful pure cases passed **owner-reported**;
neither editor nor method reviewer reran them. Fresh conversation does not imply
MCP-server cold restart, cache eviction or erasure of all external workdir files.

## Failed attempts remain failed

| Earlier actual attempt | Bounded disposition; never retroactively PASS |
| --- | --- |
| CLI 1 | Follow-up turn failed; native result/answer unavailable for attribution. |
| MCP 1 | Follow-up-answer assertion failed before answer retention; cause unknown. |
| MCP 2 | RPC produced no text result; missing native evidence prevents product/provider attribution. |
| MCP 3 | Independently valid direct-source 30-day answer rejected by a literal synthesis-filename observer requirement; later purity checks did not execute. |
| CLI 2 | Valid 30-day hyphenated answer rejected by the unaccepted prose-v2 pattern; later purity checks did not execute. |
| MCP 4 | Recorded retryable provider capacity turn rejection; no second answer or completed follow-up purity. HTTP status was absent. |

Prose-v2 also had independently identified current/historical-claim, path/revision
substring and cross-citation line-borrowing weaknesses and was **not accepted**.
Structured-v3 removes prose ambiguity with explicit types and exact per-citation
binding; it does not change product source, numerical qualification thresholds
or expected policy. Final fresh passes are distinct observations under the
justified observer, not relabeled old reports. All six failed hashes/check sets
are retained in the evidence.

A separate reviewer HTTP 429 had a 362-second retry instruction that was
respected, coordinator-confirmed. It is not the MCP4 HTTP status or evidence that
all earlier unknown failures were provider failures. No confirmed-state loss or
product defect is inferred from these observer/provider/unattributed outcomes.

## Review and dated documentation boundary

Independent **automated OMP**, not human, prepublication review accepted bounded
candidate.5 source/evidence and closed F1. Actual exact PR/main CI and the guarded
release satisfied its publication conditions. Independent bounded review has
now accepted both final installed-consumer results. Final postpublication V08
approved exact documentation head `b33739b` and this bounded receipt/status
projection. Its only receipt finding P01 is corrected: consumer-review SHA-256
`1a5301b060cb1d9ef7b4232e7e8260ff34ffbd29de51389f2c5b800e34e80f32`.
The final review receipt SHA-256 is
`45a18f0aca18d0e104aa1d22f3858351265a4bec4394f6514d7c1074955f68f3`.
The [structured record](../evidence/v1-delivery.json) identifies automated roles,
unchanged scope and the required protected-integration conditions. No human
approval or future CI result is invented.

The immutable v1.0.0 tag/source archive, published **2026-10-26 07:56:16 UTC**,
contains the dated prepublication documentation snapshot. This **2026-10-26
postpublication record** adds subsequently observed delivery/consumer facts;
it does not move the tag, replace release assets or rewrite old reports.
V07 outcomes and the bounded automated V08 review are complete. Actual final
Git integration and checks are recorded in [PR25](https://github.com/matheus3301/kb/pull/25);
the [stability checklist](../stability.md) centralizes qualification and scope.
Finite synthetic local-process/filesystem evidence does not establish arbitrary
history size, portable latency, hostile-editor isolation or hardware power-loss
safety. All earlier qualification failures and nonimprovements remain intact.
