package model

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateProjectRequest struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	RepoURL      string   `json:"repoUrl"`
	TemplateID   string   `json:"templateId"`
	GroupID      string   `json:"groupId"`
	Environments []string `json:"environments"`
}

type CreateTemplateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Format      string `json:"format"`
	Body        string `json:"body"`
}

type CreateConfigFileRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SourceType  string `json:"sourceType"`
	SourceID    string `json:"sourceId"`
}

type CreateConfigRequest struct {
	Environment  string `json:"environment"`
	ConfigFileID string `json:"configFileId"`
	Key          string `json:"key"`
	Value        string `json:"value"`
	ValueType    string `json:"valueType"`
	IsSensitive  bool   `json:"isSensitive"`
	ChangeReason string `json:"changeReason"`
}

type UpdateConfigRequest struct {
	ConfigFileID *string `json:"configFileId"`
	Key          *string `json:"key"`
	Value        *string `json:"value"`
	ValueType    *string `json:"valueType"`
	IsSensitive  *bool   `json:"isSensitive"`
	ChangeReason string  `json:"changeReason"`
}

type RollbackConfigRequest struct {
	VersionID    string `json:"versionId"`
	ChangeReason string `json:"changeReason"`
}

type RollbackConfigRevisionRequest struct {
	Environment  string `json:"environment"`
	RevisionID   string `json:"revisionId"`
	ChangeReason string `json:"changeReason"`
}

type ImportConfigRequest struct {
	Environment  string `json:"environment"`
	ConfigFileID string `json:"configFileId"`
	Format       string `json:"format"`
	Content      string `json:"content"`
	ChangeReason string `json:"changeReason"`
}

type CreateGroupRequest struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"memberIds"`
}

type GroupMemberRequest struct {
	UserID    string    `json:"userId"`
	GroupRole GroupRole `json:"groupRole"`
}

type ProjectMemberRequest struct {
	UserID      string      `json:"userId"`
	ProjectRole ProjectRole `json:"projectRole"`
}

type UpdateProjectMembersRequest struct {
	Members []ProjectMemberRequest `json:"members"`
}

type CreateSharedConfigRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Format      string              `json:"format"`
	Entries     []SharedConfigEntry `json:"entries"`
}

type UpdateSharedConfigRequest struct {
	Name         string              `json:"name"`
	Description  string              `json:"description"`
	Format       string              `json:"format"`
	Entries      []SharedConfigEntry `json:"entries"`
	ChangeReason string              `json:"changeReason"`
}

type SubmitSharedConfigUpdateRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Format      string              `json:"format"`
	Entries     []SharedConfigEntry `json:"entries"`
	Reason      string              `json:"reason"`
}

type CreateReviewRequest struct {
	ProjectID   string `json:"projectId"`
	Environment string `json:"environment"`
	ConfigKey   string `json:"configKey"`
	Reason      string `json:"reason"`
}

type ReviewDecisionRequest struct {
	Comment string `json:"comment"`
}

type ReviewFilters struct {
	Environment string
	ConfigKey   string
	Status      string
}
