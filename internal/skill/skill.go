// Package skill embeds the agent-facing SKILL.md into the binary so the `agent`
// subcommand can print it with no repo or network access. See contract.md §5.
package skill

import _ "embed"

//go:embed SKILL.md
var Content string
