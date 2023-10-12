[English](README.md) | 中文

# tRPC-Go CLS 远程日志插件

[![Go Reference](https://pkg.go.dev/badge/github.com/trpc-ecosystem/go-log-cls.svg)](https://pkg.go.dev/github.com/trpc-ecosystem/go-log-cls)
[![Go Report Card](https://goreportcard.com/badge/github.com/trpc.group/trpc-go/trpc-log-cls)](https://goreportcard.com/report/github.com/trpc.group/trpc-go/trpc-log-cls)
[![LICENSE](https://img.shields.io/github/license/trpc-ecosystem/go-log-cls.svg?style=flat-square)](https://github.com/trpc-ecosystem/go-log-cls/blob/main/LICENSE)
[![Releases](https://img.shields.io/github/release/trpc-ecosystem/go-log-cls.svg?style=flat-square)](https://github.com/trpc-ecosystem/go-log-cls/releases)
[![Docs](https://img.shields.io/badge/docs-latest-green)](http://test.trpc.group.woa.com/docs/)
[![Tests](https://github.com/trpc-ecosystem/go-log-cls/actions/workflows/prc.yaml/badge.svg)](https://github.com/trpc-ecosystem/go-log-cls/actions/workflows/prc.yaml)
[![Coverage](https://codecov.io/gh/trpc-ecosystem/go-log-cls/branch/main/graph/badge.svg)](https://app.codecov.io/gh/trpc-ecosystem/go-log-cls/tree/main)

插件封装了腾讯云 [CLS SDK](https://github.com/TencentCloud/tencentcloud-cls-sdk-go)，提供 tRPC-Go 日志插件，可以让你的 tRPC-Go 服务快速接入 CLS 日志系统。

## 完整配置

```yaml
plugins:
  log: #日志配置 支持多个日志 可通过 log.Get("xxx").Debug 打日志
    default: #默认日志的配置，每个日志可支持多输出
      - writer: cls #cls远程日志输出
        level: debug #远程日志的级别
        remote_config: #远程日志配置
          topic_id: b0179d73-8932-4a96-a1df-xxxxxx #cls日志主题id
          host: ap-guangzhou.cls.tencentyun.com #cls日志上报域名
          secret_id: AKIDRefNpzzYcOf7HFsj8Kxxxxxxx #腾讯云secret_id
          secret_key: jK4ZJIMEuV3IHy49zYq2yxxxxxxx #腾讯云secret_key
          total_size_ln_bytes: 104857600 #[可选,100 * 1024 * 1024] 实例能缓存的日志大小上限
          max_send_worker_count: 50 #[可选,50] 并发的最多"goroutine"的数量
          max_block_sec: 0 #[可选,0] send 方法上的最大阻塞时间，默认0不阻塞
          max_batch_size: 5242880 #[可选,5 * 1024 * 1024] Batch中缓存的日志大小大于等于MaxBatchSize时，该batch将被发
          max_batch_count: 4096 #[可选,4096] Batch中缓存的日志条数大于等于 MaxBatchCount 时，该batch将被发送
          linger_ms: 2000 #[可选,2 * 1000] Batch从创建到可发送的逗留时间
          retries: 10 #[可选,10] 如果某个Batch首次发送失败，能够对其重试的次数
          max_reserved_attempts: 11 #[可选,11] 每个Batch每次被尝试发送都对应着一个Attempt，此参数用来控制返回给用户的 attempt 个数
          base_retry_backoff_ms: 100 #[可选,100] 首次重试的退避时间
          max_retry_backoff_ms: 50000 #[可选,50 * 1000] 重试的最大退避时间
          source: 127.0.0.1 #[可选] 默认使用trpc全局local_ip,service监听ip
          field_map: #[可选,默认不设置] 自定义上报字段映射
            Level: log_level #将Level字段映射为log_level并上报
            field1: test_field #将field1字段映射为test_field字段并上报
```

## 快速上手

### 1 申请 cls 日志 topic

- 网址：https://console.cloud.tencent.com/cls/overview?region=ap-guangzhou

### 2 根据上文完整配置，配置 trpc_go.yaml

### 3 开发代码

- 首先需要 import 本插件

```golang
    import _ "trpc.group/trpc-go/trpc-log-cls"
```

- 打印日志

```golang
    log.WithFields("key1","value1").Info("message1")
    log.Warn("warning message1")
```

- 如果需要对上报的字段进行重新映射，可以通过配置的 field_map 进行设置

```yaml
field_map:                    #[可选,默认不设置] 自定义上报字段映射
  Msg: log_content            #将Msg字段映射为log_content并上报
  Caller: file_line           #将Caller字段映射为file_line并上报
  Level: log_level            #将Level字段映射为log_level并上报
  Time: ts                    #将Time字段映射为ts并上报
  key1: test_field            #将key1字段映射为test_field并上报
  ...
```

若需要更复杂的字段重映射或过滤，可以通过重写`cls.GetReportCLSField`实现，比如为了规范上报到远程日志的字段都采用小写+下划线风格：

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

### 4 查看远程日志

- https://console.cloud.tencent.com/cls/search
