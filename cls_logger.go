// Package cls is tencent cloud cls log plugin for trpc-go
package cls

import (
	"errors"
	"time"

	clssdk "github.com/tencentcloud/tencentcloud-cls-sdk-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/metrics"
	"trpc.group/trpc-go/trpc-go/plugin"
)

const (
	pluginName = "cls"
	pluginType = "log"
)

const (
	sourceDefault      = "default_source"
	logTimeKey         = "Time"
	logTimeTemplate    = "2006-01-02 15:04:05"
	metricsBufferFull  = "cls.BufferFull"
	metricsSendFail    = "cls.SendFail"
	metricsSendSuccess = "cls.SendSuccess"
)

// LoggerPlugin is CLS logger plugin.
type LoggerPlugin struct {
}

// Type returns the type of logger plugin.
func (lp *LoggerPlugin) Type() string {
	return pluginType
}

func init() {
	log.RegisterWriter(pluginName, &LoggerPlugin{})
}

// Config is the configuration for the cls log plugin.
type Config struct {
	// TopicID is the log reporting topic ID.
	TopicID string `yaml:"topic_id"`
	// Host is the log reporting host.
	Host string `yaml:"host"`
	// SecretID is the log reporting secret ID used when calling Tencent Cloud APIs.
	SecretID string `yaml:"secret_id"`
	// SecretKey is the log reporting secret key used when calling Tencent Cloud APIs.
	SecretKey string `yaml:"secret_key"`
	// TotalSizeLnBytes is the maximum log size limit that an instance can cache, default is 100MB.
	TotalSizeLnBytes int64 `yaml:"total_size_ln_bytes"`
	// MaxSendWorkerCount is the maximum number of goroutines that the client can use concurrently, default is 50.
	MaxSendWorkerCount int64 `yaml:"max_send_worker_count"`
	// MaxBlockSec is the maximum blocking time on the send method if available buffer space is insufficient, default is non-blocking.
	MaxBlockSec int `yaml:"max_block_sec"`
	// MaxBatchSize is the size at which a batch of cached logs will be sent when greater than or equal to MaxBatchSize, default is 512KB, with a maximum of 5MB.
	MaxBatchSize int64 `yaml:"max_batch_size"`
	// MaxBatchCount is the number of logs in a batch that triggers sending when greater than or equal to MaxBatchCount, default is 4096, with a maximum of 40960.
	MaxBatchCount int `yaml:"max_batch_count"`
	// LingerMs is the time a batch stays in an available state before being sent, default is 2 seconds, with a minimum of 100 milliseconds.
	LingerMs int64 `yaml:"linger_ms"`
	// Retries is the number of retries allowed if a batch fails to send on the first attempt, default is 10 times.
	Retries int `yaml:"retries"`
	// MaxReservedAttempts is the number of attempts retained for each batch, each send attempt corresponds to an attempt, this parameter controls the number of attempts returned to the user, default is to keep only the latest 11 attempt records.
	MaxReservedAttempts int `yaml:"max_reserved_attempts"`
	// BaseRetryBackoffMs is the initial backoff time for the first retry, default is 100 milliseconds. The client uses an exponential backoff algorithm, where the planned wait time for the Nth retry is baseRetryBackoffMs * 2^(N-1).
	BaseRetryBackoffMs int64 `yaml:"base_retry_backoff_ms"`
	// MaxRetryBackoffMs is the maximum backoff time for retries, default is 50 seconds.
	MaxRetryBackoffMs int64 `yaml:"max_retry_backoff_ms"`
	// FieldMap is the mapping of log reporting fields.
	FieldMap map[string]string `yaml:"field_map"`
	// Source is the log source, typically the machine's IP address.
	Source string `yaml:"source"`
}

// Setup setups the plugin.
func (lp *LoggerPlugin) Setup(name string, configDec plugin.Decoder) error {
	if configDec == nil {
		return errors.New("cls log writer decoder empty")
	}
	decoder, ok := configDec.(*log.Decoder)
	if !ok {
		return errors.New("cls log writer log decoder type invalid")
	}

	conf := &log.OutputConfig{}
	err := decoder.Decode(&conf)
	if err != nil {
		return err
	}

	clsLogger, err := lp.SetupCls(conf)
	if err != nil {
		return err
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "Time",
		LevelKey:       "Level",
		NameKey:        "Name",
		CallerKey:      "Caller",
		MessageKey:     "Msg",
		StacktraceKey:  "StackTrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     log.NewTimeEncoder(conf.FormatConfig.TimeFmt),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	encoder := zapcore.NewJSONEncoder(encoderCfg)

	zl := zap.NewAtomicLevelAt(log.Levels[conf.Level])
	decoder.Core = zapcore.NewCore(
		encoder,
		zapcore.AddSync(clsLogger),
		zl,
	)
	decoder.ZapLevel = zl

	return nil
}

// Logger is cls logger.
type Logger struct {
	clsConfig Config
	client    *clssdk.AsyncProducerClient
}

// SetupCls setups cls logger.
func (lp *LoggerPlugin) SetupCls(conf *log.OutputConfig) (*Logger, error) {
	var clsConfig Config
	err := conf.RemoteConfig.Decode(&clsConfig)
	if err != nil {
		return nil, err
	}
	clsConfig.withSourceDefault()

	cli, err := clssdk.NewAsyncProducerClient(&clssdk.AsyncProducerClientConfig{
		TotalSizeLnBytes:    clsConfig.TotalSizeLnBytes,
		MaxSendWorkerCount:  clsConfig.MaxSendWorkerCount,
		MaxBlockSec:         clsConfig.MaxBlockSec,
		MaxBatchSize:        clsConfig.MaxBatchSize,
		MaxBatchCount:       clsConfig.MaxBatchCount,
		LingerMs:            clsConfig.LingerMs,
		Retries:             clsConfig.Retries,
		MaxReservedAttempts: clsConfig.MaxReservedAttempts,
		BaseRetryBackoffMs:  clsConfig.BaseRetryBackoffMs,
		MaxRetryBackoffMs:   clsConfig.MaxRetryBackoffMs,
		Endpoint:            clsConfig.Host,
		AccessKeyID:         clsConfig.SecretID,
		AccessKeySecret:     clsConfig.SecretKey,
		Source:              clsConfig.Source,
		Timeout:             10000,
		IdleConn:            50,
	})
	if err != nil {
		return nil, err
	}
	cli.Start()

	logger := &Logger{
		clsConfig: clsConfig,
		client:    cli,
	}

	return logger, nil
}

// Write trpc写日志接口实现
func (l *Logger) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	body := make(map[string]string)
	if err := codec.Unmarshal(codec.SerializationTypeJSON, p, &body); err != nil {
		return 0, err
	}
	if len(body) <= 0 {
		return 0, nil
	}

	bodyRe := make(map[string]string)
	var lTime time.Time
	for key, value := range body {
		if key == logTimeKey {
			lTime, _ = time.ParseInLocation(logTimeTemplate, value, time.Local)
		}
		keyIn, needReport := GetReportCLSField(key, &l.clsConfig)
		if !needReport {
			continue
		}
		bodyRe[keyIn] = value
	}

	cLog := clssdk.NewCLSLog(lTime.UnixNano(), bodyRe)
	callBack := &Callback{}
	if err := l.client.SendLog(l.clsConfig.TopicID, cLog, callBack); err != nil {
		metrics.Counter(metricsBufferFull).Incr()
		return 0, err
	}

	return len(p), nil
}

// GetReportCLSField is used to override the logging fields reported to CLS.
// For example: log.WithFields("field1", "value1").Info("message1").
// In this case, the field reported to CLS is "field1," and you can override it using the
// GetReportCLSField function. The default implementation supports field mapping through
// framework configuration, as detailed in the README.md.
// Parameter sourceField represents the original field, such as "field1" in the example above,
// cfg is the configuration information for CLS framework. The returned reportField is
// the field after remapping for reporting to CLS. If needReport is false, it means that this
// field will be ignored and not reported to CLS.
var GetReportCLSField = func(sourceField string, cfg *Config) (reportField string, needReport bool) {
	if cfg == nil || len(cfg.FieldMap) == 0 {
		return sourceField, true
	}

	reportField, exist := cfg.FieldMap[sourceField]
	if !exist {
		reportField = sourceField
	}

	return reportField, true
}

// Callback provides callback function.
type Callback struct {
}

// Success is the success callback function.
func (callback *Callback) Success(result *clssdk.Result) {
	metrics.Counter(metricsSendSuccess).Incr()
}

// Fail is the failure callback function.
func (callback *Callback) Fail(result *clssdk.Result) {
	metrics.Counter(metricsSendFail).Incr()
}

// withSourceDefault sets default source.
func (c *Config) withSourceDefault() {
	if c.Source == "" {
		c.Source = trpc.GlobalConfig().Global.LocalIP
	}
	if c.Source == "" {
		if len(trpc.GlobalConfig().Server.Service) > 0 {
			c.Source = trpc.GlobalConfig().Server.Service[0].IP
		}
	}
	if c.Source == "" {
		c.Source = sourceDefault
	}
}
