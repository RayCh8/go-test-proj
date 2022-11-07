package main

import (
	"github.com/AmazingTalker/go-rpc-kit/flagkit"
	"github.com/AmazingTalker/go-rpc-kit/logkit"
)

type AirbrakeConfig struct {
	ProjectID  int64  `long:"projectId" default:"0" env:"PROJECT_ID"`
	ProjectKey string `long:"projectKey" default:"" env:"PROJECT_KEY"`
}

type LoggerConfig struct {
	Level       logkit.LoggerLevel `long:"level" description:"set log level" default:"info" env:"LEVEL"`
	Development bool               `long:"development" description:"enable development mode" env:"DEVELOPMENT"`
	Airbrake    AirbrakeConfig     `group:"airbrake" namespace:"airbrake" env-namespace:"AIRBRAKE"`
}

type EnvConfig struct {
	Namespace string `long:"namespace" description:"Environment namespace. ex: Prod, Stag, Dev" env:"NAMESPACE"`
	PodName   string `long:"pod" description:"pod name or host name in k8s" env:"POD_NAME"`
	JObName   string `long:"service" description:"service name" env:"JOB_NAME"` // !!
}

type MonitorConfig struct {
	PeriodSecs uint `long:"period-in-secs" description:"period of collecting in seconds" default:"10" env:"PERIOD_SECONDS"`
}

type MetricConfig struct {
	URL            string  `long:"url" description:"metric URL" env:"URL"`
	APIKey         string  `long:"apikey" description:"api key" env:"API_KEY"`
	RefleshSeconds float64 `long:"refleshseconds" description:"reflesh seconds" env:"REFLESH_SECONDS"`
}

var env struct {
	LoggerConfig  `group:"logger" namespace:"logger" env-namespace:"LOGGER"`
	EnvConfig     `group:"env" namespace:"env" env-namespace:"ENV"`
	MetricConfig  `group:"metric" namespace:"metric" env-namespace:"METRIC"`
	MonitorConfig `group:"monitor" namespace:"monitor" env-namespace:"MONITOR"`
}

func init() {
	flagkit.Register(&env)
}
