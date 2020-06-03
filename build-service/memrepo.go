package main

import (
	"time"

	"github.com/hailo-platform/H2O/build-service/models"
)

type memoryRepo struct {
	builds []*models.Build
	called string
	filter string
}

func (r *memoryRepo) Create(b *models.Build) error {
	r.called = "Create"
	r.builds = append(r.builds, b)
	return nil
}

func (r *memoryRepo) GetAll(limit int) ([]*models.Build, error) {
	r.called = "GetAll"
	return r.builds, nil
}

func (r *memoryRepo) GetAllWithName(name string, limit int) ([]*models.Build, error) {
	r.called = "GetAllWithName"
	return nil, nil
}

func (r *memoryRepo) GetVersion(name, version string) (*models.Build, error) {
	r.called = "GetVersion"
	return nil, nil
}

func (r *memoryRepo) Delete(name, version string) error {
	r.called = "Delete"
	return nil
}

func (r *memoryRepo) GetNames(filter string) ([]string, error) {
	r.called = "GetNames"
	r.filter = filter
	return []string{}, nil
}

func (r *memoryRepo) GetCoverage(name, version string) (map[string]float64, error) {
	r.called = "GetCoverage"
	return nil, nil
}

func (r *memoryRepo) GetCoverageTrend(name string, since time.Time) (models.CoverageSnapshots, error) {
	r.called = "GetCoverageTrend"
	return nil, nil
}

func (r *memoryRepo) SetMergeBaseDate(service, version, importPath, commit string, date time.Time) error {
	r.called = "SetMergeBaseDate"
	return nil
}

func newTestRepo() *memoryRepo {
	return &memoryRepo{
		builds: make([]*models.Build, 0),
	}
}

type memCommitRepo struct{}

func (r *memCommitRepo) MergeBaseDate(importPath, sha, base string) (*time.Time, error) {
	return nil, nil
}

func newTestCommitRepo() *memCommitRepo {
	return &memCommitRepo{}
}
