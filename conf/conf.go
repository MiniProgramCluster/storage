package conf

import (
	"fmt"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
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

var storageConfs map[string]*StorageConf

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
	storageConfs = make(map[string]*StorageConf)
	if err := viper.Unmarshal(&storageConfs); err != nil {
		fmt.Println("Failed to unmarshal app config")
	}
	fmt.Printf("storageConfs:%+v\n", storageConfs)
}

func Init(file string) {
	// 使用 viper 加载 JSON 配置文件
	viper.SetConfigFile(file)   // 配置文件路径
	viper.SetConfigType("json") // 配置文件类型
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("config file changed:%s\n", e.Name)
		parseConfig()
	})
	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Failed to unmarshal app config")
	}
	parseConfig()
}

func FileInfo(url string) (*StorageConf, string) {
	if !strings.HasPrefix(url, PREFIX) {
		fmt.Printf("url:%s is not valid\n", url)
		return nil, ""
	}
	url = url[len(PREFIX):]
	n := strings.Index(url, "/")
	if n == -1 {
		fmt.Printf("url:%s is not valid\n", url)
		return nil, ""
	}
	dir := url[:n+1]
	conf := storageConfs[dir]
	if conf == nil {
		fmt.Printf("url:%s is not valid dir:%s\n", url, dir)
		return nil, ""
	}
	fileName := url[n+1:]
	fmt.Printf("filePath:%s, maxAge:%d %s\n", conf.Dir, conf.MaxAge, fileName)
	return conf, fileName
}
