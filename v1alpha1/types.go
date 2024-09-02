// SPDX-License-Identifier: MPL-2.0
/*
 * Copyright (C) 2024 Damian Peckett <damian@pecke.tt>.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

package v1alpha1

import "time"

// TelemetryEventKind represents the kind of event.
type TelemetryEventKind string

const (
	// The event is informational.
	TelemetryEventKindInfo TelemetryEventKind = "info"
	// The event is a warning.
	TelemetryEventKindWarning TelemetryEventKind = "warning"
	// The event is an error.
	TelemetryEventKindError TelemetryEventKind = "error"
)

type TelemetryEvent struct {
	// The session ID associated with the event. The session id is short-lived and not persisted.
	// It is only used to link events together (as there might be a relationship between them).
	SessionID string `json:"session_id,omitempty"`
	// Timestamp when the event occurred.
	Timestamp *time.Time `json:"timestamp,omitempty"`
	// The kind of event.
	Kind TelemetryEventKind `json:"kind,omitempty"`
	// The name of the event.
	Name string `json:"name,omitempty"`
	// A message associated with the event.
	Message string `json:"message,omitempty"`
	// Any values associated with the event.
	Values map[string]string `json:"values,omitempty"`
	// If an error, the stack trace associated with the event.
	StackTrace []*StackFrame `json:"stack_trace,omitempty"`
	// A set of tags associated with the event.
	Tags []string `json:"tags,omitempty"`
}

type StackFrame struct {
	// The file name where the error occurred.
	File string `json:"file,omitempty"`
	// The name of the method where the error occurred.
	Function string `json:"function,omitempty"`
	// The line number in the file where the error occurred.
	Line int32 `json:"line,omitempty"`
	// The column number in the line where the error occurred.
	Column int32 `json:"column,omitempty"`
}
