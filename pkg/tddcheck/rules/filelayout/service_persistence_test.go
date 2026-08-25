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

	violations, err := checkRoot(filepath.Join(root, "internal"))
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

	violations, err := checkRoot(filepath.Join(root, "internal"))
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
		"internal/repository/x.store.repository.go": `package repository
type Store struct { last *DeviceModel }
func NewStore() *Store { return &Store{} }
`,
		"internal/repository/device.schema.go": `package repository
type DeviceModel struct{}
`,
		"internal/repository/device.store.go": `package repository
import "context"
func (s *Store) FetchDevice(ctx context.Context, id string) (*string, error) { return &id, nil }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %#v", violations)
	}
}

func TestViolationsAllowsRepositoryRowFieldNamedModel(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/repository/model_alias.support.go": `package repository
type ModelAliasRow struct { Model string }
`,
		"internal/service/device.service.go": `package service
import "example.com/app/internal/repository"
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
func (s *DeviceService) Resolve(row repository.ModelAliasRow) string { return row.Model }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertNoViolationContains(t, violations, "repository schema models")
}

func TestViolationsAllowsBusinessMethodNamedQuery(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.service.go": `package service
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
func (s *DeviceService) Query() {}
func (s *DeviceService) List() { s.Query() }
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertNoViolationContains(t, violations, "persistence APIs directly")
}

func TestViolationsChecksPersistenceAcrossServiceFileRoles(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.service.go": `package service
type DeviceService struct{}
func NewDeviceService() *DeviceService { return &DeviceService{} }
`,
		"internal/service/device.commands.go": `package service
import "database/sql"
type DeviceRequest struct { DB *sql.DB }
`,
		"internal/service/device.types.go": `package service
import "example.com/app/internal/repository"
type DeviceResult struct { Row repository.DeviceRow }
`,
		"internal/service/device.support.go": `package service
import "github.com/uptrace/bun"
type DeviceSupport struct { DB *bun.DB }
`,
		"internal/repository/device.support.go": `package repository
type DeviceRow struct{}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "service files must not import persistence package database/sql")
	assertViolationContains(t, violations, "service files must not import persistence package github.com/uptrace/bun")
	assertViolationContains(t, violations, "service public type DeviceResult must not expose repository type")
}

func TestViolationsRejectsRepositoryModelHiddenInExportedServiceType(t *testing.T) {
	root := fixture(t, map[string]string{
		"internal/service/device.service.go": `package service
import "example.com/app/internal/repository"
type DeviceService struct{}
type DeviceResult struct { Row repository.DeviceRow }
func NewDeviceService() *DeviceService { return &DeviceService{} }
func (s *DeviceService) Get() DeviceResult { return DeviceResult{} }
`,
		"internal/repository/device.support.go": `package repository
type DeviceRow struct{}
`,
	})

	violations, err := checkRoot(filepath.Join(root, "internal"))
	if err != nil {
		t.Fatal(err)
	}
	assertViolationContains(t, violations, "service public type DeviceResult must not expose repository type")
}
