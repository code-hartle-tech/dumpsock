# Runbook — Cut a release

Use this when shipping a new `vX.Y.Z` tag.

## Pre-flight

```bash
# 1. on a clean working tree
git status                            # clean
git checkout develop && git pull
go test ./...                         # green
gofmt -l ./... | (! grep .)           # clean
go vet ./...                          # quiet
golangci-lint run ./... 2>/dev/null   # quiet (optional but preferred)
```

If any of the above fails, fix before continuing.

## 1. PR develop → main

```bash
gh pr create \
  --base main --head develop \
  --title "release: vX.Y.Z" \
  --body "$(cat CHANGELOG-vX.Y.Z.md)"
```

Wait for CI green, then merge.

## 2. Tag the merge commit on main

```bash
git checkout main && git pull
git tag -a vX.Y.Z -m "vX.Y.Z"
git push origin vX.Y.Z
```

## 3. Build all artifacts locally

```bash
export VERSION=vX.Y.Z

# CLI matrix
scripts/build.sh

# GUI (macOS only on this machine; Windows GUI happens on a separate runner)
scripts/build-gui.sh

# Compress the .app for GitHub
cd dist && zip -r DumpSock.app.zip DumpSock.app && cd -
```

Verify each binary:
```bash
./dist/dumpsock-darwin-arm64 version
./dist/dumpsock-darwin-arm64 devices
```

## 4. Write the changelog

`CHANGELOG-vX.Y.Z.md` at repo root. Format:

```markdown
# v0.X.Y — YYYY-MM-DD

## Highlights
- One-line user-facing summary of the headliner.

## New
- feat(gui): real-time device disconnect polling (#7)
- feat(backup): --hash sha256 for paranoid dedup (#2)

## Fixed
- fix(backup): delete-after correctly skips pre-skipped files (#3)

## Internal
- docs: full VitePress wiki (internal + external) (#10)

## Known issues
- The Compare & Merge tab still uses placeholder counts. Real engine lands in v0.3.

Full commit log: [vX.Y-1.Y...vX.Y.Z](https://github.com/code-hartle-tech/dumpsock/compare/v0.X.Y-1...v0.X.Y)
```

## 5. Create the GitHub release

```bash
gh release create vX.Y.Z \
  --title "DumpSock vX.Y.Z" \
  --notes-file CHANGELOG-vX.Y.Z.md \
  dist/dumpsock-darwin-universal \
  dist/dumpsock-darwin-amd64 \
  dist/dumpsock-darwin-arm64 \
  dist/dumpsock-linux-amd64 \
  dist/dumpsock-linux-arm64 \
  dist/dumpsock-windows-amd64.exe \
  dist/DumpSock.app.zip
```

## 6. Compute & post SHA-256s

```bash
shasum -a 256 dist/* > SHA256SUMS-vX.Y.Z.txt
gh release upload vX.Y.Z SHA256SUMS-vX.Y.Z.txt
```

## 7. Bump dev branch

```bash
git checkout develop
# bump VERSION constant or git tag-based version pin to next dev marker
git commit -am "chore(release): bump to vX.Y.Z+dev"
git push
```

## 8. Update docs

- External: bump version in `docs/external/guide/install.md` Homebrew tap formula reference.
- Internal: append a row to `docs/internal/sessions/` index.
- Wiki: open this runbook page, append "Last shipped: vX.Y.Z, YYYY-MM-DD" to the top.

## 9. Announce

- HARTLE.TECH internal Slack (or whichever channel is current).
- If user-facing changes are notable, a short tweet from the `@hartle_tech` handle. Mention only product changes, never operator handles.

## Rollback

If shipped binary is broken:
1. **Don't delete the tag.** Tag = historical fact.
2. Create `vX.Y.Z+1` with the fix. Note in changelog: "Replaces broken vX.Y.Z release".
3. Edit the broken release on GitHub: prepend `**SUPERSEDED — use vX.Y.Z+1**` to the title and notes.
4. Delete the old release's binaries (NOT the tag) so accidental downloads stop.
5. Post-mortem in `docs/internal/discoveries/` if it was a non-obvious failure.

## Code-signing addendum (when we get there)

Insert between steps 3 and 4:

```bash
# macOS — sign + notarize
codesign --options=runtime --timestamp --sign "Developer ID Application: HARTLE TECH ..." \
    dist/DumpSock.app
ditto -c -k --keepParent dist/DumpSock.app dist/DumpSock.app.zip
xcrun notarytool submit dist/DumpSock.app.zip \
    --keychain-profile "hartle-tech-notary" --wait
xcrun stapler staple dist/DumpSock.app
```

Tracked in issue #5.
