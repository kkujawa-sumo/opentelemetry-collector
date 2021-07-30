// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filterprocessor

import (
	"context"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filtermatcher"
	"go.opentelemetry.io/collector/internal/processor/filtermetric"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/internal/processor/filterspan"
	"go.opentelemetry.io/collector/model/pdata"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

type filterProcessor struct {
	cfg              *Config
	includeMetrics   filtermetric.Matcher
	excludeMetrics   filtermetric.Matcher
	includeAttribute filtermatcher.AttributesMatcher
	excludeAttribute filtermatcher.AttributesMatcher
	includeSpans     filterspan.Matcher
	excludeSpans     filterspan.Matcher
	logger           *zap.Logger
}

func newFilterMetricProcessor(logger *zap.Logger, cfg *Config) (*filterProcessor, error) {
	inc, includeAttr, err := createMetricsMatcher(cfg.Metrics.Include)
	if err != nil {
		return nil, err
	}

	exc, excludeAttr, err := createMetricsMatcher(cfg.Metrics.Exclude)
	if err != nil {
		return nil, err
	}

	includeMatchType := ""
	var includeExpressions []string
	var includeMetricNames []string
	var includeResourceAttributes []filterconfig.Attribute
	if cfg.Metrics.Include != nil {
		includeMatchType = string(cfg.Metrics.Include.MatchType)
		includeExpressions = cfg.Metrics.Include.Expressions
		includeMetricNames = cfg.Metrics.Include.MetricNames
		includeResourceAttributes = cfg.Metrics.Include.ResourceAttributes
	}

	excludeMatchType := ""
	var excludeExpressions []string
	var excludeMetricNames []string
	var excludeResourceAttributes []filterconfig.Attribute
	if cfg.Metrics.Exclude != nil {
		excludeMatchType = string(cfg.Metrics.Exclude.MatchType)
		excludeExpressions = cfg.Metrics.Exclude.Expressions
		excludeMetricNames = cfg.Metrics.Exclude.MetricNames
		excludeResourceAttributes = cfg.Metrics.Exclude.ResourceAttributes
	}

	logger.Info(
		"Metric filter configured",
		zap.String("include match_type", includeMatchType),
		zap.Strings("include expressions", includeExpressions),
		zap.Strings("include metric names", includeMetricNames),
		zap.Any("include metrics with resource attributes", includeResourceAttributes),
		zap.String("exclude match_type", excludeMatchType),
		zap.Strings("exclude expressions", excludeExpressions),
		zap.Strings("exclude metric names", excludeMetricNames),
		zap.Any("exclude metrics with resource attributes", excludeResourceAttributes),
	)

	return &filterProcessor{
		cfg:              cfg,
		includeMetrics:   inc,
		excludeMetrics:   exc,
		includeAttribute: includeAttr,
		excludeAttribute: excludeAttr,
		logger:           logger,
	}, nil
}

func newFilterSpanProcessor(logger *zap.Logger, cfg *Config) (*filterProcessor, error) {
	inc, err := createSpansMatcher(cfg.Spans.Include)
	if err != nil {
		return nil, err
	}

	exc, err := createSpansMatcher(cfg.Spans.Exclude)
	if err != nil {
		return nil, err
	}

	includeMatchType := ""
	var includeServices []string
	var includeSpanNames []string
	if cfg.Spans.Include != nil {
		includeMatchType = string(cfg.Spans.Include.MatchType)
		includeServices = cfg.Spans.Include.Services
		includeSpanNames = cfg.Spans.Include.SpanNames
	}

	excludeMatchType := ""
	var excludeServices []string
	var excludeSpanNames []string
	if cfg.Spans.Exclude != nil {
		excludeMatchType = string(cfg.Spans.Exclude.MatchType)
		excludeServices = cfg.Spans.Exclude.Services
		excludeSpanNames = cfg.Spans.Exclude.SpanNames
	}

	logger.Info(
		"Span filter configured",
		zap.String("includeSpans match_type", includeMatchType),
		zap.Strings("includeSpans services", includeServices),
		zap.Strings("includeSpans span names", includeSpanNames),
		zap.String("excludeSpans match_type", excludeMatchType),
		zap.Strings("excludeSpans services", excludeServices),
		zap.Strings("excludeSpans span names", excludeSpanNames),
	)

	return &filterProcessor{
		cfg:          cfg,
		includeSpans: inc,
		excludeSpans: exc,
		logger:       logger,
	}, nil
}

func createSpansMatcher(mp *filterconfig.MatchProperties) (filterspan.Matcher, error) {
	if mp == nil {
		return nil, nil
	}
	return filterspan.NewMatcher(mp)
}

func createMetricsMatcher(mp *filtermetric.MatchProperties) (filtermetric.Matcher, filtermatcher.AttributesMatcher, error) {
	// Nothing specified in configuration
	if mp == nil {
		return nil, nil, nil
	}
	var attributeMatcher filtermatcher.AttributesMatcher
	attributeMatcher, err := filtermatcher.NewAttributesMatcher(
		filterset.Config{
			MatchType:    filterset.MatchType(mp.MatchType),
			RegexpConfig: mp.RegexpConfig,
		},
		mp.ResourceAttributes,
	)
	if err != nil {
		return nil, attributeMatcher, err
	}

	nameMatcher, err := filtermetric.NewMatcher(mp)
	return nameMatcher, attributeMatcher, err
}

// ProcessMetrics filters the given metrics based off the filterProcessor's filters.
func (fmp *filterProcessor) ProcessMetrics(_ context.Context, pdm pdata.Metrics) (pdata.Metrics, error) {
	pdm.ResourceMetrics().RemoveIf(func(rm pdata.ResourceMetrics) bool {
		keepMetricsForResource := fmp.shouldKeepMetricsForResource(rm.Resource())
		if !keepMetricsForResource {
			return true
		}
		rm.InstrumentationLibraryMetrics().RemoveIf(func(ilm pdata.InstrumentationLibraryMetrics) bool {
			ilm.Metrics().RemoveIf(func(m pdata.Metric) bool {
				keep, err := fmp.shouldKeepMetric(m)
				if err != nil {
					fmp.logger.Error("shouldKeepMetric failed", zap.Error(err))
					// don't `return`, keep the metric if there's an error
				}
				return !keep
			})
			// Filter out empty InstrumentationLibraryMetrics
			return ilm.Metrics().Len() == 0
		})
		// Filter out empty ResourceMetrics
		return rm.InstrumentationLibraryMetrics().Len() == 0
	})
	if pdm.ResourceMetrics().Len() == 0 {
		return pdm, processorhelper.ErrSkipProcessingData
	}
	return pdm, nil
}

// ProcessTraces filters the given spans based off the filterProcessor's filters.
func (fmp *filterProcessor) ProcessTraces(_ context.Context, pdt pdata.Traces) (pdata.Traces, error) {
	pdt.ResourceSpans().RemoveIf(func(rs pdata.ResourceSpans) bool {
		resource := rs.Resource()
		rs.InstrumentationLibrarySpans().RemoveIf(func(ils pdata.InstrumentationLibrarySpans) bool {
			ils.Spans().RemoveIf(func(span pdata.Span) bool {
				return !fmp.shouldKeepSpan(span, resource, ils.InstrumentationLibrary())
			})
			// Filter out empty InstrumentationLibraryMetrics
			return ils.Spans().Len() == 0
		})
		// Filter out empty ResourceMetrics
		return rs.InstrumentationLibrarySpans().Len() == 0
	})
	if pdt.ResourceSpans().Len() == 0 {
		return pdt, processorhelper.ErrSkipProcessingData
	}
	return pdt, nil
}

func (fmp *filterProcessor) shouldKeepMetric(metric pdata.Metric) (bool, error) {
	if fmp.includeMetrics != nil {
		matches, err := fmp.includeMetrics.MatchMetric(metric)
		if err != nil {
			// default to keep if there's an error
			return true, err
		}
		if !matches {
			return false, nil
		}
	}

	if fmp.excludeMetrics != nil {
		matches, err := fmp.excludeMetrics.MatchMetric(metric)
		if err != nil {
			return true, err
		}
		if matches {
			return false, nil
		}
	}

	return true, nil
}

func (fmp *filterProcessor) shouldKeepMetricsForResource(resource pdata.Resource) bool {
	resourceAttributes := resource.Attributes()

	if fmp.includeMetrics != nil && fmp.includeAttribute != nil {
		matches := fmp.includeAttribute.Match(resourceAttributes)
		if !matches {
			return false
		}
	}

	if fmp.excludeMetrics != nil && fmp.excludeAttribute != nil {
		matches := fmp.excludeAttribute.Match(resourceAttributes)
		if matches {
			return false
		}
	}

	return true
}

func (fmp *filterProcessor) shouldKeepSpan(span pdata.Span, resource pdata.Resource, library pdata.InstrumentationLibrary) bool {
	if fmp.includeSpans != nil {
		matches := fmp.includeSpans.MatchSpan(span, resource, library)
		if !matches {
			return false
		}
	}

	if fmp.excludeSpans != nil {
		matches := fmp.excludeSpans.MatchSpan(span, resource, library)
		if matches {
			return false
		}
	}

	return true
}
