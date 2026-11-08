package optional

import (
	"encoding/json"
	"testing"
)

func TestStringJSONRoundTripDoesNotRetainLexicalQuotes(t *testing.T) {
	want := `Living "Room" Speaker`
	var got Optional[string]
	if err := json.Unmarshal([]byte(`"Living \"Room\" Speaker"`), &got); err != nil {
		t.Fatal(err)
	}
	if got.Value() != want {
		t.Fatalf("got %q, want %q", got.Value(), want)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `"Living \"Room\" Speaker"` {
		t.Fatalf("encoded %s", encoded)
	}
}

func TestPrimitiveJSONRoundTrip(t *testing.T) {
	var got Optional[bool]
	if err := json.Unmarshal([]byte(`true`), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Value() {
		t.Fatal("decoded false, want true")
	}
}

func TestJSONNullIsRejectedWithoutMutation(t *testing.T) {
	got := New(true)
	if err := json.Unmarshal([]byte(`null`), &got); err == nil {
		t.Fatal("expected null to be rejected")
	}
	value, present := got.Get()
	if !present || !value {
		t.Fatalf("optional mutated after null: value=%v present=%v", value, present)
	}
}
