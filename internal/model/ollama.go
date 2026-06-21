package model

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Think  bool   `json:"think"`
	// Format constrains the output, e.g. "json" for structured extraction.
	Format string `json:"format,omitempty"`
}

type GenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type TagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}
