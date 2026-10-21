package skillgap

import (
	"reflect"
	"testing"
	"time"
)

var t0 = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

func level(v int) *int { return &v }

func leadProfile() PositionProfile {
	return PositionProfile{ID: "p-lead", Version: 2, Name: "Team Lead", Requirements: []Requirement{
		{CompetencyID: "go", RequiredLevel: 3},
		{CompetencyID: "lead", RequiredLevel: 4},
		{CompetencyID: "comm", RequiredLevel: 3},
		{CompetencyID: "sql", RequiredLevel: 2},
		{CompetencyID: "arch", RequiredLevel: 3},
	}}
}

func leadRecords() []LevelRecord {
	return []LevelRecord{
		{CompetencyID: "go", Level: 2, Status: StatusFinal, SetAt: t0},
		{CompetencyID: "go", Level: 3, Status: StatusFinal, SetAt: t0.Add(24 * time.Hour)},
		{CompetencyID: "lead", Level: 1, Status: StatusFinal, SetAt: t0},
		{CompetencyID: "lead", Level: 3, Status: StatusPending, SetAt: t0.Add(48 * time.Hour)},
		{CompetencyID: "comm", Level: 2, Status: StatusFinal, SetAt: t0},
		{CompetencyID: "comm", Level: 3, Status: StatusPending, SetAt: t0.Add(-24 * time.Hour)},
		{CompetencyID: "sql", Level: 4, Status: StatusPending, SetAt: t0},
	}
}

func leadPlans() []DevelopmentPlan {
	return []DevelopmentPlan{
		{ID: "pl-1", Title: "Leadership", CompetencyIDs: []string{"lead"}},
		{ID: "pl-2", Title: "Team lead track", CompetencyIDs: []string{"lead", "comm"}},
		{ID: "pl-3", Title: "Communication", CompetencyIDs: []string{"comm"}},
		{ID: "pl-4", Title: "Data and architecture", CompetencyIDs: []string{"sql", "arch"}},
		{ID: "pl-5", Title: "Go", CompetencyIDs: []string{"go"}},
		{ID: "pl-0", Title: "Public speaking", CompetencyIDs: []string{"comm"}},
	}
}

func competencyIDs(items []Item) []string {
	ids := []string{}
	for _, item := range items {
		ids = append(ids, item.CompetencyID)
	}
	return ids
}

func TestReferenceExample(t *testing.T) {
	report := Calculate(leadProfile(), leadRecords())

	if report.ProfileID != "p-lead" || report.ProfileVersion != 2 {
		t.Fatalf("unexpected profile: %q v%d", report.ProfileID, report.ProfileVersion)
	}
	wantItems := []Item{
		{CompetencyID: "arch", RequiredLevel: 3, Status: GapNotAssessed},
		{CompetencyID: "comm", RequiredLevel: 3, Status: GapDeficit, Effective: level(2), Delta: 1},
		{CompetencyID: "go", RequiredLevel: 3, Status: GapMet, Effective: level(3)},
		{CompetencyID: "lead", RequiredLevel: 4, Status: GapDeficit, Effective: level(1), Pending: level(3), Delta: 3},
		{CompetencyID: "sql", RequiredLevel: 2, Status: GapNotAssessed, Pending: level(4)},
	}
	if !reflect.DeepEqual(report.Items, wantItems) {
		t.Fatalf("Items = %s\nwant %s", formatItems(report.Items), formatItems(wantItems))
	}
	if got, want := competencyIDs(report.Deficits()), []string{"lead", "comm"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Deficits() = %v, want %v", got, want)
	}
	if got, want := competencyIDs(report.NotAssessed()), []string{"arch", "sql"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("NotAssessed() = %v, want %v", got, want)
	}
	if report.HasSufficientData() {
		t.Fatal("HasSufficientData() = true, want false")
	}

	recommendations := Recommend(report, leadPlans(), 10)
	if want := []string{"arch", "sql"}; !reflect.DeepEqual(recommendations.AssessFirst, want) {
		t.Fatalf("AssessFirst = %v, want %v", recommendations.AssessFirst, want)
	}
	want := []struct {
		id     string
		score  int
		covers []string
	}{
		{id: "pl-2", score: 4, covers: []string{"lead", "comm"}},
		{id: "pl-1", score: 3, covers: []string{"lead"}},
		{id: "pl-0", score: 1, covers: []string{"comm"}},
		{id: "pl-3", score: 1, covers: []string{"comm"}},
	}
	// pl-4 develops only not assessed competencies, pl-5 only a met one.
	if len(recommendations.Plans) != len(want) {
		t.Fatalf("got %d plans, want %d: %+v", len(recommendations.Plans), len(want), recommendations.Plans)
	}
	for i, w := range want {
		got := recommendations.Plans[i]
		if got.Plan.ID != w.id || got.Score != w.score || !reflect.DeepEqual(got.Covers, w.covers) {
			t.Fatalf("plan %d = %s (%d, %v), want %s (%d, %v)", i, got.Plan.ID, got.Score, got.Covers, w.id, w.score, w.covers)
		}
	}
}

func TestLevelAboveRequiredIsMet(t *testing.T) {
	report := Calculate(
		PositionProfile{ID: "p", Version: 1, Requirements: []Requirement{{CompetencyID: "go", RequiredLevel: 2}}},
		[]LevelRecord{{CompetencyID: "go", Level: 4, Status: StatusFinal, SetAt: t0}},
	)
	want := Item{CompetencyID: "go", RequiredLevel: 2, Status: GapMet, Effective: level(4)}
	if !reflect.DeepEqual(report.Items, []Item{want}) {
		t.Fatalf("Items = %s", formatItems(report.Items))
	}
	if !report.HasSufficientData() || len(report.Deficits()) != 0 {
		t.Fatal("met competency must not be a deficit or missing data")
	}
}

func TestOnlyPendingIsNotAssessed(t *testing.T) {
	report := Calculate(
		PositionProfile{ID: "p", Version: 1, Requirements: []Requirement{{CompetencyID: "go", RequiredLevel: 2}}},
		[]LevelRecord{
			{CompetencyID: "go", Level: 1, Status: StatusPending, SetAt: t0},
			{CompetencyID: "go", Level: 3, Status: StatusPending, SetAt: t0.Add(time.Hour)},
		},
	)
	want := Item{CompetencyID: "go", RequiredLevel: 2, Status: GapNotAssessed, Pending: level(3)}
	if !reflect.DeepEqual(report.Items, []Item{want}) {
		t.Fatalf("Items = %s", formatItems(report.Items))
	}
}

func TestResolveLevels(t *testing.T) {
	tests := []struct {
		name          string
		records       []LevelRecord
		wantEffective *int
		wantPending   *int
	}{
		{name: "no records"},
		{
			name: "latest final wins",
			records: []LevelRecord{
				{CompetencyID: "go", Level: 3, Status: StatusFinal, SetAt: t0.Add(time.Hour)},
				{CompetencyID: "go", Level: 1, Status: StatusFinal, SetAt: t0},
			},
			wantEffective: level(3),
		},
		{
			name: "equal set_at keeps first record",
			records: []LevelRecord{
				{CompetencyID: "go", Level: 2, Status: StatusFinal, SetAt: t0},
				{CompetencyID: "go", Level: 4, Status: StatusFinal, SetAt: t0},
				{CompetencyID: "go", Level: 1, Status: StatusPending, SetAt: t0.Add(time.Hour)},
				{CompetencyID: "go", Level: 3, Status: StatusPending, SetAt: t0.Add(time.Hour)},
			},
			wantEffective: level(2), wantPending: level(1),
		},
		{
			name: "pending at the same time as final is hidden",
			records: []LevelRecord{
				{CompetencyID: "go", Level: 2, Status: StatusFinal, SetAt: t0},
				{CompetencyID: "go", Level: 3, Status: StatusPending, SetAt: t0},
			},
			wantEffective: level(2),
		},
		{
			name: "only latest pending is considered",
			records: []LevelRecord{
				{CompetencyID: "go", Level: 2, Status: StatusFinal, SetAt: t0},
				{CompetencyID: "go", Level: 4, Status: StatusPending, SetAt: t0.Add(time.Hour)},
				{CompetencyID: "go", Level: 3, Status: StatusPending, SetAt: t0.Add(2 * time.Hour)},
			},
			wantEffective: level(2), wantPending: level(3),
		},
		{
			name: "other competencies and unknown statuses ignored",
			records: []LevelRecord{
				{CompetencyID: "sql", Level: 4, Status: StatusFinal, SetAt: t0},
				{CompetencyID: "go", Level: 4, SetAt: t0},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			effective, pending := resolveLevels(test.records, "go")
			if !reflect.DeepEqual(effective, test.wantEffective) || !reflect.DeepEqual(pending, test.wantPending) {
				t.Fatalf("resolveLevels = (%v, %v), want (%v, %v)", formatLevel(effective), formatLevel(pending), formatLevel(test.wantEffective), formatLevel(test.wantPending))
			}
		})
	}
}

func TestRecommendLimit(t *testing.T) {
	report := Calculate(leadProfile(), leadRecords())

	limited := Recommend(report, leadPlans(), 2)
	if got := planIDs(limited.Plans); !reflect.DeepEqual(got, []string{"pl-2", "pl-1"}) {
		t.Fatalf("limit 2: plans = %v", got)
	}
	unlimited := Recommend(report, leadPlans(), 0)
	if got := planIDs(unlimited.Plans); !reflect.DeepEqual(got, []string{"pl-2", "pl-1", "pl-0", "pl-3"}) {
		t.Fatalf("limit 0: plans = %v", got)
	}

	empty := Recommend(report, nil, 10)
	if empty.Plans == nil || len(empty.Plans) != 0 {
		t.Fatalf("no plans: Plans = %#v, want empty", empty.Plans)
	}
	if !reflect.DeepEqual(empty.AssessFirst, []string{"arch", "sql"}) {
		t.Fatalf("no plans: AssessFirst = %v", empty.AssessFirst)
	}

	assessed := Recommend(Report{}, leadPlans(), 10)
	if assessed.AssessFirst == nil || len(assessed.AssessFirst) != 0 || len(assessed.Plans) != 0 {
		t.Fatalf("empty report: %#v", assessed)
	}
}

func TestInputNotMutated(t *testing.T) {
	profile := leadProfile()
	records := leadRecords()
	plans := leadPlans()

	report := Calculate(profile, records)
	itemsBefore := append([]Item(nil), report.Items...)
	report.Deficits()
	report.NotAssessed()
	recommendations := Recommend(report, plans, 10)

	if !reflect.DeepEqual(profile, leadProfile()) {
		t.Fatalf("profile mutated: %+v", profile)
	}
	if !reflect.DeepEqual(records, leadRecords()) {
		t.Fatalf("records mutated: %+v", records)
	}
	if !reflect.DeepEqual(plans, leadPlans()) {
		t.Fatalf("plans mutated: %+v", plans)
	}
	if !reflect.DeepEqual(report.Items, itemsBefore) {
		t.Fatalf("report items mutated: %s", formatItems(report.Items))
	}

	recommendations.Plans[0].Plan.CompetencyIDs[0] = "changed"
	*report.Items[1].Effective = 4
	if !reflect.DeepEqual(plans, leadPlans()) || !reflect.DeepEqual(records, leadRecords()) {
		t.Fatal("results share memory with the input")
	}
}

func planIDs(plans []PlanRecommendation) []string {
	ids := []string{}
	for _, plan := range plans {
		ids = append(ids, plan.Plan.ID)
	}
	return ids
}

func formatLevel(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func formatItems(items []Item) []any {
	result := make([]any, 0, len(items))
	for _, item := range items {
		result = append(result, []any{item.CompetencyID, item.RequiredLevel, item.Status, formatLevel(item.Effective), formatLevel(item.Pending), item.Delta})
	}
	return result
}
