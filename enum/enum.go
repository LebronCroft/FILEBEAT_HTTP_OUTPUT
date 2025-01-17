package enum

// 模块默认路径映射表
var DefaultPaths = map[string]map[string][]string{
	"nginx": {
		"access": {"/var/log/nginx/access.log*"},
		"error":  {"/var/log/nginx/error.log*"},
	},
	"mysql": {
		"slowlog":  {"/var/log/mysql/mysql-slow.log*"},
		"errorlog": {"/var/log/mysql/error.log*"},
	},
	"redis": {
		"slowlog": {"/var/log/redis/redis-slowlog.log*"},
		"general": {"/var/log/redis/redis.log*"},
	},
	"apache": {
		"access": {"/var/log/apache2/access.log*"},
		"error":  {"/var/log/apache2/error.log*"},
	},
	"mongodb": {
		"logs": {"/var/log/mongodb/mongodb.log"},
	},
	"kafka": {
		"log": {
			"/var/log/kafka/controller.log*",
			"/var/log/kafka/server.log*",
			"/var/log/kafka/state-change.log*",
			"/var/log/kafka/kafka-*.log*",
		},
	},
	"zookeeper": {
		"audit": {"/var/log/zookeeper/zookeeper_audit.log*"},
		"log":   {"/var/log/zookeeper/zookeeper.log*"},
	},
	"rabbitmq": {
		"audit": {"/var/log/rabbitmq/*.log*"},
	},
	"postgresql": {
		"log": {"/var/log/postgres/*.log*"},
	},
	"logstash": {
		"log":     {"/var/log/logstash/logstash.log*"},
		"slowlog": {"/var/log/logstash/logstash-slowlog.log*"},
	},
}
