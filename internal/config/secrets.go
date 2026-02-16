package config

import "git.server.lan/pkg/config/realtimeconfig"

type secretKey realtimeconfig.Key

const ()

func GetSecret(key secretKey) (realtimeconfig.Value, error) {
	return realtimeconfig.Get(realtimeconfig.Key(key))
}
