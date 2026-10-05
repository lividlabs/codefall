// What the driver does on GitHub and in the project in the person's place: merges the equip pull
// requests, waits for the Action to merge each document pull request, merges the code stack at the
// end, returns the checkout to main after each landing, and gathers evidence for the judge.
// Every gh command runs with the run directory as its working directory and GH_REPO naming the
// repository, so it is a session outside the project directory, where codefall's guard hook does
// not apply. The guard is a Claude Code hook; it never saw these commands anyway.

import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync, cpSync, existsSync } from "node:fs";
import { join } from "node:path";

export type Gh = ReturnType<typeof makeGh>;

export type Pr = { number: number; title: string; headRefName: string; baseRefName: string; state: string; isDraft: boolean; mergedAt: string | null; url: string };

export function makeGh(repo: string, cwd: string, log: (line: string) => void) {
  const env = { ...process.env, GH_REPO: repo, GH_PROMPT_DISABLED: "1" };
  const gh = (...args: string[]): string => {
    try {
      return execFileSync("gh", args, { cwd, env, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] }).trim();
    } catch (error: any) {
      throw new Error(`gh ${args.join(" ")} failed: ${error.stderr ?? error.message}`);
    }
  };
  const ghJson = <T = any>(...args: string[]): T => JSON.parse(gh(...args) || "null");

  const listPrs = (state = "all"): Pr[] =>
    ghJson("pr", "list", "--state", state, "--limit", "200", "--json", "number,title,headRefName,baseRefName,state,isDraft,mergedAt,url");

  return {
    gh,
    ghJson,
    listPrs,

    /** The newest pull request, open or merged, whose branch starts with `prefix`. */
    latestPr(prefix: string): Pr | undefined {
      return listPrs("all")
        .filter((p) => p.headRefName.startsWith(prefix))
        .sort((a, b) => b.number - a.number)[0];
    },

    /** Waits for a pull request to be merged. Returns true when merged, false on timeout. */
    async waitForMerge(pr: number, minutes: number): Promise<boolean> {
      const until = Date.now() + minutes * 60_000;
      let saidDraft = false;
      while (Date.now() < until) {
        const view = ghJson("pr", "view", String(pr), "--json", "state,mergedAt,isDraft");
        if (view.state === "MERGED") return true;
        if (view.state === "CLOSED") return false;
        if (view.isDraft && !saidDraft) {
          log(`PR #${pr} is still a draft; the Action fires only when a verb marks it ready`);
          saidDraft = true;
        }
        await new Promise((r) => setTimeout(r, 30_000));
      }
      return false;
    },

    /** Merges one pull request with a squash, as the person would. */
    mergePr(pr: number): string {
      return gh("pr", "merge", String(pr), "--squash");
    },

    /** Merges the code stack up to `pr`, as the person would. */
    mergeStack(pr: number): string {
      return gh("stack", "merge", String(pr), "--squash", "--yes");
    },

    /** The top of the open code stack: the feat/ PR no other open PR bases on. */
    codeStackTop(): Pr | undefined {
      const open = listPrs("open").filter((p) => p.headRefName.startsWith("feat/"));
      const bases = new Set(open.map((p) => p.baseRefName));
      return open.find((p) => !bases.has(p.headRefName));
    },

    /**
     * Puts the project checkout back on main, current, the way a person would after a merge. The
     * verbs leave the checkout on the branch they wrote; the next verb's landing procedure branches
     * from wherever it stands and would ask about another branch. Plain git, not /codefall-refresh,
     * so no session is spent on it; it is recorded in deviations.txt as not a deviation.
     */
    returnToMain(projectDir: string): void {
      const git = (...args: string[]) => execFileSync("git", args, { cwd: projectDir, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
      git("fetch", "origin", "--prune", "--quiet");
      git("checkout", "--quiet", "main");
      git("pull", "--ff-only", "--quiet", "origin", "main");
    },

    /** Gathers what the judge reads into <runDir>/evidence/. */
    gatherEvidence(projectDir: string, runDir: string, epicGuess: string): void {
      const dir = join(runDir, "evidence");
      mkdirSync(dir, { recursive: true });
      const save = (name: string, text: string) => writeFileSync(join(dir, name), text + "\n");
      const sh = (cmd: string, args: string[], cwdOverride = projectDir): string => {
        try {
          return execFileSync(cmd, args, { cwd: cwdOverride, env, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
        } catch (error: any) {
          return `(${cmd} ${args.join(" ")} failed: ${(error.stderr ?? error.message).toString().trim()})`;
        }
      };

      save("prs.json", JSON.stringify(listPrs("all"), null, 2));
      save("issues.json", gh("issue", "list", "--state", "all", "--limit", "200", "--json", "number,title,state,labels,url"));
      save("action-runs.json", gh("run", "list", "--workflow", "codefall-land-documents", "--limit", "30", "--json", "status,conclusion,event,headBranch,url,createdAt"));
      save("stack.txt", sh("gh", ["stack", "list"]));
      sh("git", ["fetch", "origin", "--quiet"]);
      save("git-log.txt", sh("git", ["log", "--oneline", "--decorate", "-n", "80", "origin/main"]));
      save("docs-tree.txt", sh("git", ["ls-tree", "-r", "--name-only", "origin/main", "docs"]));
      save("src-tree.txt", sh("git", ["ls-tree", "-r", "--name-only", "origin/main", "src", "testing", "test"]));
      save("settings.json", sh("git", ["show", "origin/main:.codefall/settings.json"]));
      save("epic.txt", sh("bd", ["epic", "status", "--json"]));
      save("epic-show.txt", sh("bd", ["show", epicGuess]));
      save("children.txt", sh("bd", ["list", "--parent", epicGuess, "--all", "--limit", "0"]));
      save("ready.txt", sh("bd", ["ready"]));
      for (const sub of [".codefall/reviews", ".codefall/tests", "testing/test-cases"]) {
        const from = join(projectDir, sub);
        if (existsSync(from)) cpSync(from, join(dir, sub.split("/").at(-1)!), { recursive: true });
      }
    },
  };
}
