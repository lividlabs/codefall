// The page shell every screen renders inside, and the app's whole visual language: a handful of
// CSS variables, one type scale, and three building blocks (card, button, empty state). A new
// screen uses these and adds nothing global.

export const styles = `
  :root {
    --bg: #f6f4ef;
    --paper: #ffffff;
    --ink: #1f1d1a;
    --muted: #6b665e;
    --line: #e2ddd3;
    --accent: #b5471f;
    --accent-ink: #ffffff;
    --radius: 8px;
    --space: 16px;
    --font: ui-sans-serif, system-ui, -apple-system, "Segoe UI", Helvetica, Arial, sans-serif;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0;
    background: var(--bg);
    color: var(--ink);
    font: 16px/1.5 var(--font);
  }
  header.site {
    display: flex;
    align-items: baseline;
    gap: var(--space);
    padding: var(--space) calc(var(--space) * 1.5);
    border-bottom: 1px solid var(--line);
    background: var(--paper);
  }
  header.site h1 { margin: 0; font-size: 20px; font-weight: 600; }
  header.site nav a { color: var(--muted); text-decoration: none; margin-right: var(--space); }
  header.site nav a[aria-current="page"] { color: var(--ink); font-weight: 600; }
  main { max-width: 720px; margin: 0 auto; padding: calc(var(--space) * 1.5); }
  h2 { font-size: 18px; margin: 0 0 var(--space); }
  .card {
    background: var(--paper);
    border: 1px solid var(--line);
    border-radius: var(--radius);
    padding: var(--space);
    margin-bottom: var(--space);
  }
  .button {
    display: inline-block;
    border: 0;
    border-radius: var(--radius);
    padding: 8px 14px;
    background: var(--accent);
    color: var(--accent-ink);
    font: inherit;
    cursor: pointer;
  }
  .button.quiet { background: transparent; color: var(--accent); border: 1px solid var(--line); }
  .empty { color: var(--muted); }
  .error { color: var(--accent); }
  input[type="text"] {
    width: 100%;
    font: inherit;
    padding: 8px 10px;
    border: 1px solid var(--line);
    border-radius: var(--radius);
    background: var(--paper);
  }
  label { display: block; font-size: 14px; color: var(--muted); margin-bottom: 4px; }
  .muted { color: var(--muted); font-size: 14px; }
`;

export function escapeHtml(text) {
  return String(text)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

// `nav` is a list of { href, label }; `current` is the href of the page being rendered.
export function layout({ title, body, nav = [{ href: "/", label: "Home" }], current = "/" }) {
  const links = nav
    .map(
      (item) =>
        `<a href="${item.href}"${item.href === current ? ' aria-current="page"' : ""}>${escapeHtml(item.label)}</a>`,
    )
    .join("");
  return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${escapeHtml(title)}</title>
<style>${styles}</style>
</head>
<body>
<header class="site"><h1>Forfeit</h1><nav>${links}</nav></header>
<main>${body}</main>
</body>
</html>`;
}
