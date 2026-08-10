package engine

import (
	"errors"
	"testing"
)

func TestApplyReturnsErrorWhenAResourceFails(t *testing.T) {
	var ran []string
	resources := []Resource{
		{ID: "ok", Kind: fakeKind{pending: true, desc: "ok", ran: &ran, id: "ok"}},
		{ID: "bad", Kind: fakeKind{pending: true, desc: "bad", ran: &ran, id: "bad", failErr: errors.New("boom")}},
	}

	err := Apply(resources)
	if err == nil {
		t.Fatal("expected an error when a resource fails")
	}
	if len(ran) != 1 || ran[0] != "ok" {
		t.Fatalf("expected the unrelated resource to still have executed, ran = %v", ran)
	}
}

func TestApplyNoErrorWhenEverythingSucceeds(t *testing.T) {
	var ran []string
	resources := []Resource{
		{ID: "a", Kind: fakeKind{pending: true, desc: "a", ran: &ran, id: "a"}},
		{ID: "b", Kind: fakeKind{pending: true, desc: "b", ran: &ran, id: "b"}},
	}

	if err := Apply(resources); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ran) != 2 {
		t.Fatalf("expected both resources to execute, ran = %v", ran)
	}
}
