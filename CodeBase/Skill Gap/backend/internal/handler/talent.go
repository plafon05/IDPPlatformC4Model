package handler

import (
	"errors"
	"net/http"
	"strconv"

	"idp-platform/backend/internal/httpjson"
	"idp-platform/backend/internal/talent"
	"idp-platform/backend/internal/talent/skillgap"
)

type talentHandler struct {
	service *talent.Service
	users   usersHandler
}

type skillGapResponse struct {
	ProfileID         string                    `json:"profile_id"`
	ProfileVersion    int                       `json:"profile_version"`
	HasSufficientData bool                      `json:"has_sufficient_data"`
	Items             []skillGapItemResponse    `json:"items"`
	AssessFirst       []string                  `json:"assess_first"`
	RecommendedPlans  []recommendedPlanResponse `json:"recommended_plans"`
}

type skillGapItemResponse struct {
	CompetencyID   string `json:"competency_id"`
	RequiredLevel  int    `json:"required_level"`
	Status         string `json:"status"`
	EffectiveLevel *int   `json:"effective_level"`
	PendingLevel   *int   `json:"pending_level"`
	Delta          int    `json:"delta"`
}

type recommendedPlanResponse struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Covers []string `json:"covers"`
	Score  int      `json:"score"`
}

func (h talentHandler) skillGap(w http.ResponseWriter, r *http.Request) {
	claims, ok := accessClaimsFromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid access token")
		return
	}
	employeeID := r.PathValue("id")
	if !isUUID(employeeID) {
		httpjson.WriteError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}
	allowed, err := h.users.canReadUserIDPs(r, claims.UserID, claims.Roles, employeeID)
	if err != nil {
		httpjson.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		return
	}
	if !allowed {
		httpjson.WriteError(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions")
		return
	}

	query := r.URL.Query()
	profileID := query.Get("profile_id")
	if !isUUID(profileID) {
		httpjson.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "profile_id must be a UUID")
		return
	}
	version := 0
	if query.Has("version") {
		parsed, err := strconv.ParseInt(query.Get("version"), 10, 32)
		if err != nil || parsed < 0 {
			httpjson.WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "version must be a non-negative integer")
			return
		}
		version = int(parsed)
	}

	result, err := h.service.SkillGap(r.Context(), employeeID, profileID, version)
	if err != nil {
		writeTalentError(w, err)
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, newSkillGapResponse(result))
}

func newSkillGapResponse(result *talent.SkillGapResult) skillGapResponse {
	report := result.Report
	response := skillGapResponse{
		ProfileID:         report.ProfileID,
		ProfileVersion:    report.ProfileVersion,
		HasSufficientData: report.HasSufficientData(),
		Items:             make([]skillGapItemResponse, 0, len(report.Items)),
		AssessFirst:       append([]string{}, result.Recommendations.AssessFirst...),
		RecommendedPlans:  make([]recommendedPlanResponse, 0, len(result.Recommendations.Plans)),
	}
	for _, item := range report.Items {
		response.Items = append(response.Items, skillGapItemResponse{
			CompetencyID:   item.CompetencyID,
			RequiredLevel:  item.RequiredLevel,
			Status:         gapStatusName(item.Status),
			EffectiveLevel: item.Effective,
			PendingLevel:   item.Pending,
			Delta:          item.Delta,
		})
	}
	for _, plan := range result.Recommendations.Plans {
		response.RecommendedPlans = append(response.RecommendedPlans, recommendedPlanResponse{
			ID:     plan.Plan.ID,
			Title:  plan.Plan.Title,
			Covers: append([]string{}, plan.Covers...),
			Score:  plan.Score,
		})
	}
	return response
}

func gapStatusName(status skillgap.GapStatus) string {
	switch status {
	case skillgap.GapMet:
		return "met"
	case skillgap.GapDeficit:
		return "deficit"
	default:
		return "not_assessed"
	}
}

func writeTalentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, talent.ErrProfileNotFound):
		httpjson.WriteError(w, http.StatusNotFound, "PROFILE_NOT_FOUND", "Position profile not found")
	case errors.Is(err, talent.ErrEmployeeNotFound):
		httpjson.WriteError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
	default:
		httpjson.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	}
}

// isUUID reports whether value is a canonical textual UUID, so that malformed
// identifiers are rejected before they reach PostgreSQL.
func isUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, c := range value {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F') {
				return false
			}
		}
	}
	return true
}
