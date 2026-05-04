package fetch_root

import "github.com/ruohao1/penta/internal/actions"

const InputKind = "fetch_root.input"

var Spec = actions.ActionSpec{
	Type:       actions.ActionFetchRoot,
	Permission: actions.PermissionSafeProbe,
	InputKind:  InputKind,
	RequiresEvidence: []actions.EvidenceRequirement{
		{Kind: actions.EvidenceService},
	},
	ProducesEvidence: []actions.EvidenceKind{
		actions.EvidenceHTTPResponse,
	},
}
