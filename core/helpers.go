package core

import (
	"errors"
	"github.com/labstack/echo"
	"net"
)

type CustomContext struct {
	Core *Core
	echo.Context
}

func routeHandler(fn func(*CustomContext) error) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		c := ctx.(*CustomContext)
		return fn(c)
	}
}

func GetLocalIp() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	var ip net.IP
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}

		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}

		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}

			return ip.String(), nil
		}
	}

	return "", errors.New("are you connected to the network")
}
