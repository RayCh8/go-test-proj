package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"

	"github.com/AmazingTalker/go-cache"
	"github.com/AmazingTalker/go-rpc-kit/cachekit"
	"github.com/AmazingTalker/go-rpc-kit/dockerkit"
	"github.com/AmazingTalker/go-rpc-kit/logkit"
	"github.com/AmazingTalker/go-rpc-kit/migrationkit"
	"github.com/AmazingTalker/go-rpc-kit/mysqlkit"
)

const (
	migrationDir = "../../database/migrations"
	sqlURLFormat = "root:root@tcp(localhost:%s)/mysql?charset=utf8mb4&parseTime=True"
	rdsURLFormat = ":%s"
)

var (
	mockCTX     = context.Background()
	mockUUID    = uuid.New()
	mockTimeNow time.Time
	mockLoc     *time.Location
)

func init() {
	mockLoc, _ = time.LoadLocation("")
	mockTimeNow = time.Unix(1629446406, 0).In(mockLoc)
}

type daoSuite struct {
	suite.Suite

	ring  *redis.Ring
	db    *gorm.DB
	cache cache.Service
	im    *impl

	redisPort string
	mysqlPort string
}

func (s *daoSuite) migrationDir() string {
	if dockerkit.RunCITest() {
		return os.Getenv("MIGRATION_DIR")
	}

	return migrationDir
}

func (s *daoSuite) mysqlURL() string {
	if dockerkit.RunCITest() {
		return os.Getenv("MYSQL_DSN")
	}

	return fmt.Sprintf(sqlURLFormat, s.mysqlPort)
}

func (s *daoSuite) redisAddrs() map[string]string {
	if dockerkit.RunCITest() {
		addrs := strings.Split(os.Getenv("REDIS_ADDRS"), ",")

		m := map[string]string{}
		for _, addr := range addrs {
			strs := strings.SplitN(addr, ":", 2)
			m[strs[0]] = strs[1]
		}

		return m
	}

	return map[string]string{"server1": fmt.Sprintf(rdsURLFormat, s.redisPort)}
}

func (s *daoSuite) SetupSuite() {
	// setup logger
	logkit.RegisterAmazingLogger(&logkit.Config{
		Logger:              logkit.LoggerZap,
		Development:         true,
		IntegrationAirbrake: &logkit.IntegrationAirbrake{},
	})

	// run dockerkit when dealing with go test locally
	if dockerkit.RunLocalTest() {
		ports, err := dockerkit.RunExtDockers(mockCTX, []dockerkit.Image{
			dockerkit.ImageMySQL,
			dockerkit.ImageRedis,
		})
		s.Require().NoError(err)
		s.mysqlPort = ports[0]
		s.redisPort = ports[1]
	}

	// setup mysql
	logkit.Info(mockCTX, "init mysql", logkit.Payload{"dir": s.migrationDir(), "mysqlURL": s.mysqlURL()})
	migration := migrationkit.NewGooseMigrationKit(migrationkit.GooseMysqlDriver, migrationkit.GooseMigrationOpt{
		Dir:      s.migrationDir(),
		DBString: s.mysqlURL(),
	})
	s.Require().NoError(migration.Up())
	migration.Close()

	db, err := mysqlkit.NewMySqlConn(mysqlkit.MySqlConnOpt{
		Config: &mysqlkit.MysqlConnConf{
			DSN: s.mysqlURL(),
		},
	})
	s.Require().NoError(err)
	s.db = db

	// setup redis
	logkit.Info(mockCTX, "init redis", logkit.Payload{"redisAddrs": s.redisAddrs()})
	s.ring = redis.NewRing(&redis.RingOptions{
		Addrs: s.redisAddrs(),
	})
}

func (s *daoSuite) TearDownSuite() {
	sqlDB, _ := s.db.DB()
	sqlDB.Close()
	s.ring.Close()

	if dockerkit.RunLocalTest() {
		dockerkit.PurgeExtDockers(mockCTX, []dockerkit.Image{
			dockerkit.ImageRedis,
			dockerkit.ImageMySQL,
		})
	}

	logkit.Flush()
}

func (s *daoSuite) SetupTest() {
	cache.ClearPrefix()
	s.cache = cachekit.NewCache(
		cachekit.NewSharedCache(s.ring),
		cachekit.NewLocalCache(1024),
	)

	s.im = NewRecordDAO(s.db, s.cache).(*impl)
}

func (s *daoSuite) TearDownTest() {
	cache.ClearPrefix()

	// clean all in redis
	s.Require().NoError(s.ring.ForEachShard(mockCTX, func(ctx context.Context, client *redis.Client) error {
		return client.FlushDB(ctx).Err()
	}))

	// clean all in mysql
	s.Require().NoError(s.db.Where("1 = 1").Delete(&Record{}).Error)
}

func TestDAOSuite(t *testing.T) {
	suite.Run(t, new(daoSuite))
}

func (s *daoSuite) TestCreateRecord() {
	tests := []struct {
		Desc      string
		Record    *Record
		CheckFunc func(string)
	}{
		{
			Desc: "normal case",
			Record: &Record{
				TheNum:    1,
				TheStr:    "normal",
				CreatedAt: &mockTimeNow,
				UpdatedAt: &mockTimeNow,
			},
			CheckFunc: func(desc string) {
				records := []Record{}
				s.Require().NoError(s.db.Find(&records).Error, desc)
				s.Require().Equal(1, len(records), desc)

				record := records[0]
				s.Require().Equal(mockTimeNow, *record.CreatedAt, desc)
				s.Require().Equal(int64(1), record.TheNum, desc)
				s.Require().Equal("normal", record.TheStr, desc)
			},
		},
	}

	for _, t := range tests {
		s.SetupTest()

		err := s.im.CreateRecord(mockCTX, t.Record)
		s.Require().NoError(err, t.Desc)

		if t.CheckFunc != nil {
			t.CheckFunc(t.Desc)
		}

		s.TearDownTest()
	}
}

func (s *daoSuite) TestGetRecord() {
	tests := []struct {
		Desc      string
		SetupTest func(string)
		ID        string
		ExpErr    error
		ExpRecord *Record
		CheckFunc func(string)
	}{
		{
			Desc:   "not existed",
			ID:     "nothing",
			ExpErr: fmt.Errorf("record not found"),
		},
		{
			Desc: "normal case",
			SetupTest: func(desc string) {
				rs := []Record{
					{ID: mockUUID, CreatedAt: &mockTimeNow, UpdatedAt: &mockTimeNow, TheNum: 80, TheStr: "AT"},
				}
				s.Require().NoError(s.db.Create(&rs).Error, desc)
			},
			ID:     mockUUID.String(),
			ExpErr: nil,
			ExpRecord: &Record{
				ID:        mockUUID,
				TheNum:    int64(80),
				TheStr:    "AT",
				CreatedAt: &mockTimeNow,
				UpdatedAt: &mockTimeNow,
			},
			CheckFunc: func(desc string) {
				// check cache
				b, err := s.ring.Get(mockCTX, fmt.Sprintf("ca:records:%s", mockUUID.String())).Bytes()
				s.Require().NoError(err, desc)

				r := Record{}
				s.Require().NoError(json.Unmarshal(b, &r), desc)
				s.Require().Equal(Record{
					ID:        mockUUID,
					TheNum:    int64(80),
					TheStr:    "AT",
					CreatedAt: &mockTimeNow,
					UpdatedAt: &mockTimeNow,
				}, r, desc)
			},
		},
	}

	for _, t := range tests {
		s.SetupTest()

		if t.SetupTest != nil {
			t.SetupTest(t.Desc)
		}

		record, err := s.im.GetRecord(mockCTX, t.ID)
		s.Require().Equal(t.ExpErr, err, t.Desc)
		if err == nil {
			s.Require().Equal(t.ExpRecord, record, t.Desc)
		}

		if t.CheckFunc != nil {
			t.CheckFunc(t.Desc)
		}

		s.TearDownTest()
	}
}

func (s *daoSuite) TestListRecords() {
	tests := []struct {
		Desc       string
		SetupTest  func(string)
		Opt        ListRecordsOpt
		ExpErr     error
		ExpRecords []Record
		CheckFunc  func(string)
	}{
		{
			Desc:       "no records",
			Opt:        ListRecordsOpt{Size: 10, Page: 0},
			ExpErr:     nil,
			ExpRecords: []Record{},
		},
		{
			Desc: "normal case",
			SetupTest: func(desc string) {
				rs := []Record{
					{ID: mockUUID, CreatedAt: &mockTimeNow, UpdatedAt: &mockTimeNow, TheNum: 80, TheStr: "AT"},
				}
				s.Require().NoError(s.db.Create(&rs).Error, desc)
			},
			Opt:    ListRecordsOpt{Size: 10, Page: 0},
			ExpErr: nil,
			ExpRecords: []Record{
				{
					ID:        mockUUID,
					CreatedAt: &mockTimeNow,
					UpdatedAt: &mockTimeNow,
					TheNum:    80,
					TheStr:    "AT",
				},
			},
			CheckFunc: func(desc string) {
				// check cache
				b, err := s.ring.Get(mockCTX, "ca:records:0-10").Bytes()
				s.Require().NoError(err, desc)

				rs := []Record{}
				s.Require().NoError(json.Unmarshal(b, &rs), desc)
				s.Require().Equal([]Record{{
					ID:        mockUUID,
					TheNum:    int64(80),
					TheStr:    "AT",
					CreatedAt: &mockTimeNow,
					UpdatedAt: &mockTimeNow,
				}}, rs, desc)
			},
		},
	}

	for _, t := range tests {
		s.SetupTest()

		if t.SetupTest != nil {
			t.SetupTest(t.Desc)
		}

		records, err := s.im.ListRecords(mockCTX, t.Opt)
		s.Require().Equal(t.ExpErr, err, t.Desc)
		if err == nil {
			s.Require().Equal(t.ExpRecords, records, t.Desc)
		}

		if t.CheckFunc != nil {
			t.CheckFunc(t.Desc)
		}

		s.TearDownTest()
	}
}
