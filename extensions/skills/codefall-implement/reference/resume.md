# Picking up an interrupted run

Read for the Resume mode, and at step 2 whenever the epic in scope already has closed children.

State lives in three places — beads, git, GitHub — and a resumed session reconciles them rather
than re-running anything:

| Found | Meaning | Do |
| --- | --- | --- |
| Bead closed, PR merged | Finished and landed | `bd gate check` records it; nothing else |
| Bead closed, PR open | Done, awaiting the human | Leave it; it is in the merge order |
| Bead claimed, branch pushed, no PR | Worker stopped before `gh pr create` | Verify the branch, open the PR from the root — do not re-run the work |
| Bead claimed, no branch | Work never started or never landed anywhere | Relaunch the worker with the same rendered prompt |
| Bead unclaimed but `bd ready` says ready | Never started | Normal flow |
| Bead `deferred` with a `discovered-from` edge | A later round's child, filed by implement, review, or test | Not resumed; the next round's go reopens and claims it |

A stacked chain resumes from its highest link with an open PR; everything below is merged or
awaiting merge, and everything above follows the normal sequence.

**A crashed round and a new one look alike until the children are read.** Both show closed children
beside open ones. A child that is `open` or claimed belongs to the round that crashed, and resumes
as the table says; a child that is `deferred` was filed for the next round and waits for its go. A
run that finds both reconciles the first kind, then asks whether to start the next round in the same
run.
