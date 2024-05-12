package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/jessevdk/go-flags"
	bolt "go.etcd.io/bbolt"

	"github.com/umputun/feed-master/app/api"
	"github.com/umputun/feed-master/app/config"
	"github.com/umputun/feed-master/app/proc"
)

type options struct {
	Port int    `short:"p" long:"port" description:"port to listen" default:"8080"`
	DB   string `short:"c" long:"db" env:"FM_DB" default:"var/feed-master.bdb" description:"bolt db file"`
	Conf string `short:"f" long:"conf" env:"FM_CONF" default:"feed-master.yml" description:"config file (yml)"`

	// single feed overrides
	Feed           string        `long:"feed" env:"FM_FEED" description:"single feed, overrides config"`
	UpdateInterval time.Duration `long:"update-interval" env:"UPDATE_INTERVAL" default:"1m" description:"update interval, overrides config"`

	TelegramServer  string        `long:"telegram_server" env:"TELEGRAM_SERVER" default:"https://api.telegram.org" description:"telegram bot api server"`
	TelegramToken   string        `long:"telegram_token" env:"TELEGRAM_TOKEN" description:"telegram token"`
	TelegramTimeout time.Duration `long:"telegram_timeout" env:"TELEGRAM_TIMEOUT" default:"1m" description:"telegram timeout"`

	Dbg bool `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision = "local"

func main() {
	fmt.Printf("feed-master %s\n", revision)
	var opts options
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}
	setupLog(opts.Dbg)

	var conf = &config.Conf{}
	var err error
	if opts.Feed == "" {
		conf, err = config.Load(opts.Conf)
		if err != nil {
			log.Fatalf("[ERROR] can't load config %s, %v", opts.Conf, err)
		}
	}

	db, err := makeBoltDB(opts.DB)
	if err != nil {
		log.Fatalf("[ERROR] can't open db %s, %v", opts.DB, err)
	}
	procStore := &proc.BoltDB{DB: db}

	telegramNotif, err := proc.NewTelegramClient(
		opts.TelegramToken,
		opts.TelegramServer,
		opts.TelegramTimeout,
		&proc.TelegramSenderImpl{},
	)
	if err != nil {
		log.Fatalf("[ERROR] failed to initialize telegram client %s, %v", opts.TelegramToken, err)
	}

	p := &proc.Processor{Conf: conf, Store: procStore, TelegramNotif: telegramNotif}
	go func() {
		if err := p.Do(context.Background()); err != nil {
			log.Printf("[ERROR] processor failed: %v", err)
		}
	}()

	server := api.Server{
		Version: revision,
		Conf:    *conf,
		Store:   procStore,
	}
	server.Run(context.Background(), opts.Port)
}

func makeBoltDB(dbFile string) (*bolt.DB, error) {
	log.Printf("[INFO] bolt (persistent) store, %s", dbFile)

	if dbFile == "" {
		return nil, fmt.Errorf("empty db")
	}
	if err := os.MkdirAll(path.Dir(dbFile), 0o700); err != nil {
		return nil, err
	}
	db, err := bolt.Open(dbFile, 0o600, &bolt.Options{Timeout: 1 * time.Second}) // nolint
	if err != nil {
		return nil, err
	}

	return db, err
}

func setupLog(dbg bool) {
	if dbg {
		log.Setup(log.Debug, log.CallerFile, log.Msec, log.LevelBraces)
		return
	}
	log.Setup(log.Msec, log.LevelBraces)
}
