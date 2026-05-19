package model

type ValidationEntry struct {
	Environment string `json:"environment"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	ValueType   string `json:"valueType"`
	IsSensitive bool   `json:"isSensitive"`
}

type ValidationIssue struct {
	Environment string `json:"environment"`
	Key         string `json:"key"`
	Code        string `json:"code"`
	Message     string `json:"message"`
}

type ValidationResult struct {
	ProjectID    string            `json:"projectId"`
	Environments []string          `json:"environments"`
	Valid        bool              `json:"valid"`
	Errors       []ValidationIssue `json:"errors"`
	Warnings     []ValidationIssue `json:"warnings"`
}
