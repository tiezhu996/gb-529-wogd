package model

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestBalanceRunEstimatedBOGColumnName(t *testing.T) {
	parsed, err := schema.Parse(&BalanceRun{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse BalanceRun schema: %v", err)
	}
	field := parsed.LookUpField("EstimatedBOGKG")
	if field == nil {
		t.Fatal("EstimatedBOGKG field is missing from the GORM schema")
	}
	if field.DBName != "estimated_bog_kg" {
		t.Fatalf("EstimatedBOGKG column = %q, want estimated_bog_kg", field.DBName)
	}
}
