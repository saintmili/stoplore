# 01: Configure exchanges, accounts, and fee defaults

**What to build:** Create and retrieve exchange accounts and separate entry/exit percentage fee settings, using decimal values that survive reopening either backend.

**Blocked by:** None (can start immediately).

**Status:** claimed

**Spec coverage:** User stories 4, 27, 59, 60, 61, 68.

- [ ] Public operations create and retrieve exchanges and their accounts, with meaningful validation and missing-record errors.
- [ ] Separate entry and exit percentage rates preserve exact decimal values; an explicit zero remains distinguishable from missing configuration.
- [ ] Both CSV and PostgreSQL preserve settings and identities across reopening.
- [ ] Document the public decimal representation contract for these settings and preserve existing journal source records.
- [ ] Establish the shared public API behavior suite while verifying this configuration workflow.

