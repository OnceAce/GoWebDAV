package main

import (
	"GoWebdav/model"
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"golang.org/x/net/webdav"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"strconv"
)

type WebDAVConfig struct {
	Prefix   string
	PathDir  string
	Username string
	Password string
	ReadOnly bool
	Handler  *webdav.Handler
}

func (WebDAVConfig *WebDAVConfig) Init(prefix string, pathDir string, username string, password string, readonly bool) {
	WebDAVConfig.Prefix = prefix
	WebDAVConfig.PathDir = pathDir
	WebDAVConfig.Username = username
	WebDAVConfig.Password = password
	WebDAVConfig.ReadOnly = readonly

	WebDAVConfig.Handler = &webdav.Handler{
		FileSystem: webdav.Dir(pathDir),
		LockSystem: webdav.NewMemLS(),
		Prefix:     prefix,
	}
}
func (WebDAVConfig *WebDAVConfig) InitByConfigStr(str string) {
	davConfigArray := strings.Split(str, ",")
	prefix := davConfigArray[0]
	pathDir := davConfigArray[1]
	username := davConfigArray[2]
	password := davConfigArray[3]

	readonly, err := strconv.ParseBool(davConfigArray[4])
	if err != nil {
		readonly = false
	}

	WebDAVConfig.Init(prefix, pathDir, username, password, readonly)
}
func WebDAVConfigFindOneByPrefix(WebDAVConfigs []*WebDAVConfig, prefix string) *WebDAVConfig {
	for _, WebDAVConfig := range WebDAVConfigs {
		if WebDAVConfig.Prefix == prefix {
			return WebDAVConfig
		}
	}
	return nil
}






type Config struct {
	dav string
}

func (c *Config) Load() {
	pflag.String("dav", "/dav1,./TestDir1,user1,pass1;/dav2,./TestDir2,user2,pass2", "like /dav1,./TestDir1,user1,pass1;/dav2,./TestDir2,user2,pass2")
	pflag.Parse()

	err := viper.BindPFlags(pflag.CommandLine)    //从pflag检索“命令行”并处理错误
	if err != nil {
		fmt.Println(err)
	}
	err = viper.ReadInConfig()
	if err != nil {
		fmt.Println(err)
	}
	viper.SetConfigType("yaml")             //设置配置文件类型
	viper.AddConfigPath(".")                //添加配置文件所在的路径
	viper.SetConfigName("config")           //设置配置文件的名字
	
	c.dav = viper.GetString("dav")          //通过viper从pflag中获取值
}

var AppConfig *Config = &Config{}





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
func main() {
	AppConfig.Load()
	fmt.Print("AppConfig.dav ")
	fmt.Println(AppConfig.dav)
	davConfigs := strings.Split(AppConfig.dav, ";")

	WebDAVConfigs := make([]*model.WebDAVConfig, 0)

	for _, davConfig := range davConfigs {
		WebDAVConfig := &model.WebDAVConfig{}
		WebDAVConfig.InitByConfigStr(davConfig)
		WebDAVConfigs = append(WebDAVConfigs, WebDAVConfig)
	}

	sMux := http.NewServeMux()
	sMux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {

		webDAVConfig := model.WebDAVConfigFindOneByPrefix(WebDAVConfigs, parsePrefixFromURL(req.URL))

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

		if webDAVConfig.ReadOnly && req.Method != "GET" && req.Method != "OPTIONS" {
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

// /dav1/123.txt -> dav1
func parsePrefixFromURL(url *url.URL) string {
	u := fmt.Sprint(url)
	return "/" + strings.Split(u, "/")[1]
}
