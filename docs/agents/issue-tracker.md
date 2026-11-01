# Issue tracker: Local Markdown

Issues and specs live in `.scratch/`.

## Conventions

- One feature per directory: `.scratch/<feature-slug>/`.
- The specification is `<feature-directory>/spec.md`.
- Implementation tickets are separate files under
  `<feature-directory>/issues/<NN>-<slug>.md`, numbered from `01`.
- Record triage state with a `Status:` line near the top of each spec or ticket.
  Use the vocabulary in `triage-labels.md`.
- Append comments and conversation history under `## Comments`.

## Publishing and fetching

Publishing means creating or updating the appropriate local Markdown file.
Fetch a spec or ticket by reading its referenced path. Resolve a bare ticket
number within its feature directory; ask for the feature if ambiguous.

## Wayfinding

- Store maps at `.scratch/<effort>/map.md`.
- Store child tickets at `.scratch/<effort>/issues/NN-<slug>.md`.
- Record ticket type with `Type:` and dependencies with `Blocked by: NN, NN`.
- A ticket is unblocked when all its blockers are resolved.
- Claim a ticket with `Status: claimed` before working on it.
- Resolve it by adding an `## Answer`, setting `Status: resolved`, and updating
  the map's decisions with a summary and link.
- Choose the lowest-numbered unblocked, unclaimed ticket first.
