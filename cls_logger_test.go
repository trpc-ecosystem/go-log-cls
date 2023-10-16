//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package cls

import (
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	cloudsdk "github.com/tencentcloud/tencentcloud-cls-sdk-go"
	"gopkg.in/yaml.v3"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/log"
)

func TestSetup(t *testing.T) {
	p := &LoggerPlugin{}
	err := p.Setup("", nil)
	if err == nil {
		t.Errorf("setup err:%+v", err)
	}

	patch := gomonkey.ApplyMethod(reflect.TypeOf(p), "SetupCls", func(*LoggerPlugin, *log.OutputConfig) (*Logger, error) {
		return &Logger{}, nil
	})
	err = p.Setup("", &log.Decoder{
		OutputConfig: &log.OutputConfig{
			FormatConfig: log.FormatConfig{},
			RemoteConfig: yaml.Node{
				Kind: yaml.ScalarNode,
			},
		},
	})
	if err != nil {
		t.Errorf("setup err:%+v", err)
	}
	patch.Reset()

	producerConfig := cloudsdk.GetDefaultAsyncProducerClientConfig()
	producerConfig.Endpoint = "ap-guangzhou.cls.tencentcs.com"
	producerConfig.AccessKeyID = "11"
	producerConfig.AccessKeySecret = "11"
	producerInstance, err := cloudsdk.NewAsyncProducerClient(producerConfig)
	if err != nil {
		t.Error(err)
	}
	patch = gomonkey.ApplyFunc(cloudsdk.NewAsyncProducerClient, func(*cloudsdk.AsyncProducerClientConfig) (*cloudsdk.AsyncProducerClient, error) {
		return &cloudsdk.AsyncProducerClient{}, nil
	})
	patch = gomonkey.ApplyMethod(reflect.TypeOf(producerInstance), "Start", func(*cloudsdk.AsyncProducerClient) {
		return
	})
	_, err = p.SetupCls(&log.OutputConfig{
		FormatConfig: log.FormatConfig{},
		RemoteConfig: yaml.Node{
			Kind: yaml.ScalarNode,
		},
	})
	if err != nil {
		t.Errorf("setupCls err:%+v", err)
	}
	patch.Reset()

	_ = p.Type()
}

func TestWrite(t *testing.T) {
	log := &Logger{}
	producerConfig := cloudsdk.GetDefaultAsyncProducerClientConfig()
	producerConfig.Endpoint = "ap-guangzhou.cls.tencentcs.com"
	producerConfig.AccessKeyID = "11"
	producerConfig.AccessKeySecret = "11"
	producerInstance, err := cloudsdk.NewAsyncProducerClient(producerConfig)
	if err != nil {
		t.Error(err)
	}
	log.client = producerInstance

	patch := gomonkey.ApplyMethod(reflect.TypeOf(log.client), "SendLog", func(*cloudsdk.AsyncProducerClient, string, *cloudsdk.Log, cloudsdk.CallBack) error {
		return nil
	})
	info := []byte(`{"L":"INFO","T":"2020-12-08 11:32:48.051","C":"app/main.go:37","M":"test info:"}`)
	n, err := log.Write(info)
	if err != nil {
		t.Errorf("Write err:%+v", err)
	}
	if n <= 0 {
		t.Errorf("Write err")
	}
	patch.Reset()

	patch = gomonkey.ApplyMethod(reflect.TypeOf(log.client), "SendLog", func(*cloudsdk.AsyncProducerClient, string, *cloudsdk.Log, cloudsdk.CallBack) error {
		return errs.New(1, "Over producer set maximum blocking time")
	})
	info1 := []byte(`{"L":"INFO","Time":"2020-12-08 11:32:48.051","C":"app/main.go:37","M":"test info"}`)
	_, err = log.Write(info1)
	if err == nil {
		t.Errorf("Write err:%+v", err)
	}

	infoe2 := []byte(`{}`)
	_, err = log.Write(infoe2)
	if err != nil {
		t.Errorf("Write err:%+v", err)
	}

	infoe3 := []byte(``)
	_, err = log.Write(infoe3)
	if err != nil {
		t.Errorf("Write err:%+v", err)
	}

}

func TestDefaultGetReportCLSField(t *testing.T) {
	cfg := Config{
		FieldMap: map[string]string{
			"Level": "level",
		},
	}
	tests := []struct {
		name            string
		sourceField     string
		wantReportField string
		wantNeedReport  bool
	}{
		{name: "remap field", sourceField: "Level", wantReportField: "level", wantNeedReport: true},
		{name: "no remap", sourceField: "field", wantReportField: "field", wantNeedReport: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotReportField, gotNeedReport := GetReportCLSField(tt.sourceField, &cfg)
			if gotReportField != tt.wantReportField {
				t.Errorf("GetReportCLSField() gotReportField = %v, want %v", gotReportField, tt.wantReportField)
			}
			if gotNeedReport != tt.wantNeedReport {
				t.Errorf("GetReportCLSField() gotNeedReport = %v, want %v", gotNeedReport, tt.wantNeedReport)
			}
		})
	}
}

func TestCallback(t *testing.T) {
	var c Callback
	c.Success(&cloudsdk.Result{})
	c.Fail(&cloudsdk.Result{})
}
