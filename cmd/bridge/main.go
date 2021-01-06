package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/cloudkucooland/toofar"
	"github.com/cloudkucooland/toofar/accessory"
	"github.com/cloudkucooland/toofar/config"
	"github.com/cloudkucooland/toofar/platform"

	"github.com/brutella/hc/log"
	"github.com/urfave/cli/v2"
)

func main() {
	var dir, file string

	app := cli.App{
		Name:  "scot's home automation server",
		Usage: "server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "dir",
				Value:       "config",
				Usage:       "configuration directory",
				Destination: &dir,
			},
			&cli.StringFlag{
				Name:        "config",
				Value:       "server.json",
				Usage:       "configuration file",
				Destination: &file,
			},
		},
		Action: func(c *cli.Context) error {
			// log.Debug.Enable()
			// log.Info.Printf("config dir: %s", dir)
			// log.Info.Printf("config file: %s", file)

			fulldir, err := filepath.Abs(dir)
			if err != nil {
				log.Info.Panic("unable to get config directory", dir)
			}
			cfd := filepath.Join(fulldir, file)
			confFile, err := os.Open(cfd)
			if err != nil {
				log.Info.Panic("unable to open config: ", cfd)
			}
			raw, err := ioutil.ReadAll(confFile)
			if err != nil {
				log.Info.Panic(err)
			}
			confFile.Close()

			var conf config.Config
			err = json.Unmarshal(raw, &conf)
			if err != nil {
				log.Info.Panic(err, string(raw))
			}

			conf.ConfigDir = fulldir
			conf.ConfigFile = cfd

			// spin up platforms to listen to devices
			toofar.BootstrapPlatforms(conf)

			// load accessory configs
			var accdir = filepath.Join(fulldir, "accessories")
			files, err := ioutil.ReadDir(accdir)
			if err != nil {
				log.Info.Panic(err)
			}
			for _, f := range files {
				acc, err := fileToAccessory(filepath.Join(accdir, f.Name()), f.Name())
				if err != nil {
					log.Info.Printf(err.Error())
					continue
				}
				toofar.AddAccessory(acc)
			}

			// HC can only be started once all accessories are known
			toofar.StartHC(conf)

			// run all the background processes
			platform.Background()

			// wait for signal to shut down
			sigch := make(chan os.Signal, 3)
			signal.Notify(sigch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGHUP, os.Interrupt)

			// loop until signal sent
			sig := <-sigch

			log.Info.Printf("shutdown requested by signal: %s", sig)
			platform.ShutdownAllPlatforms()
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Info.Panic(err)
	}
}

func fileToAccessory(file string, name string) (*accessory.TFAccessory, error) {
	f, err := os.Open(file)
	if err != nil {
		log.Info.Printf("unable to open accessory config file: %s %s", file, err.Error())
	}
	defer f.Close()

	raw, err := ioutil.ReadAll(f)
	if err != nil {
		log.Info.Printf(err.Error())
	}

	var acc accessory.TFAccessory
	err = json.Unmarshal(raw, &acc)
	if err != nil {
		log.Info.Println(err, string(raw))
	}

	acc.Name = name[:strings.LastIndex(name, ".")]
	// log.Info.Printf("%+v", acc)
	return &acc, nil
}
