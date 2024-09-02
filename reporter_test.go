// SPDX-License-Identifier: MPL-2.0
/*
 * Copyright (C) 2024 The Noisy Sockets Authors.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */
package telemetry_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/dpeckett/telemetry"
	"github.com/dpeckett/telemetry/v1alpha1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReporter(t *testing.T) {
	// Start a mock telemetry server.
	server, eventCh := mockTelemetryServer(t)
	t.Cleanup(server.Close)

	// Create a new telemetry reporter.
	conf := telemetry.Configuration{
		BaseURL:   server.URL,
		AuthToken: "test-token",
		Tags:      []string{"test-tag"},
	}

	ctx := context.Background()
	reporter := telemetry.NewReporter(ctx, slog.Default(), conf)

	// Report a telemetry event.
	reporter.ReportEvent(&v1alpha1.TelemetryEvent{
		Name: "TestEvent",
		Values: map[string]string{
			"key1": "value1",
		},
	})

	select {
	case event := <-eventCh:
		// Assert that the event received by the server matches the event sent.
		assert.Equal(t, "TestEvent", event.Name)
		assert.Equal(t, "value1", event.Values["key1"])
		assert.Equal(t, "test-tag", event.Tags[0])
		assert.NotEmpty(t, event.SessionID, "SessionID should be set")
		assert.NotNil(t, event.Timestamp, "Timestamp should be set")

	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for telemetry event")
	}

	// Shutdown the reporter to ensure graceful exit.
	require.NoError(t, reporter.Shutdown(ctx))
}

func TestReporter_DoNotTrack(t *testing.T) {
	// Set the DO_NOT_TRACK environment variable.
	os.Setenv("DO_NOT_TRACK", "1")
	t.Cleanup(func() {
		os.Unsetenv("DO_NOT_TRACK")
	})

	// Start a mock telemetry server.
	server, eventCh := mockTelemetryServer(t)
	t.Cleanup(server.Close)

	// Create a new telemetry reporter.
	conf := telemetry.Configuration{
		BaseURL:   server.URL,
		AuthToken: "test-token",
		Tags:      []string{"test-tag"},
	}

	ctx := context.Background()
	reporter := telemetry.NewReporter(ctx, slog.Default(), conf)

	// Report a telemetry event.
	reporter.ReportEvent(&v1alpha1.TelemetryEvent{
		Name: "TestEvent",
		Values: map[string]string{
			"key1": "value1",
		},
	})

	// Set a timeout for receiving the event and expect no event to be received.
	select {
	case event := <-eventCh:
		t.Fatalf("Expected no telemetry event, but got: %v", event)

	case <-time.After(100 * time.Millisecond):
		// Expected timeout as no event should be received.
	}

	// Shutdown the reporter to ensure graceful exit.
	err := reporter.Shutdown(ctx)
	require.NoError(t, err)
}

func mockTelemetryServer(t *testing.T) (*httptest.Server, chan *v1alpha1.TelemetryEvent) {
	eventCh := make(chan *v1alpha1.TelemetryEvent, 1)

	// Create a mock server to handle incoming telemetry events.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		require.Equal(t, "/v1alpha1/events", r.URL.Path)
		require.Equal(t, "POST", r.Method)

		var event v1alpha1.TelemetryEvent
		require.NoError(t, json.NewDecoder(r.Body).Decode(&event))

		eventCh <- &event

		w.WriteHeader(http.StatusOK)
	}))

	return server, eventCh
}
