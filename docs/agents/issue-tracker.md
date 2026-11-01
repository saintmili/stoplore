# Issue tracker: GitHub

Specs, implementation tickets, and wayfinding maps live in GitHub Issues
at https://github.com/saintmili/stoplore. Use the `gh` CLI.

## Target repository

Run `gh repo view --json nameWithOwner` before the first issue creation
in a session. The target must be `saintmili/stoplore`.
Use `--repo saintmili/stoplore` when operating outside this clone.

## Publishing and fetching

- Publish a spec or ticket by creating a GitHub issue.
- Create: `gh issue create --title "..." --body-file <file>`.
- Read: `gh issue view <number> --json number,title,body,labels,comments`.
- List: `gh issue list --state open --json number,title,labels,assignees`.
- Comment: `gh issue comment <number> --body-file <file>`.
- Update a body: `gh issue edit <number> --body-file <file>`.
- Apply or remove labels with `gh issue edit --add-label` or `--remove-label`.
  Read `triage-labels.md` for the label vocabulary.
- Resolve: post the explanation as a comment, then close the issue.
  Post the comment separately even if the issue is already closed.
- Resolve a bare ticket number as a GitHub issue number.

## Relationships and dependencies

Use a parent issue for the spec or map and sub-issues for its tickets.
Use native GitHub issue dependencies for blocking relationships.
Check `gh issue create --help` and `gh issue edit --help` for support
in the installed CLI; use `gh api` when necessary.

If native relationships are unavailable, link children in a task list
in the parent body, put `Part of #<parent>` in each child, and record
`Blocked by: #<number>, #<number>` near the top of its body.

A ticket is unblocked when every blocker is closed.

## Wayfinding

- Label the map `wayfinder:map`.
- Label child tickets `wayfinder:<type>`, where type is
  `research`, `prototype`, `grilling`, or `task`.
- Create missing wayfinding labels when needed.
- Choose the first unblocked, unassigned child in map order.
- Claim it with `gh issue edit <number> --add-assignee @me`.
- Resolve it by posting its answer, closing it, and updating the map's
  decisions with a summary and link.

## Pull requests as a triage surface

PRs as a request surface: no.

## Existing local records

Existing `.scratch/` files are historical records until explicitly migrated.
Their local ticket numbers are scoped to their feature directory;
they are not GitHub issue numbers.
