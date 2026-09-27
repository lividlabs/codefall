package extensions

import "maps"

// skillRenames is what every current skill directory used to be called: the former directory name
// under skills/, and the name it has now. `codefall upgrade` reads it to report a skill it removes as
// a rename rather than a deletion, when the files an earlier install wrote under the former name are
// gone from the tree this binary ships (ADR-010).
//
// Maintenance rule: a pull request that renames a skill adds a row here and a row to the "Skill
// renames" table in skills/codefall-graft/lineage.md, in the same pull request. The two records say
// the same thing to two readers: this one to the binary, that one to the graft skill and to a
// person. Without the row here, upgrade reports the old directory as no longer shipped, which is
// true and unhelpful.
var skillRenames = map[string]string{
	"conceptualize":          "codefall-envision",
	"codefall-conceptualize": "codefall-envision",
	"design":                 "codefall-design",
	"graft":                  "codefall-graft",
	"implement":              "codefall-implement",
	"mock-up":                "codefall-mock-up",
	"scaffold":               "codefall-scaffold",
	"specify":                "codefall-specify",
}

// SkillRenames returns the former skill directory names and the name each has now. Each call returns
// its own copy, so a caller that edits what it gets back changes nothing for the next one.
func SkillRenames() map[string]string {
	return maps.Clone(skillRenames)
}
