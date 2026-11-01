# 02: Define and version trading setups

**What to build:** Create named setups, revise their rules into new versions, and retrieve earlier versions through the library on either backend.

**Blocked by:** None (can start immediately).

**Status:** resolved

**Spec coverage:** User stories 15, 17, 18, 60, 61.

- [x] Public operations create named setups with identifiable rule versions.
- [x] Changing rules preserves earlier versions and makes the new version distinguishable.
- [x] Both storage backends preserve setup versions across reopening.
- [x] Tests verify caller-visible version behavior without introducing a separate strategy hierarchy.


## Answer

Added public `Service.CreateSetup`, `ReviseSetup`, `GetSetup`,
`GetSetupVersion`, and `ListSetups` operations. Each named setup starts at version
1; revisions append preserved rules with a new UUID and increasing version
number. Trades can retain the specific setup-version UUID without a separate
strategy hierarchy.

Setup documents use the shared atomic journal persistence introduced by ticket
01. Public behavior tests verify historical and current rules after reopening
CSV and PostgreSQL, independent named setups, and meaningful errors for invalid
or missing records.

Validation: initial missing-API test failed before implementation; the listing
slice failed before `ListSetups` was added. Final `go test ./...` passed with
`STOPLORE_TEST_POSTGRES_DSN` configured, covering both backends.
