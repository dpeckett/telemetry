// SPDX-License-Identifier: MPL-2.0
/*
 * Copyright (C) 2024 The Noisy Sockets Authors.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package telemetry

import (
	"context"
	_ "embed"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/dpeckett/telemetry/internal/util"
	"github.com/dpeckett/telemetry/v1alpha1"
	"golang.org/x/sync/errgroup"
)

const (
	// The environment variable name to disable telemetry reporting.
	doNotTrackEnvName = "DO_NOT_TRACK"
	// The maximum number of in-flight telemetry reports.
	maxConcurrentReports = 16
)

// Configuration is the telemetry reporter configuration.
type Configuration struct {
	// BaseURL is the telemetry server base URL.
	BaseURL string
	// AuthToken is the telemetry API auth bearer token.
	AuthToken string
	// Tags is a list of optional tags to include in all telemetry reports.
	Tags []string
	// HTTPClient is the optional HTTP client to use for telemetry reporting.
	HTTPClient *http.Client
}

// Reporter is a telemetry reporter.
type Reporter struct {
	logger       *slog.Logger
	client       *v1alpha1.TelemetryEventClient
	authToken    string
	sessionID    string
	tags         []string
	reportsCtx   context.Context
	reports      *errgroup.Group
	shuttingDown atomic.Bool
}

// NewReporter creates a new telemetry reporter.
func NewReporter(ctx context.Context, logger *slog.Logger, conf Configuration) *Reporter {
	httpClient := conf.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	reports, reportsCtx := errgroup.WithContext(ctx)
	reports.SetLimit(maxConcurrentReports)

	return &Reporter{
		logger:     logger,
		client:     v1alpha1.NewTelemetryEventClient(httpClient, conf.BaseURL),
		authToken:  conf.AuthToken,
		sessionID:  util.GenerateID(16),
		tags:       conf.Tags,
		reportsCtx: reportsCtx,
		reports:    reports,
	}
}

// Close aborts any ongoing telemetry reporting.
func (r *Reporter) Close() error {
	r.reports.Go(func() error {
		return context.Canceled
	})

	if err := r.reports.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the telemetry reporter.
func (r *Reporter) Shutdown(ctx context.Context) error {
	// Stop accepting new reports.
	r.shuttingDown.Store(true)

	reportsDone := make(chan error, 1)
	go func() {
		defer close(reportsDone)

		reportsDone <- r.reports.Wait()
	}()

	select {
	case <-ctx.Done():
		// Abort any ongoing reports.
		return r.Close()
	case err := <-reportsDone:
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}

		return nil
	}
}

// ReportEvent reports a telemetry event.
func (r *Reporter) ReportEvent(event *v1alpha1.TelemetryEvent) {
	if os.Getenv(doNotTrackEnvName) != "" {
		r.logger.Debug("Telemetry is disabled, dropping event")
		return
	}

	now := time.Now()
	event.Timestamp = &now

	if event.SessionID == "" {
		event.SessionID = r.sessionID
	}

	event.Tags = append(event.Tags, r.tags...)

	if r.shuttingDown.Load() {
		r.logger.Debug("Shutting down, dropping event")
		return
	}

	started := r.reports.TryGo(func() error {
		// Absolute maximum limit.
		ctx, cancel := context.WithTimeout(r.reportsCtx, 30*time.Second)
		defer cancel()

		if err := r.client.ReportEvent(ctx, event); err != nil {
			// Don't spam the logs when the user is offline.
			r.logger.Debug("Failed to report event", slog.Any("error", err))
		}

		return nil
	})
	if !started {
		r.logger.Warn("Too many in-flight telemetry reports, dropping event")
	}
}
