package talent

import (
	"context"
	"fmt"

	"idp-platform/backend/internal/talent/skillgap"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository reads the skill gap engine input from PostgreSQL.
type Repository struct {
	db *pgxpool.Pool
}

// GetProfile returns the requirements of an active profile version; version 0
// means the latest one.
func (r *Repository) GetProfile(ctx context.Context, profileID string, version int) (skillgap.PositionProfile, error) {
	rows, err := r.db.Query(ctx, `
		SELECT p.name, pr.version, pr.competency_id::text, pr.required_level
		FROM position_profiles p
		JOIN profile_requirements pr ON pr.profile_id = p.id
		WHERE p.id = $1 AND p.is_active = true
			AND pr.version = CASE WHEN $2::int = 0
				THEN (SELECT MAX(version) FROM profile_requirements WHERE profile_id = p.id)
				ELSE $2::int END
		ORDER BY pr.competency_id
	`, profileID, version)
	if err != nil {
		return skillgap.PositionProfile{}, err
	}
	defer rows.Close()

	profile := skillgap.PositionProfile{ID: profileID}
	for rows.Next() {
		var requirement skillgap.Requirement
		if err := rows.Scan(&profile.Name, &profile.Version, &requirement.CompetencyID, &requirement.RequiredLevel); err != nil {
			return skillgap.PositionProfile{}, err
		}
		profile.Requirements = append(profile.Requirements, requirement)
	}
	if err := rows.Err(); err != nil {
		return skillgap.PositionProfile{}, err
	}
	if len(profile.Requirements) == 0 {
		return skillgap.PositionProfile{}, ErrProfileNotFound
	}
	return profile, nil
}

// GetLevelRecords returns the final and pending competency levels of an
// employee ordered by set_at. Rejected levels are not returned.
func (r *Repository) GetLevelRecords(ctx context.Context, employeeID string) ([]skillgap.LevelRecord, error) {
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, employeeID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrEmployeeNotFound
	}

	rows, err := r.db.Query(ctx, `
		SELECT competency_id::text, level, status, set_at
		FROM employee_competency_levels
		WHERE employee_id = $1 AND status IN ('final', 'pending_confirmation')
		ORDER BY set_at, id
	`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []skillgap.LevelRecord
	for rows.Next() {
		var record skillgap.LevelRecord
		var status string
		if err := rows.Scan(&record.CompetencyID, &record.Level, &status, &record.SetAt); err != nil {
			return nil, err
		}
		switch status {
		case "final":
			record.Status = skillgap.StatusFinal
		case "pending_confirmation":
			record.Status = skillgap.StatusPending
		default:
			return nil, fmt.Errorf("unexpected competency level status %q", status)
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

// ListPlans returns active IDP templates with the competencies they develop.
// Templates without competencies are skipped.
func (r *Repository) ListPlans(ctx context.Context) ([]skillgap.DevelopmentPlan, error) {
	rows, err := r.db.Query(ctx, `
		SELECT t.id::text, t.title, tc.competency_id::text
		FROM idp_templates t
		JOIN template_competencies tc ON tc.template_id = t.id
		WHERE t.is_active = true
		ORDER BY t.id, tc.competency_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []skillgap.DevelopmentPlan
	for rows.Next() {
		var id, title, competencyID string
		if err := rows.Scan(&id, &title, &competencyID); err != nil {
			return nil, err
		}
		if len(plans) == 0 || plans[len(plans)-1].ID != id {
			plans = append(plans, skillgap.DevelopmentPlan{ID: id, Title: title})
		}
		last := &plans[len(plans)-1]
		last.CompetencyIDs = append(last.CompetencyIDs, competencyID)
	}
	return plans, rows.Err()
}
