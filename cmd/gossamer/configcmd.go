// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.
package main

import (
	"fmt"
	"github.com/ChainSafe/gossamer/p2p"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"github.com/ChainSafe/gossamer/cmd/utils"
	cfg "github.com/ChainSafe/gossamer/config"
	"github.com/ChainSafe/gossamer/dot"
	"github.com/ChainSafe/gossamer/polkadb"
	log "github.com/inconshreveable/log15"
	"github.com/naoina/toml"
	"github.com/urfave/cli"
	"unicode"
)

var (
	dumpConfigCommand = cli.Command{
		Action:      dumpConfig,
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       append(append(nodeFlags, rpcFlags...)),
		Category:    "CONFIGURATION DEBUGGING",
		Description: `The dumpconfig command shows configuration values.`,
	}

	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file",
	}
)

// makeNode sets up node; opening badgerDB instance and returning the Goss container
func makeNode(ctx *cli.Context) (*dot.Dot, error) {
	fig, err := setConfig(ctx)
	if err != nil {
		log.Error("unable to extract required config", "err", err)
	}
	srv := setP2PConfig(ctx, fig.ServiceConfig)
	datadir := setDatabaseDir(ctx, fig)
	db, err := polkadb.NewBadgerDB(datadir)
	if err != nil {
		fmt.Println(err)
	}
	return &dot.Dot{
		ServerConfig: fig.ServiceConfig,
		Server:       srv,
		Polkadb:      db,
	}, nil
}

// setConfig checks for config.toml if --config flag is specified
func setConfig(ctx *cli.Context) (*cfg.Config, error) {
	var fig *cfg.Config
	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		config, err := loadConfig(file)
		if err != nil {
			log.Warn("err loading toml file", "err", err.Error())
			return fig, err
		}
		return config, nil
	}
	return fig, nil
}

// setDatabaseDir initializes directory for BadgerDB logs
func setDatabaseDir(ctx *cli.Context, cfg *cfg.Config) string {
	if cfg.DbConfig.Datadir != "" {
		return cfg.DbConfig.Datadir
	} else if file := ctx.GlobalString(utils.DataDirFlag.Name); file != "" {
		return file
	} else {
		log.Error("must specify data directory")
		return ""
	}
}

// loadConfig loads the contents from config.toml and inits Config object
func loadConfig(file string) (*cfg.Config, error) {
	fp, err := filepath.Abs(file)
	filepath.Clean(fp)
	f, err := os.Open(fp)
	if err != nil {
		panic(err)
	}
	defer func() {
		err = f.Close()
		if err != nil {
			log.Warn("err closing conn", "err", err.Error())
		}
	}()
	var config *cfg.Config
	if err = tomlSettings.NewDecoder(f).Decode(&config); err != nil {
		log.Error("decoding toml error", "err", err.Error())
	}
	return config, err
}

// setBootstrapNodes creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
func setBootstrapNodes(ctx *cli.Context, cfg *p2p.ServiceConfig) {
	var urls []string
	switch {
	case ctx.GlobalIsSet(utils.BootnodesFlag.Name):
		urls = strings.Split(ctx.GlobalString(utils.BootnodesFlag.Name), ",")
	case cfg.BootstrapNodes != nil:
		return // already set, don't apply defaults.
	}
	cfg.BootstrapNodes = append(cfg.BootstrapNodes, urls...)
}

// SetP2PConfig sets up the configurations required for P2P service
func setP2PConfig(ctx *cli.Context, cfg *p2p.ServiceConfig) *p2p.Service {
	setBootstrapNodes(ctx, cfg)
	srv := startP2PService(cfg)
	return srv
}

// startP2PService starts a p2p network layer from provided config
func startP2PService(cfg *p2p.ServiceConfig) *p2p.Service {
	srv, err := p2p.NewService(cfg)
	if err != nil {
		log.Error("error starting p2p", "err", err.Error())
	}
	return srv
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	fig, err := setConfig(ctx)
	if err != nil {
		return err
	}
	comment := ""

	out, err := tomlSettings.Marshal(&fig)
	if err != nil {
		return err
	}

	dump := os.Stdout
	if ctx.NArg() > 0 {
		dump, err = os.OpenFile(ctx.Args().Get(0), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer dump.Close()
	}
	dump.WriteString(comment)
	dump.Write(out)

	return nil
}

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		link := ""
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}