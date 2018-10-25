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
