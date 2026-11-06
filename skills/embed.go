// Package skills bundles the version-matched CLI skill without network access.
package skills

import "embed"

// KnowledgeBase contains the complete skill and its five reference documents.
//
//go:embed knowledge-base/SKILL.md knowledge-base/references/*.md
var KnowledgeBase embed.FS
