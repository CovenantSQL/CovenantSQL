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
	"os/signal"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/utils/log"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	_ "bazil.org/fuse/fs/fstestutil"
)

var usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s <db URL> <mountpoint>\n\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 2 {
		usage()
		os.Exit(2)
	}

	log.SetLevel(log.DebugLevel)
	var config, dsn, mountPoint, password string

	flag.StringVar(&config, "config", "./conf/config.yaml", "config file path")
	flag.StringVar(&mountPoint, "mount", "./", "dir to mount")
	flag.StringVar(&dsn, "dsn", "", "database url")
	flag.StringVar(&password, "password", "", "master key password for covenantsql")
	flag.Parse()

	err := client.Init(config, []byte(password))
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("covenantsql", dsn)
	if err != nil {
		log.Fatal(err)
	}

	defer func() { _ = db.Close() }()

	if err := initSchema(db); err != nil {
		log.Fatal(err)
	}

	cfs := CFS{db}
	// Mount filesystem.
	c, err := fuse.Mount(
		mountPoint,
		fuse.FSName("CovenantFS"),
		fuse.Subtype("CovenantFS"),
		fuse.LocalVolume(),
		fuse.VolumeName(""),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = c.Close()
	}()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		for range sig {
			if err := fuse.Unmount(mountPoint); err != nil {
				log.Printf("Signal received, but could not unmount: %s", err)
			} else {
				break
			}
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
