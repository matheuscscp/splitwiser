package openaipkg

// CleanOpenAIJSONResponse cleans the JSON response from OpenAI API.
func CleanOpenAIJSONResponse(content string) string {
	// eliminate everything before the first '{'
	for i, c := range content {
		if c == '{' {
			content = content[i:]
			break
		}
	}

	// eliminate everything after the last '}'
	for i := len(content) - 1; i >= 0; i-- {
		if content[i] == '}' {
			content = content[:i+1]
			break
		}
	}

	return content
}
