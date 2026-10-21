package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"idp-platform/backend/internal/auth"
	"idp-platform/backend/internal/config"
	"idp-platform/backend/internal/handler"
	"idp-platform/backend/internal/migrations"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	skillGapProfileID         = "90000000-0000-0000-0000-0000000000a1"
	skillGapInactiveProfileID = "90000000-0000-0000-0000-0000000000a2"
	skillGapMissingProfileID  = "90000000-0000-0000-0000-0000000000a3"
	skillGapGoID              = "c1000000-0000-0000-0000-0000000000a1"
	skillGapSQLID             = "c1000000-0000-0000-0000-0000000000a2"
	skillGapCommID            = "c1000000-0000-0000-0000-0000000000a3"
)

type skillGapPayload struct {
	ProfileID         string `json:"profile_id"`
	ProfileVersion    int    `json:"profile_version"`
	HasSufficientData bool   `json:"has_sufficient_data"`
	Items             []struct {
		CompetencyID   string `json:"competency_id"`
		RequiredLevel  int    `json:"required_level"`
		Status         string `json:"status"`
		EffectiveLevel *int   `json:"effective_level"`
		PendingLevel   *int   `json:"pending_level"`
		Delta          int    `json:"delta"`
	} `json:"items"`
	AssessFirst      []string `json:"assess_first"`
	RecommendedPlans []struct {
		ID     string   `json:"id"`
		Title  string   `json:"title"`
		Covers []string `json:"covers"`
		Score  int      `json:"score"`
	} `json:"recommended_plans"`
}

func TestTalentSkillGap(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	dbConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(dbConfig.Database, "_test") {
		t.Fatalf("integration tests require a database ending in _test, got %q", dbConfig.Database)
	}

	cfg := config.Config{
		AppEnv: "test", DatabaseURL: dsn,
		JWTSecrets:   []config.JWTSigningKey{{KeyID: "integration", Secret: "integration-test-secret"}},
		JWTAccessTTL: time.Hour, JWTRefreshTTL: time.Hour, CORSOrigins: []string{"http://example.test"},
	}
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	current, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Up(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(current); err != nil {
		t.Fatal(err)
	}

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	f := &fixture{db: db, cfg: cfg, publisher: &recordingPublisher{}}
	f.resetSkillGap(t)
	defer f.resetSkillGap(t)
	templateID := f.seedSkillGap(t)
	f.router = handler.NewRouter(cfg, db, nil, f.publisher)

	employeeToken := f.token(t, f.employeeID, "employee")
	managerToken := f.token(t, f.managerID, "employee", "manager")
	outsiderToken := f.token(t, f.outsiderID, "employee", "manager")
	hrToken := f.token(t, f.hrID, "employee", "hr_admin")
	path := func(employeeID, query string) string {
		return "/api/v1/employees/" + employeeID + "/skill-gap" + query
	}

	t.Run("access", func(t *testing.T) {
		latest := path(f.employeeID, "?profile_id="+skillGapProfileID)
		f.request(t, employeeToken, http.MethodGet, latest, nil, http.StatusOK)
		f.request(t, managerToken, http.MethodGet, latest, nil, http.StatusOK)
		f.request(t, outsiderToken, http.MethodGet, latest, nil, http.StatusForbidden)
		f.request(t, hrToken, http.MethodGet, latest, nil, http.StatusOK)
	})

	t.Run("latest version by default", func(t *testing.T) {
		response := f.request(t, employeeToken, http.MethodGet, path(f.employeeID, "?profile_id="+skillGapProfileID), nil, http.StatusOK)
		var payload skillGapPayload
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if payload.ProfileID != skillGapProfileID || payload.ProfileVersion != 2 || payload.HasSufficientData {
			t.Fatalf("unexpected profile summary: %+v", payload)
		}
		if len(payload.Items) != 3 {
			t.Fatalf("got %d items, want 3: %+v", len(payload.Items), payload.Items)
		}
		goItem, sqlItem, commItem := payload.Items[0], payload.Items[1], payload.Items[2]
		if goItem.CompetencyID != skillGapGoID || goItem.Status != "deficit" || goItem.RequiredLevel != 3 ||
			goItem.EffectiveLevel == nil || *goItem.EffectiveLevel != 2 || goItem.PendingLevel == nil || *goItem.PendingLevel != 3 || goItem.Delta != 1 {
			t.Fatalf("unexpected go item: %+v", goItem)
		}
		if sqlItem.CompetencyID != skillGapSQLID || sqlItem.Status != "met" || sqlItem.Delta != 0 || sqlItem.PendingLevel != nil {
			t.Fatalf("unexpected sql item: %+v", sqlItem)
		}
		if commItem.CompetencyID != skillGapCommID || commItem.Status != "not_assessed" || commItem.EffectiveLevel != nil || commItem.PendingLevel != nil {
			t.Fatalf("rejected level must not count: %+v", commItem)
		}
		if !reflect.DeepEqual(payload.AssessFirst, []string{skillGapCommID}) {
			t.Fatalf("assess_first = %v", payload.AssessFirst)
		}
		if len(payload.RecommendedPlans) != 1 || payload.RecommendedPlans[0].ID != templateID ||
			payload.RecommendedPlans[0].Score != 1 || !reflect.DeepEqual(payload.RecommendedPlans[0].Covers, []string{skillGapGoID}) {
			t.Fatalf("unexpected recommended plans: %+v", payload.RecommendedPlans)
		}
	})

	t.Run("old version keeps its requirements", func(t *testing.T) {
		response := f.request(t, managerToken, http.MethodGet, path(f.employeeID, "?profile_id="+skillGapProfileID+"&version=1"), nil, http.StatusOK)
		body := response.Body.String()
		if !strings.Contains(body, `"recommended_plans":[]`) || !strings.Contains(body, `"assess_first":[]`) {
			t.Fatalf("empty lists must be encoded as []: %s", body)
		}
		var payload skillGapPayload
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		if payload.ProfileVersion != 1 || !payload.HasSufficientData || len(payload.Items) != 2 {
			t.Fatalf("unexpected version 1 report: %+v", payload)
		}
		for _, item := range payload.Items {
			if item.Status != "met" || item.RequiredLevel != 2 {
				t.Fatalf("unexpected version 1 item: %+v", item)
			}
		}
	})

	t.Run("validation and not found", func(t *testing.T) {
		f.request(t, hrToken, http.MethodGet, path(f.employeeID, ""), nil, http.StatusBadRequest)
		f.request(t, hrToken, http.MethodGet, path(f.employeeID, "?profile_id="), nil, http.StatusBadRequest)
		f.request(t, hrToken, http.MethodGet, path(f.employeeID, "?profile_id=not-a-uuid"), nil, http.StatusBadRequest)
		f.request(t, hrToken, http.MethodGet, path(f.employeeID, "?profile_id="+skillGapProfileID+"&version=-1"), nil, http.StatusBadRequest)
		f.request(t, hrToken, http.MethodGet, path(f.employeeID, "?profile_id="+skillGapProfileID+"&version=abc"), nil, http.StatusBadRequest)

		response := f.request(t, hrToken, http.MethodGet, path(f.employeeID, "?profile_id="+skillGapMissingProfileID), nil, http.StatusNotFound)
		if !strings.Contains(response.Body.String(), "PROFILE_NOT_FOUND") {
			t.Fatalf("unexpected error: %s", response.Body.String())
		}
		f.request(t, hrToken, http.MethodGet, path(f.employeeID, "?profile_id="+skillGapInactiveProfileID), nil, http.StatusNotFound)
		f.request(t, hrToken, http.MethodGet, path(f.employeeID, "?profile_id="+skillGapProfileID+"&version=3"), nil, http.StatusNotFound)

		response = f.request(t, hrToken, http.MethodGet, path(skillGapMissingProfileID, "?profile_id="+skillGapProfileID), nil, http.StatusNotFound)
		if !strings.Contains(response.Body.String(), "USER_NOT_FOUND") {
			t.Fatalf("unexpected error: %s", response.Body.String())
		}
	})
}

func (f *fixture) seedSkillGap(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	hash, err := auth.HashPassword("TestPassword1")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := f.db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	f.managerID = insertUser(t, tx, "manager@test.local", hash, "Manager", "One", "Lead")
	f.employeeID = insertUser(t, tx, "employee@test.local", hash, "Employee", "One", "Developer")
	f.outsiderID = insertUser(t, tx, "outsider@test.local", hash, "Manager", "Other", "Lead")
	f.hrID = insertUser(t, tx, "hr@test.local", hash, "HR", "Admin", "HR")
	if _, err := tx.Exec(ctx, `UPDATE users SET manager_id=$1 WHERE id=$2`, f.managerID, f.employeeID); err != nil {
		t.Fatal(err)
	}
	insertRoles(t, tx, f.managerID, "employee", "manager")
	insertRoles(t, tx, f.employeeID, "employee")
	insertRoles(t, tx, f.outsiderID, "employee", "manager")
	insertRoles(t, tx, f.hrID, "employee", "hr_admin")

	statements := []struct {
		sql  string
		args []any
	}{
		{sql: `INSERT INTO competencies(id,name,category) VALUES($1,'Skill gap Go','technical'),($2,'Skill gap SQL','technical'),($3,'Skill gap communication','soft')`,
			args: []any{skillGapGoID, skillGapSQLID, skillGapCommID}},
		{sql: `INSERT INTO position_profiles(id,name,is_active) VALUES($1,'Skill gap backend',true),($2,'Skill gap archived',false)`,
			args: []any{skillGapProfileID, skillGapInactiveProfileID}},
		{sql: `INSERT INTO profile_requirements(profile_id,version,competency_id,required_level) VALUES
			($1,1,$3,2),($1,1,$4,2),
			($1,2,$3,3),($1,2,$4,2),($1,2,$5,2),
			($2,1,$3,1)`,
			args: []any{skillGapProfileID, skillGapInactiveProfileID, skillGapGoID, skillGapSQLID, skillGapCommID}},
		{sql: `INSERT INTO employee_competency_levels(employee_id,competency_id,level,status,method,basis,set_by,set_at) VALUES
			($1,$3,2,'final','manager_decision','review',$2,NOW()-INTERVAL '3 days'),
			($1,$3,3,'pending_confirmation','idp_rule','task completed',NULL,NOW()-INTERVAL '1 day'),
			($1,$4,3,'final','idp_rule','plan completed',$2,NOW()-INTERVAL '2 days'),
			($1,$5,4,'rejected','manager_decision','not confirmed',$2,NOW()-INTERVAL '1 day')`,
			args: []any{f.employeeID, f.managerID, skillGapGoID, skillGapSQLID, skillGapCommID}},
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}

	var templateID string
	if err := tx.QueryRow(ctx, `INSERT INTO idp_templates(creator_id,title,is_active) VALUES($1,'Go track',true) RETURNING id::text`, f.managerID).Scan(&templateID); err != nil {
		t.Fatal(err)
	}
	var archivedTemplateID string
	if err := tx.QueryRow(ctx, `INSERT INTO idp_templates(creator_id,title,is_active) VALUES($1,'Archived Go track',false) RETURNING id::text`, f.managerID).Scan(&archivedTemplateID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO template_competencies(template_id,competency_id,target_level) VALUES($1,$3,3),($1,$4,3),($2,$3,4)`,
		templateID, archivedTemplateID, skillGapGoID, skillGapSQLID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return templateID
}

func (f *fixture) resetSkillGap(t *testing.T) {
	t.Helper()
	f.reset(t)
	ctx := context.Background()
	if _, err := f.db.Exec(ctx, `DELETE FROM position_profiles WHERE id IN ($1,$2)`, skillGapProfileID, skillGapInactiveProfileID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `DELETE FROM competencies WHERE id IN ($1,$2,$3)`, skillGapGoID, skillGapSQLID, skillGapCommID); err != nil {
		t.Fatal(err)
	}
}
