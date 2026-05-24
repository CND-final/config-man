package processor

import (
	"fmt"
	"strings"

	appctx "config-man/backend/internal/context"
	"config-man/backend/internal/logger"
	"config-man/backend/model"
	"config-man/backend/pkg/util"
)

func (p *Processor) ListUsers(ctx appctx.RequestContext) ([]model.User, *model.ErrorDetail) {
	if ctx.Actor.Role != model.RoleSystemAdmin && ctx.Actor.Role != model.RoleUserGroupAdmin {
		return nil, model.Forbidden("Only system_admin or group_admin can list users")
	}
	return p.store.ListUsers(), nil
}

func (p *Processor) ListGroups(ctx appctx.RequestContext) ([]model.Group, *model.ErrorDetail) {
	groups := p.store.ListGroups()
	if ctx.Actor.Role == model.RoleSystemAdmin {
		logger.Processor.Info("groups listed")
		return groups, nil
	}

	visible := make([]model.Group, 0)
	for _, group := range groups {
		if util.GroupHasMember(group, ctx.Actor.ID) {
			visible = append(visible, group)
		}
	}
	logger.Processor.Info("groups listed")
	return visible, nil
}

func (p *Processor) GetGroup(ctx appctx.RequestContext, groupID string) (model.Group, *model.ErrorDetail) {
	group, err := p.requireGroup(groupID)
	if err != nil {
		return model.Group{}, err
	}
	if !util.CanReadGroup(ctx.Actor, group) {
		return model.Group{}, model.Forbidden("You cannot view this group")
	}
	return group, nil
}

func (p *Processor) CreateGroup(ctx appctx.RequestContext, name string, memberIDs []string) (model.Group, *model.ErrorDetail) {
	if ctx.Actor.Role != model.RoleSystemAdmin && ctx.Actor.Role != model.RoleUserGroupAdmin {
		return model.Group{}, model.Forbidden("Only system_admin or group_admin can create groups")
	}

	name = strings.TrimSpace(name)
	if len(name) < 2 {
		return model.Group{}, model.InvalidInput("group name is required")
	}
	if p.store.GroupNameExists(name) {
		return model.Group{}, model.Conflict(fmt.Sprintf("Group %q already exists", name))
	}

	members, err := p.membersFromIDs(memberIDs, model.RoleGroupMember)
	if err != nil {
		return model.Group{}, err
	}
	if ctx.Actor.Role == model.RoleUserGroupAdmin && !util.GroupMemberExists(members, ctx.Actor.ID) {
		members = append(members, model.GroupMember{User: ctx.Actor, GroupRole: model.RoleGroupAdmin})
	}

	group := model.Group{
		ID:          util.NewID("grp"),
		Name:        name,
		Members:     members,
		MemberCount: len(members),
	}

	if err := p.store.SaveGroup(group, newAudit(ctx.ActorName(), "group.create", "group", group.ID, "", map[string]any{
		"name":        group.Name,
		"memberCount": len(group.Members),
	})); err != nil {
		return model.Group{}, model.InternalError("database persistence failed: " + err.Error())
	}
	created, _ := p.store.FindGroup(group.ID)
	return created, nil
}

func (p *Processor) DeleteGroup(ctx appctx.RequestContext, groupID string) *model.ErrorDetail {
	group, err := p.requireGroup(groupID)
	if err != nil {
		return err
	}
	if ctx.Actor.Role != model.RoleSystemAdmin {
		return model.Forbidden("Only system_admin can delete groups")
	}
	if err := p.store.DeleteGroup(group.ID, newAudit(ctx.ActorName(), "group.delete", "group", group.ID, "", map[string]any{"name": group.Name})); err != nil {
		return model.InternalError("database persistence failed: " + err.Error())
	}
	return nil
}

func (p *Processor) AddGroupMember(ctx appctx.RequestContext, groupID, userID string, role model.GroupRole) (model.Group, *model.ErrorDetail) {
	group, err := p.requireGroup(groupID)
	if err != nil {
		return model.Group{}, err
	}
	if !util.CanManageGroup(ctx.Actor, group) {
		return model.Group{}, model.Forbidden("You cannot edit group members")
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return model.Group{}, model.InvalidInput("userId is required")
	}
	if _, ok := p.store.FindUserByID(userID); !ok {
		return model.Group{}, model.NotFound(fmt.Sprintf("User %q not found", userID))
	}
	if role == "" {
		role = model.RoleGroupMember
	}
	if !util.ValidGroupRole(role) {
		return model.Group{}, model.InvalidInput("groupRole must be group_admin or member")
	}
	if err := p.store.AddGroupMember(group.ID, userID, role, newAudit(ctx.ActorName(), "group_member.add", "group", group.ID, "", map[string]any{"userId": userID, "groupRole": role})); err != nil {
		return model.Group{}, model.InternalError("database persistence failed: " + err.Error())
	}
	updated, _ := p.store.FindGroup(group.ID)
	return updated, nil
}

func (p *Processor) RemoveGroupMember(ctx appctx.RequestContext, groupID, userID string) (model.Group, *model.ErrorDetail) {
	group, err := p.requireGroup(groupID)
	if err != nil {
		return model.Group{}, err
	}
	if !util.CanManageGroup(ctx.Actor, group) {
		return model.Group{}, model.Forbidden("You cannot edit group members")
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return model.Group{}, model.InvalidInput("userId is required")
	}
	if err := p.store.RemoveGroupMember(group.ID, userID, newAudit(ctx.ActorName(), "group_member.remove", "group", group.ID, "", map[string]any{"userId": userID})); err != nil {
		return model.Group{}, model.InternalError("database persistence failed: " + err.Error())
	}
	updated, _ := p.store.FindGroup(group.ID)
	return updated, nil
}

func (p *Processor) requireGroup(groupID string) (model.Group, *model.ErrorDetail) {
	groupID = strings.TrimSpace(groupID)
	group, ok := p.store.FindGroup(groupID)
	if !ok {
		return model.Group{}, model.NotFound(fmt.Sprintf("Group %q not found", groupID))
	}
	return group, nil
}

func (p *Processor) membersFromIDs(userIDs []string, role model.GroupRole) ([]model.GroupMember, *model.ErrorDetail) {
	seen := map[string]bool{}
	members := make([]model.GroupMember, 0, len(userIDs))
	for _, rawID := range userIDs {
		userID := strings.TrimSpace(rawID)
		if userID == "" || seen[userID] {
			continue
		}
		user, ok := p.store.FindUserByID(userID)
		if !ok {
			return nil, model.NotFound(fmt.Sprintf("User %q not found", userID))
		}
		seen[userID] = true
		members = append(members, model.GroupMember{User: user, GroupRole: role})
	}
	return members, nil
}
