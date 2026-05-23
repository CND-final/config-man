package app

import (
	"net/http"

	"config-man/backend/internal/response"
	"config-man/backend/model"

	"github.com/gin-gonic/gin"
)

func (s *Server) getGroupsRoutes() Routes {
	return Routes{
		{
			Name:    "ListGroups",
			Method:  http.MethodGet,
			Pattern: "/groups",
			APIFunc: s.handleListGroups,
		},
		{
			Name:    "GetGroup",
			Method:  http.MethodGet,
			Pattern: "/groups/:groupId",
			APIFunc: s.handleGetGroup,
		},
		{
			Name:    "CreateGroup",
			Method:  http.MethodPost,
			Pattern: "/groups",
			APIFunc: s.handleCreateGroup,
		},
		{
			Name:    "DeleteGroup",
			Method:  http.MethodDelete,
			Pattern: "/groups/:groupId",
			APIFunc: s.handleDeleteGroup,
		},
		{
			Name:    "AddGroupMember",
			Method:  http.MethodPost,
			Pattern: "/groups/:groupId/members",
			APIFunc: s.handleAddGroupMember,
		},
		{
			Name:    "RemoveGroupMember",
			Method:  http.MethodDelete,
			Pattern: "/groups/:groupId/members/:userId",
			APIFunc: s.handleDeleteGroupMember,
		},
	}
}

func (s *Server) handleListGroups(c *gin.Context) {
	groups, err := s.processor.ListGroups(requestContextFromGin(c))
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteJSON(c, http.StatusOK, gin.H{"groups": groups})
}

func (s *Server) handleGetGroup(c *gin.Context) {
	group, err := s.processor.GetGroup(requestContextFromGin(c), c.Param("groupId"))
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteJSON(c, http.StatusOK, gin.H{"group": group})
}

func (s *Server) handleCreateGroup(c *gin.Context) {
	var req model.CreateGroupRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}
	group, err := s.processor.CreateGroup(requestContextFromGin(c), req.Name, req.MemberIDs)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteJSON(c, http.StatusCreated, group)
}

func (s *Server) handleDeleteGroup(c *gin.Context) {
	if err := s.processor.DeleteGroup(requestContextFromGin(c), c.Param("groupId")); err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteJSON(c, http.StatusOK, gin.H{"deleted": true})
}

func (s *Server) handleAddGroupMember(c *gin.Context) {
	var req model.GroupMemberRequest
	if err := response.DecodeJSON(c, &req); err != nil {
		response.WriteError(c, err)
		return
	}
	group, err := s.processor.AddGroupMember(requestContextFromGin(c), c.Param("groupId"), req.UserID, req.GroupRole)
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteJSON(c, http.StatusOK, group)
}

func (s *Server) handleDeleteGroupMember(c *gin.Context) {
	group, err := s.processor.RemoveGroupMember(requestContextFromGin(c), c.Param("groupId"), c.Param("userId"))
	if err != nil {
		response.WriteError(c, err)
		return
	}
	response.WriteJSON(c, http.StatusOK, group)
}
