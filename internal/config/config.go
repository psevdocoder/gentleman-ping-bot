package config

import "git.server.lan/pkg/config/realtimeconfig"

type (
	configKey         realtimeconfig.Key
	realtimeConfigKey realtimeconfig.Key
)

const (
	// CurlFile File location with copied from DevTools cURL request for sending message
	CurlFile configKey = "values.curl_file"
	// CronExpr cron expr
	CronExpr realtimeConfigKey = "realtime_config.cron_expr"
	// MessageText Текст сообщения
	MessageText realtimeConfigKey = "realtime_config.message_text"
	// Markup Задает форматирование????
	Markup realtimeConfigKey = "realtime_config.markup"
	// ChatId Определяет, кому слать сообщение
	ChatId realtimeConfigKey = "realtime_config.chat_id"
	// SendEnabled Включает или выключает отправку сообщений
	SendEnabled realtimeConfigKey = "realtime_config.send_enabled"
)

func GetValue[T configKey | realtimeConfigKey](key T) (realtimeconfig.Value, error) {
	return realtimeconfig.Get(realtimeconfig.Key(key))
}

func Watch(key realtimeConfigKey, callback realtimeconfig.WatchCallback) {
	realtimeconfig.Watch(realtimeconfig.Key(key), callback)
}
