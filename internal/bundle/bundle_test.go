package bundle_test

import (
	"testing"

	"github.com/a-kaibu/enbu-poc/internal/bundle"
)

func TestMarshalUnmarshal(t *testing.T) {
	secrets := map[string]string{
		"DB_URL":  "postgres://localhost/dev",
		"API_KEY": "sk-1234",
	}

	data := bundle.Marshal(secrets)
	got, err := bundle.Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got["DB_URL"] != secrets["DB_URL"] || got["API_KEY"] != secrets["API_KEY"] {
		t.Fatalf("round-trip mismatch: got %v", got)
	}
}

func TestToDotEnv(t *testing.T) {
	secrets := map[string]string{
		"B_KEY": "val2",
		"A_KEY": "val1",
	}

	result := string(bundle.ToDotEnv(secrets))
	expected := "A_KEY=val1\nB_KEY=val2\n"

	if result != expected {
		t.Fatalf("got %q, want %q", result, expected)
	}
}
