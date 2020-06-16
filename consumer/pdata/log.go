// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pdata

import (
	"github.com/gogo/protobuf/proto"

	logsproto "go.opentelemetry.io/collector/internal/data/opentelemetry-proto-gen/logs/v1"
)

// NewResourceLogsSliceFromOrig creates ResourceLogsSlice from logsproto.ResourceLogs.
// This function simply makes generated newResourceLogsSlice() function publicly
// available for internal.data.Log to call. We intentionally placed data.Log in the
// internal package so that it is not available publicly while it is experimental.
// Once the expriment is over data.Log should move to this package (pdata) and
// NewResourceLogsSliceFromOrig function will no longer be needed.
func NewResourceLogsSliceFromOrig(orig *[]*logsproto.ResourceLogs) ResourceLogsSlice {
	return ResourceLogsSlice{orig}
}


// This file defines in-memory data structures to represent logs.

// Logs is the top-level struct that is propagated through the logs pipeline.
//
// This is a reference type (like builtin map).
//
// Must use NewLogs functions to create new instances.
// Important: zero-initialized instance is not valid for use.
type Logs struct {
	orig *[]*logsproto.ResourceLogs
}

// LogsFromProto creates the internal Logs representation from the ProtoBuf.
func LogsFromProto(orig []*logsproto.ResourceLogs) Logs {
	return Logs{&orig}
}

// LogsToProto converts the internal Logs to the ProtoBuf.
func LogsToProto(ld Logs) []*logsproto.ResourceLogs {
	return *ld.orig
}

// NewLogs creates a new Logs.
func NewLogs() Logs {
	orig := []*logsproto.ResourceLogs(nil)
	return Logs{&orig}
}

// Clone returns a copy of Logs.
func (ld Logs) Clone() Logs {
	otlp := LogsToProto(ld)
	resourceSpansClones := make([]*logsproto.ResourceLogs, 0, len(otlp))
	for _, resourceSpans := range otlp {
		resourceSpansClones = append(resourceSpansClones,
			proto.Clone(resourceSpans).(*logsproto.ResourceLogs))
	}
	return LogsFromProto(resourceSpansClones)
}

// LogRecordCount calculates the total number of log records.
func (ld Logs) LogRecordCount() int {
	logCount := 0
	rss := ld.ResourceLogs()
	for i := 0; i < rss.Len(); i++ {
		rs := rss.At(i)
		if rs.IsNil() {
			continue
		}
		logCount += rs.Logs().Len()
	}
	return logCount
}

func (ld Logs) ResourceLogs() ResourceLogsSlice {
	return NewResourceLogsSliceFromOrig(ld.orig)
}
