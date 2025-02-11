package conf

import (
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	"storage/log"
)

var PREFIX = "/storage/"

type privateConf struct {
	Secret string   `json:"secret"`
	Pathes []string `json:"pathes"`
}

type StorageConf struct {
	Dir     string      `json:"dir"`
	MaxAge  uint64      `json:"maxAge"`
	Private privateConf `json:"private"`
}

type Conf struct {
	Folders map[string]*StorageConf `json:"folders"`
	AuthKey string                  `json:"authKey"`
}

var conf Conf

func (c *StorageConf) String() string {
	return fmt.Sprintf("dir:%s, maxAge:%d, private:%+v", c.Dir, c.MaxAge, c.Private)
}

func (c *privateConf) String() string {
	return fmt.Sprintf("secret:%s, pathes:%+v", c.Secret, c.Pathes)
}

func (c *StorageConf) IsPrivate(path string) bool {
	for _, p := range c.Private.Pathes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func (c *StorageConf) FilePath(fileName string) string {
	return c.Dir + fileName
}

func parseConfig() {
	var tmp Conf
	if err := viper.Unmarshal(&tmp); err != nil {
		log.Logger().Error().Err(err).Msg("Failed to unmarshal app config")
	}
	conf = tmp
	log.Logger().Debug().Msgf("conf:%+v", conf)
}

func Init(file string) {
	// 使用 viper 加载 JSON 配置文件
	viper.SetConfigFile(file)   // 配置文件路径
	viper.SetConfigType("json") // 配置文件类型
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Logger().Info().Msgf("config file changed: %s", e.Name)
		parseConfig()
	})
	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		log.Logger().Error().Err(err).Msg("Failed to read config file")
	}
	parseConfig()
}

func FileInfo(url string) (*StorageConf, string) {
	if !strings.HasPrefix(url, PREFIX) {
		log.Logger().Error().Msgf("invalid url: %s", url)
		return nil, ""
	}
	url = url[len(PREFIX):]
	n := strings.Index(url, "/")
	if n == -1 {
		log.Logger().Error().Msgf("invalid url format: %s", url)
		return nil, ""
	}
	dir := url[:n+1]
	folder := conf.Folders[dir]
	if folder == nil {
		log.Logger().Error().Msgf("invalid directory in url: %s (dir: %s)", url, dir)
		return nil, ""
	}
	fileName := url[n+1:]
	log.Logger().Debug().Msgf("file request - path:%s maxAge:%d filename:%s", folder.Dir, folder.MaxAge, fileName)
	return folder, fileName
}

func GetAuthKey() string {
	return conf.AuthKey
}
