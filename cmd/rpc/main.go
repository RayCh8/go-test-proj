package main

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rafaelhl/gorm-newrelic-telemetry-plugin/telemetry"
	etcd "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"

	"github.com/AmazingTalker/go-amazing/pkg/dao"
	"github.com/AmazingTalker/go-amazing/pkg/pb"
	"github.com/AmazingTalker/go-amazing/pkg/rpc"
	"github.com/AmazingTalker/go-rpc-kit/cachekit"
	"github.com/AmazingTalker/go-rpc-kit/configkit"
	"github.com/AmazingTalker/go-rpc-kit/envkit"
	"github.com/AmazingTalker/go-rpc-kit/flagkit"
	"github.com/AmazingTalker/go-rpc-kit/logkit"
	"github.com/AmazingTalker/go-rpc-kit/metrickit"
	"github.com/AmazingTalker/go-rpc-kit/migrationkit"
	"github.com/AmazingTalker/go-rpc-kit/monitorkit"
	"github.com/AmazingTalker/go-rpc-kit/mysqlkit"
	"github.com/AmazingTalker/go-rpc-kit/rediskit"
	"github.com/AmazingTalker/go-rpc-kit/validatorkit"
)

const (
	configRoot = "/configs/envs/development"
)

type ServiceLauncher struct {
	Run    func() error
	Labels []string
}

func main() {

	// get env
	if err := flagkit.Parse(); err != nil {
		panic(err)
	}

	// register env first, it will benefit ctx, logkit, and metric
	envkit.Register(envkit.Config{
		EnvNamespace: env.EnvConfig.Namespace,
		PodName:      env.EnvConfig.PodName,
		ServiceName:  env.EnvConfig.ServiceName,
	})

	// init context
	ctx := context.Background()

	// init logger
	logkit.RegisterAmazingLogger(&logkit.Config{
		Logger:      logkit.LoggerZap,
		Level:       env.LoggerConfig.Level,
		Development: env.LoggerConfig.Development,
		IntegrationAirbrake: &logkit.IntegrationAirbrake{
			ProjectID:  env.LoggerConfig.Airbrake.ProjectID,
			ProjectKey: env.LoggerConfig.Airbrake.ProjectKey,
			Env:        envkit.EnvNamespace(),
		},
	})
	defer logkit.Flush()

	// init etcd
	logkit.Info(ctx, "init etcd", logkit.Payload{
		"addrs":              env.EtcdConfig.Addrs,
		"dialTimeoutSeconds": env.EtcdConfig.DialTimeoutSeconds,
	})

	etcdCli, err := etcd.New(etcd.Config{
		Username:    env.EtcdConfig.Username,
		Password:    env.EtcdConfig.Password,
		Endpoints:   env.EtcdConfig.Addrs,
		DialTimeout: time.Second * time.Duration(env.EtcdConfig.DialTimeoutSeconds),
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
	})
	if err != nil {
		// Because eks-staging-v2 haven't deployed the ETCD. I ignore this Fatal, use normal Error instead.
		logkit.ErrorV2(ctx, "init etcd failed", err, nil)
	}
	defer etcdCli.Close()

	// publishing the configs to etcd in development env
	if envkit.Namespace() == envkit.EnvDevelopment {
		publisher := configkit.NewPublisher(etcdCli, configRoot, configkit.RenderRoot(envkit.EnvDevelopment))
		if err := publisher.Publish(ctx); err != nil {
			logkit.FatalV2(ctx, "config.Publish failed", err, nil)
		}
	}

	// init dynamic config watcher
	logkit.Info(ctx, "init config watcher", logkit.Payload{
		"projectName": envkit.ProjectName(),
		"env":         envkit.Namespace(),
	})

	if err := configkit.LaunchWatcher(ctx, configkit.Params{
		ProjectName: envkit.ProjectName(),
		Env:         envkit.Namespace(),
		Client:      etcdCli,
	}); err != nil {
		logkit.ErrorV2(ctx, "config.LaunchWatcher failed, and stopped listening changes on remote", err, nil)
		logkit.Errorf(ctx, "check prject name in amazing-configs, or the path parameter in configkit.Register()")
	}

	// init metric
	logkit.Info(ctx, "init metric", logkit.Payload{
		"url":            env.MetricConfig.URL,
		"refleshSeconds": env.MetricConfig.RefleshSeconds,
	})

	metrickit.Register(metrickit.Config{
		Exporter:      metrickit.ExporterNewRelic,
		URL:           env.MetricConfig.URL,
		APIKey:        env.MetricConfig.APIKey,
		RefleshPeriod: time.Duration(time.Duration(env.RefleshSeconds) * time.Second),
	})

	// init monitoring
	logkit.Info(ctx, "init monitor", logkit.Payload{
		"period-in-secs": env.MonitorConfig.PeriodSecs,
	})

	monitorkit.Register(monitorkit.Config{
		Metric:                   metrickit.New("monitor"),
		RuntimeCollectorInterval: time.Duration(time.Duration(env.MonitorConfig.PeriodSecs) * time.Second),
	})
	monitorkit.Run()
	defer monitorkit.GracefulStop()

	// init redis
	logkit.Info(ctx, "init redis", logkit.Payload{"addrs": env.RedisConfig.Addrs})
	ring, err := rediskit.NewRedisRing(env.RedisConfig.Addrs)
	if err != nil {
		logkit.FatalV2(ctx, "init redis failed", err, nil)
	}

	// init cache
	logkit.Info(ctx, "init cache", logkit.Payload{"size": env.LocalCacheConfig.Size})
	cacheSrv := cachekit.NewCache(
		cachekit.NewSharedCache(ring),
		cachekit.NewLocalCache(env.LocalCacheConfig.Size),
	)

	mysqlCfg, err := mysqlkit.NewMySQLConfig(mysqlkit.MysqlConnConf{
		Protocol: env.MysqlConnConfig.Protocol,
		Host:     env.MysqlConnConfig.Host,
		Port:     env.MysqlConnConfig.Port,
		User:     env.MysqlConnConfig.User,
		Password: env.MysqlConnConfig.Password,
		DBName:   env.MysqlConnConfig.DBName,
		DSN:      env.MysqlConnConfig.DSN,
	})
	if err != nil {
		logkit.FatalV2(ctx, "init mysql config failed", err, nil)
	}

	db, err := mysqlkit.NewGORM(
		mysqlCfg,
		telemetry.NewNrTracer(
			mysqlCfg.DBName, // db name
			mysqlCfg.Addr,   // Addr is the name of the server hosting the datastore.
			"MySQL",         // product name: fixed string defined by new relic
		),
		metrickit.NewGORMTracer(metrickit.Setting{
			DBName: mysqlCfg.DBName,
			Metric: metrickit.New("gorm"),
		}),
	)
	if err != nil {
		logkit.FatalV2(ctx, "init gorm failed", err, nil)
	}

	// https://gorm.io/docs/generic_interface.html
	sqlDB, err := db.DB()
	if err != nil {
		logkit.FatalV2(ctx, "invalid db", err, nil)
	}
	defer sqlDB.Close()

	// db migration check
	logkit.Infof(ctx, "start migration")
	migrationKit := migrationkit.NewGooseMigrationKit(migrationkit.GooseMysqlDriver, migrationkit.GooseMigrationOpt{
		Dir:      "/database/migrations",
		DBString: mysqlCfg.FormatDSN(),
	})
	if err := migrationKit.Up(); err != nil {
		logkit.FatalV2(ctx, "db migration failed", err, nil)
	}
	migrationKit.Close()

	// init validator
	logkit.Infof(ctx, "init validator")
	validator := validatorkit.NewGoPlaygroundValidator()

	// init server base
	logkit.Infof(ctx, "init server")
	serv := rpc.NewGoAmazingServer(rpc.GoAmazingServerOpt{
		Validator: validator,
		RecordDao: dao.NewRecordDAO(db, cacheSrv),
	})

	// init service
	var wg sync.WaitGroup

	launchers := []*ServiceLauncher{
		NewGrpcSvcLauncher(env.GRPCAddr, serv),
		NewHttpSvcLauncher(env.HTTPAddr, serv),
	}

	logkit.Infof(ctx, "launching service")

	// launch service
	for i := range launchers {
		l := launchers[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := l.Run(); err != nil {
				logkit.FatalV2(ctx, "launcher failed", err, logkit.Payload{"labels": l.Labels})
			}
		}()
	}

	wg.Wait()
}

// NewGrpcSvcLauncher 3-1. You need add a gRPC listener and register the service.
func NewGrpcSvcLauncher(addr string, serv pb.GoAmazingServer) *ServiceLauncher {

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logkit.FatalV2(context.TODO(), "failed to start listen tpc", err, nil)
	}

	s := grpc.NewServer()

	pb.RegisterGoAmazingGrpcService(s, serv) // 3-2. Run "RegisterGoAmazingGrpcService"

	return &ServiceLauncher{
		Labels: []string{"grpc"},
		Run: func() error {
			return s.Serve(lis)
		},
	}
}

// NewHttpSvcLauncher 4-1. You need add a HTTP listener and register the service.
func NewHttpSvcLauncher(addr string, serv pb.GoAmazingServer) *ServiceLauncher {

	// TODO: move details into RegisterGoAmazingHttpService

	s := gin.New()
	s.Use(gin.Recovery())
	s.Use(metrickit.Middleware(metrickit.New("gin")))

	pb.RegisterGoAmazingHttpService(s, serv) // 4-2. Run "RegisterGoAmazingHttpService"

	return &ServiceLauncher{
		Labels: []string{"http"},
		Run: func() error {
			return s.Run(addr)
		},
	}
}
