package model

type TemplateEntry struct {
	Key          string `json:"key"`
	DefaultValue string `json:"defaultValue"`
	ValueType    string `json:"valueType"`
	Required     bool   `json:"required"`
	IsSensitive  bool   `json:"isSensitive"`
	Description  string `json:"description"`
}

type Template struct {
	Name    string          `json:"name"`
	Entries []TemplateEntry `json:"entries"`
}

func BaseTemplate() Template {
	return Template{
		Name: "Group Base Template",
		Entries: []TemplateEntry{
			{
				Key:          "app.timezone",
				DefaultValue: "Asia/Taipei",
				ValueType:    "string",
				Required:     true,
				Description:  "Default application timezone",
			},
			{
				Key:          "log.level",
				DefaultValue: "info",
				ValueType:    "string",
				Required:     true,
				Description:  "Application logging level",
			},
			{
				Key:          "api.baseUrl",
				DefaultValue: "https://api.example.com",
				ValueType:    "string",
				Required:     true,
				Description:  "Backend API base URL",
			},
			{
				Key:          "feature.newCheckoutEnabled",
				DefaultValue: "false",
				ValueType:    "boolean",
				Required:     false,
				Description:  "Feature flag for staged rollout",
			},
			{
				Key:          "database.url",
				DefaultValue: "",
				ValueType:    "string",
				Required:     true,
				IsSensitive:  true,
				Description:  "Database connection string",
			},
		},
	}
}
