package config

import "net"

type Config struct {
	Host        string
	Port        string
	DatabaseURL string
}

func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, c.Port)
}
