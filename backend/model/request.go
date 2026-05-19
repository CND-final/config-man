package model

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateProjectRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	RepoURL       string   `json:"repoUrl"`
	OwnerName     string   `json:"ownerName"`
	DefaultFormat string   `json:"defaultFormat"`
	Environments  []string `json:"environments"`
}

type CreateConfigRequest struct {
	Environment  string `json:"environment"`
	Key          string `json:"key"`
	Value        string `json:"value"`
	ValueType    string `json:"valueType"`
	IsSensitive  bool   `json:"isSensitive"`
	ChangeReason string `json:"changeReason"`
}

type UpdateConfigRequest struct {
	Value        *string `json:"value"`
	ValueType    *string `json:"valueType"`
	IsSensitive  *bool   `json:"isSensitive"`
	ChangeReason string  `json:"changeReason"`
}

type ImportConfigRequest struct {
	Environment  string `json:"environment"`
	Format       string `json:"format"`
	Content      string `json:"content"`
	ChangeReason string `json:"changeReason"`
}

type ValidateProjectRequest struct {
	Environment  string            `json:"environment"`
	DraftEntries []ValidationEntry `json:"draftEntries"`
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
