package main

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"golang.org/x/net/webdav"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var AppConfig = &Config{}

func main() {
	WebDAVConfigs := AppConfig.Load()

	sMux := http.NewServeMux()
	sMux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {

		webDAVConfig := WebDAVConfigFindOneByPrefix(WebDAVConfigs, req.URL)

		if webDAVConfig == nil {

			w.Header().Set("Content-Type", "text/html; charset=utf-8")

			//index

			_, err := fmt.Fprintf(w, "<pre>\n")
			if err != nil {
				fmt.Println(err)
			}

			for _, config := range WebDAVConfigs {
				_, err = fmt.Fprintf(w, "<a href=\"%s\" >%s</a>\n", config.Prefix+"/", config.Prefix)
				if err != nil {
					fmt.Println(err)
				}
			}

			_, err = fmt.Fprintf(w, "<pre>\n")
			if err != nil {
				fmt.Println(err)
			}

			return
		}

		if webDAVConfig.Username != "null" && webDAVConfig.Password != "null" {
			// 配置中的 用户名 密码 都为 null 时 不进行身份检查
			// 不都为 null 进行身份检查

			username, password, ok := req.BasicAuth()

			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if username == "" || password == "" {
				http.Error(w, "username missing or password missing", http.StatusUnauthorized)
				return
			}

			if username != webDAVConfig.Username || password != webDAVConfig.Password {
				http.Error(w, "username wrong or password wrong", http.StatusUnauthorized)
				return
			}
		}

		if webDAVConfig.ReadOnly && req.Method != "GET" && req.Method != "OPTIONS" && req.Method != "PROPFIND"{
			// ReadOnly
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = w.Write([]byte("Readonly, Method " + req.Method + " Not Allowed"))
			return
		}

		// show files of directory
		if req.Method == "GET" && handleDirList(webDAVConfig.Handler.FileSystem, w, req, webDAVConfig.Handler.Prefix) {
			return
		}

		// handle file
		webDAVConfig.Handler.ServeHTTP(w, req)
	})

	err := http.ListenAndServe(":80", sMux)
	if err != nil {
		fmt.Println(err)
	}
}



func handleDirList(fs webdav.FileSystem, w http.ResponseWriter, req *http.Request, prefix string) bool {
	ctx := context.Background()

	path := req.URL.Path
	path = strings.Replace(path, prefix, "/", 1)

	f, err := fs.OpenFile(ctx, path, os.O_RDONLY, 0)
	if err != nil {
		return false
	}
	defer f.Close()

	if fi, _ := f.Stat(); fi != nil && !fi.IsDir() {

		return false

	}

	dirs, err := f.Readdir(-1)

	if err != nil {
		log.Print(w, "Error reading directory", http.StatusInternalServerError)
		return false
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	_, err = fmt.Fprintf(w, "<pre>\n")
	if err != nil {
		fmt.Println(err)
	}

	for _, d := range dirs {
		name := d.Name()
		if d.IsDir() {
			name += "/"
		}
		_, err = fmt.Fprintf(w, "<a href=\"%s\" >%s</a>\n", prefix+"/"+path+"/"+name, name)
		if err != nil {
			fmt.Println(err)
		}
	}

	_, err = fmt.Fprintf(w, "</pre>\n")
	if err != nil {
		fmt.Println(err)
	}
	return true
}

type Config struct {
	dav string
}

type WebDAVConfigure struct {
	Prefix   string
	PathDir  string
	Username string
	Password string
	ReadOnly bool
	Handler  *webdav.Handler
}

func (config *Config) Load() []*WebDAVConfigure {
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.SetConfigName("config")

	pflag.String("dav", "/dav1,./TestDir1,user1,pass1;/dav2,./TestDir2,user2,pass2", "like /dav1,./TestDir1,user1,pass1;/dav2,./TestDir2,user2,pass2")
	pflag.Parse()

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		fmt.Println(err)
	}
	err = viper.ReadInConfig()
	if err != nil {
		fmt.Println(err)
	}

	config.dav = viper.GetString("dav")
	fmt.Print("AppConfig.dav ")
	fmt.Println(config.dav)

	WebDAVConfigs := ParseInitialConfig(config.dav)
	return WebDAVConfigs


}

func ParseInitialConfig (str string) []*WebDAVConfigure {
	WebDAVConfigs := make([]*WebDAVConfigure, 0)
	for _, davConfig := range strings.Split(str, ";") {
		WebDAVConfig := &WebDAVConfigure{}
		davConfigArray := strings.Split(davConfig, ",")
		WebDAVConfig.Prefix = davConfigArray[0]
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

		WebDAVConfigs = append(WebDAVConfigs, WebDAVConfig)
	}
	return WebDAVConfigs
}

func WebDAVConfigFindOneByPrefix(WebDAVConfigs []*WebDAVConfigure, url *url.URL) *WebDAVConfigure {
	prefix := "/" + strings.Split(fmt.Sprint(url), "/")[1]     //parse Prefix From URL
	for _, WebDAVConfig := range WebDAVConfigs {
		if WebDAVConfig.Prefix == prefix {
			return WebDAVConfig
		}
	}
	return nil
}




