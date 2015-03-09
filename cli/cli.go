package cli

import (
	. "fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/codegangsta/cli"
	"github.com/ugorji/go/codec"

	"polydawn.net/repeatr/def"
	"polydawn.net/repeatr/executor/dispatch"
)

var App *cli.App

func init() {
	App = cli.NewApp()

	App.Name = "repeatr"
	App.Usage = "Run it. Run it again."
	App.Version = "0.0.1"

	App.Commands = []cli.Command{
		{
			Name:   "run",
			Usage:  "Run a formula",
			Action: Run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "executor, e",
					Value: "nsinit",
					Usage: "Which executor to use",
				},
				cli.StringFlag{
					Name:  "input, i",
					Value: "formula.json",
					Usage: "Location of input formula (json format)",
				},
			},
		},
	}
}

func Run(c *cli.Context) {

	executor := *executors.Get(c.String("executor"))
	filename, _ := filepath.Abs(c.String("input"))

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		Println(err)
		Println("Could not read file", filename)
		os.Exit(1)
	}

	dec := codec.NewDecoderBytes(content, &codec.JsonHandle{})

	formula := def.Formula{}
	dec.MustDecode(&formula)

	executor.Run(formula)
}
