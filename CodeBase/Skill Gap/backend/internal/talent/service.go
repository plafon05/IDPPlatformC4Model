// Package talent exposes the skill gap engine over the project database.
package talent

import (
	"context"
	"errors"

	"idp-platform/backend/internal/talent/skillgap"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProfileNotFound  = errors.New("position profile not found")
	ErrEmployeeNotFound = errors.New("employee not found")
)

const recommendationsLimit = 10

// Service calculates skill gaps and plan recommendations.
type Service struct {
	repo *Repository
}

// SkillGapResult is the skill gap report with its recommendations.
type SkillGapResult struct {
	Report          skillgap.Report
	Recommendations skillgap.Recommendations
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{repo: &Repository{db: db}}
}

// SkillGap calculates the skill gap of an employee against a profile version
// (0 means the latest version) and recommends development plans.
func (s *Service) SkillGap(ctx context.Context, employeeID, profileID string, version int) (*SkillGapResult, error) {
	records, err := s.repo.GetLevelRecords(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	profile, err := s.repo.GetProfile(ctx, profileID, version)
	if err != nil {
		return nil, err
	}
	report := skillgap.Calculate(profile, records)

	plans, err := s.repo.ListPlans(ctx)
	if err != nil {
		return nil, err
	}
	return &SkillGapResult{
		Report:          report,
		Recommendations: skillgap.Recommend(report, plans, recommendationsLimit),
	}, nil
}
