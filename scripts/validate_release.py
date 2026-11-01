"""Validate release inputs against a checkout containing all remote refs and tags."""

import argparse
import re
import subprocess
import sys


VERSION = re.compile(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")


def git(*args):
    return subprocess.check_output(["git", *args], text=True).strip()


def validate(version, commit, retry):
    match = VERSION.fullmatch(version)
    if not match or match[1] != "0":
        raise ValueError("Use a stable v0.MINOR.PATCH version; v1 needs a new policy.")
    if not re.fullmatch(r"[0-9a-f]{40}", commit):
        raise ValueError("Select an exact, full 40-character commit SHA.")
    commit = git("rev-parse", "--verify", f"{commit}^{{commit}}")
    branch = f"release/v0.{match[2]}"
    git("rev-parse", "--verify", f"refs/remotes/origin/{branch}")
    if subprocess.run([
        "git", "merge-base", "--is-ancestor", commit,
        f"refs/remotes/origin/{branch}",
    ]).returncode:
        raise ValueError(f"Selected commit does not belong to {branch}.")
    tags = git("tag", "--list").splitlines()
    exists = version in tags
    if exists:
        if not retry:
            raise ValueError("Version already exists; select retry to resume publication.")
        if git("rev-parse", f"refs/tags/{version}^{{commit}}") != commit:
            raise ValueError("Existing tag identifies a different commit.")
    elif retry:
        raise ValueError("Retry requires an existing version tag.")
    versions = sorted(
        (tuple(map(int, parsed.groups())), tag)
        for tag in tags if (parsed := VERSION.fullmatch(tag))
    )
    selected = tuple(map(int, match.groups()))
    latest = versions[-1][0] if versions else None
    if latest and (latest[0] != 0 or selected[:2] < latest[:2]):
        raise ValueError("Only the latest stable v0 minor line receives fixes.")
    previous = ""
    if not exists:
        if not latest:
            if version != "v0.1.0":
                raise ValueError("The first release must be v0.1.0.")
        elif selected <= latest:
            raise ValueError("A new version must be greater than the latest release.")
        elif selected[:2] != latest[:2] and selected[2] != 0:
            raise ValueError("A new minor line must start with patch zero.")
    earlier = [(number, tag) for number, tag in versions if number < selected]
    if earlier:
        previous = earlier[-1][1]
        if subprocess.run([
            "git", "merge-base", "--is-ancestor",
            f"refs/tags/{previous}^{{commit}}", commit,
        ]).returncode:
            raise ValueError("Selected commit must include the preceding release.")
    return {
        "commit": commit, "branch": branch, "version": version,
        "exists": str(exists).lower(), "previous": previous,
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("version")
    parser.add_argument("commit")
    parser.add_argument("--retry", action="store_true")
    args = parser.parse_args()
    try:
        for key, value in validate(args.version, args.commit, args.retry).items():
            print(f"{key}={value}")
    except (ValueError, subprocess.CalledProcessError) as error:
        print(f"Release rejected: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
