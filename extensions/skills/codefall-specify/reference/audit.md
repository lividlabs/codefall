# Reading the code while specifying

Read at step 6, when auditing what already exists.

The code is a source of questions, never of answers. It tells you what exists today, what reaches
the same outcome, and what depends on what. It never tells you what should happen; that is the
user's, and a case the code raises that the user cannot settle goes under **Open questions**.
`codefall-plan` holds the same line from the other side: the implementation is never a source for
what a criterion says.

The audit is the first of two reads. The second, at step 10, is the engineering review in
`engineering-review.md`: a subagent reads the drafted criteria against the code and the project's
stance and comes back with an engineer's questions. The audit finds what the feature touches; the
review finds what the criteria get wrong about it.

## Which surfaces

The ones the user named, and the ones the step 3 search found. The second set matters most: the
user names the surface they are thinking about, and the search finds the ones they are not. A feature
that controls who can reach some outcome — create an account, see a list, place an order — is
audited by outcome: every path in the code that reaches it is a surface, whether or not the user
mentioned it.

## What to enumerate

Tabs, fields, states, columns, branches, and every other path that reaches the outcome the feature
controls. Read the artifact's contents, not its name.

## The question

Intersect what is there with what the user described, and ask about the difference:

> "The settings screen has three tabs today: Account, Notifications, and Shared Lists. You
> mentioned the first two. Is Shared Lists preserved as-is, folded into something new, or
> intentionally going away?"

> "There are two ways to create an account today: the sign-up page, and the landing page a shared
> list opens for someone without an account. You described the gate on the first. Does it apply to
> the second?"

| The user says | Record it as |
| --- | --- |
| Preserved | a line under **Existing behavior preserved** |
| Merged | how it composes, covered by a criterion |
| Removed | its own acceptance criterion |
| The gate applies there too | a criterion naming that path |
| They do not know | an open question |

Audit the surfaces the feature touches, not the whole application. Keep the list of artifacts you
read, with the surface each one is: the engineering review at step 10 is handed that list.
