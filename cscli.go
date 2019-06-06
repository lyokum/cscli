package main

import (
	"errors"
	"fmt"
	"github.com/urfave/cli"
	"io/ioutil"
	"log"
	"os"
)

var (
	ErrInvalidDirectory = errors.New("Invalid directory specified")
	ErrNoCRNs           = errors.New("No CRNs given")
	ErrServerNotFound   = errors.New("Server not found")
	ErrPhoneInvalid     = errors.New("Invalid phone number")
	ErrProviderNotFound = errors.New("Provider not known")

	Providers = map[string]string{"att": "txt.att.net", "tmobile": "tmomail.net", "sprint": "messaging.sprintpcs.com", "verizon": "vtext.com"}
	Storage   ClassCache
)

/* FIXME: remove these after testing */
var _ = fmt.Printf
var _ = log.Printf

func main() {
	// Create app
	app := cli.NewApp()

	// Fill help fields
	app.Name = "cscli"
	app.Usage = "command line interface for ND Class Search"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Logan Yokum",
			Email: "lyokum@nd.edu",
		},
	}

	// Fill flags
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "no-cache, n",
			Usage: "do not cache all classes into json files for quicker searches",
		},
		cli.BoolFlag{
			Name:  "debug, i",
			Usage: "enable debugging messages",
		},
		cli.StringFlag{
			Name:  "directory, d",
			Usage: "specify directory to put cache json files in",
		},
	}

	// Fill commands
	app.Commands = []cli.Command{
		cli.Command{
			Name:  "search",
			Usage: "search for class CRNs by names and provided filters with regex",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "open, o",
					Usage: "restrict search to open courses",
				},
				cli.IntSliceFlag{
					Name:  "id, c",
					Usage: "specify `CRN` to search",
				},
				cli.StringSliceFlag{
					Name:  "department, d",
					Usage: "restrict search to `DEPT` (3 or 4 letter abbreviations)",
				},
				cli.StringSliceFlag{
					Name:  "professor, p",
					Usage: "restrict search to first or last name of `PROF`",
				},
				cli.BoolFlag{
					Name:  "info, i",
					Usage: "display more info than just the CRNs of found classes",
				},
				cli.BoolFlag{
					Name:  "update, u",
					Usage: "update class info of classes in search results from website",
				},
			},
			Action:                 performSearch,
			UseShortOptionHandling: true,
		},
		cli.Command{
			Name:  "check",
			Usage: "check to see if classes with specified CRNs are open",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "update, u",
					Usage: "send update to server (requires update-send)",
				},
				cli.StringFlag{
					Name:  "server, s",
					Usage: "specify update `SERVER` domain name/ip address and port (required when using --update/-u flag)",
					Value: "",
				},
				cli.BoolFlag{
					Name:  "text, t",
					Usage: "send text to phone (requires mail-send which uses msmtp-mta)",
				},
				cli.StringFlag{
					Name:  "cellphone, c",
					Usage: "specify 9-digit cellphone `NUMBER` (no spaces) (required when using --text/-t flag)",
					Value: "",
				},
				cli.StringFlag{
					Name:  "provider, p",
					Usage: "specify phone `PROVIDER` (company) (required when using --text/-t flag)",
					Value: "",
				},
			},
			Action:                 checkCRNs,
			UseShortOptionHandling: true,
		},
		cli.Command{
			Name:   "refresh",
			Usage:  "refresh cache files",
			Action: refreshCache,
		},
	}

	// setup app
	app.Before = func(ctx *cli.Context) (err error) {
		// set debug
		log.SetFlags(log.Flags() | log.Lshortfile)
		if !ctx.Bool("debug") {
			log.SetOutput(ioutil.Discard)
		}

		// init cache
		Storage.Init()

		// set cache dir
		if dir := ctx.String("directory"); dir != "" {
			log.Println("Setting directory")
			err = setDirectory(ctx, dir)
			if err != nil {
				return err
			}
			log.Println("Directories set to", Storage.Info.Directory, "and", Storage.OptCache.Info.Directory)
		}

		// restore cache from file
		err = Storage.Restore()
		if err != nil {
			return
		}

		return nil
	}

	// Run app
	err := app.Run(os.Args)
	if err != nil {
		log.SetOutput(os.Stderr)
		log.Fatal("ERROR: " + err.Error())
	}
}
