package store

import (
	"encoding/json"
	"errors"
	"time"

	"phone-task-client/internal/domain"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func Open(path string) (*Store, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(
		&settingModel{},
		&profileModel{},
		&apiTemplateModel{},
		&taskTemplateModel{},
		&jobModel{},
		&jobItemModel{},
		&eventModel{},
		&devicePoolSnapshotModel{},
	); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *Store) SaveGlobalSettings(settings domain.GlobalSettings) error {
	values := map[string]string{
		"base_url":        settings.BaseURL,
		"reserve_devices": int64String(settings.ReserveDevices),
		"interval_ms":     int64String(settings.Interval.Milliseconds()),
		"timeout_ms":      int64String(settings.Timeout.Milliseconds()),
		"log_dir":         settings.LogDir,
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		for key, value := range values {
			model := settingModel{Key: key, Value: value, UpdatedAt: time.Now()}
			if err := tx.Save(&model).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) LoadGlobalSettings() (domain.GlobalSettings, error) {
	var rows []settingModel
	if err := s.db.Find(&rows).Error; err != nil {
		return domain.GlobalSettings{}, err
	}
	values := map[string]string{}
	for _, row := range rows {
		values[row.Key] = row.Value
	}
	return domain.GlobalSettings{
		BaseURL:        values["base_url"],
		ReserveDevices: parseInt64(values["reserve_devices"]),
		Interval:       time.Duration(parseInt64(values["interval_ms"])) * time.Millisecond,
		Timeout:        time.Duration(parseInt64(values["timeout_ms"])) * time.Millisecond,
		LogDir:         values["log_dir"],
	}, nil
}

func (s *Store) SaveProfile(profile domain.Profile) (domain.Profile, error) {
	model := profileModelFromDomain(profile)
	if err := s.db.Save(&model).Error; err != nil {
		return domain.Profile{}, err
	}
	return model.toDomain(), nil
}

func (s *Store) GetProfile(id int64) (domain.Profile, error) {
	var model profileModel
	if err := s.db.First(&model, id).Error; err != nil {
		return domain.Profile{}, err
	}
	return model.toDomain(), nil
}

func (s *Store) ListProfiles() ([]domain.Profile, error) {
	var models []profileModel
	if err := s.db.Order("id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Profile, 0, len(models))
	for _, model := range models {
		items = append(items, model.toDomain())
	}
	return items, nil
}

func (s *Store) SaveAPITemplate(t domain.APITemplate) (domain.APITemplate, error) {
	model, err := apiTemplateModelFromDomain(t)
	if err != nil {
		return domain.APITemplate{}, err
	}
	if err := s.db.Save(&model).Error; err != nil {
		return domain.APITemplate{}, err
	}
	return model.toDomain()
}

func (s *Store) GetAPITemplate(id int64) (domain.APITemplate, error) {
	var model apiTemplateModel
	if err := s.db.First(&model, id).Error; err != nil {
		return domain.APITemplate{}, err
	}
	return model.toDomain()
}

func (s *Store) ListAPITemplates() ([]domain.APITemplate, error) {
	var models []apiTemplateModel
	if err := s.db.Order("id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]domain.APITemplate, 0, len(models))
	for _, model := range models {
		item, err := model.toDomain()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) SaveTaskTemplate(t domain.TaskTemplate) (domain.TaskTemplate, error) {
	model := taskTemplateModelFromDomain(t)
	if err := s.db.Save(&model).Error; err != nil {
		return domain.TaskTemplate{}, err
	}
	return model.toDomain(), nil
}

func (s *Store) GetTaskTemplate(id int64) (domain.TaskTemplate, error) {
	var model taskTemplateModel
	if err := s.db.First(&model, id).Error; err != nil {
		return domain.TaskTemplate{}, err
	}
	return model.toDomain(), nil
}

func (s *Store) ListTaskTemplates() ([]domain.TaskTemplate, error) {
	var models []taskTemplateModel
	if err := s.db.Order("id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]domain.TaskTemplate, 0, len(models))
	for _, model := range models {
		items = append(items, model.toDomain())
	}
	return items, nil
}

func (s *Store) CreateJob(job domain.Job, items []domain.JobItem) (domain.Job, []domain.JobItem, error) {
	var savedJob domain.Job
	var savedItems []domain.JobItem
	err := s.db.Transaction(func(tx *gorm.DB) error {
		jobModel := jobModelFromDomain(job)
		if jobModel.CreatedAt.IsZero() {
			jobModel.CreatedAt = time.Now()
		}
		if jobModel.UpdatedAt.IsZero() {
			jobModel.UpdatedAt = jobModel.CreatedAt
		}
		if err := tx.Create(&jobModel).Error; err != nil {
			return err
		}
		savedJob = jobModel.toDomain()
		for _, item := range items {
			item.JobID = savedJob.ID
			itemModel := jobItemModelFromDomain(item)
			if itemModel.CreatedAt.IsZero() {
				itemModel.CreatedAt = jobModel.CreatedAt
			}
			if itemModel.UpdatedAt.IsZero() {
				itemModel.UpdatedAt = itemModel.CreatedAt
			}
			if err := tx.Create(&itemModel).Error; err != nil {
				return err
			}
			savedItems = append(savedItems, itemModel.toDomain())
		}
		return nil
	})
	return savedJob, savedItems, err
}

func (s *Store) GetJob(id int64) (domain.Job, error) {
	var model jobModel
	if err := s.db.First(&model, id).Error; err != nil {
		return domain.Job{}, err
	}
	return model.toDomain(), nil
}

func (s *Store) ListJobs(limit int) ([]domain.Job, error) {
	q := s.db.Order("created_at desc, id desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var models []jobModel
	if err := q.Find(&models).Error; err != nil {
		return nil, err
	}
	jobs := make([]domain.Job, 0, len(models))
	for _, model := range models {
		jobs = append(jobs, model.toDomain())
	}
	return jobs, nil
}

func (s *Store) ListJobsPage(page int, pageSize int) ([]domain.Job, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var total int64
	if err := s.db.Model(&jobModel{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var models []jobModel
	offset := (page - 1) * pageSize
	if err := s.db.Order("created_at desc, id desc").Limit(pageSize).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	jobs := make([]domain.Job, 0, len(models))
	for _, model := range models {
		jobs = append(jobs, model.toDomain())
	}
	return jobs, total, nil
}

func (s *Store) UpdateJob(job domain.Job) error {
	if job.ID == 0 {
		return errors.New("job id is required")
	}
	model := jobModelFromDomain(job)
	if model.UpdatedAt.IsZero() {
		model.UpdatedAt = time.Now()
	}
	return s.db.Save(&model).Error
}

func (s *Store) DeleteJob(jobID int64) error {
	if jobID == 0 {
		return errors.New("job id is required")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var job jobModel
		if err := tx.First(&job, jobID).Error; err != nil {
			return err
		}
		if domain.JobStatus(job.Status) == domain.JobStatusRunning {
			return errors.New("执行中的任务不能删除，请先停止")
		}
		if err := tx.Where("job_id = ?", jobID).Delete(&eventModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("job_id = ?", jobID).Delete(&jobItemModel{}).Error; err != nil {
			return err
		}
		return tx.Delete(&jobModel{}, jobID).Error
	})
}

func (s *Store) ListJobItems(jobID int64) ([]domain.JobItem, error) {
	var models []jobItemModel
	if err := s.db.Where("job_id = ?", jobID).Order("id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]domain.JobItem, 0, len(models))
	for _, model := range models {
		items = append(items, model.toDomain())
	}
	return items, nil
}

func (s *Store) ListRunnableJobs() ([]domain.Job, error) {
	var models []jobModel
	if err := s.db.Where("status = ? AND paused = ? AND stopped = ?", string(domain.JobStatusRunning), false, false).Order("created_at asc, id asc").Find(&models).Error; err != nil {
		return nil, err
	}
	jobs := make([]domain.Job, 0, len(models))
	for _, model := range models {
		jobs = append(jobs, model.toDomain())
	}
	return jobs, nil
}

func (s *Store) CountItemsByStatus(jobID int64, statuses ...domain.JobItemStatus) (int, error) {
	var count int64
	values := make([]string, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, string(status))
	}
	if err := s.db.Model(&jobItemModel{}).Where("job_id = ? AND status IN ?", jobID, values).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *Store) ListItemsByStatus(jobID int64, limit int, statuses ...domain.JobItemStatus) ([]domain.JobItem, error) {
	values := make([]string, 0, len(statuses))
	for _, status := range statuses {
		values = append(values, string(status))
	}
	q := s.db.Where("job_id = ? AND status IN ?", jobID, values).Order("updated_at asc, id asc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	var models []jobItemModel
	if err := q.Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]domain.JobItem, 0, len(models))
	for _, model := range models {
		items = append(items, model.toDomain())
	}
	return items, nil
}

func (s *Store) AddJobItem(item domain.JobItem) (domain.JobItem, error) {
	model := jobItemModelFromDomain(item)
	if model.CreatedAt.IsZero() {
		model.CreatedAt = time.Now()
	}
	if model.UpdatedAt.IsZero() {
		model.UpdatedAt = model.CreatedAt
	}
	if err := s.db.Create(&model).Error; err != nil {
		return domain.JobItem{}, err
	}
	return model.toDomain(), nil
}

func (s *Store) UpdateJobItem(item domain.JobItem) error {
	if item.ID == 0 {
		return errors.New("job item id is required")
	}
	model := jobItemModelFromDomain(item)
	if model.UpdatedAt.IsZero() {
		model.UpdatedAt = time.Now()
	}
	return s.db.Save(&model).Error
}

func (s *Store) AddEvent(event domain.Event) (domain.Event, error) {
	model := eventModelFromDomain(event)
	if model.CreatedAt.IsZero() {
		model.CreatedAt = time.Now()
	}
	if err := s.db.Create(&model).Error; err != nil {
		return domain.Event{}, err
	}
	return model.toDomain(), nil
}

func mustJSON(value map[string]string) string {
	if len(value) == 0 {
		return "{}"
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func parseJSONMap(raw string) map[string]string {
	if raw == "" {
		return map[string]string{}
	}
	out := map[string]string{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return map[string]string{}
	}
	return out
}
