package resolve_dns

import (
	"github.com/ruohao1/penta/internal/actions"
	"github.com/ruohao1/penta/internal/targets"
)

const InputKind = "resolve_dns.input"

var Spec = actions.ActionSpec{
	Type:       actions.ActionResolveDNS,
	Permission: actions.PermissionPassive,
	InputKind:  InputKind,
	RequiresEvidence: []actions.EvidenceRequirement{
		{
			Kind:        actions.EvidenceTarget,
			TargetTypes: []targets.Type{targets.TypeDomain},
		},
	},
	ProducesEvidence: []actions.EvidenceKind{
		actions.EvidenceDNSRecord,
	},
}
