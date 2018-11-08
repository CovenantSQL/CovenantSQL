/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"time"

	"github.com/sirupsen/logrus"
)

// Entry defines alias for logrus entry.
type Entry logrus.Entry

// NewEntry returns new entry for logrus logger.
func NewEntry(logger *Logger) *Entry {
	return &Entry{
		Logger: (*logrus.Logger)(logger),
		// Default is five fields, give a little extra room
		Data: make(logrus.Fields, 5),
	}
}

// Returns the string representation from the reader and ultimately the formatter.
func (entry *Entry) String() (string, error) {
	return (*logrus.Entry)(entry).String()
}

// WithError adds an error as single field (using the key defined in ErrorKey) to the Entry.
func (entry *Entry) WithError(err error) *Entry {
	return (*Entry)((*logrus.Entry)(entry).WithError(err))
}

// WithField add a single field to the Entry.
func (entry *Entry) WithField(key string, value interface{}) *Entry {
	return (*Entry)((*logrus.Entry)(entry).WithField(key, value))
}

// WithFields add a map of fields to the Entry.
func (entry *Entry) WithFields(fields Fields) *Entry {
	return (*Entry)((*logrus.Entry)(entry).WithFields((logrus.Fields)(fields)))
}

// WithTime overrides the time of the Entry.
func (entry *Entry) WithTime(t time.Time) *Entry {
	return &Entry{Logger: entry.Logger, Data: entry.Data, Time: t}
}

// Debug record a new debug level log.
func (entry *Entry) Debug(args ...interface{}) {
	(*logrus.Entry)(entry).Debug(args...)
}

// Print record a new non-level log.
func (entry *Entry) Print(args ...interface{}) {
	(*logrus.Entry)(entry).Print(args...)
}

// Info record a new info level log.
func (entry *Entry) Info(args ...interface{}) {
	(*logrus.Entry)(entry).Info(args...)
}

// Warn record a new warning level log.
func (entry *Entry) Warn(args ...interface{}) {
	(*logrus.Entry)(entry).Warn(args...)
}

// Warning record a new warning level log.
func (entry *Entry) Warning(args ...interface{}) {
	(*logrus.Entry)(entry).Warning(args...)
}

// Error record a new error level log.
func (entry *Entry) Error(args ...interface{}) {
	(*logrus.Entry)(entry).Error(args...)
}

// Fatal record a fatal level log.
func (entry *Entry) Fatal(args ...interface{}) {
	(*logrus.Entry)(entry).Fatal(args...)
}

// Panic record a panic level log.
func (entry *Entry) Panic(args ...interface{}) {
	(*logrus.Entry)(entry).Panic(args...)
}

// Entry Printf family functions

// Debugf record a debug level log.
func (entry *Entry) Debugf(format string, args ...interface{}) {
	(*logrus.Entry)(entry).Debugf(format, args...)
}

// Infof record a info level log.
func (entry *Entry) Infof(format string, args ...interface{}) {
	(*logrus.Entry)(entry).Infof(format, args...)
}

// Printf record a new non-level log.
func (entry *Entry) Printf(format string, args ...interface{}) {
	(*logrus.Entry)(entry).Printf(format, args...)
}

// Warnf record a warning level log.
func (entry *Entry) Warnf(format string, args ...interface{}) {
	(*logrus.Entry)(entry).Warnf(format, args...)
}

// Warningf record a warning level log.
func (entry *Entry) Warningf(format string, args ...interface{}) {
	(*logrus.Entry)(entry).Warningf(format, args...)
}

// Errorf record a error level log.
func (entry *Entry) Errorf(format string, args ...interface{}) {
	(*logrus.Entry)(entry).Errorf(format, args...)
}

// Fatalf record a fatal level log.
func (entry *Entry) Fatalf(format string, args ...interface{}) {
	(*logrus.Entry)(entry).Fatalf(format, args...)
}

// Panicf record a panic level log.
func (entry *Entry) Panicf(format string, args ...interface{}) {
	(*logrus.Entry)(entry).Panicf(format, args...)
}

// Entry Println family functions

// Debugln record a debug level log.
func (entry *Entry) Debugln(args ...interface{}) {
	(*logrus.Entry)(entry).Debugln(args...)
}

// Infoln record a info level log.
func (entry *Entry) Infoln(args ...interface{}) {
	(*logrus.Entry)(entry).Infoln(args...)
}

// Println record a non-level log.
func (entry *Entry) Println(args ...interface{}) {
	(*logrus.Entry)(entry).Println(args...)
}

// Warnln record a warning level log.
func (entry *Entry) Warnln(args ...interface{}) {
	(*logrus.Entry)(entry).Warnln(args...)
}

// Warningln record a warning level log.
func (entry *Entry) Warningln(args ...interface{}) {
	(*logrus.Entry)(entry).Warningln(args...)
}

// Errorln record a error level log.
func (entry *Entry) Errorln(args ...interface{}) {
	(*logrus.Entry)(entry).Errorln(args...)
}

// Fatalln record a fatal level log.
func (entry *Entry) Fatalln(args ...interface{}) {
	(*logrus.Entry)(entry).Fatalln(args...)
}

// Panicln record a panic level log.
func (entry *Entry) Panicln(args ...interface{}) {
	(*logrus.Entry)(entry).Panicln(args...)
}
