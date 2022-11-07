package config

import (
	"encoding/json"

	"github.com/AmazingTalker/go-rpc-kit/configkit"
)

const (
	dynamicCfgPath = "go-amazing/rpc/dynamic_config.json"
)

var (
	dynamicConfig = DynamicConfig{}
)

type DynamicConfig struct {
	Enable bool   `json:"enable,omitempty"`
	Num    int64  `json:"num,omitempty"`
	Str    string `json:"str,omitempty"`
}

func init() {
	configkit.Register(dynamicCfgPath, &dynamicConfig)
}

func (c *DynamicConfig) Check(data []byte) (interface{}, []string, error) {
	cfg := DynamicConfig{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, nil, err
	}

	return cfg, nil, nil
}

func (c *DynamicConfig) Apply(v interface{}) {
	*c = v.(DynamicConfig)
}

func Config() DynamicConfig {
	return dynamicConfig
}
