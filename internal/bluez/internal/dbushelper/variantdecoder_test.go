//go:build linux

package dbushelper

import (
	"testing"

	"github.com/bluetuith-org/bluetooth-classic/api/optional"
	"github.com/godbus/dbus/v5"
)

func TestDecodeVariantMapPreservesStringContentWithoutLexicalQuotes(t *testing.T) {
	type properties struct {
		Name optional.Optional[string] `codec:"Name,omitempty"`
	}

	want := `Living "Room" Speaker`
	var got properties
	if err := DecodeVariantMap(
		map[string]dbus.Variant{"Name": dbus.MakeVariant(want)},
		&got,
		"Name",
	); err != nil {
		t.Fatal(err)
	}
	if got.Name.Value() != want {
		t.Fatalf("got %q, want %q", got.Name.Value(), want)
	}
}
