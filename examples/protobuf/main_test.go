package main

import (
	"errors"
	"io"
	"testing"

	"github.com/shiwano/errdef"
	"github.com/shiwano/errdef/resolver"
	"github.com/shiwano/errdef/unmarshaler"
	"google.golang.org/protobuf/proto"
)

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	orig := ErrNotFound.WithOptions(
		UserID("u123"),
		errdef.Details{"info": "additional info"},
	).Wrapf(io.EOF, "user not found")

	protoBytes, err := marshalProto(orig.(errdef.Error))
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var protoMsg Error
	if err := proto.Unmarshal(protoBytes, &protoMsg); err != nil {
		t.Fatalf("failed to unmarshal proto: %v", err)
	}

	r := resolver.New(ErrNotFound)
	u := unmarshaler.New(r, protoDecoder,
		unmarshaler.WithBuiltinFields(),
		unmarshaler.WithStandardSentinelErrors(),
	)

	restored, err := u.Unmarshal(&protoMsg)
	if err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if restored.Error() != orig.Error() {
		t.Errorf("message mismatch: got %q, want %q", restored.Error(), orig.Error())
	}

	origErr := orig.(errdef.Error)
	restoredErr := restored.(errdef.Error)
	if restoredErr.Kind() != origErr.Kind() {
		t.Errorf("kind mismatch: got %q, want %q", restoredErr.Kind(), origErr.Kind())
	}

	if uid := UserIDFrom.OrZero(restored); uid != "u123" {
		t.Errorf("user_id mismatch: got %q, want %q", uid, "u123")
	}

	details := errdef.DetailsFrom.OrZero(restored)
	if details["info"] != "additional info" {
		t.Errorf("details mismatch: got %v, want %q", details["info"], "additional info")
	}

	if !errors.Is(restored, io.EOF) {
		t.Error("restored error should wrap io.EOF")
	}
}
