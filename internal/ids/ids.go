package ids

import "crypto/rand"

const (
	PrefixRun      = "run_"
	PrefixSession  = "ses_"
	PrefixTask     = "tsk_"
	PrefixEvidence = "evd_"
	PrefixArtifact = "art_"
	PrefixScope    = "scp_"
	PrefixEvent    = "evt_"

	LegacyPrefixSession  = "session_"
	LegacyPrefixTask     = "task_"
	LegacyPrefixEvidence = "evidence_"
	LegacyPrefixArtifact = "artifact_"
	LegacyPrefixScope    = "scope_"
	LegacyPrefixEvent    = "event_"

	TokenLength = 10
)

const alphabet = "abcdefghjkmnpqrstuvwxyz"

func New(prefix string) string {
	return prefix + Token()
}

func Token() string {
	buf := make([]byte, TokenLength)
	for i := range buf {
		buf[i] = alphabet[randomIndex()]
	}
	return string(buf)
}

func IsEvidenceID(value string) bool {
	return hasAnyPrefix(value, PrefixEvidence, LegacyPrefixEvidence)
}

func IsArtifactID(value string) bool {
	return hasAnyPrefix(value, PrefixArtifact, LegacyPrefixArtifact)
}

func randomIndex() byte {
	var b [1]byte
	limit := byte(256 - (256 % len(alphabet)))
	for {
		if _, err := rand.Read(b[:]); err != nil {
			panic(err)
		}
		if b[0] < limit {
			return b[0] % byte(len(alphabet))
		}
	}
}

func hasAnyPrefix(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if len(value) >= len(prefix) && value[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}
