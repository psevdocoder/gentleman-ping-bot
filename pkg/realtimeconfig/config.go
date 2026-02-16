package realtimeconfig

import (
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

type Key string

type Value struct {
	raw any
}

func (v Value) Int() (int, error)                { return toInt(v.raw) }
func (v Value) Int64() (int64, error)            { return toInt64(v.raw) }
func (v Value) Float32() (float32, error)        { return toFloat32(v.raw) }
func (v Value) Float64() (float64, error)        { return toFloat64(v.raw) }
func (v Value) Bool() (bool, error)              { return toBool(v.raw) }
func (v Value) String() (string, error)          { return toString(v.raw) }
func (v Value) Duration() (time.Duration, error) { return toDuration(v.raw) }

type WatchCallback func(newValue, oldValue Value)

const defaultConfigPath = "values/config.yaml"

var (
	callbacks  = make(map[Key][]WatchCallback)
	lastValues = make(map[Key]Value)
	mu         sync.RWMutex
)

func Watch(key Key, callback WatchCallback) {
	mu.Lock()
	defer mu.Unlock()
	callbacks[key] = append(callbacks[key], callback)
}

func StartWatching() error {
	return startWithPath(defaultConfigPath)
}

func Get(key Key) (Value, error) {
	return getFromPath(defaultConfigPath, key)
}

func startWithPath(path string) error {
	if err := loadInitial(path); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if err := watcher.Add(path); err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&(fsnotify.Write|fsnotify.Create) > 0 {
					checkForChanges(path)
				}
			case err := <-watcher.Errors:
				log.Printf("fsnotify error: %v", err)
			}
		}
	}()

	return nil
}

func getFromPath(path string, key Key) (Value, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Value{}, err
	}

	var full map[string]any
	if err := yaml.Unmarshal(data, &full); err != nil {
		return Value{}, err
	}

	keyStr := string(key)

	var prefix, section string

	switch {
	case strings.HasPrefix(keyStr, "realtime_config."):
		prefix = "realtime_config."
		section = "realtime_config"
	case strings.HasPrefix(keyStr, "values."):
		prefix = "values."
		section = "values"
	case strings.HasPrefix(keyStr, "secrets."):
		prefix = "secrets."
		section = "secrets"
	default:
		return Value{}, errors.New("unknown key prefix")
	}

	rawList, ok := full[section].([]any)
	if !ok {
		return Value{}, fmt.Errorf("invalid or missing %s section", section)
	}

	suffix := strings.TrimPrefix(keyStr, prefix)
	for _, item := range rawList {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, ok := entry["name"].(string)
		if ok && name == suffix {
			return Value{entry["value"]}, nil
		}
	}

	return Value{}, fmt.Errorf("key not found: %s", key)
}

func loadInitial(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var full map[string]any
	if err := yaml.Unmarshal(data, &full); err != nil {
		return err
	}
	rawList, ok := full["realtime_config"].([]any)
	if !ok {
		return errors.New("invalid or missing realtime_config section")
	}
	mu.Lock()
	defer mu.Unlock()
	for _, item := range rawList {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, ok := entry["name"].(string)
		if !ok {
			continue
		}
		val := entry["value"]
		lastValues[Key("realtime_config."+name)] = Value{val}
	}
	return nil
}

func checkForChanges(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("error reading config: %v", err)
		return
	}
	var full map[string]any
	if err := yaml.Unmarshal(data, &full); err != nil {
		log.Printf("error parsing yaml: %v", err)
		return
	}
	rawList, ok := full["realtime_config"].([]any)
	if !ok {
		log.Printf("invalid or missing realtime_config section")
		return
	}

	mu.Lock()
	defer mu.Unlock()
	for _, item := range rawList {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name, ok := entry["name"].(string)
		if !ok {
			continue
		}
		val := entry["value"]
		key := Key("realtime_config." + name)
		oldVal, exists := lastValues[key]
		v := Value{val}
		if !exists || !equal(oldVal.raw, val) {
			lastValues[key] = v
			for _, cb := range callbacks[key] {
				go cb(v, oldVal)
			}
		}
	}
}

func toInt(v any) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case int64:
		return int(val), nil
	case float64:
		return int(val), nil
	case string:
		return strconv.Atoi(val)
	default:
		return 0, fmt.Errorf("cannot convert %T to int", v)
	}
}

func toInt64(v any) (int64, error) {
	switch val := v.(type) {
	case int:
		return int64(val), nil
	case int64:
		return val, nil
	case float64:
		return int64(val), nil
	case string:
		return strconv.ParseInt(val, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", v)
	}
}

func toFloat32(v any) (float32, error) {
	f, err := toFloat64(v)
	return float32(f), err
}

func toFloat64(v any) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

func toBool(v any) (bool, error) {
	switch val := v.(type) {
	case bool:
		return val, nil
	case string:
		return strconv.ParseBool(val)
	default:
		return false, fmt.Errorf("cannot convert %T to bool", v)
	}
}

func toString(v any) (string, error) {
	switch val := v.(type) {
	case string:
		return val, nil
	case fmt.Stringer:
		return val.String(), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func toDuration(v any) (time.Duration, error) {
	switch val := v.(type) {
	case string:
		return time.ParseDuration(val)
	case int:
		return time.Duration(val), nil
	case int64:
		return time.Duration(val), nil
	case float64:
		return time.Duration(int64(val)), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to time.Duration", v)
	}
}

func equal(a, b any) bool {
	return reflect.DeepEqual(a, b)
}
