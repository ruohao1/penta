package probe_http

import (
	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/targets"
)

const InputKind = "probe_http.input"

var Spec = actions.ActionSpec{
	Type:       actions.ActionProbeHTTP,
	Permission: actions.PermissionSafeProbe,
	InputKind:  InputKind,
	RequiresEvidence: []actions.EvidenceRequirement{
		{
			Kind: actions.EvidenceTarget,
			TargetTypes: []targets.Type{
				targets.TypeDomain,
				targets.TypeIP,
				targets.TypeURL,
			},
		},
	},
	ProducesEvidence: []actions.EvidenceKind{
		actions.EvidenceService,
	},
}
