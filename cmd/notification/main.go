package main

import (
	"flag"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
)

var (
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path")
}

func newApp(logger log.Logger) *kratos.App {
	return kratos.New(
		kratos.Name("notification"),
		kratos.Logger(logger),
	)
}

func main() {
	flag.Parse()

	logger := log.NewStdLogger(os.Stdout)
	log := log.NewHelper(logger)

	app := newApp(logger)
	if err := app.Run(); err != nil {
		log.Error(err)
	}
}
