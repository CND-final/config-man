package test

import (
    "net/http"
    "testing"

    "config-man/backend/model"
)

// Data-driven integration tests covering several edge cases for project APIs.
// These reuse the helpers in integration_test.go (openTestDB, prepareDatabase,
// newIntegrationHandler, request, decodeBody).
func TestDataDrivenProjectEdgeCases(t *testing.T) {
    cases := []struct {
        name     string
        setup    func(t *testing.T, handler http.Handler) // optional precondition
        method   string
        path     string
        token    string
        body     any
        wantCode int
    }{
        {
            name:     "create project success",
            method:   http.MethodPost,
            path:     "/api/v1/projects",
            token:    "alice",
            body:     map[string]any{"name": "dd-success-project", "groupId": "platform-team"},
            wantCode: http.StatusCreated,
        },
        {
            name: "create duplicate name",
            setup: func(t *testing.T, handler http.Handler) {
                // create initial project
                res := request(t, handler, http.MethodPost, "/api/v1/projects", "alice", map[string]any{
                    "name":    "dd-dup-project",
                    "groupId": "platform-team",
                })
                if res.Code != http.StatusCreated {
                    t.Fatalf("setup create failed: %d body=%s", res.Code, res.Body.String())
                }
            },
            method:   http.MethodPost,
            path:     "/api/v1/projects",
            token:    "alice",
            body:     map[string]any{"name": "dd-dup-project", "groupId": "platform-team"},
            wantCode: http.StatusConflict,
        },
        {
            name:     "create missing group",
            method:   http.MethodPost,
            path:     "/api/v1/projects",
            token:    "alice",
            body:     map[string]any{"name": "no-group-project"},
            wantCode: http.StatusBadRequest,
        },
        {
            name:     "create short name",
            method:   http.MethodPost,
            path:     "/api/v1/projects",
            token:    "alice",
            body:     map[string]any{"name": "A", "groupId": "platform-team"},
            wantCode: http.StatusBadRequest,
        },
        {
            name:     "create invalid template",
            method:   http.MethodPost,
            path:     "/api/v1/projects",
            token:    "alice",
            body:     map[string]any{"name": "invalid-template-project", "groupId": "platform-team", "templateId": "no-such"},
            wantCode: http.StatusNotFound,
        },
        {
            name:     "get non-existent project",
            method:   http.MethodGet,
            path:     "/api/v1/projects/fake-project-id-xyz",
            token:    "alice",
            body:     nil,
            wantCode: http.StatusNotFound,
        },
        {
            name: "update members remove only admin",
            setup: func(t *testing.T, handler http.Handler) {
                // create project with alice as admin (default)
                res := request(t, handler, http.MethodPost, "/api/v1/projects", "alice", map[string]any{
                    "name":    "dd-member-project",
                    "groupId": "platform-team",
                })
                if res.Code != http.StatusCreated {
                    t.Fatalf("setup create failed: %d body=%s", res.Code, res.Body.String())
                }
            },
            method:   http.MethodPut,
            path:     "/api/v1/projects/dd-member-project/members", // will replace in test with actual ID
            token:    "alice",
            body:     map[string]any{"members": []map[string]any{{"userId": "alice", "projectRole": string(model.RoleProjectViewer)}}},
            wantCode: http.StatusBadRequest,
        },
        {
            name: "update members invalid role",
            setup: func(t *testing.T, handler http.Handler) {
                res := request(t, handler, http.MethodPost, "/api/v1/projects", "alice", map[string]any{
                    "name":    "dd-invalid-role-project",
                    "groupId": "platform-team",
                })
                if res.Code != http.StatusCreated {
                    t.Fatalf("setup create failed: %d body=%s", res.Code, res.Body.String())
                }
            },
            method:   http.MethodPut,
            path:     "/api/v1/projects/dd-invalid-role-project/members",
            token:    "alice",
            body:     map[string]any{"members": []map[string]any{{"userId": "alice", "projectRole": "super_hacker_role"}}},
            wantCode: http.StatusBadRequest,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            db := openTestDB(t)
            defer db.Close()
            prepareDatabase(t, db)

            handler := newIntegrationHandler(t, db)

            // run setup if provided
            if tc.setup != nil {
                tc.setup(t, handler)
            }

            // special handling: some setups create projects and we need the real ID
            path := tc.path
            // if path uses placeholder project name, resolve it by listing projects
            if pathContainsPlaceholder(path) {
                // find created project with the name indicated in path
                res := request(t, handler, http.MethodGet, "/api/v1/projects", "alice", nil)
                if res.Code != http.StatusOK {
                    t.Fatalf("list projects failed: %d body=%s", res.Code, res.Body.String())
                }
                projects := decodeBody[[]model.Project](t, res)
                // replace placeholder with found ID by name
                for _, p := range projects {
                    if placeholderMatchesPath(p.Name, path) {
                        // build members path
                        path = "/api/v1/projects/" + p.ID + "/members"
                        break
                    }
                }
            }

            res := request(t, handler, tc.method, path, tc.token, tc.body)
            if res.Code != tc.wantCode {
                t.Fatalf("case %q: status = %d body=%s, want %d", tc.name, res.Code, res.Body.String(), tc.wantCode)
            }
        })
    }
}

// helper: detect placeholder usage in path
func pathContainsPlaceholder(path string) bool {
    return path == "/api/v1/projects/dd-member-project/members" || path == "/api/v1/projects/dd-invalid-role-project/members"
}

// helper: check whether project name corresponds to path placeholder
func placeholderMatchesPath(name, path string) bool {
    if path == "/api/v1/projects/dd-member-project/members" && name == "dd-member-project" {
        return true
    }
    if path == "/api/v1/projects/dd-invalid-role-project/members" && name == "dd-invalid-role-project" {
        return true
    }
    return false
}
