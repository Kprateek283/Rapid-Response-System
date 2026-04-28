package mock

// RoleRequirement defines a staff role needed for dispatch.
type RoleRequirement struct {
	Role            string             `json:"role"`
	MinCount        int                `json:"min_count"`
	SkillThresholds map[string]float64 `json:"skill_thresholds,omitempty"`
}

// MockAssessmentResult is the output of AI assessment (Time 1 or Time 2).
type MockAssessmentResult struct {
	IncidentID    string            `json:"incident_id"`
	FinalSeverity int               `json:"final_severity"`
	AISummary     string            `json:"ai_summary"`
	RequiredStaff []RoleRequirement `json:"required_staff"`
}

// MockSummary is the output of the rolling summary mock.
type MockSummary struct {
	Severity int      `json:"severity"`
	Summary  string   `json:"summary"`
	Actions  []string `json:"actions"`
}
