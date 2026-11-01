# 12: Migrate and confirm existing CSV records

**What to build:** Import current and legacy MTL CSV records into either backend, preserve their source and original information, and review and confirm missing or inferred essentials before reporting.

**Blocked by:** 07: Preserve original plans and dated revisions; 10: Report completed-trade performance by setup.

**Status:** ready-for-agent

**Spec coverage:** User stories 53, 60, 61, 63, 64, 65, 66, 67.

- [ ] A public migration operation accepts supported current and legacy MTL CSV records and can target either configured backend.
- [ ] Preserve source records and all available original information; show imported records in the journal immediately.
- [ ] Identify missing or inferred essentials such as account, currency, quantity, and fees and keep affected records out of performance statistics pending confirmation.
- [ ] A public review-and-confirmation workflow makes completed records eligible when their essential information is confirmed and fees are complete.
- [ ] Retain unknown historical setup rules and plans as unknown and keep missing funding optional, with no zero-funding confirmation required.
- [ ] Expose exclusion counts and demonstrate that confirmation updates reports without modifying the source.
- [ ] Use representative synthetic fixtures and verify reopened data and reports through the public API, establishing the shared migration workflow for additional sources.

