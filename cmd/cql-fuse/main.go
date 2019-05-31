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

// Copyright 2015 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License. See the AUTHORS file
// for names of contributors.
//
// Author: Marc Berhault (marc@cockroachlabs.com)

// This is a simple fuse filesystem that stores all metadata and data
// in cockroach.
//
// Inode relationships are stored in the `namespace` table, and inodes
// themselves in the `inode` table.
//
// Data blocks are stored in the `block` table, indexed by inode ID
// and block number.
//
// Basic functionality is implemented, including:
// - mk/rm directory
// - create/rm files
// - read/write files
// - rename
// - symlinks
//
// WARNING: concurrent access on a single mount is fine. However,
// behavior is undefined (read broken) when mounted more than once at the
// same time. Specifically, read/writes will not be seen right away and
// may work on out of date information.
//
// One caveat of the implemented features is that handles are not
// reference counted so if an inode is deleted, all open file descriptors
// pointing to it become invalid.
//
// Some TODOs (definitely not a comprehensive list):
// - support basic attributes (mode, timestamps)
// - support other types: hard links
// - add ref counting (and handle open/release)
// - sparse files: don't store empty blocks
// - sparse files 2: keep track of holes

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/utils"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
)

var usage = func() {
	_, _ = fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "  %s -config <config> -dsn <dsn> -mount <mountpoint>\n\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var (
		configFile string
		dsn        string
		mountPoint string
		password   string
		readOnly   bool
	)
	flag.StringVar(&configFile, "config", "~/.cql/config.yaml", "Config file path")
	flag.StringVar(&mountPoint, "mount", "./", "Dir to mount")
	flag.StringVar(&dsn, "dsn", "", "Database url")
	flag.StringVar(&password, "password", "", "Master key password for covenantsql")
	flag.BoolVar(&readOnly, "readonly", false, "Mount read only volume")
	flag.Usage = usage
	flag.Parse()

	log.SetLevel(log.InfoLevel)

	configFile = utils.HomeDirExpand(configFile)

	err := client.Init(configFile, []byte(password))
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := client.ParseDSN(dsn)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("covenantsql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	if err := initSchema(db); err != nil {
		log.Fatal(err)
	}

	cfs := CFS{db}
	opts := make([]fuse.MountOption, 0, 5)
	opts = append(opts, fuse.FSName("CovenantFS"))
	opts = append(opts, fuse.Subtype("CovenantFS"))
	opts = append(opts, fuse.LocalVolume())
	opts = append(opts, fuse.VolumeName(cfg.DatabaseID))
	if readOnly {
		opts = append(opts, fuse.ReadOnly())
	}
	// Mount filesystem.
	c, err := fuse.Mount(
		mountPoint,
		opts...,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = c.Close()
	}()

	log.Infof("DB: %s mount on %s succeed", dsn, mountPoint)

	go func() {
		<-utils.WaitForExit()
		if err := fuse.Unmount(mountPoint); err != nil {
			log.Printf("Signal received, but could not unmount: %s", err)
		}
	}()

	// Serve root.
	err = fs.Serve(c, cfs)
	if err != nil {
		log.Fatal(err)
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		log.Fatal(err)
	}
}
