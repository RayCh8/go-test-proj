package rpc

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	codes "github.com/AmazingTalker/at-error-code"
	mockDAO "github.com/AmazingTalker/go-amazing/internal/pkg/dao"
	"github.com/AmazingTalker/go-amazing/pkg/dao"
	"github.com/AmazingTalker/go-amazing/pkg/pb"
	"github.com/AmazingTalker/go-rpc-kit/errorkit"
	"github.com/AmazingTalker/go-rpc-kit/logkit"
	"github.com/AmazingTalker/go-rpc-kit/validatorkit"
)

var (
	mockCTX    = context.Background()
	mockUUID   = uuid.New()
	mockRecord = &dao.Record{
		TheNum: 3838,
		TheStr: "AT",
	}
)

type ExpAtError struct {
	ExpStatus int64
	ExpCode   codes.ATErrorCode
}

type rpcSuite struct {
	suite.Suite

	// mocks
	mockRecord *mockDAO.RecordDAO

	serv GoAmazingServer
}

func (s *rpcSuite) SetupSuite() {
	logkit.RegisterAmazingLogger(&logkit.Config{
		Logger:              logkit.LoggerZap,
		Development:         true,
		IntegrationAirbrake: &logkit.IntegrationAirbrake{},
	})
}

func (s *rpcSuite) TearDownSuite() {
	logkit.Flush()
}

func (s *rpcSuite) SetupTest() {
	// setup mock
	s.mockRecord = mockDAO.NewRecordDAO(s.T())

	s.serv = NewGoAmazingServer(GoAmazingServerOpt{
		Validator: validatorkit.NewGoPlaygroundValidator(),
		RecordDao: s.mockRecord,
	})
}

func (s *rpcSuite) TearDownTest() {
	s.mockRecord.AssertExpectations(s.T())
}

func TestRPCSuite(t *testing.T) {
	suite.Run(t, new(rpcSuite))
}

func (s *rpcSuite) TestHealth() {
	tests := []struct {
		Desc     string
		Req      *pb.HealthReq
		ExpError *ExpAtError
		ExpRes   *pb.HealthRes
	}{
		{
			Desc:     "normal case",
			ExpError: nil,
			ExpRes:   &pb.HealthRes{Ok: true},
		},
	}

	for _, t := range tests {
		resp, err := s.serv.Health(mockCTX, t.Req)

		if err == nil {
			s.Require().Equal(nil, err, t.Desc)
			s.Require().Equal(t.ExpRes, resp, t.Desc)
		} else {
			atErr := errorkit.FormatError(err)
			s.Require().Equal(t.ExpError.ExpCode, atErr.ATErrorCode(), t.Desc)
			s.Require().Equal(int(t.ExpError.ExpStatus), atErr.HttpStatus(), t.Desc)
			s.Require().Equal(t.ExpRes, resp, t.Desc)
		}

		s.TearDownTest()
	}
}

func (s *rpcSuite) TestCreateRecord() {
	tests := []struct {
		Desc      string
		SetupTest func(string)
		Req       *pb.CreateRecordReq
		ExpError  error
		ExpResp   *pb.CreateRecordRes
	}{
		{
			Desc: "create failed",
			SetupTest: func(desc string) {
				s.mockRecord.On(
					"CreateRecord", mock.Anything, mockRecord,
				).Return(
					errors.New("XD"),
				).Once()
			},
			Req: &pb.CreateRecordReq{
				TheNum: mockRecord.TheNum,
				TheStr: mockRecord.TheStr,
			},
			ExpError: errors.New("XD"),
		},
		{
			Desc: "normal case",
			SetupTest: func(desc string) {
				s.mockRecord.On(
					"CreateRecord", mock.Anything, mockRecord,
				).Return(
					nil,
				).Once()
			},
			Req: &pb.CreateRecordReq{
				TheNum: mockRecord.TheNum,
				TheStr: mockRecord.TheStr,
			},
			ExpError: nil,
			ExpResp: &pb.CreateRecordRes{
				Record: mockRecord.FormatPb(),
			},
		},
	}

	for _, t := range tests {
		if t.SetupTest != nil {
			t.SetupTest(t.Desc)
		}

		resp, err := s.serv.CreateRecord(mockCTX, t.Req)
		s.Require().Equal(t.ExpError, err, t.Desc)

		if err == nil {
			s.Require().Equal(t.ExpResp, resp, t.Desc)
		}

		s.TearDownTest()
	}
}

func (s *rpcSuite) TestGetRecord() {
	tests := []struct {
		Desc      string
		SetupTest func(string)
		Req       *pb.GetRecordReq
		ExpError  error
		ExpResp   *pb.GetRecordRes
	}{
		{
			Desc: "get failed",
			SetupTest: func(desc string) {
				s.mockRecord.On(
					"GetRecord", mock.Anything, mockUUID.String(),
				).Return(
					nil, errors.New("XD"),
				).Once()
			},
			Req:      &pb.GetRecordReq{ID: mockUUID.String()},
			ExpError: errors.New("XD"),
		},
		{
			Desc: "normal case",
			SetupTest: func(desc string) {
				s.mockRecord.On(
					"GetRecord", mock.Anything, mockUUID.String(),
				).Return(
					mockRecord, nil,
				).Once()
			},
			Req:      &pb.GetRecordReq{ID: mockUUID.String()},
			ExpError: nil,
			ExpResp: &pb.GetRecordRes{
				Record: mockRecord.FormatPb(),
			},
		},
	}

	for _, t := range tests {
		if t.SetupTest != nil {
			t.SetupTest(t.Desc)
		}

		resp, err := s.serv.GetRecord(mockCTX, t.Req)
		s.Require().Equal(t.ExpError, err, t.Desc)

		if err == nil {
			s.Require().Equal(t.ExpResp, resp, t.Desc)
		}

		s.TearDownTest()
	}
}

func (s *rpcSuite) TestListRecord() {
	tests := []struct {
		Desc      string
		SetupTest func(string)
		Req       *pb.ListRecordReq
		ExpError  error
		ExpResp   *pb.ListRecordRes
	}{
		{
			Desc: "parse size failed",
			Req: &pb.ListRecordReq{
				PageSize: "abc",
				Page:     "1",
			},
			ExpError: &strconv.NumError{
				Func: "ParseInt",
				Num:  "abc",
				Err:  errors.New("invalid syntax"),
			},
		},
		{
			Desc: "parse size failed",
			Req: &pb.ListRecordReq{
				PageSize: "1",
				Page:     "abc",
			},
			ExpError: &strconv.NumError{
				Func: "ParseInt",
				Num:  "abc",
				Err:  errors.New("invalid syntax"),
			},
		},
		{
			Desc: "list failed",
			Req: &pb.ListRecordReq{
				PageSize: "10",
				Page:     "1",
			},
			SetupTest: func(desc string) {
				s.mockRecord.On(
					"ListRecords", mock.Anything, dao.ListRecordsOpt{Size: 10, Page: 1},
				).Return(
					nil, errors.New("XD"),
				).Once()
			},
			ExpError: errors.New("XD"),
		},
		{
			Desc: "normal case",
			Req: &pb.ListRecordReq{
				PageSize: "10",
				Page:     "1",
			},
			SetupTest: func(desc string) {
				s.mockRecord.On(
					"ListRecords", mock.Anything, dao.ListRecordsOpt{Size: 10, Page: 1},
				).Return(
					[]dao.Record{*mockRecord}, nil,
				).Once()
			},
			ExpError: nil,
			ExpResp: &pb.ListRecordRes{
				Records: []*pb.Record{mockRecord.FormatPb()},
			},
		},
	}

	for _, t := range tests {
		if t.SetupTest != nil {
			t.SetupTest(t.Desc)
		}

		resp, err := s.serv.ListRecord(mockCTX, t.Req)
		s.Require().Equal(t.ExpError, err, t.Desc)

		if err == nil {
			s.Require().Equal(t.ExpResp, resp, t.Desc)
		}

		s.TearDownTest()
	}
}
