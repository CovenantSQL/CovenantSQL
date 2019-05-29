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

package main

import (
	"expvar"
	"io/ioutil"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/conf"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var (
	diskUsageMetric = expvar.NewInt("service:miner:disk:usage")
)

func collectDiskUsage() (err error) {
	// run du on linux and mac
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return
	}

	if conf.GConf == nil || conf.GConf.Miner == nil || conf.GConf.Miner.RootDir == "" {
		log.Error("miner config is empty, disk usage report is disabled")
		return
	}

	duBin, err := exec.LookPath("du")
	if err != nil {
		log.WithError(err).Error("could not found du command")
		return
	}

	cmd := exec.Command(duBin, "-sk", conf.GConf.Miner.RootDir)
	duOutput, err := cmd.StdoutPipe()
	if err != nil {
		log.WithError(err).Error("could not get result of disk usage")
		return
	}

	err = cmd.Start()
	if err != nil {
		log.WithError(err).Error("could not start disk usage command")
		return
	}

	duResult, err := ioutil.ReadAll(duOutput)
	if err != nil {
		log.WithError(err).Error("get disk usage result failed")
		return
	}

	err = cmd.Wait()
	if err != nil {
		log.WithError(err).Error("run disk usage command failed")
		return
	}

	splitResult := strings.SplitN(string(duResult), "\t", 2)
	if len(splitResult) == 0 || len(strings.TrimSpace(splitResult[0])) == 0 {
		log.Error("could not get disk usage result")
		return
	}

	usedKiloBytes, err := strconv.ParseInt(strings.TrimSpace(splitResult[0]), 10, 64)
	if err != nil {
		log.WithError(err).Error("could not parse usage bytes result")
		return
	}

	diskUsageMetric.Set(usedKiloBytes)

	return
}
