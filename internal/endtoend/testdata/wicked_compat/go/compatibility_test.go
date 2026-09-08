package compat

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// Check type identity, not only the spelling "uuid.UUID" in generated source.
var (
	_ uuid.UUID  = Record{}.ExternalID
	_ *uuid.UUID = Record{}.OptionalUuid
	_            = CreateRecordRow{ID: 42, Name: "example"}
	_            = CreateRecordParams{ExternalID: uuid.UUID{}, OptionalUuid: new(uuid.UUID)}
)

func TestLegacyJSONPayload(t *testing.T) {
	const id = "e9114100-51ea-4e6c-8f88-189f4f365dfe"
	u := uuid.MustParse(id)
	record := Record{ID: 42, ExternalID: u, OptionalUuid: &u, Name: "example", Data: json.RawMessage(`{"n":1}`)}
	const legacy = `{"id":42,"external_id":"` + id + `","optional_uuid":"` + id + `","display_name":"example","data":{"n":1}}`
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != legacy {
		t.Fatalf("JSON contract changed: %s", encoded)
	}
	var decoded Record
	if err := json.Unmarshal([]byte(legacy), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ID != 42 || decoded.ExternalID != u || decoded.OptionalUuid == nil || *decoded.OptionalUuid != u || decoded.Name != "example" || string(decoded.Data) != `{"n":1}` {
		t.Fatalf("old cache payload did not round-trip: %+v", decoded)
	}
}
