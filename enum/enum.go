package enum

import "sync"

var FileBeatConfigFile = "/filebeat.yml"
var Mu sync.Mutex // 全局锁
