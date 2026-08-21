# Verify — prove the binary against a confirmed block

`deshred verify` measures the decoder's output against ground truth derived
from a confirmed block and reports the result as separate accuracy matrices.
Every released artefact must pass the bundled fixture before it ships; the same
command lets you re-prove it on your own machine, offline.

## Run the shipped fixture

Download `verify-kit-<version>.tar.gz` from the release you installed:

```sh
tar xzf verify-kit-<version>.tar.gz
./deshred verify \
  --capture slice.dzcap --truth truth.json --alt-journal alt-journal.jsonl \
  --pools pools.json --expect expected.json
```

Expected output:

```text
VERIFY GREEN — matrices match expected.json
```

`manifest.json` inside the kit lists the SHA-256 of every file and the exact
command line. The capture is a real mainnet shred slice (slot 440061516); the
truth file is what that slot's confirmed block actually contained. Nothing in
the kit requires network access.

On a mismatch there is no green line: the process exits non-zero with
`Error: verify: measured matrices differ from the pinned expectation in
expected.json` after printing the measured matrices. That means your download
is corrupt or mixed from different releases — re-check `checksums.txt` and the
manifest hashes, re-download, and make sure binary and kit come from the same
tag. A genuine mismatch on an intact, same-release pair is a bug — please
report it.

## What the matrices mean

The matrices are deliberately **never merged into one score** — each answers a
different question, and a single blended number would hide exactly the failures
you care about:

- **A — fidelity.** Of the swap intents the decoder claimed, how many are
  confirmed by the block (true positives), and how many are wrong (false
  positives)? This is the "when it speaks, is it right?" axis.
- **B — coverage.** Of the venue swaps provably in the block, how many did the
  decoder see? Misses are split by cause — the dominant, structural cause is
  CPI blindness (see [limits.md](limits.md)).
- **C — reality.** Venue instructions on transactions that **reverted** on
  chain. A pre-execution decoder cannot know a transaction will fail; C makes
  that population explicit instead of letting it pollute A or B.
- **D — everything else** that fits none of the above (structurally empty on
  healthy runs).

## Verifying against your own archive

Instead of `--truth`, you can point `verify` at an archive RPC endpoint with
`--archive-url` and let it derive ground truth from `getBlock` for the slots in
your own capture:

```sh
./deshred verify --capture your-capture.dzcap --archive-url https://your-archive-endpoint
```

Two rules are enforced, not just recommended:

- **Exactly one archive endpoint.** Different archives can return materially
  different transaction sets for the same slot; mixing providers corrupts every
  ratio, so `verify` refuses more than one `--archive-url`.
- **Votes are excluded from the denominators.** Archives return no vote
  transactions, while the shred pipeline reconstructs them; `verify` subtracts
  votes before taking any ratio so coverage can never read above 100%.

`--emit-truth` saves the derived truth to a file so later runs can repeat the
measurement offline with `--truth`.
