package environmenttext

import "strings"

type Variable struct {
	Name    string
	Default string
}

func TemplateOf(variables []Variable) string {
	var template strings.Builder
	for _, variable := range variables {
		template.WriteString(variable.Name + "=" + variable.Default + "\n")
	}
	return template.String()
}
