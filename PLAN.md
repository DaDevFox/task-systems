Implementaiton plan for:

* **Primary trigger:** the **milestone closed** webhook (bumps `MAJOR` per your rule).
* **Optional trigger:** **manual dispatch** (also bumps `MAJOR` or can switch to commit-driven SemVer if you want).
* **No version file** — tags are the source of truth.
* Supports your commit grammar and the `!` breaking modifier.

---

# 0) Rules we’ll encode

* **Tag format:** `vSUPERMAJOR.MAJOR.MINOR.PATCH`
* **Default policy (recommended):**

  * **Milestone closed** → **always** bump `MAJOR` (reset `MINOR`, `PATCH` to `0`).
  * **Manual dispatch** → by default also bump `MAJOR`, but may be switched to “SemVer from commits” for one-off cuts.
* **SemVer from commits (optional mode):**

  * If any commit since the previous tag is **breaking** (`[!]` anywhere in the subject/body *or* `BREAKING CHANGE:` in body *or* `type!:` style), bump **MAJOR**.
  * Else if any commit starts with `FEAT.` or `FEAT(`, bump **MINOR** (and reset `PATCH`).
  * Else if any commit starts with `FIX_` or `ENH_`, bump **PATCH**.
  * Else **no bump**.
* **SUPERMAJOR**: only **manual** — via workflow inputs either **bump** or **set**.

Your recognized ACTION prefixes:

* **non-versioned:** `ICM_ FMT_ CFG_ IMP_ REF_ DEL_ MERG DOC_`
* **PATCH:** `FIX_ ENH_`
* **MINOR:** `FEAT.` (and we also accept `FEAT(scope):`)

---

# 1) Create a dedicated versioning workflow

> **File:** `.github/workflows/release-versioning.yml`

This workflow:

* Fires when a **milestone is closed** or via **workflow\_dispatch**.
* Finds the latest `v*.*.*.*` tag (or seeds a baseline).
* Parses commit history since that tag for your ACTIONs and `!`.
* Computes the next version.
* Creates an **annotated tag** and a **GitHub Release** with grouped notes.

```yaml
name: Release Versioning (Tags)

on:
  milestone:
    types: [closed]
  workflow_dispatch:
    inputs:
      mode:
        description: "Bump policy"
        type: choice
        options:
          - major-per-release
          - semver-from-commits
        default: major-per-release
      force_major:
        description: "Force MAJOR bump"
        type: boolean
        default: false
      supermajor_action:
        description: "SUPERMAJOR control"
        type: choice
        options: [none, bump, set]
        default: none
      new_supermajor:
        description: "If set, SUPERMAJOR becomes this value"
        type: number
        default: 1
      dry_run:
        description: "Do not tag/release, just print"
        type: boolean
        default: false

permissions:
  contents: write        # needed to push tags & create releases
  issues: read           # we read milestone info
  pull-requests: read

concurrency:
  group: release-versioning
  cancel-in-progress: false

jobs:
  tag-and-release:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout default branch with history
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Determine latest version tag
        id: latest
        run: |
          set -euo pipefail
          # Find the latest tag matching vX.Y.Z.W
          tag=$(git tag -l 'v[0-9]*.[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | head -n1)
          if [[ -z "$tag" ]]; then
            # First-time seed — before you “publish the app”
            # This is the baseline from which you will bump MAJOR on first release.
            tag="v1.0.0.0"
            echo "No existing tags detected. Seeding baseline: $tag"
          fi
          echo "tag=$tag" >> $GITHUB_OUTPUT

      - name: Compute next version
        id: next
        env:
          # Default to major-per-release for milestone events:
          DEFAULT_MODE: major-per-release
          INPUT_MODE: ${{ github.event_name == 'workflow_dispatch' && inputs.mode || '' }}
          FORCE_MAJOR: ${{ github.event_name == 'workflow_dispatch' && inputs.force_major || 'false' }}
          SUPERMAJOR_ACTION: ${{ github.event_name == 'workflow_dispatch' && inputs.supermajor_action || 'none' }}
          NEW_SUPERMAJOR: ${{ github.event_name == 'workflow_dispatch' && inputs.new_supermajor || '1' }}
        run: |
          set -euo pipefail
          prev="${{ steps.latest.outputs.tag }}"
          IFS='.' read -r S M m p <<< "${prev#v}"

          # SUPERMAJOR management (manual only)
          case "$SUPERMAJOR_ACTION" in
            bump) S=$((S+1)); M=0; m=0; p=0 ;;
            set)  S="${NEW_SUPERMAJOR}"; M=0; m=0; p=0 ;;
            none) : ;;
          esac

          # Decide mode
          MODE="$DEFAULT_MODE"
          if [[ -n "$INPUT_MODE" ]]; then MODE="$INPUT_MODE"; fi

          # Determine comparison range
          range="${prev}..HEAD"
          if [[ "$prev" == "v1.0.0.0" && -z "$(git tag -l 'v[0-9]*.[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | sed -n '2p')" ]]; then
            # If this is the seeded baseline and there are no earlier tags,
            # compare from root commit
            range="$(git rev-list --max-parents=0 HEAD | tail -n1)..HEAD"
          fi

          # Gather commit subjects + bodies (exclude merge commits)
          log=$(git log --no-merges --pretty=format:'%s%n%b%n----' $range || true)

          # Flags
          breaking=false
          feat=false
          patch=false

          # Breaking if:
          #  - '[!]' appears in subject/body (your MODIFIER)
          #  - 'type!:' style (compat)
          #  - 'BREAKING CHANGE:' in body (compat)
          if echo "$log" | grep -Eiq '(\[[^]]*!\])|(^[A-Z]+(\([^)]*\))?(!):)|BREAKING CHANGE:'; then
            breaking=true
          fi
          # Minor if FEAT. or FEAT(scope):
          if echo "$log" | grep -Eiq '(^|\n)FEAT(\.|[[:space:]]*\()'; then
            feat=true
          fi
          # Patch if FIX_ or ENH_:
          if echo "$log" | grep -Eiq '(^|\n)(FIX_|ENH_)'; then
            patch=true
          fi

          bump="none"
          if [[ "$MODE" == "major-per-release" ]]; then
            bump="major"                    # your default rule
          else
            if $breaking; then bump="major"
            elif $feat;  then bump="minor"
            elif $patch; then bump="patch"
            else bump="none"
            fi
          fi

          # Manual override
          if [[ "$FORCE_MAJOR" == "true" ]]; then bump="major"; fi

          # Apply bump
          case "$bump" in
            major) M=$((M+1)); m=0; p=0 ;;
            minor) m=$((m+1)); p=0 ;;
            patch) p=$((p+1)) ;;
            none)  : ;;
          esac

          NEW="v${S}.${M}.${m}.${p}"
          echo "new=$NEW"   >> $GITHUB_OUTPUT
          echo "bump=$bump" >> $GITHUB_OUTPUT
          echo "range=$range" >> $GITHUB_OUTPUT

      - name: Generate grouped release notes
        id: notes
        run: |
          set -euo pipefail
          prev="${{ steps.latest.outputs.tag }}"
          new="${{ steps.next.outputs.new }}"
          range="${{ steps.next.outputs.range }}"

          {
            echo "## ${new}"
            if [[ "${{ github.event_name }}" == "milestone" ]]; then
              echo ""
              echo "Milestone closed: **${{ github.event.milestone.title }}** (#${{ github.event.milestone.number }})"
            fi
            echo ""
            echo "Changes since ${prev}:"
            echo ""

            echo "### ✨ Features"
            git log $range --no-merges --pretty=format:'- %s' | grep -E '^- FEAT(\.|[[:space:]]*\()' || echo "- (none)"

            echo ""
            echo "### 🛠️ Enhancements"
            git log $range --no-merges --pretty=format:'- %s' | grep -E '^- ENH_' || echo "- (none)"

            echo ""
            echo "### 🐞 Fixes"
            git log $range --no-merges --pretty=format:'- %s' | grep -E '^- FIX_' || echo "- (none)"

            echo ""
            echo "### 🧹 Docs / Chore"
            git log $range --no-merges --pretty=format:'- %s' | grep -E '^- (ICM_|FMT_|CFG_|IMP_|REF_|DEL_|MERG|DOC_)' || echo "- (none)"

            echo ""
            echo "### ⚠️ Breaking"
            git log $range --no-merges --pretty=format:'- %s%n%b%n' | \
              awk 'BEGIN{RS="----"} /(\[[^]]*!\])|(^[A-Z]+(\([^)]*\))?(!):)|BREAKING CHANGE:/ {gsub(/\n/," "); print "- " $0}' \
              || true
          } > release_notes.md

          echo "notes_path=release_notes.md" >> $GITHUB_OUTPUT

      - name: Create tag and release (or dry run)
        env:
          DRY: ${{ github.event_name == 'workflow_dispatch' && inputs.dry_run || 'false' }}
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          set -euo pipefail
          new="${{ steps.next.outputs.new }}"
          if [[ "$DRY" == "true" ]]; then
            echo "Would create tag $new"
            echo "----- RELEASE NOTES -----"
            cat release_notes.md
            exit 0
          fi

          git config user.name  "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git tag -a "$new" -m "Release $new"
          git push origin "$new"

          title="$new"
          if [[ "${{ github.event_name }}" == "milestone" ]]; then
            title="$new · ${GITHUB_REPOSITORY#*/} – Milestone #${{ github.event.milestone.number }}"
          fi

          # GitHub CLI is preinstalled on ubuntu-latest
          gh release create "$new" -F release_notes.md --verify-tag --title "$title"
```

### Notes

* It **seeds** `v1.0.0.0` if no prior tags exist, so your **first milestone-closed** event bumps to `v1.1.0.0`.
  If you prefer to start at `v0.0.0.0`, change the seed in the “Determine latest version tag” step.
* **SUPERMAJOR**: use **workflow\_dispatch** and set `supermajor_action` to `bump` or `set`. That resets `MAJOR.MINOR.PATCH` to `0.0`.
* **Breaking detection** supports your `[!]` modifier and also conventional `type!:` / `BREAKING CHANGE:`.

---

# 2) Make your existing CI build on tags

Your current CI runs on `push` (branches) and PRs. To build/publish artifacts **when a version tag is created**, add:

```yaml
on:
  push:
    branches: [main, develop]
    tags:
      - "v*.*.*.*"
  pull_request:
    branches: [main]
```

That way the same pipeline runs for releases (the tag push) and can attach build artifacts to the GitHub Release, if you wish.

> Tip: gate publish steps with `if: startsWith(github.ref, 'refs/tags/v')`.

---

# 3) How this maps to your policy

* **Milestone closed** (primary): bumps **MAJOR** once every release, no matter what happened in commits. `MINOR/PATCH` are reset. (Exactly your “MAJOR per release” rule.)
* **Manual dispatch**: defaults to **MAJOR per release**, but you can choose `semver-from-commits` for a one-off tag that reflects `FEAT.`/`FIX_`/`ENH_`/`[!]`.
* **Breaking `!`**: in `semver-from-commits` mode, it forces a **MAJOR**. In the default mode, your **release already bumps MAJOR**, so `!` is effectively an annotation (still listed under “Breaking” in notes).
* **SUPERMAJOR**: only manual, via inputs, with a clean reset.

---

# 4) Commit examples it will recognize

```
FEAT.(scheduler): [ ] Add cron-based trigger
ENH_UI: Improve button focus ring
FIX_core: Handle nil pointer deref in planner
DOC_: Update README with setup notes
REF_utils: Extract date parsing

# breaking via your MODIFIER
FEAT.(protocol): [!] Switch wire format to v2

# breaking via conventional style (also detected)
FEAT!(auth): rotate password hashing scheme
chore: bump deps

# body-based breaking (also detected)
FIX_core: reject invalid SIDs

BREAKING CHANGE: SID format now base32
```

---

# 5) Common tweaks (optional)

* **Always include PR titles instead of commit titles**: replace `git log` with `gh pr list/gh api` calls to fetch merged PRs between tags and use their titles.
* **Only cut releases from `main`**: add a guard step to confirm `git rev-parse --abbrev-ref HEAD` is `main`.
* **Prevent double fires**: the `concurrency` block already serializes runs; you can also limit milestone scope (e.g., only milestones whose title starts with `release/`).
* **Changelog categories**: add more sections or sort within sections; you can also hyperlink PRs/issues.

---

# 6) Security & permissions

* The workflow declares `contents: write` so it can push tags & create releases.
* It uses the built-in `GITHUB_TOKEN` (scoped to the repo).
* `issues: read` is required to display milestone info on milestone runs.

---

If you want, I can adapt your **existing CI file** to:

* Build only on tag pushes for release artifacts.
* Publish those artifacts directly to the **GitHub Release** created by the versioning workflow (e.g., `gh release upload` or `softprops/action-gh-release`).

# GOTCHAs

Milestone Hook Gotchas

Gotcha: pre-receive hook runs before refs are updated, so you can’t inspect the new tag/milestone in the repo.
→ Solution: Parse the incoming refs from stdin (<old> <new> <ref>), detect tags/milestones there, and fetch metadata from the API if needed.

Gotcha: GitHub/GitLab milestone events aren’t part of Git hooks; only refs (branches/tags) trigger them.
→ Solution: Implement milestone trigger in CI (GitHub Actions / GitLab CI) instead of raw Git hooks. Use “milestone closed” webhook event as the real hook.

Gotcha: Closing a milestone might not map cleanly to a tag (off-by-one commits, or multiple tags).
→ Solution: Enforce convention: milestone X.Y corresponds to annotated tag vX.Y at the same commit. Validate in CI.

Gotcha: If the milestone is closed without all issues/PRs resolved, CI will still run.
→ Solution: Add guard: check open issues count via API before running release pipeline. Fail if unresolved.

Manual Trigger Gotchas

Gotcha: Tags can be lightweight (no annotation), which breaks metadata extraction.
→ Solution: Require annotated tags. Validate in CI that tag has message, author, and matches version regex.

Gotcha: A manual push of a tag could bypass code review and CI checks.
→ Solution: Restrict tag creation to protected branches / release managers only.

Gotcha: GitHub/GitLab Actions only trigger on tag creation, not updates (retagging won’t retrigger).
→ Solution: Enforce immutable tags. If a mistake occurs, bump to a new patch/minor release.

Gotcha: Running manual trigger repeatedly can cause duplicate artifacts/releases.
→ Solution: Check existing release by tag name via API before publishing. Skip if exists.

Shared Gotchas (Both Approaches)

Gotcha: CI/CD runners need access to private registries/repos for packaging.
→ Solution: Configure scoped tokens or deploy keys in secrets.

Gotcha: Time skew between milestone closure, tag push, and pipeline execution can desync artifacts.
→ Solution: Decide on a single source of truth (milestone closure must be accompanied by tag push, or tag push auto-closes milestone).

Gotcha: Semantic version drift (milestone 1.2 vs tag v1.2.1).
→ Solution: Enforce regex checks on both tags and milestone names; reject mismatch.

Gotcha: Human error in naming (typo in tag/milestone).
→ Solution: Provide scripts/CLI helpers (make release VERSION=1.2.0) that automate tagging + milestone closure.

Gotcha: CI/CD pipeline might fail mid-release, leaving a half-published version.
→ Solution: Make pipeline idempotent and verify artifact existence before upload.