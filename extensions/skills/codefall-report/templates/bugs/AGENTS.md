# docs/bugs — operative rules

- `archive/` is history. Do not read it unless the user asks about an archived report by name.
- Report and criterion identifiers are permanent and append-only. A retired number is never reused,
  and `BUG-012` means the same document after it is archived.
- The report is canonical. Its tracker issue is generated from it and is regenerated, not
  hand-edited.
- A report's attachments live in `BUG-NNN-slug/` beside it and move with it when it is archived.
- A report says what is wrong. The cause and the fix belong to the design that cites it.
- Status transitions are `/codefall-report`'s to make, never a hand edit.
