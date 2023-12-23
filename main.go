package main

import (
	"db2struct/pkg"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli/v2"

	"db2struct/config"
)

var (
	outputFolder string
	packageName  string
	tableNames   cli.StringSlice
)

func main() {
	if err := run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// var app *cli.App
var cfg *config.Config

func run(_ []string) error {
	// Create channels for signals and stop signal
	signals := make(chan os.Signal, 1)

	// Register signals to receive
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	var err error
	cfg, err = config.Load()
	if err != nil {
		return err
	}

	app := cli.NewApp()
	app.Name = "db2struct"
	app.Usage = "This command will gen table mysql from db to a struct model golang. Ex: make db2struct tableName1 tableName2"
	app.Action = func(_ *cli.Context) (err error) {
		if len(tableNames.Value()) == 0 {
			log.Println(err, "Enter the table name")
			return
		}
		err = pkg.DB2Struct(cfg.MySQL.DSN(), cfg.MySQL.Database, tableNames.Value(), outputFolder, packageName)
		if err != nil {
			log.Println(err, "Error generate model from db")
		}
		return
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "out",
			Aliases:     []string{"o"},
			Usage:       "output model",
			Value:       "./model/",
			Destination: &outputFolder,
		},
		&cli.StringSliceFlag{
			Name:        "tables",
			Aliases:     []string{"t"},
			Usage:       "list table names",
			Destination: &tableNames,
		},
		&cli.StringFlag{
			Name:        "package name",
			Aliases:     []string{"p"},
			Usage:       "package name",
			Value:       "model",
			Destination: &packageName,
		},
	}
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
	return nil
}
