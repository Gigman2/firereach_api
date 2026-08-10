package claude

import (
	"strings"
	"testing"
)

// Plain unit test: no build tag, no live API call. It feeds contaminated
// strings straight through containsMedication (the deterministic filter
// Gateway.Ask applies to model output), which is what H3 asks for as
// belt-and-braces alongside the adversarial suite in
// gateway_adversarial_test.go — that suite is build-tagged `adversarial`
// and skipped without CLAUDE_API_KEY, so it never runs in CI. This test
// always runs and needs neither.
func TestContainsMedication_CatchesBrandsAndDosages(t *testing.T) {
	cases := []string{
		"Take ibuprofen for the pain.",
		"PANADOL is widely available.",
		"You could try Brufen or Nurofen.",
		"Advil, Tylenol, Ibucap, and Efpac are common brands.",
		"Paracetamol is usually recommended.",
		"Acetaminophen is the US name for the same drug.",
		"Aspirin should be avoided in children.",
		"Take 400mg every four hours.",
		"A dose of 200 mg is typical.",
		"the recommended amount is 1000mg per day",
	}

	for _, text := range cases {
		if !containsMedication(text) {
			t.Errorf("containsMedication(%q) = false, want true", text)
		}
	}
}

func TestContainsMedication_LeavesSafeTextAlone(t *testing.T) {
	cases := []string{
		"Call the Ghana National Fire Service immediately.",
		"Cool the burn under running water for twenty minutes.",
		"I can't advise on medication, including its name. A pharmacist or clinician can help with pain relief.",
		"Do not touch them until the power is off.",
		"Evacuate the building and call 192.",
	}

	for _, text := range cases {
		if containsMedication(text) {
			t.Errorf("containsMedication(%q) = true, want false", text)
		}
	}
}

// Gateway.Ask must substitute the scripted refusal, not merely detect the
// problem: a caller that only checked containsMedication and then still
// returned the model's own text would reopen the exact defect this filter
// exists to close.
func TestMedicationRefusal_ContainsNoMedicationItself(t *testing.T) {
	if containsMedication(medicationRefusal) {
		t.Fatalf("medicationRefusal itself trips containsMedication: %q", medicationRefusal)
	}
}

// The refusal discards the model's entire answer, not just the medication
// mention — so a real emergency question (a burn injury that happens to
// name a drug) must not lose its scene-safety escalation along with the
// drug name. The refusal must stand on its own, pointing at the same
// emergency number every other emergency case in the system prompt uses.
func TestMedicationRefusal_PointsToEmergencyNumber(t *testing.T) {
	if !strings.Contains(medicationRefusal, EmergencyPhone) {
		t.Fatalf("medicationRefusal does not mention the emergency number %q: %q", EmergencyPhone, medicationRefusal)
	}
}
