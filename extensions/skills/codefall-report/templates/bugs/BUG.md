# BUG-NNN: {Title}

**Status:** Ready — {date}
**Severity:** major
**Reproduced:** Yes — {date}, at a1b2c3d, the browser tools in this session
**Spec:** SPEC-003-REQ-01
**Issue:** #151

[The title says what goes wrong, not where the fault is: "Declined card empties the cart", not
"Cart store reset on 402". The Spec row names the requirement the expected behaviour belongs to, and
is omitted when there is none. The Issue row is filled in once the tracker mirror exists.]

## Summary

[One or two sentences in the reporter's terms: what goes wrong, and for whom.]

## Environment

- **Version:** <version, commit, or build>
- **Where:** <local, staging, production>
- **Platform:** <OS, browser or device, as it applies>
- **Account and data:** <role, plan, or the state the steps start from>

## Steps to reproduce

1. <from a starting point someone else can reach>
2. <each action, in order>

## Expected result

<what should have happened, and what says so: a spec criterion, the docs, earlier behaviour>

## Actual result

<what happened, exactly: message text word for word, the wrong value, what was missing>

## Evidence

[Each attachment linked by its path in `BUG-NNN-slug/`, with a line saying what it shows and where it
came from. An image the reporter could not supply as a file is described in words here.]

- ![Cart empty after the decline message](BUG-NNN-slug/reporter-01-empty-cart.png) — from the
  reporter, production, 2026-09-27
- ![Same state reproduced locally](BUG-NNN-slug/repro-01-empty-cart.png) — from the reproduction

## Reproduction

[What the attempt ran and where it went wrong, or where it went differently from the reporter's
account. Omitted when the Reproduced row is `Not attempted` and its reason says it all.]

## Frequency and impact

<every time, N of M tries, or once>. <who it affects, what it stops them doing>. <the workaround, or
that there is none>.

## Regression

[Only when the reporter knows the last version or date it worked. Otherwise omit.]

Last worked in <version or date>; first seen in <version or date>.

## Acceptance criteria

- **BUG-NNN-AC-01** — WHEN <the reporter's trigger>, the system SHALL <the expected result>
- **BUG-NNN-AC-02** — IF <the failing condition>, THEN the system SHALL <what it does instead>
  (SPEC-003-REQ-01-AC-02)

## Open questions

[Only when something stayed unresolved. A cause the reporter suspects goes here, marked as theirs.]

- <unresolved question>
