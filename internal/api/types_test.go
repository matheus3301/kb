package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestDomainErrorRemainsStructuredWhenWrapped(t *testing.T) {
	original := &DomainError{Code: "REVISION_CONFLICT", Message: "the note changed", Retryable: true}
	err := fmt.Errorf("apply: %w", original)
	var domain *DomainError
	if !errors.As(err, &domain) || domain != original {
		t.Fatal("wrapped errors must preserve the domain error")
	}
	if original.Error() != "REVISION_CONFLICT: the note changed" {
		t.Fatalf("unexpected error text: %s", original.Error())
	}
	data, marshalErr := json.Marshal(domain)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	var value map[string]any
	if decodeErr := json.Unmarshal(data, &value); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	if value["code"] != "REVISION_CONFLICT" || value["retryable"] != true {
		t.Fatalf("domain fields lost in transport: %s", data)
	}
	if _, exists := value["details"]; exists {
		t.Fatal("absent details must be omitted")
	}
}
