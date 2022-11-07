package dao

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/AmazingTalker/go-amazing/pkg/pb"
	"github.com/AmazingTalker/go-rpc-kit/daokit"
)

type ListRecordsOpt struct {
	Size int
	Page int
}

type RecordDAO interface {
	CreateRecord(context.Context, *Record, ...daokit.Enrich) error
	GetRecord(context.Context, string) (*Record, error)
	ListRecords(context.Context, ListRecordsOpt) ([]Record, error)
}

type Record struct {
	ID        uuid.UUID
	TheNum    int64
	TheStr    string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func (r *Record) FormatPb() *pb.Record {
	return &pb.Record{
		ID:        r.ID.String(),
		TheNum:    r.TheNum,
		TheStr:    r.TheStr,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
