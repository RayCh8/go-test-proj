package dao

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/AmazingTalker/go-rpc-kit/daokit"
	"github.com/AmazingTalker/go-rpc-kit/logkit"
)

type MySqlRecordDAO struct {
	db *gorm.DB
}

func NewMySqlRecordDAO(db *gorm.DB) MySqlRecordDAO {
	return MySqlRecordDAO{db: db}
}

func (dao MySqlRecordDAO) CreateRecord(ctx context.Context, record *Record, enrich ...daokit.Enrich) error {
	defer met.RecordDuration([]string{"mysql", "time"}, map[string]string{}).End()

	record.ID = uuid.New()

	db, _ := daokit.UseTxOrDB(dao.db, enrich...)

	err := db.Create(record).Error

	if err != nil {
		return err
	}
	return nil
}

func (dao MySqlRecordDAO) GetRecord(ctx context.Context, id string) (*Record, error) {
	defer met.RecordDuration([]string{"mysql", "time"}, map[string]string{}).End()

	record := &Record{}

	err := dao.db.First(record, "id = ?", id).Error

	if err != nil {
		logkit.Debug(ctx, "get record failed", logkit.Payload{"id": id, "err": err})
		return nil, err
	}

	return record, nil
}

func (dao MySqlRecordDAO) ListRecords(ctx context.Context, opt ListRecordsOpt) ([]Record, error) {
	defer met.RecordDuration([]string{"mysql", "time"}, map[string]string{}).End()

	query := dao.db

	if opt.Size > 0 {
		query = query.Limit(opt.Size)

		if opt.Page > 0 {
			query = query.Offset(opt.Page * opt.Size)
		}
	}

	list := []Record{}
	if err := query.Find(&list).Error; err != nil {
		logkit.Debug(ctx, "list record failed", logkit.Payload{"options": opt, "err": err})
		return nil, err
	}

	return list, nil
}
