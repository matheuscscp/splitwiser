package openaipkg

// CleanOpenAIJSONResponse cleans the JSON response from OpenAI API
// expecting it to be an array of objects.
// In particular, this function is good for eliminating the enclosing
// ```json{content}``` structure that OpenAI API returns, returning only
// {content}.
func CleanOpenAIJSONResponse(content string) string {
	// eliminate everything before the first '['
	for i, c := range content {
		if c == '[' {
			content = content[i:]
			break
		}
	}

	// eliminate everything after the last ']'
	for i := len(content) - 1; i >= 0; i-- {
		if content[i] == ']' {
			content = content[:i+1]
			break
		}
	}

	return content
}
