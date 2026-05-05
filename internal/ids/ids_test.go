package ids

import "testing"

func TestNewUsesPrefixAndReadableToken(t *testing.T) {
	id := New(PrefixEvidence)
	if len(id) != len(PrefixEvidence)+TokenLength {
		t.Fatalf("New() length = %d, want %d", len(id), len(PrefixEvidence)+TokenLength)
	}
	if id[:len(PrefixEvidence)] != PrefixEvidence {
		t.Fatalf("New() prefix = %q, want %q", id[:len(PrefixEvidence)], PrefixEvidence)
	}

	allowed := map[rune]bool{}
	for _, ch := range alphabet {
		allowed[ch] = true
	}
	for _, ch := range id[len(PrefixEvidence):] {
		if !allowed[ch] {
			t.Fatalf("New() token contains disallowed character %q in %q", ch, id)
		}
	}
}

func TestNewIDsAreUniqueInSmallSample(t *testing.T) {
	seen := map[string]bool{}
	for range 1000 {
		id := New(PrefixTask)
		if seen[id] {
			t.Fatalf("New() generated duplicate ID %q", id)
		}
		seen[id] = true
	}
}

func TestSelectorPrefixHelpersAcceptOldAndNewIDs(t *testing.T) {
	if !IsEvidenceID("evidence_old") || !IsEvidenceID("evd_new") {
		t.Fatal("IsEvidenceID should accept old and new evidence prefixes")
	}
	if IsEvidenceID("artifact_old") {
		t.Fatal("IsEvidenceID accepted artifact prefix")
	}
	if !IsArtifactID("artifact_old") || !IsArtifactID("art_new") {
		t.Fatal("IsArtifactID should accept old and new artifact prefixes")
	}
	if IsArtifactID("evidence_old") {
		t.Fatal("IsArtifactID accepted evidence prefix")
	}
}
