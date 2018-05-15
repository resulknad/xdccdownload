package main

import (
    "encoding/json"
    "io/ioutil"
    "os"
    "path"
    "fmt"
	"golang.org/x/net/proxy" 
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
	Proxy string
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

func (c *Config) GetProxyDial() *proxy.Dialer {
	if c.Proxy == "" {
		return nil
	}
	proxyDial, err := proxy.SOCKS5("tcp", c.Proxy, nil, proxy.Direct)
	if err != nil {
		return nil
	}
	return &proxyDial
}

func (c *Config) GetDirs() []string {
	dirs := []string{}
	for _,d := range(c.TargetPaths) {
		dirs = append(dirs,d.Dir)
	}
	return dirs
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
