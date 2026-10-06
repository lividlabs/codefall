package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lividlabs/codefall/cli/internal/initcmd/internal/domain"
	"github.com/lividlabs/codefall/cli/internal/shared/settings"
)

// skillPrefix is the prefix a run names the skills with: the answer the request carries, and the
// format's default when it carries none.
//
// Presentation reads it from the settings and asks for it at init, so a request with no answer is a
// run over settings written before the field existed. The fallback is the format's own default rather
// than a second opinion about what it should be, and it is also what every one of those projects has
// installed (ADR-015).
func skillPrefix(request Request) string {
	if request.SkillPrefix == "" {
		return settings.DefaultSkillPrefix
	}

	return request.SkillPrefix
}

// skillRename is the rewrite this run applies to everything it copies or splices out of the embedded
// tree. The skills it renames are the ones the tree ships, read from the tree each time rather than
// kept in a list, so a skill added to the tree is covered without anyone remembering it.
func (i *Initialize) skillRename(request Request) (domain.SkillRename, error) {
	skills, err := i.source.Skills()
	if err != nil {
		return domain.SkillRename{}, fmt.Errorf("read the skills the extension ships: %w", err)
	}

	rename, err := domain.NewSkillRename(skillPrefix(request), skills)
	if err != nil {
		return domain.SkillRename{}, err
	}

	return rename, nil
}

// DeclaredSkillPrefix reports the prefix .codefall/settings.json names the skills with, so a rerun
// installs under the prefix the project chose rather than asking again. A file that is not there, or
// one that names no prefix, declares the default — which is what a file written before the field
// existed looks like, and what every such project has installed. A prefix the format does not know is
// an error, as it is everywhere else init reads the settings: the run would otherwise install under a
// name the file does not say.
//
// Like ChosenHarnesses, it decodes the one field it needs: what the rest of the file may hold is the
// format's business, and this is a question about one answer the project gave.
func (i *Initialize) DeclaredSkillPrefix(dir string) (string, error) {
	data, err := i.files.ReadFile(settingsPath(dir))

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return settings.DefaultSkillPrefix, nil
	case err != nil:
		return "", fmt.Errorf("read %s: %w", settingsName, err)
	}

	var document struct {
		SkillPrefix string `json:"skillPrefix"`
	}

	if err := json.Unmarshal(data, &document); err != nil {
		return "", fmt.Errorf("decode %s: %w", settingsName, err)
	}

	if document.SkillPrefix == "" {
		return settings.DefaultSkillPrefix, nil
	}

	prefix, err := settings.ParseSkillPrefix(document.SkillPrefix)
	if err != nil {
		return "", fmt.Errorf("%s: %w", settingsName, err)
	}

	return prefix, nil
}

// customizationsDir is where a project keeps the per-skill procedure a verb reads before it starts,
// one directory per installed skill name (shared/customizations.md). The directory is named for the
// skill as installed, so it follows the prefix, and a project that changes its prefix has to move it.
const customizationsDir = codefallDir + "/skills"

// customizationName is the file a skill reads from its directory under customizationsDir.
const customizationName = "CUSTOMIZE.md"

// mentionsNamed is how many tracked files the former-prefix report names before it counts the rest.
const mentionsNamed = 5

// formerPrefixReport is what the cleanup step adds to its sentence when this run moved the skills from
// one prefix to another: the project's own files that still name a skill under the former prefix,
// which codefall never rewrites because they are the project's, and the customization files the
// skills will no longer find under their old directory names. Each clause is there so the person can
// decide; nothing here changes a file (ADR-015).
//
// The mentions come from git, which knows which files are the project's: tracked files outside the
// skills directories this run wrote and outside .codefall/, whose shared files and hook scripts the
// run rewrote and whose review and test records are history. A directory that is not a repository, or
// a machine with no git, has no tracked files to report and the clause is left out rather than
// guessed at.
func (i *Initialize) formerPrefixReport(
	ctx context.Context, request Request, former string, rename domain.SkillRename, dests map[string]string,
) ([]string, error) {
	skills, err := i.source.Skills()
	if err != nil {
		return nil, fmt.Errorf("read the skills the extension ships: %w", err)
	}

	var clauses []string

	mentions, err := i.formerMentions(ctx, request.Dir, former, skills, rename, dests)
	if err != nil {
		return nil, err
	}

	if len(mentions) > 0 {
		clauses = append(clauses, fmt.Sprintf("%d tracked %s still %s a skill as %s-<verb> and %s left as %s, the project's own: %s",
			len(mentions), plural(len(mentions), "file", "files"), plural(len(mentions), "names", "name"), former,
			plural(len(mentions), "is", "are"), plural(len(mentions), "it is", "they are"), someFiles(mentions, mentionsNamed)))
	}

	for _, skill := range skills {
		formerName := strings.Replace(skill, settings.SkillPrefixCodefall+"-", former+"-", 1)
		current := rename.Text(skill)

		if formerName == current {
			continue
		}

		at := path.Join(customizationsDir, formerName, customizationName)

		_, err := i.files.ReadFile(filepath.Join(request.Dir, filepath.FromSlash(at)))

		switch {
		case errors.Is(err, fs.ErrNotExist):
			continue
		case err != nil:
			return nil, fmt.Errorf("read %s: %w", at, err)
		}

		clauses = append(clauses, fmt.Sprintf("move %s to %s/ for %s to read it", at, path.Join(customizationsDir, current), current))
	}

	return clauses, nil
}

// formerMentions is the tracked files, by the path a person reads them at and sorted, that still name
// a skill under former, outside what this run wrote. git grep exits 1 for no match and something else
// for a directory it cannot search; neither is an error here, because the report is advice and a
// project outside a repository has nothing tracked to advise about.
//
// The project's AGENTS.md is read here rather than by git, because codefall's own sections in it still
// carry the former names when the cleanup step runs — the agents step rewrites them later in the same
// run — and only what the project wrote outside the markers is the project's.
func (i *Initialize) formerMentions(
	ctx context.Context, dir, former string, skills []string, rename domain.SkillRename, dests map[string]string,
) ([]string, error) {
	if i.runner.LookPath("git").IsAbsent() {
		return nil, nil
	}

	verbs := make([]string, 0, len(skills))
	for _, skill := range skills {
		verbs = append(verbs, strings.TrimPrefix(skill, settings.SkillPrefixCodefall+"-"))
	}

	args := []string{"grep", "-l", "-I", "-w", "-E", former + "-(" + strings.Join(verbs, "|") + ")", "--", ".",
		":(exclude)" + codefallDir, ":(exclude)" + agentsName}

	for _, dest := range dests {
		args = append(args, ":(exclude)"+dest+"/"+skillsSource)
	}

	result, err := i.runner.Run(ctx, dir, "git", args...)
	if err != nil {
		return nil, fmt.Errorf("run git grep: %w", err)
	}

	var files []string

	if result.ExitCode == 0 {
		for line := range strings.SplitSeq(strings.TrimSpace(result.Stdout), "\n") {
			if line != "" {
				files = append(files, line)
			}
		}
	}

	ownWords, err := i.agentsOutsideSections(dir)
	if err != nil {
		return nil, err
	}

	for _, name := range rename.Names(ownWords) {
		if strings.HasPrefix(name, former+"-") {
			files = append(files, agentsName)

			break
		}
	}

	slices.Sort(files)

	return files, nil
}

// agentsOutsideSections is what the project's AGENTS.md says outside codefall's marked sections: the
// project's own words, which no run rewrites. A file that is not there says nothing.
func (i *Initialize) agentsOutsideSections(dir string) (string, error) {
	body, err := i.readProjectFile(dir, agentsName)
	if err != nil {
		return "", err
	}

	text := body.OrEmpty()

	for _, spec := range sectionSpecs {
		begin := strings.Index(text, spec.begin)
		if begin < 0 {
			continue
		}

		offset := strings.Index(text[begin:], spec.end)
		if offset < 0 {
			// A section with no closing marker is the agents step's to refuse; here the rest of the
			// file is read as codefall's, which errs toward saying less.
			text = text[:begin]

			break
		}

		text = text[:begin] + text[begin+offset+len(spec.end):]
	}

	return text, nil
}

// someFiles is "docs/a.md and docs/b.md", or the first few of a longer list and how many more.
func someFiles(files []string, named int) string {
	if len(files) <= named {
		return sentenceList(files)
	}

	return fmt.Sprintf("%s and %d more", strings.Join(files[:named], ", "), len(files)-named)
}

// plural picks the form a count takes.
func plural(count int, one, many string) string {
	if count == 1 {
		return one
	}

	return many
}
