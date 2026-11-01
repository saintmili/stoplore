# 02: Define and version trading setups

**What to build:** Create named setups, revise their rules into new versions, and retrieve earlier versions through the library on either backend.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

**Spec coverage:** User stories 15, 17, 18, 60, 61.

- [ ] Public operations create named setups with identifiable rule versions.
- [ ] Changing rules preserves earlier versions and makes the new version distinguishable.
- [ ] Both storage backends preserve setup versions across reopening.
- [ ] Tests verify caller-visible version behavior without introducing a separate strategy hierarchy.

