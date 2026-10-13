# Reading the code while specifying

Read at step 6, when auditing what already exists, and at step 10, when reading the draft criteria
against the code before the document is composed.

The code is a source of questions, never of answers. It tells you what exists today, what reaches
the same outcome, and what depends on what. It never tells you what should happen; that is the
user's, and a case the code raises that the user cannot settle goes under **Open questions**.
`codefall-plan` holds the same line from the other side: the implementation is never a source for
what a criterion says.

## Step 6: audit what already exists

A specification describes what the user wants **added**, so anything the feature touches and the
spec does not mention is at risk of being removed by accident, or left in place by accident. The
audit finds those.

**Which surfaces.** The ones the user named, and the ones the step 3 search found. The second set
matters most: the user names the surface they are thinking about, and the search finds the ones they
are not. A feature that controls who can reach some outcome — create an account, see a trip, place
an order — is audited by outcome: every path in the code that reaches it is a surface, whether or
not the user mentioned it.

**What to enumerate.** Tabs, fields, states, columns, branches, and every other path that reaches
the outcome the feature controls. Read the artifact's contents, not its name.

**The question.** Intersect what is there with what the user described, and ask about the
difference:

> "The profile screen has three tabs today: Account Info, Travel Preferences, and Saved Passengers.
> You mentioned the first two. Is Saved Passengers preserved as-is, folded into something new, or
> intentionally going away?"

> "There are two ways to create an account today: the sign-up page, and the landing page a shared
> trip opens for someone without an account. You described the gate on the first. Does it apply to
> the second?"

| The user says | Record it as |
| --- | --- |
| Preserved | a line under **Existing behavior preserved** |
| Merged | how it composes, covered by a criterion |
| Removed | its own acceptance criterion |
| The gate applies there too | a criterion naming that path |
| They do not know | an open question |

Audit the surfaces the feature touches, not the whole application.

## Step 10: read the criteria against the code

The recap at step 9 is confirmed and the criteria are drafted. Before composing the document, read
each criterion with the code open and ask the question its shape calls for:

| A criterion that | Ask the code |
| --- | --- |
| removes or changes something | what depends on it: records that reference it, flows that assume it, states that hold it |
| gates an outcome | what other paths reach the same outcome |
| touches something another criterion also touches | what happens when both hold at once |

The third row is a read of the criteria against each other, with the code there to make the
consequence concrete rather than hypothetical.

Each hit is one question to the user, put the way step 5 puts them, and its answer is a criterion
or an open question. Nothing is patched into the spec silently. Say what the read found before
moving on; "the read found nothing" is a result, and it is said out loud.

### Two worked examples

**A gate with a second door.** The feature makes account creation invite-only. The criteria cover
the sign-up page. The search at step 3 finds a second place accounts are created: the landing page
a shared link opens, which lets a newcomer create an account and signs them in. Left alone, that
path bypasses the gate. The audit at step 6 surfaces it as a path to the gated outcome, and the read
at step 10 catches it again under the second row. The question to the user is whether the gate
applies to shared-link sign-ups, and the answer is a criterion either way.

**A removal that orphans something.** One criterion lets a pending account sign in while the gate is
off. Another says that declining a pending account removes it. Read as a pair, they mean a pending
account can own trips and bookings and then be removed. The code shows that trips and bookings
reference their owner, so the consequence is a real table and not a worry. The question to the
user is what happens to that activity, and the answer is a criterion under the decline requirement.
