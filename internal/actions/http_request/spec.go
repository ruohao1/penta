package http_request

import "github.com/ruohao1/penta/internal/actions"

const InputKind = "http_request.input"

var Spec = actions.ActionSpec{
	Type:       actions.ActionHTTPRequest,
	Permission: actions.PermissionSafeProbe,
	InputKind:  InputKind,
	RequiresEvidence: []actions.EvidenceRequirement{
		{Kind: actions.EvidenceService},
	},
	ProducesEvidence: []actions.EvidenceKind{
		actions.EvidenceHTTPResponse,
	},
}
