package grok_exporter //nolint:golint

import (
	v3 "github.com/fstab/grok_exporter/config/v3"
	"github.com/fstab/grok_exporter/tailer"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCustomHandlersForWebhookInputType(t *testing.T) {
	webhookTailer := tailer.InitWebhookTailer(&v3.InputConfig{})
	e := &Exporter{
		config: &Config{
			GrokConfig: v3.Config{
				Input: v3.InputConfig{
					Type:        "webhook",
					WebhookPath: "/test_webhook",
				},
			},
		},
	}

	handlers := e.CustomHandlers()

	assert.Equal(t, webhookTailer, handlers["/integrations/grok_exporter/test_webhook"])
}

func TestCustomHandlersForNonWebhookInputType(t *testing.T) {
	e := &Exporter{
		config: &Config{
			GrokConfig: v3.Config{
				Input: v3.InputConfig{
					Type: "file",
				},
			},
		},
	}

	handlers := e.CustomHandlers()

	assert.Nil(t, handlers["/integrations/grok_exporter/test_webhook"])
}

func TestScrapeConfigs(t *testing.T) {
	e := &Exporter{config: &Config{}}

	scrapeConfigs := e.ScrapeConfigs()

	assert.Equal(t, "grok_exporter", scrapeConfigs[0].JobName)
	assert.Equal(t, "/metrics", scrapeConfigs[0].MetricsPath)
}

func TestInitializingGrokExporter(t *testing.T) {
	logger := log.NewNopLogger()
	config := &Config{
		GrokConfig: v3.Config{
			Global: v3.GlobalConfig{
				ConfigVersion:          3,
				RetentionCheckInterval: 10,
			},
			Input: v3.InputConfig{
				Type: "file",
			},
			GrokPatterns: []string{"HOUR (?:2[0123]|[01]?[0-9])"},
			AllMetrics: v3.MetricsConfig{{
				Type:          "counter",
				Name:          "test_counter",
				Help:          "test_counter",
				PathsAndGlobs: v3.PathsAndGlobs{},
				Match:         "%{HOUR}",
			}},
		},
	}

	exporter, err := New(logger, config)

	assert.NoError(t, err)
	assert.Equal(t, config, exporter.(*Exporter).config)
	assert.Equal(t, logger, exporter.(*Exporter).logger)
	assert.Equal(t, 1, len(exporter.(*Exporter).metrics))
	if _, ok := exporter.(*Exporter).metrics[0].Collector().(prometheus.Counter); !ok {
		assert.Fail(t, "Failed to create counter metric")
	}

	nLinesTotal, err := exporter.(*Exporter).selfMetrics.nLinesTotal.GetMetricWithLabelValues("status")
	assert.NoError(t, err)
	assert.Equal(t, "Desc{fqName: \"grok_exporter_lines_total\", help: \"Total number of logger lines processed by grok_exporter.\", constLabels: {}, variableLabels: [status]}", nLinesTotal.Desc().String())

	nMatchesByMetric, err := exporter.(*Exporter).selfMetrics.nMatchesByMetric.GetMetricWithLabelValues("metric")
	assert.NoError(t, err)
	assert.Equal(t, "Desc{fqName: \"grok_exporter_lines_matching_total\", help: \"Number of lines matched for each metric. Note that one line can be matched by multiple metrics.\", constLabels: {}, variableLabels: [metric]}", nMatchesByMetric.Desc().String())

	procTimeMicrosecondsByMetric, err := exporter.(*Exporter).selfMetrics.procTimeMicrosecondsByMetric.GetMetricWithLabelValues("status")
	assert.NoError(t, err)
	assert.Equal(t, "Desc{fqName: \"grok_exporter_lines_processing_time_microseconds_total\", help: \"Processing time in microseconds for each metric. Divide by grok_exporter_lines_matching_total to get the averge processing time for one logger line.\", constLabels: {}, variableLabels: [metric]}", procTimeMicrosecondsByMetric.Desc().String())

	nErrorsByMetric, err := exporter.(*Exporter).selfMetrics.nErrorsByMetric.GetMetricWithLabelValues("status")
	assert.NoError(t, err)
	assert.Equal(t, "Desc{fqName: \"grok_exporter_line_processing_errors_total\", help: \"Number of errors for each metric. If this is > 0 there is an error in the configuration file. Check grok_exporter's console output.\", constLabels: {}, variableLabels: [metric]}", nErrorsByMetric.Desc().String())

	assert.NotNil(t, exporter.(*Exporter).logTailer)
	assert.NotNil(t, exporter.(*Exporter).retentionTicker)
}

func TestInitializingGrokExporterMetricError(t *testing.T) {
	logger := log.NewNopLogger()
	config := &Config{
		GrokConfig: v3.Config{
			Global: v3.GlobalConfig{
				ConfigVersion:          3,
				RetentionCheckInterval: 10,
			},
			Input: v3.InputConfig{
				Type: "file",
			},
			GrokPatterns: []string{"HOUR (?:2[0123]|[01]?[0-9])"},
			AllMetrics: v3.MetricsConfig{{
				Type:          "invalid",
				Name:          "test_counter",
				Help:          "test_counter",
				PathsAndGlobs: v3.PathsAndGlobs{},
				Match:         "%{HOUR}",
			}},
		},
	}

	exporter, err := New(logger, config)

	assert.Error(t, err)
	assert.Equal(t, "failed to initialize metrics: Metric type invalid is not supported", err.Error())
	assert.Nil(t, exporter)
}

func TestInitializingGrokExporterTailerError(t *testing.T) {
	logger := log.NewNopLogger()
	config := &Config{
		GrokConfig: v3.Config{
			Global: v3.GlobalConfig{
				ConfigVersion:          3,
				RetentionCheckInterval: 10,
			},
			Input: v3.InputConfig{
				Type: "invalid",
			},
			GrokPatterns: []string{"HOUR (?:2[0123]|[01]?[0-9])"},
			AllMetrics: v3.MetricsConfig{{
				Type:          "counter",
				Name:          "test_counter",
				Help:          "test_counter",
				PathsAndGlobs: v3.PathsAndGlobs{},
				Match:         "%{HOUR}",
			}},
		},
	}

	exporter, err := New(logger, config)

	assert.Error(t, err)
	assert.Equal(t, "config error: Input type 'invalid' unknown", err.Error())
	assert.Nil(t, exporter)
}

