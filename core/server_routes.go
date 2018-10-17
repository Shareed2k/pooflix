package core

import (
	"fmt"
	"github.com/labstack/echo"
	"log"
	"net/http"
	"strconv"
	"time"
)

func routes(e *echo.Echo) {
	api := e.Group("/api/v1")

	// endpoint to start download torrent from magnet link
	api.POST("/torrents/magnet", routeHandler(func(ctx *CustomContext) error {
		c := ctx.Core
		link := ctx.FormValue("link")

		if err := c.engine.NewMagnet(link); err != nil {
			return echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error())
		}

		return echo.NewHTTPError(http.StatusAccepted)
	}))

	// endpoint of torrents in pooflix
	api.GET("/torrents", routeHandler(func(ctx *CustomContext) error {
		c := ctx.Core
		return ctx.JSON(http.StatusOK, c.engine.GetTorrents())
	}))

	// endpoint for generation m3u8 file list of streams
	api.GET("/torrents/:hash/.m3u", routeHandler(func(ctx *CustomContext) error {
		c := ctx.Core
		hash := ctx.Param("hash")

		if t, ok := c.engine.GetTorrents()[hash]; ok {
			ctx.Response().Header().Set(echo.HeaderContentType, "application/x-mpegurl; charset=utf-8")

			ip, err := GetLocalIp()
			if err != nil {
				log.Printf("Core: can't get internal ip, %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError, err)
			}

			var str string
			for i, file := range t.Files {
				str += fmt.Sprintf("#EXTINF:-1,%s\nhttp://%s:%s/api/v1/torrents/%s/stream/%d\n", file.Path, ip, c.config.HttpServerPort, hash, i)
			}

			return ctx.String(http.StatusOK, "#EXTM3U\n"+str)
		}

		return echo.ErrNotFound
	}))

	// endpoint for stream specific file from torrent slice of files
	api.GET("/torrents/:hash/stream/:id", routeHandler(func(ctx *CustomContext) error {
		c := ctx.Core
		hash := ctx.Param("hash")
		id, err := strconv.Atoi(ctx.Param("id"))

		if t, ok := c.engine.GetTorrents()[hash]; err == nil && ok && len(t.Files) > id {
			rr := t.Files[id].GetFile()
			entry := rr.NewReader()

			defer func() {
				if err := entry.Close(); err != nil {
					log.Printf("Error closing file reader: %s\n", err)
				}
			}()

			ctx.Response().Header().Set("Accept-Ranges", "bytes")
			ctx.Response().Header().Set("transferMode.dlna.org", "Streaming")
			ctx.Response().Header().Set("contentFeatures.dlna.org", "DLNA.ORG_OP=01;DLNA.ORG_CI=0;DLNA.ORG_FLAGS=01700000000000000000000000000000")

			http.ServeContent(ctx.Response(), ctx.Request(), rr.Path(), time.Now(), entry)
			return nil
		}

		return echo.ErrNotFound
	}))
}
