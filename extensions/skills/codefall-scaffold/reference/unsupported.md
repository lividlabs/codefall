# A surface with no supported profile

What scaffold does when a surface's stack has no supported profile: it asks codefall for the
profile, then offers to scaffold the surface on the project's own stack. Read at step 1, when a
surface does not match. ADR-017 in codefall's repository records the decision.

## Contents

- [Say so](#say-so)
- [The request](#the-request)
- [The project's own stack](#the-projects-own-stack)
- [What gets written](#what-gets-written)

## Say so

Say plainly:

> Sorry — we don't support your stack yet.

Name the supported profiles, then take the request and offer the own stack, in that order. With
several unsupported surfaces, batch each step across all of them rather than repeating it per
surface. Do not steer toward a supported profile the person did not pick.

## The request

Codefall's tracker is GitHub Issues on `lividlabs/codefall`, a public repository. Its maintainers
put the `profile request` label on every request for a profile. Search it once per stack before
offering anything, with both queries, because a request filed recently may not carry the label yet:

```
gh issue list -R lividlabs/codefall --state all --label "profile request" --search "<stack>" \
  --json number,title,state,stateReason,url
gh issue list -R lividlabs/codefall --state all --search "<stack> profile in:title" \
  --json number,title,state,stateReason,url
```

Search for the stack as the person named it, and by its language when the framework finds nothing:
`FastAPI`, then `python`. Read the titles and bodies of what comes back: an issue counts when it
asks for a profile for this stack. The result decides what is offered:

- **An open issue asks for this stack.** Name it with its link, and offer to add the person's voice
  to it: a 👍 reaction and a comment. Show the comment before posting it. On yes, post what the
  person took; they may take the reaction and decline the comment.

  ```
  gh api -X POST repos/lividlabs/codefall/issues/<n>/reactions -f content=+1
  gh issue comment <n> -R lividlabs/codefall --body-file <file>
  ```

- **A closed issue, completed.** A later codefall ships the profile. Say so, name the issue, and
  stop: the next step is `codefall upgrade`, then this verb again.
- **A closed issue, not planned.** Name it and its closing comment, and go on to the own stack.
- **No issue.** Hand the person a finished issue to paste at
  `https://github.com/lividlabs/codefall/issues/new`: the title `scaffold: a <stack> surface profile`
  and the body below. Do not file it and do not offer to. The tracker rule in
  `../../../../.codefall/shared/workflow.md`, under *Who is authoritative for what*, allows a
  reaction and a comment on an existing request and nothing more.

When `gh` is missing, is not signed in, or cannot reach GitHub, hand over the issue text and the
search link, `https://github.com/lividlabs/codefall/issues?q=label%3A%22profile+request%22+<stack>`.

**A comment or an issue holds the stack and nothing about the product.** The tracker is public, and
the product is the person's. Name the stack and its version, the kind of surface, such as an HTTP
API, a mobile app, or a CLI, and the codefall version from `.codefall/manifest.json`. Nothing from the
vision, the decision log, or the description goes in.

```
Requesting a `<stack>` surface profile.

- Surface: <kind>, holding domain logic
- Stack: <language, framework, and versions>
- codefall: <version>
```

## The project's own stack

Offer it once the request is settled:

> We can still scaffold this surface on the stance. Your stack's architecture is recorded as this
> project's own ADRs instead of a codefall profile, and codefall checks none of it.

On no, scaffold follows its step 1: it stops, or emits the stack-agnostic core when the person asks
for it. Upgrade keeps the surface at the stack-agnostic core and names it as undecided.
On yes, ask the person to describe the architecture and best practices for their stack. Ask once,
covering what every profile settles, and take an answer in prose:

| Concern | What the answer says |
| --- | --- |
| Layout | Where each component's facade, the four rings, the shared modules, and the composition root live |
| Dependency injection | A container or hand wiring, and how the composition root binds implementations |
| Boundary enforcement | Which tool or language feature enforces each of the five boundary rules: facade-only imports, inward-only layers, cross-component through facades only, shared modules import no component, and inner rings touch neither the UI framework nor gateways |
| Errors and absence | How an error travels, and how a missing value is written |
| Tests | The runner, and how a test replaces a gateway |
| Client state | Server state, shared client state, and local state, for a surface with a UI |
| Practices | The idioms and conventions every agent working on this surface should follow |

"Not decided" is an answer. It goes to the decision log's `Open`.

**Offer a search in the same question:** *I can look up how `<stack>` projects apply Clean
Architecture and what they use for each of these.* On yes, search with WebSearch and read with
WebFetch. Read the language's and the framework's own documentation and the package pages first,
then write-ups from people who maintain the tools. Report per concern what the sources use, how it
maps onto ADR-BASE-01's rings and ADR-BASE-02's facade, and where the stack works against the
stance, with each source linked. A framework that wants to own the layering, or a language with no
module visibility, are common cases.

Present the options flat. Codefall has no stance on a stack it has no profile for, so no option is
labelled recommended. The person decides each concern. An answer they gave before the search stands
unless they change it.

**The stance does not bend for the stack.** Sources that organize code by layer, or skip boundary
enforcement, still produce package-by-component with enforced boundaries here. Say so where it
comes up.

## What gets written

- **One project ADR per concern decided**, from `../templates/adrs/TEMPLATE.md`, numbered after the
  seam ADR when there is one. Its Context opens by saying codefall has no profile for the stack and
  this decision is the project's own. It names where the decision came from: the person's
  description, or the sources, which are linked under Related. Each ADR is Accepted and dated, and
  drafted and shown before it is written, as the seam ADR is.
- **A scoped `AGENTS.md`** for the surface. No skeleton exists for the stack, so read a supported
  profile's `AGENTS.md.skeleton` for its sections and nothing else. Fill each section from the ADRs
  just written, link them, and leave out a section they do not cover.
- **Provenance** in `.codefall/scaffold.json`: the surface under `ownStack`, keyed by its name, with
  `stack` as the person named it and `adrs` listing its ADR identifiers. Its ADRs also go under
  `adrs`, as every emitted ADR does.
- **Docs only for this surface**, whatever depth the other surfaces take. The code tiers follow a
  profile's depth notes, and this surface has none. The report names boundary enforcement as the
  first task the person owes the project, with the tool the ADR chose.

The report names each surface on its own stack, its ADRs, and the request: the issue it reacted to
or commented on, or the issue text it handed over.
