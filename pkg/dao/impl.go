package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/AmazingTalker/go-cache"
	"github.com/AmazingTalker/go-rpc-kit/daokit"
	"github.com/AmazingTalker/go-rpc-kit/logkit"
	"github.com/AmazingTalker/go-rpc-kit/metrickit"
)

const (
	pfxRecord = "records"
)

var (
	met = metrickit.NewWithPkgName(
		metrickit.EnableAutoFillInFuncName(true),
	)
)

type impl struct {
	mysql MySqlRecordDAO
	cache cache.Cache
}

func NewRecordDAO(db *gorm.DB, cacheSrv cache.Service) RecordDAO {
	im := &impl{mysql: NewMySqlRecordDAO(db)}

	im.cache = cacheSrv.Create([]cache.Setting{
		{
			Prefix: pfxRecord,
			CacheAttributes: map[cache.Type]cache.Attribute{
				cache.SharedCacheType: {TTL: time.Minute},
				cache.LocalCacheType:  {TTL: 10 * time.Second},
			},
		},
	})

	/*
		Use cases:
			ctx := context.Background()

			1) Get()

			record := Record{}
			if err := im.cache.Get(ctx, pfxRecord, "key", &record); err != nil {
				return err
			}

			---

			2) GetM()

			records := []*Record{}
			res, err := im.cache.MGet(ctx, pfxRecord, "key1", "key2", "key3")
			if err != nil {
				return err
			}

			for i := 0; i < res.Len(); i++ {
				r := &Record{}
				if err := res.Get(ctx, i, r); err != nil {
					return err // It may be ErrCacheMiss or other errors
				}
				records = append(records, r)
			}

		More examples: https://github.com/AmazingTalker/go-cache
	*/

	return im
}

func (im *impl) CreateRecord(ctx context.Context, record *Record, enrich ...daokit.Enrich) error {
	defer met.RecordDuration([]string{"time"}, map[string]string{}).End()

	return im.mysql.CreateRecord(ctx, record, enrich...)
}

func (im *impl) GetRecord(ctx context.Context, id string) (*Record, error) {
	defer met.RecordDuration([]string{"time"}, map[string]string{}).End()

	record := &Record{}
	ctx = logkit.EnrichPayload(ctx, logkit.Payload{"usingCachePrefix": pfxRecord})

	if err := im.cache.GetByFunc(ctx, pfxRecord, id, record, func() (interface{}, error) {
		// TODO: cache GetByFunc should pass the context
		ctx = logkit.EnrichPayload(ctx, logkit.Payload{"cacheHit": false})
		return im.mysql.GetRecord(ctx, id)
	}); err != nil {
		return nil, err
	}

	return record, nil
}

func (im *impl) ListRecords(ctx context.Context, opt ListRecordsOpt) ([]Record, error) {
	defer met.RecordDuration([]string{"time"}, map[string]string{}).End()

	var records []Record

	key := fmt.Sprintf("%v-%v", opt.Page, opt.Size)
	if err := im.cache.GetByFunc(ctx, pfxRecord, key, &records, func() (interface{}, error) {
		// TODO: cache GetByFunc should pass the context
		return im.mysql.ListRecords(ctx, opt)
	}); err != nil {
		return nil, err
	}

	return records, nil
}
