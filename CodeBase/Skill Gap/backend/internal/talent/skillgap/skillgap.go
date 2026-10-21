package skillgap

import "sort"

// Calculate compares the profile requirements with the employee level records.
// Items are sorted by CompetencyID. The arguments are not modified.
func Calculate(profile PositionProfile, records []LevelRecord) Report {
	items := make([]Item, 0, len(profile.Requirements))
	for _, requirement := range profile.Requirements {
		effective, pending := resolveLevels(records, requirement.CompetencyID)
		item := Item{
			CompetencyID:  requirement.CompetencyID,
			RequiredLevel: requirement.RequiredLevel,
			Effective:     effective,
			Pending:       pending,
		}
		switch {
		case effective == nil:
			item.Status = GapNotAssessed
		case *effective < requirement.RequiredLevel:
			item.Status = GapDeficit
			item.Delta = requirement.RequiredLevel - *effective
		default:
			item.Status = GapMet
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CompetencyID < items[j].CompetencyID
	})

	return Report{ProfileID: profile.ID, ProfileVersion: profile.Version, Items: items}
}

// resolveLevels returns the effective level (latest final record) and the
// pending level to show (latest pending record newer than the effective one).
// On equal SetAt the record that comes first in records wins.
func resolveLevels(records []LevelRecord, competencyID string) (effective, pending *int) {
	var final, latestPending *LevelRecord
	for i := range records {
		record := &records[i]
		if record.CompetencyID != competencyID {
			continue
		}
		switch record.Status {
		case StatusFinal:
			if final == nil || record.SetAt.After(final.SetAt) {
				final = record
			}
		case StatusPending:
			if latestPending == nil || record.SetAt.After(latestPending.SetAt) {
				latestPending = record
			}
		}
	}

	if final != nil {
		level := final.Level
		effective = &level
	}
	if latestPending != nil && (final == nil || latestPending.SetAt.After(final.SetAt)) {
		level := latestPending.Level
		pending = &level
	}
	return effective, pending
}

// Deficits returns the deficit items ordered by Delta desc, CompetencyID asc.
func (r Report) Deficits() []Item {
	var deficits []Item
	for _, item := range r.Items {
		if item.Status == GapDeficit {
			deficits = append(deficits, item)
		}
	}
	sort.SliceStable(deficits, func(i, j int) bool {
		if deficits[i].Delta != deficits[j].Delta {
			return deficits[i].Delta > deficits[j].Delta
		}
		return deficits[i].CompetencyID < deficits[j].CompetencyID
	})
	return deficits
}

// NotAssessed returns the items without a final level in Items order.
func (r Report) NotAssessed() []Item {
	var notAssessed []Item
	for _, item := range r.Items {
		if item.Status == GapNotAssessed {
			notAssessed = append(notAssessed, item)
		}
	}
	return notAssessed
}

// HasSufficientData reports whether every required competency is assessed.
func (r Report) HasSufficientData() bool {
	return len(r.NotAssessed()) == 0
}

// Recommend lists the competencies to assess first and the plans that close at
// least one deficit, ordered by Score desc, Plan.ID asc and limited to limit
// when limit > 0. The arguments are not modified.
func Recommend(report Report, plans []DevelopmentPlan, limit int) Recommendations {
	result := Recommendations{AssessFirst: []string{}, Plans: []PlanRecommendation{}}
	for _, item := range report.NotAssessed() {
		result.AssessFirst = append(result.AssessFirst, item.CompetencyID)
	}

	deficits := report.Deficits()
	for _, plan := range plans {
		develops := make(map[string]struct{}, len(plan.CompetencyIDs))
		for _, competencyID := range plan.CompetencyIDs {
			develops[competencyID] = struct{}{}
		}
		recommendation := PlanRecommendation{}
		for _, deficit := range deficits {
			if _, ok := develops[deficit.CompetencyID]; ok {
				recommendation.Covers = append(recommendation.Covers, deficit.CompetencyID)
				recommendation.Score += deficit.Delta
			}
		}
		if len(recommendation.Covers) == 0 {
			continue
		}
		plan.CompetencyIDs = append([]string(nil), plan.CompetencyIDs...)
		recommendation.Plan = plan
		result.Plans = append(result.Plans, recommendation)
	}
	sort.SliceStable(result.Plans, func(i, j int) bool {
		if result.Plans[i].Score != result.Plans[j].Score {
			return result.Plans[i].Score > result.Plans[j].Score
		}
		return result.Plans[i].Plan.ID < result.Plans[j].Plan.ID
	})
	if limit > 0 && len(result.Plans) > limit {
		result.Plans = result.Plans[:limit]
	}
	return result
}
