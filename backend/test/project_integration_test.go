package test

import (
	"net/http"
	"testing"

	"config-man/backend/model"
)

// ============================================================================
// 測試：專案名稱重複 (HTTP 409 Conflict)
// ============================================================================
func TestIntegrationProjectCreateDuplicateName(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	prepareDatabase(t, db)

	handler := newIntegrationHandler(t, db)
	payload := map[string]any{
		"name":    "duplicate-test-project",
		"groupId": "platform-team",
	}

	res1 := request(t, handler, http.MethodPost, "/api/v1/projects", "alice", payload)
	if res1.Code != http.StatusCreated {
		t.Fatalf("first create should succeed, got %d", res1.Code)
	}

	res2 := request(t, handler, http.MethodPost, "/api/v1/projects", "alice", payload)
	if res2.Code != http.StatusConflict {
		t.Fatalf("duplicate create should return 409 Conflict, got %d body=%s", res2.Code, res2.Body.String())
	}
}

// ============================================================================
// 測試：必填欄位缺失與格式錯誤 (HTTP 400 Bad Request)
// ============================================================================
func TestIntegrationProjectCreateMissingFields(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	prepareDatabase(t, db)

	handler := newIntegrationHandler(t, db)

	reqNoGroup := map[string]any{
		"name": "valid-name-no-group",
	}
	res1 := request(t, handler, http.MethodPost, "/api/v1/projects", "alice", reqNoGroup)
	if res1.Code != http.StatusBadRequest {
		t.Fatalf("missing groupId should return 400 Bad Request, got %d", res1.Code)
	}

	reqShortName := map[string]any{
		"name":    "A",
		"groupId": "platform-team",
	}
	res2 := request(t, handler, http.MethodPost, "/api/v1/projects", "alice", reqShortName)
	if res2.Code != http.StatusBadRequest {
		t.Fatalf("short name should return 400 Bad Request, got %d", res2.Code)
	}
}

// ============================================================================
// 測試：無效的模板 ID (HTTP 404 Not Found)
// ============================================================================
func TestIntegrationProjectCreateInvalidTemplate(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	prepareDatabase(t, db)

	handler := newIntegrationHandler(t, db)

	reqInvalidTemplate := map[string]any{
		"name":       "invalid-template-project",
		"groupId":    "platform-team",
		"templateId": "non-existent-template-999",
	}
	res := request(t, handler, http.MethodPost, "/api/v1/projects", "alice", reqInvalidTemplate)
	if res.Code != http.StatusNotFound {
		t.Fatalf("invalid templateId should return 404 Not Found, got %d body=%s", res.Code, res.Body.String())
	}
}

// ============================================================================
// 測試：資料隔離與存取不存在的專案 (HTTP 404 Not Found)
// ============================================================================
func TestIntegrationProjectAccessNonExistent(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	prepareDatabase(t, db)

	handler := newIntegrationHandler(t, db)

	resGet := request(t, handler, http.MethodGet, "/api/v1/projects/fake-project-id-9999", "alice", nil)
	if resGet.Code != http.StatusNotFound {
		t.Fatalf("accessing non-existent project should be 404 Not Found, got %d body=%s", resGet.Code, resGet.Body.String())
	}
}

// ============================================================================
// 測試：更新成員的「自我毀滅」防護 (HTTP 400 Bad Request)
// ============================================================================
func TestIntegrationProjectMemberSelfDestructProtection(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	prepareDatabase(t, db)

	handler := newIntegrationHandler(t, db)

	resCreate := request(t, handler, http.MethodPost, "/api/v1/projects", "alice", map[string]any{
		"name":    "member-test-project",
		"groupId": "platform-team",
	})
	if resCreate.Code != http.StatusCreated {
		t.Fatalf("alice create project failed: %d", resCreate.Code)
	}
	created := decodeBody[model.Project](t, resCreate)

	updatePayload := map[string]any{
		"members": []map[string]any{
			{
				"userId":      "alice",
				"projectRole": string(model.RoleProjectViewer),
			},
		},
	}
	resUpdate := request(t, handler, http.MethodPut, "/api/v1/projects/"+created.ID+"/members", "alice", updatePayload)
	
	if resUpdate.Code != http.StatusBadRequest {
		t.Fatalf("removing the only project_admin should return 400, got %d body=%s", resUpdate.Code, resUpdate.Body.String())
	}
}

// ============================================================================
// 測試：更新成員給予無效的角色 (HTTP 400 Bad Request)
// ============================================================================
func TestIntegrationProjectMemberInvalidRole(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	prepareDatabase(t, db)

	handler := newIntegrationHandler(t, db)

	resCreate := request(t, handler, http.MethodPost, "/api/v1/projects", "alice", map[string]any{
		"name":    "invalid-role-project",
		"groupId": "platform-team",
	})
	created := decodeBody[model.Project](t, resCreate)

	updatePayload := map[string]any{
		"members": []map[string]any{
			{
				"userId":      "alice",
				"projectRole": "super_hacker_role", // 無效的角色名稱
			},
		},
	}
	resUpdate := request(t, handler, http.MethodPut, "/api/v1/projects/"+created.ID+"/members", "alice", updatePayload)
	
	if resUpdate.Code != http.StatusBadRequest {
		t.Fatalf("assigning an invalid role should return 400, got %d body=%s", resUpdate.Code, resUpdate.Body.String())
	}
}