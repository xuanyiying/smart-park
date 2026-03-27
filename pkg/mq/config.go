package mq

type MQType string

const (
	MQTypeRedis    MQType = "redis"
	MQTypeNATS     MQType = "nats"
	MQTypeRocketMQ MQType = "rocketmq"
)

type Config struct {
	Type     MQType         `json:"type"`
	Redis    RedisConfig    `json:"redis"`
	NATS     NATSConfig     `json:"nats"`
	RocketMQ RocketMQConfig `json:"rocketmq"`
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	Stream   string `json:"stream"`
}

type NATSConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type RocketMQConfig struct {
	NameServer string `json:"nameServer"`
	Group      string `json:"group"`
	AccessKey  string `json:"accessKey"`
	SecretKey  string `json:"secretKey"`
	Namespace  string `json:"namespace"`
}
