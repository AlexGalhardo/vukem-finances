# CI

1. A Windows machine does not show what Linux runners show: file-system case, build tags, path separators, Bun versus Node behaviour. A change that touches paths, processes or build tags is also checked in the Linux Docker build before hand-off.
2. CI is a gate from epic 01 on. Do not delete or weaken a workflow to make a run green.
3. A flaky test is a defect. Find the cause. Never skip it, retry it or add a sleep to it; wait on a condition, never on a duration. With no cause found, delete it in its own commit that says why.
4. A failing check whose cause is a real bug of the product is fixed in the product, not in the test.
5. In a matrix, an `include` entry whose keys are not dimensions is added to every combination and overwrites the entry before it. Make the varying key a dimension and let `include` add only what depends on it.
6. Every downloaded build input and third-party action is pinned: images by tag and digest, actions by commit, files by URL and SHA-256 measured from a real download.
7. A workflow on `main` can be dispatched on another ref to test a fix before the release merges. Dispatch with publishing off first; publish only after a green run.
8. Build scripts run on macOS bash 3.2: with `set -u`, an empty array is unbound. Expand possibly-empty arrays as `${arr[@]+"${arr[@]}"}`.
