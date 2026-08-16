package client

import "fmt"

type NotFoundError struct {
	Kind  string
	Value string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("ClearML %s %q was not found", e.Kind, e.Value)
}

type MultipleMatchesError struct {
	Kind  string
	Value string
}

func (e *MultipleMatchesError) Error() string {
	return fmt.Sprintf("ClearML %s lookup for %q returned multiple matches", e.Kind, e.Value)
}

func exactlyOne[T any](kind, value string, values []T, id func(T) string) (T, error) {
	var zero T
	switch len(values) {
	case 0:
		return zero, &NotFoundError{Kind: kind, Value: value}
	case 1:
		if id(values[0]) == "" {
			return zero, invalidResponse(kind+" lookup", "id")
		}
		return values[0], nil
	default:
		return zero, &MultipleMatchesError{Kind: kind, Value: value}
	}
}

func invalidResponse(endpoint, field string) error {
	return fmt.Errorf("ClearML %s response did not contain %s", endpoint, field)
}
