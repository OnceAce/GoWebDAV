package model

import (
	"golang.org/x/net/webdav"
	"strconv"
	"strings"
)
type WebDAVConfig struct {
	Prefix   string
	PathDir  string
	Username string
	Password string
	ReadOnly bool
	Handler  *webdav.Handler
}

func (WebDAVConfig *WebDAVConfig) InitByConfigStr(str string) {
	davConfigArray := strings.Split(str, ",")
	WebDAVConfig.Prefix= davConfigArray[0]
	WebDAVConfig.PathDir = davConfigArray[1]
	WebDAVConfig.Username = davConfigArray[2]
	WebDAVConfig.Password = davConfigArray[3]

	readonly, err := strconv.ParseBool(davConfigArray[4])
	if err != nil {
		readonly = false
	}
	WebDAVConfig.ReadOnly = readonly

	WebDAVConfig.Handler = &webdav.Handler{
		FileSystem: webdav.Dir(davConfigArray[1]),
		LockSystem: webdav.NewMemLS(),
		Prefix:     davConfigArray[0],
	}
}
