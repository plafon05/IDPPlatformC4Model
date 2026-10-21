// Package skillgap calculates the skill gap of an employee against a position
// profile and recommends development plans. It is pure: no database, HTTP,
// logging, context or clock.
package skillgap

import "time"

// Requirement is the level a position profile requires for a competency.
type Requirement struct {
	CompetencyID  string
	RequiredLevel int
}

// PositionProfile is one version of a position profile.
type PositionProfile struct {
	ID           string
	Version      int
	Name         string
	Requirements []Requirement
}

// LevelStatus is the status of an employee competency level record.
type LevelStatus int

// Level record statuses. Rejected levels and self-assessments never reach the
// engine.
const (
	StatusFinal LevelStatus = iota + 1
	StatusPending
)

// LevelRecord is a competency level set for an employee.
type LevelRecord struct {
	CompetencyID string
	Level        int
	Status       LevelStatus
	SetAt        time.Time
}

// DevelopmentPlan is a development plan (IDP template) and the competencies it
// develops.
type DevelopmentPlan struct {
	ID            string
	Title         string
	CompetencyIDs []string
}

// GapStatus is the result of comparing a required and an effective level.
type GapStatus int

// Gap statuses.
const (
	GapMet GapStatus = iota + 1
	GapDeficit
	GapNotAssessed
)

// Item is the gap for one required competency.
type Item struct {
	CompetencyID  string
	RequiredLevel int
	Status        GapStatus
	// Effective is nil when the competency has no final level.
	Effective *int
	// Pending is shown only and never used in the calculation.
	Pending *int
	Delta   int
}

// Report is the skill gap of an employee against a position profile version.
type Report struct {
	ProfileID      string
	ProfileVersion int
	Items          []Item
}

// PlanRecommendation is a development plan that closes some deficits.
type PlanRecommendation struct {
	Plan   DevelopmentPlan
	Covers []string
	Score  int
}

// Recommendations tell what to assess first and which plans to take.
type Recommendations struct {
	AssessFirst []string
	Plans       []PlanRecommendation
}
