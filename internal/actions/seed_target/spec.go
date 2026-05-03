package seed_target

import "github.com/ruohao1/penta/internal/actions"

const InputKind = "seed_target.input"

var Spec = actions.ActionSpec{
	Type:       actions.ActionSeedTarget,
	Permission: actions.PermissionPassive,
	InputKind:  InputKind,
	ProducesEvidence: []actions.EvidenceKind{
		actions.EvidenceTarget,
	},
}
