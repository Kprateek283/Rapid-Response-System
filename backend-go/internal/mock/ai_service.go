package mock

import (
	"fmt"
	"time"

	"github.com/google-hackathon/rapid-response/internal/models"
)

// MockAIAssessment simulates the Time 1 AI assessment after media-ready.
// In production, this would be a gRPC call to the Python AI service.
func MockAIAssessment(incident models.Incident) MockAssessmentResult {
	// Simulate AI processing time
	time.Sleep(2 * time.Second)

	return MockAssessmentResult{
		IncidentID:    incident.ID,
		FinalSeverity: 7, // Default high severity for hackathon demo
		AISummary:     "AI Assessment: Possible emergency detected. Guest appears distressed. Recommending immediate medical and security response.",
		RequiredStaff: []RoleRequirement{
			{Role: "guard", MinCount: 1, SkillThresholds: map[string]float64{"combat": 0.5}},
			{Role: "medical_staff", MinCount: 1, SkillThresholds: map[string]float64{"first_aid": 0.7}},
		},
	}
}

// MockGroundTruthReeval simulates the Time 2 re-evaluation after ground truth submission.
func MockGroundTruthReeval(scoutSeverity int, initialSeverity int) MockAssessmentResult {
	time.Sleep(1 * time.Second)

	// Use scout severity as final, adjusted slightly
	finalSev := scoutSeverity
	staffNeeded := []RoleRequirement{}

	if finalSev >= 7 {
		staffNeeded = append(staffNeeded,
			RoleRequirement{Role: "guard", MinCount: 2, SkillThresholds: map[string]float64{"combat": 0.5}},
			RoleRequirement{Role: "medical_staff", MinCount: 1, SkillThresholds: map[string]float64{"first_aid": 0.7}},
		)
	} else if finalSev >= 4 {
		staffNeeded = append(staffNeeded,
			RoleRequirement{Role: "guard", MinCount: 1, SkillThresholds: map[string]float64{"combat": 0.3}},
		)
	}
	// If finalSev < 4, no additional staff needed

	return MockAssessmentResult{
		FinalSeverity: finalSev,
		RequiredStaff: staffNeeded,
		AISummary:     "Ground truth re-evaluation complete. Severity adjusted based on on-scene assessment.",
	}
}

// MockTranslate simulates AI-powered translation of text.
func MockTranslate(text, sourceLang string, targetLangs []string) map[string]string {
	result := map[string]string{sourceLang: text}
	for _, lang := range targetLangs {
		if lang != sourceLang {
			result[lang] = fmt.Sprintf("[%s translation of: %s]", lang, text)
		}
	}
	return result
}

// MockRollingSummary simulates periodic AI insight updates.
func MockRollingSummary(incidentID string) MockSummary {
	return MockSummary{
		Severity: 5,
		Summary:  "Ongoing incident. Staff on scene. Situation being assessed.",
		Actions:  []string{"Monitor situation", "Stand by for updates"},
	}
}

// MockCompileReport simulates the final AI-compiled incident report.
func MockCompileReport(incident models.Incident, events []models.IncidentEvent) map[string]interface{} {
	timeline := make([]map[string]interface{}, len(events))
	for i, e := range events {
		timeline[i] = map[string]interface{}{
			"event_type": e.EventType,
			"actor_id":   e.ActorID,
			"actor_role": e.ActorRole,
			"created_at": e.CreatedAt,
			"payload":    e.Payload,
		}
	}

	severity := 0
	if incident.FinalSeverity != nil {
		severity = *incident.FinalSeverity
	}

	return map[string]interface{}{
		"incident_id":    incident.ID,
		"hotel_id":       incident.HotelID,
		"room_id":        incident.RoomID,
		"guest_id":       incident.GuestID,
		"trigger_source": incident.TriggerSource,
		"timeline":       timeline,
		"final_severity": severity,
		"ai_summary":     incident.AISummary,
		"resolution":     "Incident resolved via 2-party handshake",
		"compiled_at":    time.Now().UTC(),
		"compiled_by":    "MOCK_AI_AGENT",
	}
}
