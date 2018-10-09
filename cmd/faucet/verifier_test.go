/*
 * Copyright 2018 The CovenantSQL Authors.
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

package main

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestVerifyFacebook(t *testing.T) {
	Convey("", t, func() {
		var err error
		err = verifyFacebook("https://www.facebook.com/hupili/posts/1700877176661446",
			[]string{"xxx", "Initium Media"}, "https://github.com/initiumlab/beijinguprooted")
		So(err, ShouldBeNil)
		err = verifyFacebook("facebook.com/dualipaofficial/posts/1832797603472815",
			[]string{"xxx", "ELECTRICITY"}, "http://www.baidu.com")
		So(err, ShouldNotBeNil)
		err = verifyFacebook("facebook.com/dualipaofficial/posts/1832797603472815",
			[]string{"xxx", "哈哈"}, "http://smarturl.it/SilkCityElectricity/youtube")
		So(err, ShouldNotBeNil)
	})
}

func TestVerifyTwitter(t *testing.T) {
	Convey("", t, func() {
		var err error
		err = verifyTwitter("https://twitter.com/tualatrix/status/1040460103898394624",
			[]string{"xxx", "好奇心日报"}, "http://m.qdaily.com")
		So(err, ShouldBeNil)
		err = verifyTwitter("https://twitter.com/Fenng/status/1040487918995791873",
			[]string{"xxx", "阿里巴巴"}, "http://www.baidu.com")
		So(err, ShouldNotBeNil)
		err = verifyTwitter("https://twitter.com/Fenng/status/1040487918995791873",
			[]string{"xxx", "百度"}, "https://twitter.com")
		So(err, ShouldNotBeNil)
	})
}

func TestVerifyWeibo(t *testing.T) {
	Convey("", t, func() {
		var err error
		err = verifyWeibo("https://weibo.com/2104296457/GzhcXuPNB",
			[]string{"xxx", "Mavic"}, "https://www.chiphell.com")
		So(err, ShouldBeNil)
		err = verifyWeibo("https://weibo.com/2104296457/Gz8vO2gOc",
			[]string{"xxx", "卡西欧"}, "http://www.baidu.com")
		So(err, ShouldNotBeNil)
		err = verifyWeibo("https://weibo.com/2104296457/Gz8vO2gOc",
			[]string{"xxx", "哈哈"}, "https://www.chiphell.com")
		So(err, ShouldNotBeNil)
	})
}
