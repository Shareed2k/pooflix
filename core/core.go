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
		e.GET("/torrent", routeHandler(func(ctx echo.Context) error {
			if err := c.engine.NewMagnet("magnet:?xt=urn:btih:74B7F3F188F9CE60DE7F2B547FB60507F2F7222A&tr=http%3A%2F%2Fbt3.t-ru.org%2Fann%3Fmagnet&dn=%5Bfrontendmasters.com%5D%20Introduction%20to%20Data%20Structures%20for%20Interviews%20%5B2018%2C%20ENG%5D"); err != nil {
				return err
			}

			return nil
		}))

		e.GET("/torrent/list", routeHandler(func(ctx echo.Context) error {
			return ctx.JSON(200, c.engine.GetTorrents())
		}))

		e.GET("/torrent/m3u8", routeHandler(func(ctx echo.Context) error {
			files := c.engine.GetTorrents()["74b7f3f188f9ce60de7f2b547fb60507f2f7222a"].Files

			ctx.Response().Header().Set(echo.HeaderContentType, "application/x-mpegurl; charset=utf-8")

			var str string
			for i, file := range files {
				str += "#EXTINF:-1," + file.Path + "\n" + fmt.Sprintf("http://127.0.0.1:8007/torrent/stream/%s/%d\n", "74b7f3f188f9ce60de7f2b547fb60507f2f7222a", i)
			}

			return ctx.String(http.StatusOK, "#EXTM3U\n"+str)
		}))

		e.GET("/torrent/stream/:hash/:id", routeHandler(func(ctx echo.Context) error {
			hash := ctx.Param("hash")
			id := ctx.Param("id")

			idd, _ := strconv.Atoi(id)

			if t, ok := c.engine.GetTorrents()[hash]; ok {
				buffer := make([]byte, 512)

				rr := t.Files[idd].GetFile()
				fmt.Println(rr.Path())
				/*_, err := rr.Read(buffer)
				if err != nil {
					return echo.ErrNotFound
				}*/

				//h := ctx.Response()
				//h.Writer.Header().Set("Content-Disposition", "attachment; filename=\""+rr.Path()+"\"")
				//h.Writer.

				entry, err := NewFileReader(rr)
				if err != nil {
					return echo.ErrNotFound
				}

				defer func() {
					if err := entry.Reader.Close(); err != nil {
						log.Printf("Error closing file reader: %s\n", err)
					}
				}()

				//ctx.Response().Status = 206
				//ctx.Response().Header().Set(echo.HeaderContentType, http.DetectContentType(buffer))
				ctx.Response().Header().Set("Accept-Ranges", "bytes")
				ctx.Response().Header().Set("transferMode.dlna.org", "Streaming")
				ctx.Response().Header().Set("contentFeatures.dlna.org", "DLNA.ORG_OP=01;DLNA.ORG_CI=0;DLNA.ORG_FLAGS=01700000000000000000000000000000")
				ctx.Response().Header().Set("Content-Disposition", "attachment; filename=\""+rr.Path()+"\"")

				return ctx.Stream(200, http.DetectContentType(buffer), entry.Reader)
				//http.ServeContent(ctx.Response(), ctx.Request(), rr.Path(), time.Now(), entry.Reader)
				//return nil
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
