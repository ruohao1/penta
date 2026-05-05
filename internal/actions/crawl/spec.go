package crawl

import "github.com/ruohao1/penta/internal/actions"

const InputKind = "crawl.input"

var Spec = actions.ActionSpec{
	Type:       actions.ActionCrawl,
	Permission: actions.PermissionPassive,
	InputKind:  InputKind,
	RequiresEvidence: []actions.EvidenceRequirement{
		{Kind: actions.EvidenceHTTPResponse},
	},
	ProducesEvidence: []actions.EvidenceKind{
		actions.EvidenceCrawl,
	},
}
