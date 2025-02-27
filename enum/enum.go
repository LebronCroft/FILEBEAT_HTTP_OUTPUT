package enum

import "sync"

var FileBeatConfigFile = "/filebeat.yml"
var AgentIDAddress = "/etc/one-agent/machine-id"
var ModuleFileBeatConfigFile = "/modules.d/"
var Mu sync.Mutex // 全局锁
