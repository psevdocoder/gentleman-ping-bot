package config

import "github.com/psevdocoder/gentleman-ping-bot/pkg/realtimeconfig"

type secretKey realtimeconfig.Key

const ()

func GetSecret(key secretKey) (realtimeconfig.Value, error) {
	return realtimeconfig.Get(realtimeconfig.Key(key))
}
