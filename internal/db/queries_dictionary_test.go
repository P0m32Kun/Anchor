package db

import (
	"testing"
	"time"

	"github.com/P0m32Kun/Anchor/internal/models"
	"github.com/P0m32Kun/Anchor/internal/util"
)

func TestDictionary_CRUD(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	d := &models.Dictionary{
		ID: util.GenerateID(), Name: "top100.txt", Description: "top 100 paths",
		Category: models.DictionaryCategoryDirscan, FilePath: "/opt/dict/top100.txt",
		LineCount: 100, SizeBytes: 1024, Builtin: true, Enabled: true,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := q.CreateDictionary(d); err != nil {
		t.Fatalf("CreateDictionary: %v", err)
	}

	got, err := q.GetDictionary(d.ID)
	if err != nil {
		t.Fatalf("GetDictionary: %v", err)
	}
	if got == nil {
		t.Fatal("dictionary not found")
	}
	if got.Name != "top100.txt" {
		t.Errorf("name = %q, want top100.txt", got.Name)
	}
	if !got.Builtin {
		t.Error("expected builtin=true")
	}
	if !got.Enabled {
		t.Error("expected enabled=true")
	}

	// Update
	d.Name = "top200.txt"
	d.LineCount = 200
	d.UpdatedAt = time.Now().UTC()
	if err := q.UpdateDictionary(d); err != nil {
		t.Fatalf("UpdateDictionary: %v", err)
	}
	got2, err := q.GetDictionary(d.ID)
	if err != nil {
		t.Fatalf("GetDictionary after update: %v", err)
	}
	if got2.Name != "top200.txt" {
		t.Errorf("name = %q, want top200.txt", got2.Name)
	}

	// Delete
	if err := q.DeleteDictionary(d.ID); err != nil {
		t.Fatalf("DeleteDictionary: %v", err)
	}
	got3, err := q.GetDictionary(d.ID)
	if err != nil {
		t.Fatalf("GetDictionary after delete: %v", err)
	}
	if got3 != nil {
		t.Error("expected nil after delete")
	}
}

func TestListDictionaries(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	dicts := []*models.Dictionary{
		{ID: util.GenerateID(), Name: "a", Category: models.DictionaryCategoryDirscan, Builtin: true, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: util.GenerateID(), Name: "b", Category: models.DictionaryCategorySubdomain, Builtin: false, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: util.GenerateID(), Name: "c", Category: models.DictionaryCategoryDirscan, Builtin: false, Enabled: false, CreatedAt: now, UpdatedAt: now},
	}
	for _, d := range dicts {
		if err := q.CreateDictionary(d); err != nil {
			t.Fatalf("CreateDictionary %s: %v", d.Name, err)
		}
	}

	// All
	all, err := q.ListDictionaries("")
	if err != nil {
		t.Fatalf("ListDictionaries all: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("all len = %d, want 3", len(all))
	}

	// By category
	dirscan, err := q.ListDictionaries(string(models.DictionaryCategoryDirscan))
	if err != nil {
		t.Fatalf("ListDictionaries dirscan: %v", err)
	}
	if len(dirscan) != 2 {
		t.Errorf("dirscan len = %d, want 2", len(dirscan))
	}
}

func TestListEnabledDictionaries(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	dicts := []*models.Dictionary{
		{ID: util.GenerateID(), Name: "en1", Category: models.DictionaryCategoryDirscan, Builtin: true, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: util.GenerateID(), Name: "dis1", Category: models.DictionaryCategoryDirscan, Builtin: false, Enabled: false, CreatedAt: now, UpdatedAt: now},
	}
	for _, d := range dicts {
		if err := q.CreateDictionary(d); err != nil {
			t.Fatalf("CreateDictionary %s: %v", d.Name, err)
		}
	}

	enabled, err := q.ListEnabledDictionaries("")
	if err != nil {
		t.Fatalf("ListEnabledDictionaries: %v", err)
	}
	if len(enabled) != 1 {
		t.Errorf("enabled len = %d, want 1", len(enabled))
	}

	enabledByCategory, err := q.ListEnabledDictionaries(string(models.DictionaryCategoryDirscan))
	if err != nil {
		t.Fatalf("ListEnabledDictionaries by category: %v", err)
	}
	if len(enabledByCategory) != 1 {
		t.Errorf("enabled by category len = %d, want 1", len(enabledByCategory))
	}
}

func TestUpdateDictionaryEnabled(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	d := &models.Dictionary{
		ID: util.GenerateID(), Name: "builtin-dict", Category: models.DictionaryCategoryDirscan,
		Builtin: true, Enabled: true, CreatedAt: now, UpdatedAt: now,
	}
	if err := q.CreateDictionary(d); err != nil {
		t.Fatalf("CreateDictionary: %v", err)
	}

	if err := q.UpdateDictionaryEnabled(d.ID, false, time.Now().UTC()); err != nil {
		t.Fatalf("UpdateDictionaryEnabled: %v", err)
	}

	got, err := q.GetDictionary(d.ID)
	if err != nil {
		t.Fatalf("GetDictionary: %v", err)
	}
	if got.Enabled {
		t.Error("expected enabled=false after update")
	}
}

func TestListBuiltinDictionaries(t *testing.T) {
	q := New(openTestDB(t))
	now := time.Now().UTC()

	dicts := []*models.Dictionary{
		{ID: util.GenerateID(), Name: "bi1", Category: models.DictionaryCategoryDirscan, Builtin: true, Enabled: true, CreatedAt: now, UpdatedAt: now},
		{ID: util.GenerateID(), Name: "cu1", Category: models.DictionaryCategoryDirscan, Builtin: false, Enabled: true, CreatedAt: now, UpdatedAt: now},
	}
	for _, d := range dicts {
		if err := q.CreateDictionary(d); err != nil {
			t.Fatalf("CreateDictionary %s: %v", d.Name, err)
		}
	}

	builtins, err := q.ListBuiltinDictionaries()
	if err != nil {
		t.Fatalf("ListBuiltinDictionaries: %v", err)
	}
	if len(builtins) != 1 {
		t.Errorf("builtins len = %d, want 1", len(builtins))
	}
}
