package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/creasty/defaults"
	"github.com/imdario/mergo"
	"github.com/labstack/echo"
	"github.com/pooflix/engine"
	"github.com/pooflix/server"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

func New(cfg *Config) (*Core, error) {
	if cfg == nil {
		cfg = NewDefaultClientConfig()
	}

	c := &Core{config: cfg}

	if err := c.Initialize(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Core) Initialize() error {
	if err := c.InitializeForeground(); err != nil {
		return err
	}

	return c.InitializeBackground()
}

// InitializeForeground sets up Log and DB on *Core.
func (c *Core) InitializeForeground() error {
	if c.config.ConfigFilePath != "" {
		var configFileSettings Config
		configFile, err := os.Open(c.config.ConfigFilePath)
		if err != nil {
			return err
		}

		if err := json.NewDecoder(configFile).Decode(&configFileSettings); err != nil {
			return err
		}

		// Merge in command line settings (which overwrite respective config file settings)
		if err := mergo.Merge(c.config, configFileSettings); err != nil {
			return err
		}

		// Set Default Settings with struct tags
		if err := defaults.Set(c.config); err != nil {
			return err
		}
	} else {
		return errors.New("config is missing")
	}

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
	c.http, err = server.New()
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

	return c.http.Run(func(e *echo.Echo) {
		e.POST("/torrents/magnet", routeHandler(func(ctx echo.Context) error {
			link := ctx.FormValue("link")

			if err := c.engine.NewMagnet(link); err != nil {
				return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
			}

			return echo.NewHTTPError(http.StatusAccepted)
		}))

		e.GET("/torrents/list", routeHandler(func(ctx echo.Context) error {
			return ctx.JSON(http.StatusOK, c.engine.GetTorrents())
		}))

		e.GET("/torrents/:hash/.m3u", routeHandler(func(ctx echo.Context) error {
			hash := ctx.Param("hash")

			if t, ok := c.engine.GetTorrents()[hash]; ok {
				ctx.Response().Header().Set(echo.HeaderContentType, "application/x-mpegurl; charset=utf-8")

				var str string
				for i, file := range t.Files {
					str += fmt.Sprintf("#EXTINF:-1,%s\nhttp://127.0.0.1:8007/torrent/stream/%s/%d\n", file.Path, hash, i)
				}

				return ctx.String(http.StatusOK, "#EXTM3U\n"+str)
			}

			return echo.ErrNotFound
		}))

		e.GET("/torrents/stream/:hash/:id", routeHandler(func(ctx echo.Context) error {
			hash := ctx.Param("hash")
			id := ctx.Param("id")

			idd, _ := strconv.Atoi(id)

			if t, ok := c.engine.GetTorrents()[hash]; ok {
				//buffer := make([]byte, 512)

				rr := t.Files[idd].GetFile()
				entry := rr.NewReader()

				defer func() {
					if err := entry.Close(); err != nil {
						log.Printf("Error closing file reader: %s\n", err)
					}
				}()

				//ctx.Response().Status = 206
				ctx.Response().Header().Set("Accept-Ranges", "bytes")
				ctx.Response().Header().Set("transferMode.dlna.org", "Streaming")
				ctx.Response().Header().Set("contentFeatures.dlna.org", "DLNA.ORG_OP=01;DLNA.ORG_CI=0;DLNA.ORG_FLAGS=01700000000000000000000000000000")

				//return ctx.Stream(200, http.DetectContentType(buffer), entry.Reader)
				http.ServeContent(ctx.Response(), ctx.Request(), rr.Path(), time.Now(), entry)
				return nil
			}

			return echo.ErrNotFound
		}))
	})
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
	//b, _ := json.MarshalIndent(&c, "", "  ")
	//ioutil.WriteFile(c.config.ConfigFilePath, b, 0755)
	c.state.Config = ec

	return nil
}
