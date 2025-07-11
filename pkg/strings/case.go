package strings

import "github.com/iancoleman/strcase"

func ToKebabCase(s string) string {
	return strcase.ToKebab(s)
}

func ToSnakeCase(s string) string {
	return strcase.ToSnake(s)
}

func ToScreamingSnakeCase(s string) string {
	return strcase.ToScreamingSnake(s)
}

func ToCamelCase(s string) string {
	return strcase.ToCamel(s)
}

func ToLowerCamelCase(s string) string {
	return strcase.ToLowerCamel(s)
}
