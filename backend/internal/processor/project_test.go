package processor

import (
	"testing"

	appctx "config-man/backend/internal/context"
	"config-man/backend/model"
)

func TestListProjectsEmpty(t *testing.T) {
	proc := NewInMemory()
	ctx := appctx.RequestContext{
		Actor: model.User{ID: "alice", Name: "Alice", Role: model.RoleSystemAdmin},
	}

	projects := proc.ListProjects(ctx)

	if projects == nil {
		t.Fatal("projects should be a slice, not nil")
	}
}

func TestCreateProjectSuccess(t *testing.T) {
	proc := NewInMemory()
	ctx := appctx.RequestContext{
		Actor: model.User{ID: "alice", Name: "Alice", Role: model.RoleSystemAdmin},
	}

	project, err := proc.CreateProject(ctx, model.CreateProjectRequest{
		Name:      "my-service",
		OwnerName: "Platform Team",
	})

	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	if project.ID == "" {
		t.Fatal("project id should not be empty")
	}
	if project.Name != "my-service" {
		t.Fatalf("project name = %q, want my-service", project.Name)
	}
	if project.OwnerName != "Platform Team" {
		t.Fatalf("owner name = %q, want Platform Team", project.OwnerName)
	}
	if len(project.Environments) != 3 {
		t.Fatalf("environment count = %d, want 3 (default: dev, staging, prod)", len(project.Environments))
	}
}

func TestCreateProjectCustomEnvironments(t *testing.T) {
	proc := NewInMemory()
	ctx := appctx.RequestContext{
		Actor: model.User{ID: "alice", Name: "Alice", Role: model.RoleSystemAdmin},
	}

	project, err := proc.CreateProject(ctx, model.CreateProjectRequest{
		Name:         "my-service",
		OwnerName:    "Platform Team",
		Environments: []string{"local", "test", "production"},
	})

	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	if len(project.Environments) != 3 {
		t.Fatalf("environment count = %d, want 3", len(project.Environments))
	}
	if project.Environments[0].Name != "local" {
		t.Fatalf("first env = %q, want local", project.Environments[0].Name)
	}
	if project.Environments[2].Name != "production" {
		t.Fatalf("last env = %q, want production", project.Environments[2].Name)
	}
}

func TestCreateProjectInsufficientPermission(t *testing.T) {
	proc := NewInMemory()
	ctx := appctx.RequestContext{
		Actor: model.User{ID: "nora", Name: "Nora", Role: model.RoleDeveloper},
	}

	_, err := proc.CreateProject(ctx, model.CreateProjectRequest{
		Name:      "my-service",
		OwnerName: "Platform Team",
	})

	if err == nil {
		t.Fatal("expected create project to fail for developer")
	}
	if err.Kind != model.ErrorForbidden {
		t.Fatalf("error kind = %s, want forbidden", err.Kind)
	}
}

func TestCreateProjectMissingName(t *testing.T) {
	proc := NewInMemory()
	ctx := appctx.RequestContext{
		Actor: model.User{ID: "alice", Name: "Alice", Role: model.RoleSystemAdmin},
	}

	_, err := proc.CreateProject(ctx, model.CreateProjectRequest{
		Name:      "",
		OwnerName: "Platform Team",
	})

	if err == nil {
		t.Fatal("expected create project to fail with empty name")
	}
	if err.Kind != model.ErrorInvalidInput {
		t.Fatalf("error kind = %s, want invalid_input", err.Kind)
	}
}

func TestCreateProjectMissingOwner(t *testing.T) {
	proc := NewInMemory()
	ctx := appctx.RequestContext{
		Actor: model.User{ID: "alice", Name: "Alice", Role: model.RoleSystemAdmin},
	}

	_, err := proc.CreateProject(ctx, model.CreateProjectRequest{
		Name:      "my-service",
		OwnerName: "",
	})

	if err == nil {
		t.Fatal("expected create project to fail with empty owner")
	}
	if err.Kind != model.ErrorInvalidInput {
		t.Fatalf("error kind = %s, want invalid_input", err.Kind)
	}
}

func TestCreateProjectWhitespaceNameTrimmed(t *testing.T) {
	proc := NewInMemory()
	ctx := appctx.RequestContext{
		Actor: model.User{ID: "alice", Name: "Alice", Role: model.RoleSystemAdmin},
	}

	project, err := proc.CreateProject(ctx, model.CreateProjectRequest{
		Name:      "  my-service  ",
		OwnerName: "  Platform Team  ",
	})

	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	if project.Name != "my-service" {
		t.Fatalf("name should be trimmed: %q", project.Name)
	}
	if project.OwnerName != "Platform Team" {
		t.Fatalf("owner should be trimmed: %q", project.OwnerName)
	}
}

func TestCreateProjectDuplicateName(t *testing.T) {
	proc := NewInMemory()
	ctx := appctx.RequestContext{
		Actor: model.User{ID: "alice", Name: "Alice", Role: model.RoleSystemAdmin},
	}

	_, err := proc.CreateProject(ctx, model.CreateProjectRequest{
		Name:      "my-service",
		OwnerName: "Platform Team",
	})
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	_, err = proc.CreateProject(ctx, model.CreateProjectRequest{
		Name:      "my-service",
		OwnerName: "Another Team",
	})

	if err == nil {
		t.Fatal("expected create project to fail with duplicate name")
	}
	if err.Kind != model.ErrorConflict {
		t.Fatalf("error kind = %s, want conflict", err.Kind)
	}
}

func TestCreateProjectUnsupportedFormat(t *testing.T) {
	proc := NewInMemory()
	ctx := appctx.RequestContext{
		Actor: model.User{ID: "alice", Name: "Alice", Role: model.RoleSystemAdmin},
	}

	_, err := proc.CreateProject(ctx, model.CreateProjectRequest{
		Name:          "my-service",
		OwnerName:     "Platform Team",
		DefaultFormat: "toml",
	})

	if err == nil {
		t.Fatal("expected create project to fail with unsupported format")
	}
	if err.Kind != model.ErrorInvalidInput {
		t.Fatalf("error kind = %s, want invalid_input", err.Kind)
	}
}

func TestCreateProjectSupportedFormats(t *testing.T) {
	formats := []string{"json", "yaml", "properties"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			proc := NewInMemory()
			ctx := appctx.RequestContext{
				Actor: model.User{ID: "alice", Name: "Alice", Role: model.RoleSystemAdmin},
			}

			project, err := proc.CreateProject(ctx, model.CreateProjectRequest{
				Name:          "svc-" + format,
				OwnerName:     "Team",
				DefaultFormat: format,
			})

			if err != nil {
				t.Fatalf("create with format %s failed: %v", format, err)
			}
			if project.DefaultFormat != format {
				t.Fatalf("default format = %s, want %s", project.DefaultFormat, format)
			}
		})
	}
}

func TestCreateProjectDefaultFormatYAML(t *testing.T) {
	proc := NewInMemory()
	ctx := appctx.RequestContext{
		Actor: model.User{ID: "alice", Name: "Alice", Role: model.RoleSystemAdmin},
	}

	project, err := proc.CreateProject(ctx, model.CreateProjectRequest{
		Name:      "my-service",
		OwnerName: "Platform Team",
	})

	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	if project.DefaultFormat != "yaml" {
		t.Fatalf("default format = %s, want yaml", project.DefaultFormat)
	}
}

func TestCreateProjectProjectAdminPermission(t *testing.T) {
	proc := NewInMemory()
	ctx := appctx.RequestContext{
		Actor: model.User{ID: "bob", Name: "Bob", Role: model.RoleProjectAdmin},
	}

	project, err := proc.CreateProject(ctx, model.CreateProjectRequest{
		Name:      "my-service",
		OwnerName: "Platform Team",
	})

	if err != nil {
		t.Fatalf("create project failed: %v", err)
	}
	if project.ID == "" {
		t.Fatal("project id should not be empty")
	}
}
