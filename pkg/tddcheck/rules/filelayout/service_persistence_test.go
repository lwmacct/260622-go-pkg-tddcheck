package filelayout

import (
	"path/filepath"
	"testing"
)

func TestViolationsChecksServicePersistenceImports(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.service.go": `package service

import (
	"database/sql"
	"github.com/uptrace/bun"
	"gorm.io/gorm"
)

type DeviceService struct {
	db *bun.DB
	raw *sql.DB
	gorm *gorm.DB
}

func NewDeviceService(db *bun.DB) *DeviceService { return &DeviceService{db: db} }
func (s *DeviceService) Get() error { return nil }
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "service files must not import persistence package database/sql")
	assertViolationContains(t, violations, "service files must not import persistence package github.com/uptrace/bun")
	assertViolationContains(t, violations, "service files must not import persistence package gorm.io/gorm")
	assertViolationContains(t, violations, "service files must not depend on persistence handle types")
}

func TestViolationsChecksServicePersistenceCalls(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.service.go": `package service

import "example.com/app/internal/repository"

type DeviceService struct {
	db repository.Store
}

func NewDeviceService(db repository.Store) *DeviceService { return &DeviceService{db: db} }
func (s *DeviceService) Get() error {
	_, err := s.db.NewSelect().Exec(nil)
	return err
}
func (s *DeviceService) Create() error {
	_ = repository.DeviceModel{}
	return nil
}
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "service files must not call persistence APIs directly")
	assertViolationContains(t, violations, "service files must not reference repository schema models")
}

func TestViolationsAllowsServiceRepositoryStore(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.service.go": `package service

import (
	"context"
	"example.com/app/internal/repository"
)

type DeviceService struct {
	store *repository.Store
}

func NewDeviceService(store *repository.Store) *DeviceService { return &DeviceService{store: store} }
func (s *DeviceService) Get(ctx context.Context) error {
	_, err := s.store.FetchDevice(ctx, "device-1")
	return err
}
`,
		"internal/repository/x_store.repository.go": `package repository
type Store struct{}
`,
		"internal/repository/device.store.go": `package repository
import "context"
func (s *Store) FetchDevice(ctx context.Context, id string) (*string, error) { return &id, nil }
`,
	})

	violations, err := New(filepath.Join(root, "internal")).Violations()
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}
