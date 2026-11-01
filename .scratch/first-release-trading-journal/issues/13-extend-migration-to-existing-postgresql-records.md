# 13: Extend migration to existing PostgreSQL records

**What to build:** Use the established migration review and confirmation workflow for existing MTL PostgreSQL records, targeting either backend while preserving the source database records.

**Blocked by:** 12: Migrate and confirm existing CSV records.

**Status:** ready-for-agent

**Spec coverage:** User stories 53, 60, 61, 63, 64, 65, 66, 67.

- [ ] Extend the public migration workflow with a source reader for the existing PostgreSQL record model.
- [ ] Support importing into either CSV or PostgreSQL without rewriting source records or dropping available original information.
- [ ] Imported records use the same immediate visibility, missing/inferred-information review, and confirmation-driven report eligibility as CSV imports.
- [ ] Preserve available notes and historical information without inventing unavailable setup rules, plans, or costs.
- [ ] Funding remains optional and absent funding requires no zero-funding confirmation.
- [ ] Verify source preservation, public review operations, reopened records, and report results with isolated PostgreSQL fixtures and both target backends.

