/*
 * Copyright 2019 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mux

import (
	"net"
	"testing"

	"github.com/pkg/errors"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/CovenantSQL/CovenantSQL/rpc"
)

type testService struct{}

func (s *testService) Test(req *int, resp *int) (err error) {
	*resp = *req + 1
	return
}

func (s *testService) TestFailed(req *int, resp *interface{}) (err error) {
	return errors.New("failed")
}

func (s *testService) TestReconnect(req *int, resp *interface{}) (err error) {
	return errors.New("shut down")
}

func TestRawCaller(t *testing.T) {
	Convey("test raw caller methods", t, func() {
		s := NewServer().WithAcceptConnFunc(rpc.AcceptRawConn)
		err := s.RegisterService("Test", &testService{})
		So(err, ShouldBeNil)
		l, err := net.Listen("tcp", ":0")
		So(err, ShouldBeNil)
		s.SetListener(l)
		go s.Serve()
		defer s.Stop()
		c := NewRawCaller(l.Addr().String())
		defer c.Close()
		var resp int
		err = c.Call("Test.Test", 1, &resp)
		So(err, ShouldBeNil)
		So(resp, ShouldEqual, 2)
		err = c.Call("Test.TestFailed", 1, nil)
		So(err, ShouldNotBeNil)
		So(errors.Cause(err).Error(), ShouldEqual, "failed")
		err = c.Call("Test.TestReconnect", 1, nil)
		So(err, ShouldNotBeNil)
		So(err.Error(), ShouldContainSubstring, "shut down")
		So(c.Target(), ShouldEqual, l.Addr().String())
		err = c.Call("Test.Test", 2, &resp)
		So(err, ShouldBeNil)
		So(resp, ShouldEqual, 3)

		// test new client
		c2 := c.New()
		defer c2.Close()
		err = c2.Call("Test.Test", 4, &resp)
		So(err, ShouldBeNil)
		So(resp, ShouldEqual, 5)
	})
}
