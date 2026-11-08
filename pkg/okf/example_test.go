package okf_test

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/matheus3301/kb/pkg/okf"
)

func ExampleParse() {
	document, err := okf.Parse([]byte("---\ntype: Reference\ncustom: 1234567890.123456789\n---\n# Example\n"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(document.Metadata()["custom"])
	fmt.Println(len(document.Validate()))
	fmt.Print(document.Body())
	// Output:
	// 1234567890.123456789
	// 0
	// # Example
}

func ExampleDocument_Edit() {
	document, err := okf.Parse([]byte("---\ntype: 'Reference' # Keep\ntitle: Original\n---\nBody\n"))
	if err != nil {
		log.Fatal(err)
	}
	out, err := document.Edit(okf.Edit{Set: map[string]any{"title": "Revised"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(string(out))
	// Output:
	// ---
	// type: 'Reference' # Keep
	// title: Revised
	// ---
	// Body
}

func ExampleMarshal() {
	raw, err := okf.Marshal(map[string]any{"type": "Metric", "value": json.Number("0.1234567890123456789")}, "Synthetic example.\n")
	if err != nil {
		log.Fatal(err)
	}
	document, err := okf.Parse(raw)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(document.Metadata()["value"])
	// Output: 0.1234567890123456789
}

func ExampleValidateReserved() {
	findings := okf.ValidateReserved(okf.RootIndex, []byte("---\nokf_version: \"0.2\"\n---\n# Topics\n"))
	fmt.Println(len(findings))
	// Output: 0
}
