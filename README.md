English | [中文](README_CN.md)

# tRPC-Go CLS Remote Logging Plugin

[![Go Reference](https://pkg.go.dev/badge/github.com/trpc-ecosystem/go-log-cls.svg)](https://pkg.go.dev/github.com/trpc-ecosystem/go-log-cls)
[![Go Report Card](https://goreportcard.com/badge/trpc.group/trpc-go/trpc-log-cls)](https://goreportcard.com/report/trpc.group/trpc-go/trpc-log-cls)
[![LICENSE](https://img.shields.io/badge/license-Apache--2.0-green.svg)](https://github.com/trpc-ecosystem/go-log-cls/blob/main/LICENSE)
[![Releases](https://img.shields.io/github/release/trpc-ecosystem/go-log-cls.svg?style=flat-square)](https://github.com/trpc-ecosystem/go-log-cls/releases)
[![Tests](https://github.com/trpc-ecosystem/go-log-cls/actions/workflows/prc.yaml/badge.svg)](https://github.com/trpc-ecosystem/go-log-cls/actions/workflows/prc.yaml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-log-cls/branch/main/graph/badge.svg)](https://app.codecov.io/gh/trpc-ecosystem/go-log-cls/tree/main)

This plugin encapsulates the [Tencent Cloud CLS SDK](https://github.com/TencentCloud/tencentcloud-cls-sdk-go) and provides a tRPC-Go logging plugin to quickly integrate your tRPC-Go service with the CLS logging system.

## Complete Configuration

```yaml
plugins:
  log: # Logging configuration, supports multiple logs, can log using log.Get("xxx").Debug
    default: # Default log configuration, multiple outputs are supported for each log
      - writer: cls # CLS remote log output
        level: debug # Log level for remote logging
        remote_config: # Remote log configuration
          topic_id: b0179d73-8932-4a96-a1df-xxxxxx # CLS log topic ID
          host: ap-guangzhou.cls.tencentyun.com # CLS log reporting domain
          secret_id: AKIDRefNpzzYcOf7HFsj8Kxxxxxxx # Tencent Cloud secret_id
          secret_key: jK4ZJIMEuV3IHy49zYq2yxxxxxxx # Tencent Cloud secret_key
          total_size_ln_bytes: 104857600 # [Optional, 100 * 1024 * 1024] Maximum log size that the instance can cache
          max_send_worker_count: 50 # [Optional, 50] Maximum number of "goroutines" for concurrency
          max_block_sec: 0 # [Optional, 0] Maximum blocking time on the send method, default is 0 (non-blocking)
          max_batch_size: 5242880 # [Optional, 5 * 1024 * 1024] When the log size cached in the Batch is greater than or equal to MaxBatchSize, the batch will be sent
          max_batch_count: 4096 # [Optional, 4096] When the number of logs cached in the Batch is greater than or equal to MaxBatchCount, the batch will be sent
          linger_ms: 2000 # [Optional, 2 * 1000] The time from batch creation to being able to send
          retries: 10 # [Optional, 10] Number of retries for a batch that failed to send for the first time
          max_reserved_attempts: 11 # [Optional, 11] Each attempt to send a batch corresponds to an attempt, and this parameter controls the number of attempts returned to the user
          base_retry_backoff_ms: 100 # [Optional, 100] Backoff time for the first retry
          max_retry_backoff_ms: 50000 # [Optional, 50 * 1000] Maximum backoff time for retries
          source: 127.0.0.1 # [Optional] Default to using trpc global local_ip, service listening IP
          field_map: # [Optional, not set by default] Custom field mapping for reporting
            Level: log_level # Map the Level field to log_level and report it
            field1: test_field # Map the field1 field to test_field and report it
```

## Get started

### 1. Apply for a CLS Log Topic

- Website: https://console.cloud.tencent.com/cls/overview?region=ap-guangzhou

### 2. Configure trpc_go.yaml according to the complete configuration mentioned above.

### 3. Develop Your Code

- First, import this plugin:

```golang
import _ "trpc.group/trpc-go/trpc-log-cls"
```

- Log your messages:

```golang
log.WithFields("key1", "value1").Info("message1")
log.Warn("warning message1")
```

- If you need to remap the fields to be reported, you can do so using the field_map configuration:

```yaml
field_map: # [Optional, not set by default] Custom field mapping for reporting
  Msg: log_content # Map the Msg field to log_content and report it
  Caller: file_line # Map the Caller field to file_line and report it
  Level: log_level # Map the Level field to log_level and report it
  Time: ts # Map the Time field to ts and report it
  key1: test_field # Map the key1 field to test_field and report it
  ...
```

If you need more complex field remapping or filtering, you can implement it by overriding `cls.GetReportCLSField`. For example, to standardize the fields reported to remote logging in lowercase with underscores:

```golang
import (
	...
    cls "trpc.group/trpc-go/trpc-log-cls"
	...
)

func init() {
	cls.GetReportCLSField = func(sourceField string, _ *cls.Config) (string, bool) {
		var output []rune
		for i, r := range sourceField {
			if i == 0 {
				output = append(output, unicode.ToLower(r))
				continue
			}

			if unicode.IsUpper(r) {
				output = append(output, '_')
			}
			output = append(output, unicode.ToLower(r))
		}

		return string(output), true
	}
}
```

### 4. View Remote Logs

- https://console.cloud.tencent.com/cls/search

## Copyright

The copyright notice pertaining to the Tencent code in this repo was previously in the name of “THL A29 Limited.”  That entity has now been de-registered.  You should treat all previously distributed copies of the code as if the copyright notice was in the name of “Tencent.”
