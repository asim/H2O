package domain

import (
	"time"
)

type memoryRepository struct {
	data map[string]*ChangeSet
}

func NewMemoryRepository(data map[string]*ChangeSet) *memoryRepository {
	return &memoryRepository{data: data}
}

func (r memoryRepository) ReadConfig(ids []string) ([]*ChangeSet, error) {
	configs := make([]*ChangeSet, len(ids))
	for i := 0; i < len(ids); i++ {
		configs[i] = r.data[ids[i]]
	}
	return configs, nil
}

func (r *memoryRepository) UpdateConfig(cs *ChangeSet) error {
	r.data[cs.Id] = cs
	return nil
}

func (r *memoryRepository) ChangeLog(start, end time.Time, count int, lastId string) ([]*ChangeSet, string, error) {
	return []*ChangeSet{}, "", nil
}

func (r *memoryRepository) ServiceChangeLog(id string, start, end time.Time, count int, lastId string) ([]*ChangeSet, string, error) {
	return []*ChangeSet{}, "", nil
}
