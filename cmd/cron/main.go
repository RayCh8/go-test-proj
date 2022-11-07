package main

import (
	"context"
	"os"
	"time"

	"github.com/AmazingTalker/go-amazing/pkg/cronjob"
	"github.com/AmazingTalker/go-rpc-kit/envkit"
	"github.com/AmazingTalker/go-rpc-kit/flagkit"
	"github.com/AmazingTalker/go-rpc-kit/logkit"
	"github.com/AmazingTalker/go-rpc-kit/metrickit"
	"github.com/AmazingTalker/go-rpc-kit/monitorkit"
)

func main() {

	// get env
	if err := flagkit.Parse(); err != nil {
		panic(err)
	}
	// register env first, it will benefit ctx, logkit, and metric
	envkit.Register(envkit.Config{
		EnvNamespace: env.EnvConfig.Namespace,
		PodName:      env.EnvConfig.PodName,
		ServiceName:  env.EnvConfig.JObName,
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

	// Here is the job get started
	if err := cronjob.Execute(ctx); err != nil {
		logkit.ErrorV2(ctx, "Cronjob executed failed", err, nil)
	}

	os.Exit(0)
}
