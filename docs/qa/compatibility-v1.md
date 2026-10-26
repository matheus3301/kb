# Prior-version and package compatibility qualification

This is an executable qualification procedure, not a declaration that v1.0 is
released or universally compatible. The default upgrade origin is **v0.3.1**,
source `2ca104f48cc4a6551f4bbc4fae12d51432e42dff`. D04 qualification covers the
two release edges **v0.3.1 → v0.4.0 → v1.0.0**. The immutable
**[v0.4.0 release](https://github.com/matheus3301/kb/releases/tag/v0.4.0)**
is source `74f0d196408c1354899802e18ce569b0eab33adf`; retain its actual executable
for the second edge. The [compatibility policy](../compatibility.md) and
[exit checklist](../stability.md) track current qualification/publication facts.
The baseline's `kb.v1` wire version is independent of its product version.
Candidate results apply only to the executable hashes, source identities and host
in the report.

## Executable scope

Run `scripts/compatibility.py` with two actual executables. It creates fresh
synthetic roots and an isolated empty home; it does not open a production base,
install skills, register MCP, contact providers, or use model obedience as proof.
It exits zero only when all applicable checks complete:

- Baseline and candidate identify their exact version and 40-hex source commit;
  the default baseline must be v0.3.1 at the pinned source above. An additional
  baseline requires paired `--baseline-version` and `--baseline-commit`; either
  flag alone, an abbreviated commit or an actual identity mismatch fails.
  Hash both executables before and after the run. The report uses logical
  `$BASELINE/kb` and `$CANDIDATE/kb` paths, not physical workstation paths.
- Exercise JSON stdin requests, closed request fields, trailing JSON rejection,
  explicit flag precedence and conflicting positional inputs. Check the closed
  `kb.v1` envelope, required field types, warnings, success/error union and
  machine error codes. Test exits 0–6, including successful nonconformance report
  data at exit 1 and pending recovery at exit 6. Error prose is not a contract.
- Preserve an unknown type, a large exact integer, an exact-decimal YAML span,
  comments, flow style, CRLF bytes and SHA-256 revisions across upgrade. Check
  old successful create/patch/MCP receipts replay before stale preconditions and
  do not duplicate content, indexes, log entries or receipt bytes. Reject reused
  keys with changed requests.
- Independently initialize actual MCP stdio, negotiate `2025-11-25`, discover
  instructions and closed descriptive input/output schemas, and run
  search/read/context parity, preview/create/replay/patch/conflict workflows.
  No CLI skill or base instruction file is supplied to the server. Require the
  original ten tools on every baseline; a declared newer baseline may also
  advertise the known backup/restore pair, never arbitrary tools. Require all
  twelve tools on the write-enabled candidate and exactly six on either
  read-only server. Follow discovery pagination. Direct calls to hidden
  mutations must fail. Stdout remains JSON-RPC with matching response IDs.
- When the prior baseline advertises backup/restore, invoke that **prior
  executable** to produce a settled archive before any candidate invocation
  touches the fixture. Transfer its exact bytes into a separate initialized
  candidate staging root and restore with the candidate. Verify exact notes,
  binary source assets, configuration and durable receipts; replay old create
  and patch requests; continue both patch and create operations. Check the
  original producer root's paths, modes and bytes remain unchanged after backup,
  candidate restore, replay and continued writes.
- Use CLI and MCP backup/restore on the upgraded root. Verify exact content,
  configuration, receipt bytes and revisions in both restored roots; compare
  manifest identities; replay old requests and continue ordinary writes. Restarts
  select restored roots explicitly. Existing destinations are refused and
  restored writes cannot change the source root. CLI/MCP restore must reject a
  0755 source backup directory with `ACCESS_DENIED` (CLI exit 5), without chmod,
  content changes or creation of destination storage. The REVIEW-04 service
  regression additionally covers group/other access, missing source directories,
  missing archives, non-directory storage and symlink confinement.
- Interrupt the **unmodified declared baseline process** with SIGKILL only after
  observing its own pending journal. Stop the process first to inspect a stable
  journal;
  do not fabricate or edit its contents. Check blocked old reads, pure recovery
  preview and candidate backup refusal while pending. Recover with the old
  binary, then upgrade, retry/replay the original plan and make a settled backup.
  The report records the actually observed prepared/committed state, not assumed
  coverage of both states. Three bounded attempts are allowed; failure to catch
  a pending boundary is a failed qualification, not a skipped success.
- Confirm v0.3.1 backup/restore commands remain unsupported (`INVALID_INPUT`,
  exit 2). Newer baseline discovery can advertise those tools; that does not
  authorize running the old executable on newer state or imply a downgrade.

The JSON report is bounded to 32 KiB. Failures identify the stage and invariant
without copying raw process output, temporary roots or physical binary paths.
Reports retain passed stages but set `ok: false` on failure. Keep the local full
command mapping separately when physical-path provenance is needed. A reported
version/commit is not cryptographic source attestation: compare it independently
with the selected immutable source, archive checksums and build provenance.
Optional `--candidate-version` and `--candidate-commit` enforce those selected
expectations; omitting them records the executable's identity rather than proving
that a release was built from the intended commit.

## Independent source/candidate consumer recipe

Requirements: local full Git history containing the released source, the pinned
Go toolchain and available module dependencies, Python 3, and a native Linux or
macOS host. No installed production binary or package-manager symlink is needed.
Use a committed candidate; a label injected by `-ldflags` does not establish that
an arbitrary dirty working tree matches that commit.

From the product checkout:

```sh
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
baseline_commit=2ca104f48cc4a6551f4bbc4fae12d51432e42dff
candidate_commit=$(git rev-parse HEAD)
mkdir -p "$work/baseline-source"
test "$(git rev-parse 'v0.3.1^{}')" = "$baseline_commit"
git archive "$baseline_commit" | tar -x -C "$work/baseline-source"
(
  cd "$work/baseline-source"
  GOMAXPROCS=2 CGO_ENABLED=0 go build -trimpath \
    -ldflags "-X main.version=v0.3.1 -X main.commit=$baseline_commit" \
    -o "$work/baseline-kb" ./cmd/kb
)
GOMAXPROCS=2 CGO_ENABLED=0 go build -trimpath \
  -ldflags "-X main.version=compatibility-candidate -X main.commit=$candidate_commit" \
  -o "$work/candidate-kb" ./cmd/kb
python3 -m unittest discover -s scripts -p test_compatibility.py -v
GOMAXPROCS=2 python3 scripts/compatibility.py \
  --baseline "$work/baseline-kb" --candidate "$work/candidate-kb" \
  --candidate-version compatibility-candidate --candidate-commit "$candidate_commit" \
  --output .build/compatibility-source.json
```

`--binary` is an alias for `--candidate`. The minimal invocation is
`python3 scripts/compatibility.py --baseline BASELINE --candidate CANDIDATE
--output REPORT`. Both binaries still need real non-`dev` identities. Do not
relabel another release as v0.3.1. Preserve an actual installed v0.3.1 executable
before an upgrade if installed-to-candidate evidence is required; validate its
version/source identity and hash, then pass the preserved copy, not a moving
Homebrew symlink.

## Both D04 upgrade origins

Retain each actual installed/checksum-verified executable before changing the
package-manager installation. Independently confirm `V040_COMMIT` against its
published immutable release source and select `V100_COMMIT` from the intended
final candidate's real provenance. The v1 commit is operator-supplied, not guessed:

```sh
V040_COMMIT=74f0d196408c1354899802e18ce569b0eab33adf
python3 scripts/compatibility.py \
  --baseline "$BASELINE_V031" --candidate "$CANDIDATE_V040" \
  --candidate-version v0.4.0 --candidate-commit "$V040_COMMIT" \
  --output .build/compatibility-v031-to-v040.json

python3 scripts/compatibility.py \
  --baseline "$BASELINE_V040" \
  --baseline-version v0.4.0 --baseline-commit "$V040_COMMIT" \
  --candidate "$CANDIDATE_V100" \
  --candidate-version v1.0.0 --candidate-commit "$V100_COMMIT" \
  --output .build/compatibility-v040-to-v100.json
```

Both baseline override flags are required together. They select an exact
additional origin, never disable identity verification or relabel a binary.
Each run creates its own synthetic fixtures and qualifies that release edge;
it does not claim a single persistent production base was upgraded twice.
Do not substitute a development build labeled v0.4.0 for the actual intermediate
release. Qualify the second edge against its real preserved v0.4.0 executable and
the intended consumer's exact identity. A development-candidate run is not final
published-artifact evidence. The pinned v0.3.1 CI origin remains the default when
both override flags are absent; required CI also checks the pinned v0.4.0 origin.

### Prior-produced archive evidence

The second edge must exercise an archive actually produced by its selected
baseline. Candidate-produced round trips alone cannot establish this
cross-version archive contract. The `prior-produced-archive` stage runs between
the baseline's settled writes and the first candidate operation on the original
fixture. Its candidate restore and continued writes use a separate staging root,
not the original root or a candidate-created replacement archive.

The bounded `prior_archive` report records:

- `producer` and `consumer`: actual version, full source commit, executable
  SHA-256 and logical binary path, using the same verified identities as the
  overall qualification.
- `archive_path`, `archive_sha256`, `manifest_revision`, `files` and `bytes`:
  the prior-produced archive's identity and producer result.
- `restored_manifest_revision`, `authoritative_bytes_exact`,
  `replayed_requests`, `continued_writes`, `source_unchanged` and
  `restore_destination`: observations required before `status: "passed"`.

A corrupt archive, changed outer bytes, wrong manifest identity, mismatched
note/asset/config/receipt/file set, lost replay, failed continued write or source
mutation fails qualification. A matching outer checksum alone does not establish
archive validity. Real-process regressions also reject an otherwise byte-exact
archive missing old receipts: successful extraction is insufficient if the old
stale-revision request cannot replay without effects.

For v0.3.1, `prior_archive.status` is explicitly `not_applicable`, with a reason
that the baseline has no operational archive tools. That edge does not add a
fictional passing archive stage; candidate-produced round trips still run under
their separate existing stage. For an archive-capable baseline, a failed
production/restore attempt records `status: "failed"` and cannot be reported as
not applicable.

A same-source development-baseline smoke can exercise the archive-capable
branch using its real development version/commit and executable hash. Label it
development evidence; it does not substitute for the actual preserved
Homebrew/checksum-verified v0.4.0 producer and v1.0.0 consumer required by D04.



## Native archive and Homebrew consumer recipe

For a candidate package, first run the existing `make release-snapshot` in the
release validation environment. `scripts/check_release.py` retains its checks
for exactly four platform archives plus source, checksums, normalized archive
owners, unsafe paths, public documentation/licenses and exact formula identity.
It copies the host-native executable into a temporary `prefix/bin/kb`, verifies
unchanged bytes and exact release identity, and runs actual acceptance,
documentation and independent MCP/CLI backup/restore consumers there. It also
builds the archived source and runs those consumers. Temporary-prefix installation
is **not** a Homebrew formula installation. Foreign archives are inspected, not
executed.

For an independently obtained native release archive:

1. Compare its checksum against the selected release's published checksums and
   independently confirm the release tag's immutable source commit. Verify
   archive members before extraction using `inspect_archive` from
   `scripts/check_release.py` or the full snapshot verifier.
2. Extract into a disposable directory, copy `kb` to a temporary `prefix/bin/kb`,
   and run `prefix/bin/kb version --json`. Check the exact release version/commit,
   not merely a successful exit or the presence of `kb.v1`.
3. Pass that exact temporary-prefix executable to `scripts/compatibility.py`
   with both pinned origins and expected candidate version/commit. For v0.4.0,
   supply the paired baseline version/commit flags shown above. Retain separate
   bounded reports and the artifact checksum. Do not substitute source-build
   reports for this package execution.

For a separately authorized Homebrew release qualification:

1. Retain the preserved v0.3.1 executable and preserve the actual installed
   v0.4.0 executable **before** the final package upgrade. Verify both identities.
2. Perform the explicit Homebrew install/upgrade and formula test in the selected
   release workflow; installation/global changes are separate authority from
   this harness. Confirm the installed executable's `version --json` against the
   intended tag source and formula archive checksum.
3. Run the same harness with that installed executable as `--candidate` and each
   preserved old executable as `--baseline`. Supply expected candidate version
   and commit and the paired v0.4.0 baseline flags where applicable. All mutation
   fixtures still live in temporary synthetic roots.
4. Run the installed acceptance and executable documentation consumers:

   ```sh
   python3 scripts/acceptance.py --binary "$INSTALLED_KB" --output .build/installed-acceptance.json
   python3 scripts/check_docs.py --binary "$INSTALLED_KB"
   ```

5. Retain the installed/package/source reports separately. Repeat exact-identity
   qualification after any intermediate release and again for the final v1.0
   candidate. A passing intermediate build is not final-release evidence.

## Limited primary support matrix

| Target/channel | Evidence boundary | Support interpretation |
| --- | --- | --- |
| macOS arm64, Homebrew | Published v0.4.0 installation/formula, CLI/documentation and independent CLI + skill/MCP consumer verification are recorded in its [release record](https://github.com/matheus3301/kb/releases/tag/v0.4.0). Retain actual prior binaries for this procedure. | Primary installed channel; every new release needs its own installed exact-identity report. |
| Linux amd64, native archive and archived-source execution in CI | Existing [release snapshot validation](../../scripts/check_release.py) executes the host-native artifact/source on Ubuntu; the compatibility job must run the pinned baseline/candidate workflow there. | Primary CI runtime channel; not evidence of a Linux Homebrew production installation or this macOS run. |
| macOS amd64 | Cross-build and archive inspection only until a native run is recorded. | No installed/runtime compatibility claim. |
| Linux arm64 | Cross-build and archive inspection only until a native run is recorded. | No installed/runtime compatibility claim. |

No promise is made for Windows, network filesystems, remote/multi-host writers,
Apple signing/notarization, arbitrary OS versions, or universal disk/power-loss
recovery. This bounded process exercise does not replace transaction failpoint,
concurrency, parser fuzz, security, sustained-load or independent review evidence.

## Upgrade and rollback limits

Finish pending recovery with the old supported executable before upgrading.
Preserve content, configuration and durable receipts together; export alone is
not an operational backup. Keep protected off-disk copies separately: the new
same-root backup storage is staging, not disaster recovery. Never overlay old
notes on newer live journals/receipts or delete state to unblock recovery.

The tested candidate's backup format is explicitly versioned separately from
`kb.v1` and config version 1. v0.3.1 has no parser/commands for that archive.
An old executable rejecting a new command does not prove it can interpret future
operational versions. No automatic downgrade, state conversion or rollback of a
new operational format is implemented or promised. Restore to a fresh destination
with a binary supporting that archive; validate and explicitly select that root.

Final readiness remains conditional on the [stability checklist](../stability.md),
both prior origins, the actual prior-produced archive edge and exact per-platform
candidate evidence. Passing v0.4.0 installation or development-candidate checks
does not establish final v1 package identity, readiness or publication authority.
Release policy and the distinction between source, archive and Homebrew execution
remain unchanged.
