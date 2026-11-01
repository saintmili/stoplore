# Release policy

This release policy was approved on 2026-11-02 and is ready for pipeline
implementation.

## Agreed decisions

- Start with version `v0.1.0`.
- Maintain one branch per minor release line, named `release/vMAJOR.MINOR`.
  For example, `release/v0.1` contains releases tagged `v0.1.0`, `v0.1.1`,
  and subsequent patches in that line.
- Identify each exact release with a version tag.
- Publish the Go library with GitHub release notes. Downloadable programs and
  container images are out of scope.
- During v0, only the latest stable minor release line receives fixes. Older
  branches and published versions remain available after support ends.
- Patch releases preserve API and storage compatibility. Breaking API or storage
  changes require a new minor release and migration instructions.
- A maintainer manually starts a release workflow with a version and commit.
  The workflow validates and tests the selected commit, including PostgreSQL
  tests, before creating a version tag and publishing.
- Keep `master` for ongoing development. Create each release branch from
  development when preparing its first release. Apply maintenance fixes to both
  the supported release branch and `master`.
- New features, including compatible additions, require a minor release.
  Patch releases contain fixes only.
- Publish from the public repository `github.com/saintmili/mtl`, matching the
  module path, using GitHub Actions. The manually triggered workflow must exist
  on the repository's default branch.
- The initial pipeline supports stable versions only. Prereleases are deferred.
- Before tagging, require an unused version, a selected commit belonging to the
  matching release branch, and passing tests, including PostgreSQL tests.
- Publish the version tag, verify that Go can resolve the module at that version,
  and create the GitHub Release with release notes.

## Recovery

- Published version tags and their source commits remain unchanged.
- If a step fails after tagging, retry publication against the same tag and
  commit. Such a retry is the exception to the unused-version check: it must
  verify that the existing tag identifies the selected commit.
- Code corrections after publication require a new patch version.

## Prepare a release

The public GitHub repository must use `master` as its default branch and contain
these workflows on that branch. This checkout has no configured remote; configure
`origin` for `github.com/saintmili/mtl` before pushing. Repository rules must allow
the release workflow's `GITHUB_TOKEN` to create version tags and GitHub Releases.
The workflow grants write permission only to its publishing job.

After the completed work and pipeline have been merged to `master`, create the
first release branch from its reviewed commit:

```sh
git fetch origin
git switch -c release/v0.1 origin/master
git push -u origin release/v0.1
git rev-parse HEAD
```

Use the same procedure with a new minor branch when preparing its first release.
Merge fixes into the supported release branch and carry them to `master` before
preparing the next minor. New release commits must include the preceding stable
release in their history. Review compatibility and storage migration requirements
in the pull request; the workflow cannot infer whether a change is a feature or
a breaking change.

CI runs on pull requests and pushes to `master` and `release/**`. It checks module
identity, tidy dependencies, and module checksums, then runs `go test -count=1 ./...`
with a PostgreSQL 17 service. Go is selected from the tested commit's `go.mod`.
PostgreSQL tests must run rather than silently skip for lack of a connection.

## Run the publishing pipeline

In GitHub Actions, choose **Release**, then **Run workflow** on `master`. Supply:

- `version`: `v0.1.0` for the first release.
- `commit`: the full 40-character SHA from the matching remote release branch.
- `retry`: leave false for a new version.

The equivalent GitHub CLI command is:

```sh
gh workflow run release.yml --repo saintmili/mtl --ref master \
  -f version=v0.1.0 -f commit=FULL_COMMIT_SHA -f retry=false
```

The workflow validates inputs, tests the exact commit with read-only permissions,
then revalidates branch and version state before publishing. Only one release
workflow runs at a time. It does not execute the selected release's code in the
job holding publishing credentials. The pipeline rejects v1 and later versions
until a stable-version policy is agreed, and rejects patches for superseded minor
lines. An older patch in the current minor line can still resume publication.

If publication fails after the tag was pushed, run the workflow again with the
same version and SHA and set `retry` to true. Do not delete or move the tag.
Go proxy availability is retried for several minutes; a delayed proxy can require
a later workflow retry. An existing published GitHub Release is preserved.
If a draft or prerelease already exists for that tag, resolve it manually before
retrying; the stable publishing workflow fails instead of silently accepting it.
Rerunning the original failed run with `retry=false` will reject the existing tag;
start a new manual run with `retry=true` instead.

After publishing, consumers can select the exact module version:

```sh
go get github.com/saintmili/mtl@v0.1.0
```

Local checks for the pipeline's Git validation rules are:

```sh
python3 -B -m unittest discover -s scripts -p 'test_*.py'
```

These tests use temporary Git repositories and never publish tags or releases.

## References

- [Go module publishing](https://go.dev/doc/modules/publishing)
- [Go release workflow](https://go.dev/doc/modules/release-workflow)
- [Manually running GitHub workflows](https://docs.github.com/en/actions/how-tos/manage-workflow-runs/manually-run-a-workflow)
