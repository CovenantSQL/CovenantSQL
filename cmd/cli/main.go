/*
 * Copyright 2016-2018 Kenneth Shaw.
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
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"regexp"
	"strconv"
	"strings"

	"github.com/CovenantSQL/CovenantSQL/client"
	"github.com/CovenantSQL/CovenantSQL/utils/log"
	"github.com/CovenantSQL/go-sqlite3-encrypt"
	"github.com/xo/dburl"
	"github.com/xo/usql/drivers"
	"github.com/xo/usql/env"
	"github.com/xo/usql/handler"
	"github.com/xo/usql/rline"
	"github.com/xo/usql/text"
)

var (
	version           = "unknown"
	dsn               string
	command           string
	fileName          string
	outFile           string
	noRC              bool
	configFile        string
	password          string
	singleTransaction bool
	variables         varsFlag

	// DML variables
	createDB string // as a instance meta json string or simply a node count
	dropDB   string // database id to drop

	// regex for special sql statement
	descTableRegex       = regexp.MustCompile("(?i)^desc(?:ribe)?\\s+(\\w+)\\s*;?\\s*$")
	showCreateTableRegex = regexp.MustCompile("(?i)^show\\s+create\\s+table\\s+(\\w+)\\s*;\\s*?$")
)

type varsFlag struct {
	flag.Value
	vars []string
}

func (v *varsFlag) Get() []string {
	return append([]string{}, v.vars...)
}

func (v *varsFlag) String() string {
	return fmt.Sprintf("%#v", v.vars)
}

func (v *varsFlag) Set(value string) error {
	v.vars = append(v.vars, value)
	return nil
}

func init() {
	// set command name of usql
	text.CommandName = "covenantsql"

	// register SQLite3 database
	drivers.Register("sqlite3", drivers.Driver{
		AllowMultilineComments: true,
		ForceParams: drivers.ForceQueryParameters([]string{
			"loc", "auto",
		}),
		Version: func(db drivers.DB) (string, error) {
			var ver string
			err := db.QueryRow(`SELECT sqlite_version()`).Scan(&ver)
			if err != nil {
				return "", err
			}
			return "SQLite3 " + ver, nil
		},
		Err: func(err error) (string, string) {
			if e, ok := err.(sqlite3.Error); ok {
				return strconv.Itoa(int(e.Code)), e.Error()
			}

			code, msg := "", err.Error()
			if e, ok := err.(sqlite3.ErrNo); ok {
				code = strconv.Itoa(int(e))
			}

			return code, msg
		},
		ConvertBytes: func(buf []byte, tfmt string) (string, error) {
			// attempt to convert buf if it matches a time format, and if it
			// does, then return a formatted time string.
			s := string(buf)
			if s != "" && strings.TrimSpace(s) != "" {
				t := new(SqTime)
				if err := t.Scan(buf); err == nil {
					return t.Format(tfmt), nil
				}
			}
			return s, nil
		},
	})

	// register CovenantSQL database
	drivers.Register("covenantsql", drivers.Driver{
		AllowMultilineComments: true,
		Version: func(db drivers.DB) (string, error) {
			return version, nil
		},
		Err: func(err error) (string, string) {
			return "", err.Error()
		},
		ConvertBytes: func(buf []byte, tfmt string) (string, error) {
			// attempt to convert buf if it matches a time format, and if it
			// does, then return a formatted time string.
			s := string(buf)
			if s != "" && strings.TrimSpace(s) != "" {
				t := new(SqTime)
				if err := t.Scan(buf); err == nil {
					return t.Format(tfmt), nil
				}
			}
			return s, nil
		},
		RowsAffected: func(sql.Result) (int64, error) {
			return 0, nil
		},
		Open: func(url *dburl.URL) (func(string, string) (*sql.DB, error), error) {
			log.Infof("connecting to %v", url.DSN)
			return sql.Open, nil
		},
		Process: func(prefix string, sqlstr string) (typ string, processSqlStr string, isQuery bool, err error) {
			// override describe query
			// split prefix
			words := strings.Split(prefix, " ")

			if len(words) >= 1 && words[0] == "SHOW" {
				if len(words) == 2 && words[1] == "TABLES" {
					// show tables command
					sqlstr = "SELECT name FROM sqlite_master WHERE type = \"table\";"
					prefix = "SELECT"
				}
				if len(words) >= 3 && words[1] == "CREATE" && words[2] == "TABLE" {
					// show create table
					// extract table name from sqlstr
					if matches := showCreateTableRegex.FindStringSubmatch(sqlstr); len(matches) > 0 {
						sqlstr = "SELECT sql FROM sqlite_master WHERE type = \"table\" AND tbl_name = \"" + matches[1] + "\";"
						prefix = "SELECT"
					}
				}
			} else if len(words) >= 1 && (words[0] == "DESC" || words[0] == "DESCRIBE") {
				if matches := descTableRegex.FindStringSubmatch(sqlstr); len(matches) > 0 {
					// valid desc statement
					sqlstr = "PRAGMA table_info(" + matches[1] + ");"
					prefix = "SELECT"
				}
			}

			typ, isQuery = drivers.QueryExecType(prefix, sqlstr)
			processSqlStr = sqlstr
			return
		},
	})

	// register covenantsql:// scheme to dburl
	dburl.Register(dburl.Scheme{
		"covenantsql",
		func(url *dburl.URL) (string, error) {
			dbID, err := dburl.GenOpaque(url)
			if err != nil {
				return "", err
			}
			cfg := client.NewConfig()
			cfg.DatabaseID = dbID
			return cfg.FormatDSN(), nil
		},
		0,
		true,
		[]string{},
		"",
	})

	flag.StringVar(&dsn, "dsn", "", "database url")
	flag.StringVar(&command, "command", "", "run only single command (SQL or usql internal command) and exit")
	flag.StringVar(&fileName, "file", "", "execute commands from file and exit")
	flag.BoolVar(&noRC, "no-rc", false, "do not read start up file")
	flag.StringVar(&outFile, "out", "", "output file")
	flag.StringVar(&configFile, "config", "config.yaml", "config file for covenantsql")
	flag.StringVar(&password, "password", "", "master key password for covenantsql")
	flag.BoolVar(&singleTransaction, "single-transaction", false, "execute as a single transaction (if non-interactive)")
	flag.Var(&variables, "variable", "set variable")

	// DML flags
	flag.StringVar(&createDB, "create", "", "create database, argument can be instance requirement json or simply a node count requirement")
	flag.StringVar(&dropDB, "drop", "", "drop database, argument should be a database id (without covenantsql:// scheme is acceptable)")
}

func main() {
	flag.Parse()

	var err error

	// init covenantsql driver
	if err = client.Init(configFile, []byte(password)); err != nil {
		log.Errorf("init covenantsql client failed: %v", err)
		os.Exit(-1)
		return
	}

	if dropDB != "" {
		// drop database
		if _, err := client.ParseDSN(dropDB); err != nil {
			// not a dsn
			cfg := client.NewConfig()
			cfg.DatabaseID = dropDB
			dropDB = cfg.FormatDSN()
		}

		if err := client.Drop(dropDB); err != nil {
			// drop database failed
			log.Errorf("drop database %v failed: %v", dropDB, err)
			return
		}

		// drop database success
		log.Infof("drop database %v success", dropDB)
		return
	}

	if createDB != "" {
		// create database
		// parse instance requirement
		var meta client.ResourceMeta

		if err := json.Unmarshal([]byte(createDB), &meta); err != nil {
			// not a instance json, try if it is a number describing node count
			nodeCnt, err := strconv.ParseUint(createDB, 10, 16)
			if err != nil {
				// still failing
				log.Errorf("create database failed: %v is not a valid instance description", createDB)
				os.Exit(-1)
				return
			}

			meta = client.ResourceMeta{Node: uint16(nodeCnt)}
		}

		dsn, err := client.Create(meta)
		if err != nil {
			log.Infof("create database failed: %v", err)
			os.Exit(-1)
			return
		}

		log.Infof("the newly created database is: %v", dsn)
		return
	}

	available := drivers.Available()
	cur, err := user.Current()
	if err != nil {
		log.Errorf("get current failed: %v", err)
		os.Exit(-1)
		return
	}

	// run
	err = run(cur)
	if err != nil && err != io.EOF && err != rline.ErrInterrupt {
		log.Errorf("run cli error: %v", err)

		if e, ok := err.(*drivers.Error); ok && e.Err == text.ErrDriverNotAvailable {
			bindings := make([]string, 0, len(available))
			for name := range available {
				bindings = append(bindings, name)
			}
			log.Infof("Available drivers are: %v", bindings)
			return
		}
	}
}

func run(u *user.User) (err error) {
	// get working directory
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// handle variables
	for _, v := range variables.Get() {
		if i := strings.Index(v, "="); i != -1 {
			env.Set(v[:i], v[i+1:])
		} else {
			env.Unset(v)
		}
	}

	// create input/output
	interactive := command != "" || fileName != ""
	l, err := rline.New(interactive, outFile, env.HistoryFile(u))
	if err != nil {
		return err
	}
	defer l.Close()

	// create handler
	h := handler.New(l, u, wd, true)

	// open dsn
	if err = h.Open(dsn); err != nil {
		return err
	}

	// start transaction
	if singleTransaction {
		if h.IO().Interactive() {
			return text.ErrSingleTransactionCannotBeUsedWithInteractiveMode
		}
		if err = h.Begin(); err != nil {
			return err
		}
	}

	// rc file
	if rc := env.RCFile(u); !noRC && rc != "" {
		if err = h.Include(rc, false); err != nil && err != text.ErrNoSuchFileOrDirectory {
			return err
		}
	}

	if command != "" {
		// one liner command
		h.SetSingleLineMode(true)
		h.Reset([]rune(command))
		if err = h.Run(); err != nil && err != io.EOF {
			log.Errorf("run command failed: %v", err)
			os.Exit(-1)
			return
		}
	} else if fileName != "" {
		// file
		if err = h.Include(fileName, false); err != nil {
			log.Errorf("run file failed: %v", err)
			os.Exit(-1)
			return
		}
	} else {
		// interactive
		if err = h.Run(); err != nil {
			return
		}

	}

	// commit
	if singleTransaction {
		return h.Commit()
	}

	return nil
}
