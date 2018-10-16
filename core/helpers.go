package core

import "github.com/labstack/echo"

func routeHandler(fn func(ctx echo.Context) error) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		return fn(ctx)
	}
}