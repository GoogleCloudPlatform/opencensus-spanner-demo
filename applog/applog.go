// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package applog

/**
  Implment trace-log correlation with Stackdriver Logging and OpenCensus.
 **/

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/logging"
	"go.opencensus.io/trace"
)

const LOGNAME string = "oc-spannerlab"

var (
	client    *logging.Client
	projectId string
)

// Close and flush the logging client
func Close() {
	err := client.Close()
	if err != nil {
		fmt.Printf("Failed to close logging client: %v", err)
	}
}

// Log an error with the given context, may include trace and span
func Errorf(ctx context.Context, format string, v ...interface{}) {
	printf(ctx, logging.Error, format, v...)
}

// Log a fatal error with the given context, may include trace and span
func Fatalf(ctx context.Context, format string, v ...interface{}) {
	printf(ctx, logging.Critical, format, v...)
	log.Fatalf(format, v...)
}

// Initialize the Cloud Logging client
func Initialize(projId string) {
	projectId = projId
	ctx := context.Background()
	var err error
	client, err = logging.NewClient(ctx, projId)
	if err != nil {
		fmt.Printf("Failed to create logging client: %v", err)
		return
	}
	fmt.Printf("Stackdriver Logging initialized with project id %s, see Cloud " +
		" Console under GCE VM instance > all instance_id\n", projectId)
}

// Send to Cloud Logging service including reference to current span
func Printf(ctx context.Context, format string, v ...interface{}) {
	printf(ctx, logging.Info, format, v...)
}

// Send to Cloud Logging service including reference to current span
// [START spannerlab_trace_correlation]
func printf(ctx context.Context, severity logging.Severity, format string,
	v ...interface{}) {
	span := trace.FromContext(ctx)
	if client == nil {
		log.Printf(format, v...)
	} else if span == nil {
		lg := client.Logger(LOGNAME)
		lg.Log(logging.Entry{
			Severity: severity,
			Payload:  fmt.Sprintf(format, v...),
		})
	} else {
		sCtx := span.SpanContext()
		tr := sCtx.TraceID.String()
		lg := client.Logger(LOGNAME)
		trace := fmt.Sprintf("projects/%s/traces/%s", projectId, tr)
		lg.Log(logging.Entry{
			Severity: severity,
			Payload:  fmt.Sprintf(format, v...),
			Trace:    trace,
			SpanID:   sCtx.SpanID.String(),
		})
	}
}

// [END spannerlab_trace_correlation]
