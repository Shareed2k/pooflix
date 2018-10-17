package core

import (
	"fmt"
	"github.com/labstack/echo"
	"github.com/pooflix/engine"
	"github.com/pooflix/server"
	"path/filepath"
	"sync"
	"time"
)

type Core struct {
	config *Config
	//torrent engine
	engine *engine.Engine
	http   *server.Server
	state  struct {
		sync.Mutex
		Config engine.Config
		//SearchProviders scraper.Config
		//Downloads       *fsNode
		Torrents map[string]*engine.Torrent
		Users    map[string]string
		Stats    struct {
			Title   string
			Version string
			Runtime string
			Uptime  time.Time
			//System  stats
		}
	}
}

func New(cfg *Config) (err error) {
	if cfg == nil {
		cfg, err = NewDefaultClientConfig()
	}

	c := &Core{config: cfg}

	if err := c.Initialize(); err != nil {
		return err
	}

	return
}

func (c *Core) Initialize() error {
	if err := c.InitializeForeground(); err != nil {
		return err
	}

	return c.InitializeBackground()
}

// InitializeForeground sets up Log and DB on *Core.
func (c *Core) InitializeForeground() error {
	//torrent engine
	c.engine = engine.New()

	//configure engine
	ec := engine.Config{
		DownloadDirectory: c.config.DownloadDirectory,
		DisableEncryption: true,
		EnableUpload:      true,
		EnableSeeding:     false,
		AutoStart:         true,
	}

	if ec.IncomingPort <= 0 || ec.IncomingPort >= 65535 {
		ec.IncomingPort = 50007
	}

	if err := c.reconfigure(ec); err != nil {
		return fmt.Errorf("initial configure failed: %v", err)
	}

	//dns service
	if err := NewDns(); err != nil {
		return err
	}

	//http service
	var err error
	c.http, err = server.New(&server.Config{
		IncomingPort: c.config.HttpServerPort,
	})
	if err != nil {
		return err
	}

	return nil
}

// InitializeBackground starts Action processing and RecurringServices for *Core.
func (c *Core) InitializeBackground() error {
	//poll torrents and files
	go func() {
		for {
			c.state.Lock()
			c.state.Torrents = c.engine.GetTorrents()
			//s.state.Downloads = s.listFiles()
			c.state.Unlock()
			time.Sleep(1 * time.Second)
		}
	}()

	// Middleware set custom echo context
	c.http.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			return next(&CustomContext{
				Context: ctx,
				Core:    c,
			})
		}
	})

	//run http server
	return c.http.Run(routes)
}

func (c *Core) reconfigure(ec engine.Config) error {
	dldir, err := filepath.Abs(c.config.DownloadDirectory)

	if err != nil {
		return fmt.Errorf("invalid path: %v", err)
	}

	c.config.DownloadDirectory = dldir

	if err := c.engine.Configure(ec); err != nil {
		return err
	}

	c.state.Config = ec

	return nil
}
