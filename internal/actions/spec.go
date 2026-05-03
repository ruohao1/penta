package actions

import "github.com/ruohao1/penta/internal/targets"

type PermissionLevel string

const (
	PermissionPassive    PermissionLevel = "passive"
	PermissionSafeProbe  PermissionLevel = "safe_probe"
	PermissionActiveScan PermissionLevel = "active_scan"
	PermissionIntrusive  PermissionLevel = "intrusive"
	PermissionManualOnly PermissionLevel = "manual_only"
)

type EvidenceKind string

const (
	EvidenceTarget       EvidenceKind = "target"
	EvidenceService      EvidenceKind = "service"
	EvidenceDNSRecord    EvidenceKind = "dns_record"
	EvidenceHTTPResponse EvidenceKind = "http_response"
)

type ActionSpec struct {
	Type             ActionType
	Permission       PermissionLevel
	InputKind        string
	ProducesEvidence []EvidenceKind
	RequiresEvidence []EvidenceRequirement
}

type EvidenceRequirement struct {
	Kind        EvidenceKind
	TargetTypes []targets.Type
}
