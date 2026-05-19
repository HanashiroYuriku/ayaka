package response

import "strings"

func ParseErrorField(errStr string) map[string]string {
	errorMap := make(map[string]string)
	errors := strings.Split(errStr, ";")
	for _, e := range errors {
		parts := strings.SplitN(strings.TrimSpace(e), ":", 2)
		if len(parts) == 2 {
			errorMap[parts[0]] = strings.TrimSpace(parts[1])
		}
	}
	return errorMap
}
