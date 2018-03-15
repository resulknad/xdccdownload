package main

import (
    "encoding/json"
    "io/ioutil"
    "os"
    "path"
    "fmt"
)

type ChannelConfig struct {
    Server string
    Channel string
}

type TargetPath struct {
	Type string
	Dir string
}

type ConfigStruct struct {
    Port string
    TempPath string
    TargetPaths []TargetPath
    Channels []ChannelConfig
	SpeedLimit int
	ParallelDownloads int
	LogDir string
}

type Config struct {
    ConfigStruct
}

func (c *Config) GetTargetDir(pType string) string {
	for _,d := range(c.TargetPaths) {
		if d.Type == pType {
			return d.Dir
		}
	}
	return c.TargetPaths[0].Dir
}

func (c *Config) SaveConfig() {
    configJson, _ := json.Marshal((c))
    p := path.Join(os.Getenv("HOME"), ".config.json")
    ioutil.WriteFile(p, configJson, 0644)
}

func (c *Config) LoadConfig() {
    p := path.Join(os.Getenv("HOME"), ".config.json")
    content, _ := ioutil.ReadFile(p)
    json.Unmarshal(content, c)
    fmt.Println(c)
}
