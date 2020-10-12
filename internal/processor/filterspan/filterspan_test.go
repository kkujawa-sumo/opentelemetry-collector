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

package filterspan

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterconfig"
	"go.opentelemetry.io/collector/internal/processor/filterset"
)

func createConfig(matchType filterset.MatchType) *filterset.Config {
	return &filterset.Config{
		MatchType: matchType,
	}
}

func TestSpan_validateMatchesConfiguration_InvalidConfig(t *testing.T) {
	testcases := []struct {
		name        string
		property    filterconfig.MatchProperties
		errorString string
	}{
		{
			name:        "empty_property",
			property:    filterconfig.MatchProperties{},
			errorString: "at least one of \"services\", \"span_names\" or \"attributes\" field must be specified",
		},
		{
			name: "empty_service_span_names_and_attributes",
			property: filterconfig.MatchProperties{
				Services: []string{},
			},
			errorString: "at least one of \"services\", \"span_names\" or \"attributes\" field must be specified",
		},
		{
			name: "log_properties",
			property: filterconfig.MatchProperties{
				LogNames: []string{"log"},
			},
			errorString: "log_names should not be specified for trace spans",
		},
		{
			name: "invalid_match_type",
			property: filterconfig.MatchProperties{
				Config:   *createConfig("wrong_match_type"),
				Services: []string{"abc"},
			},
			errorString: "error creating service name filters: unrecognized match_type: 'wrong_match_type', valid types are: [regexp strict]",
		},
		{
			name: "missing_match_type",
			property: filterconfig.MatchProperties{
				Services: []string{"abc"},
			},
			errorString: "error creating service name filters: unrecognized match_type: '', valid types are: [regexp strict]",
		},
		{
			name: "regexp_match_type_for_attributes",
			property: filterconfig.MatchProperties{
				Config: *createConfig(filterset.Regexp),
				Attributes: []filterconfig.Attribute{
					{Key: "key", Value: "value"},
				},
			},
			errorString: `match_type=regexp is not supported for "attributes"`,
		},
		{
			name: "invalid_regexp_pattern",
			property: filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Regexp),
				Services: []string{"["},
			},
			errorString: "error creating service name filters: error parsing regexp: missing closing ]: `[`",
		},
		{
			name: "invalid_regexp_pattern2",
			property: filterconfig.MatchProperties{
				Config:    *createConfig(filterset.Regexp),
				SpanNames: []string{"["},
			},
			errorString: "error creating span name filters: error parsing regexp: missing closing ]: `[`",
		},
		{
			name: "empty_key_name_in_attributes_list",
			property: filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{"a"},
				Attributes: []filterconfig.Attribute{
					{
						Key: "",
					},
				},
			},
			errorString: "error creating processor. Can't have empty key in the list of attributes",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := NewMatcher(&tc.property)
			assert.Nil(t, output)
			require.NotNil(t, err)
			assert.Equal(t, tc.errorString, err.Error())
		})
	}
}

func TestSpan_Matching_False(t *testing.T) {
	testcases := []struct {
		name       string
		properties *filterconfig.MatchProperties
	}{
		{
			name: "service_name_doesnt_match_regexp",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Regexp),
				Services:   []string{"svcA"},
				Attributes: []filterconfig.Attribute{},
			},
		},

		{
			name: "service_name_doesnt_match_strict",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Strict),
				Services:   []string{"svcA"},
				Attributes: []filterconfig.Attribute{},
			},
		},

		{
			name: "span_name_doesnt_match",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Regexp),
				SpanNames:  []string{"spanNo.*Name"},
				Attributes: []filterconfig.Attribute{},
			},
		},

		{
			name: "span_name_doesnt_match_any",
			properties: &filterconfig.MatchProperties{
				Config: *createConfig(filterset.Regexp),
				SpanNames: []string{
					"spanNo.*Name",
					"non-matching?pattern",
					"regular string",
				},
				Attributes: []filterconfig.Attribute{},
			},
		},

		{
			name: "wrong_property_value",
			properties: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{},
				Attributes: []filterconfig.Attribute{
					{
						Key:   "keyInt",
						Value: 1234,
					},
				},
			},
		},
		{
			name: "incompatible_property_value",
			properties: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{},
				Attributes: []filterconfig.Attribute{
					{
						Key:   "keyInt",
						Value: "123",
					},
				},
			},
		},
		{
			name: "property_key_does_not_exist",
			properties: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{},
				Attributes: []filterconfig.Attribute{
					{
						Key:   "doesnotexist",
						Value: nil,
					},
				},
			},
		},
	}

	span := pdata.NewSpan()
	span.InitEmpty()
	span.SetName("spanName")
	span.Attributes().InitFromMap(map[string]pdata.AttributeValue{"keyInt": pdata.NewAttributeValueInt(123)})
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			matcher, err := NewMatcher(tc.properties)
			assert.Nil(t, err)
			assert.NotNil(t, matcher)

			assert.False(t, matcher.MatchSpan(span, "wrongSvc"))
		})
	}
}

func TestSpan_MatchingCornerCases(t *testing.T) {
	cfg := &filterconfig.MatchProperties{
		Config:   *createConfig(filterset.Strict),
		Services: []string{"svcA"},
		Attributes: []filterconfig.Attribute{
			{
				Key:   "keyOne",
				Value: nil,
			},
		},
	}

	mp, err := NewMatcher(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, mp)

	emptySpan := pdata.NewSpan()
	emptySpan.InitEmpty()
	assert.False(t, mp.MatchSpan(emptySpan, "svcA"))
}

func TestSpan_MissingServiceName(t *testing.T) {
	cfg := &filterconfig.MatchProperties{
		Config:   *createConfig(filterset.Regexp),
		Services: []string{"svcA"},
	}

	mp, err := NewMatcher(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, mp)

	emptySpan := pdata.NewSpan()
	emptySpan.InitEmpty()
	assert.False(t, mp.MatchSpan(emptySpan, ""))
}

func TestSpan_Matching_True(t *testing.T) {
	testcases := []struct {
		name       string
		properties *filterconfig.MatchProperties
	}{
		{
			name: "service_name_match_regexp",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Regexp),
				Services:   []string{"svcA"},
				Attributes: []filterconfig.Attribute{},
			},
		},
		{
			name: "service_name_match_strict",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Strict),
				Services:   []string{"svcA"},
				Attributes: []filterconfig.Attribute{},
			},
		},
		{
			name: "span_name_match",
			properties: &filterconfig.MatchProperties{
				Config:     *createConfig(filterset.Regexp),
				SpanNames:  []string{"span.*"},
				Attributes: []filterconfig.Attribute{},
			},
		},
		{
			name: "span_name_second_match",
			properties: &filterconfig.MatchProperties{
				Config: *createConfig(filterset.Regexp),
				SpanNames: []string{
					"wrong.*pattern",
					"span.*",
					"yet another?pattern",
					"regularstring",
				},
				Attributes: []filterconfig.Attribute{},
			},
		},
		{
			name: "property_exact_value_match",
			properties: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{},
				Attributes: []filterconfig.Attribute{
					{
						Key:   "keyString",
						Value: "arithmetic",
					},
					{
						Key:   "keyInt",
						Value: 123,
					},
					{
						Key:   "keyDouble",
						Value: 3245.6,
					},
					{
						Key:   "keyBool",
						Value: true,
					},
				},
			},
		},
		{
			name: "property_exists",
			properties: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{"svcA"},
				Attributes: []filterconfig.Attribute{
					{
						Key:   "keyExists",
						Value: nil,
					},
				},
			},
		},
		{
			name: "match_all_settings_exists",
			properties: &filterconfig.MatchProperties{
				Config:   *createConfig(filterset.Strict),
				Services: []string{"svcA"},
				Attributes: []filterconfig.Attribute{
					{
						Key:   "keyExists",
						Value: nil,
					},
					{
						Key:   "keyString",
						Value: "arithmetic",
					},
				},
			},
		},
	}

	span := pdata.NewSpan()
	span.InitEmpty()
	span.SetName("spanName")
	span.Attributes().InitFromMap(map[string]pdata.AttributeValue{
		"keyString": pdata.NewAttributeValueString("arithmetic"),
		"keyInt":    pdata.NewAttributeValueInt(123),
		"keyDouble": pdata.NewAttributeValueDouble(3245.6),
		"keyBool":   pdata.NewAttributeValueBool(true),
		"keyExists": pdata.NewAttributeValueString("present"),
	})

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			mp, err := NewMatcher(tc.properties)
			assert.Nil(t, err)
			assert.NotNil(t, mp)

			assert.NotNil(t, span)
			assert.True(t, mp.MatchSpan(span, "svcA"))
		})
	}
}

func TestSpan_validateMatchesConfigurationForAttributes(t *testing.T) {
	testcase := []struct {
		name   string
		input  filterconfig.MatchProperties
		output Matcher
	}{
		{
			name: "attributes_build",
			input: filterconfig.MatchProperties{
				Config: *createConfig(filterset.Strict),
				Attributes: []filterconfig.Attribute{
					{
						Key: "key1",
					},
					{
						Key:   "key2",
						Value: 1234,
					},
				},
			},
			output: &propertiesMatcher{
				Attributes: []attributeMatcher{
					{
						Key: "key1",
					},
					{
						Key:            "key2",
						AttributeValue: newAttributeValueInt(1234),
					},
				},
			},
		},

		{
			name: "both_set_of_attributes",
			input: filterconfig.MatchProperties{
				Config: *createConfig(filterset.Strict),
				Attributes: []filterconfig.Attribute{
					{
						Key: "key1",
					},
					{
						Key:   "key2",
						Value: 1234,
					},
				},
			},
			output: &propertiesMatcher{
				Attributes: []attributeMatcher{
					{
						Key: "key1",
					},
					{
						Key:            "key2",
						AttributeValue: newAttributeValueInt(1234),
					},
				},
			},
		},
	}
	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			output, err := NewMatcher(&tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.output, output)
		})
	}
}

func newAttributeValueInt(v int64) *pdata.AttributeValue {
	attr := pdata.NewAttributeValueInt(v)
	return &attr
}
