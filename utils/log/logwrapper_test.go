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
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func init() {
	AddHook(&CallerHook{})
}

func TestStandardLogger(t *testing.T) {
	SetLevel(DebugLevel)
	if GetLevel() != DebugLevel {
		t.Fail()
	}
	Debug("Debug")
	Debugln("Debugln")
	Debugf("Debugf %d", 1)
	Print("Print")
	Println("Println")
	Printf("Printf %d", 1)
	Info("Info")
	Infoln("Infoln")
	Infof("Infof %d", 1)
	Warning("Warning")
	Warningln("Warningln")
	Warningf("Warningf %d", 1)
	Warn("Warn")
	Warnln("Warnln")
	Warnln("Warnln")
	Warnf("Warnf %d", 1)
	Error("Error")
	Errorln("Errorln")
	Errorf("Errorf %d", 1)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered in f", r)
			}
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Recovered in f", r)
				}
				n := NilFormatter{}
				a, b := n.Format(&logrus.Entry{})
				if a != nil || b != nil {
					t.Fail()
				}
			}()
			Panicf("Panicf %d", 1)
		}()
		Panicln("Panicln")

	}()

	Panic("Panic")

}

func call0() {
	call1()
}

func call1() {
	call2()
}

func call2() {
	WithField("k", "v").Error("error")
	Error("call2 error")
}

func TestWithField(t *testing.T) {
	SetLevel(DebugLevel)
	if GetLevel() != DebugLevel {
		t.Fail()
	}

	call0()

	WithError(errors.New("new")).WithField("newfieldkey", "newfieldvalue").WithTime(time.Now()).Debug("debug")
	f := new(Fields)
	WithError(errors.New("new")).WithFields(*f).WithTime(time.Now()).Debug("debug")

	WithFields(*f).Debug("debug")
	WithTime(time.Now()).WithError(errors.New("new")).Debug("debug")
	entry := NewEntry(StandardLogger())
	entry.WithTime(time.Now()).String()
	entry.Printf("entry printf %d", 1)

	WithField("k", "v").Debug("debug")
	WithField("k", "v").Debugln("Debugln")
	WithField("k", "v").Debugf("debugf %d", 1)
	WithField("k", "v").Print("Print")
	WithField("k", "v").Println("Println")
	WithField("k", "v").Printf("Printf %d", 1)
	WithField("k", "v").Info("info")
	WithField("k", "v").Infoln("Infoln")
	WithField("k", "v").Infof("infof %d", 1)
	WithField("k", "v").Warning("warning")
	WithField("k", "v").Warningln("Warningln")
	WithField("k", "v").Warningf("Warningf %d", 1)
	WithField("k", "v").Warn("warn")
	WithField("k", "v").Warnln("Warnln")
	WithField("k", "v").Warnln("Warnln")
	WithField("k", "v").Warnf("warnf %d", 1)
	WithField("k", "v").Error("error")
	WithField("k", "v").Errorln("Errorln")
	WithField("k", "v").Errorf("errorf %d", 1)
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
		}
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered in f", r)
			}
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("Recovered in f", r)
				}
				n := NilFormatter{}
				a, b := n.Format(&logrus.Entry{})
				if a != nil || b != nil {
					t.Fail()
				}
			}()
			WithField("k", "v").Panicf("panicf %d", 1)
		}()
		WithField("k", "v").Panicln("Panicln")
	}()

	WithField("k", "v").Panic("panic")
}

func TestSimpleLog(t *testing.T) {
	SetStringLevel("error", ErrorLevel)
	if GetLevel() != ErrorLevel {
		t.Fail()
	}
	Debug("Debug")
	Debugln("Debugln")
	Debugf("Debugf %d", 1)
	logger := StandardLogger()
	logger.Printf("StandardLogger Printf %d", 1)

	SimpleLog = "Y"
	SetLevel(DebugLevel)
	Debug("Debug")
	Debugln("Debugln")
	Debugf("Debugf %d", 1)

	SimpleLog = "N"
	SetOutput(&NilWriter{})
	SetFormatter(&NilFormatter{})
	Debug("Debug")
	Debugln("Debugln")
	Debugf("Debugf %d", 1)
}

func TestFatalLog(t *testing.T) {
	SetStringLevel("willusenextparam", ErrorLevel)
	if GetLevel() != ErrorLevel {
		t.Fail()
	}

	logrus.StandardLogger().ExitFunc = func(code int) {}
	Fatal("Fatal")
	Fatalln("Fatalln")
	Fatalf("Fatalf %d", 1)

	entry := WithError(errors.New("new"))
	entry.Fatal("entry Fatal")
	entry.Fatalln("entry Fatalln")
	entry.Fatalf("entry Fatal %d", 1)

	logrus.StandardLogger().ExitFunc = nil
}
