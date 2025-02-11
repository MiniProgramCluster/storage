package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"storage/conf"
	"storage/filecache"
	"storage/log"

	"github.com/golang-jwt/jwt/v5"
	"gopkg.in/natefinch/lumberjack.v2"
)

var fileCache = filecache.New()

func HMAC(msg string, secret string) string {
	key := []byte(secret)
	message := []byte(msg)
	h := hmac.New(sha256.New, key)
	h.Write(message)
	hmacValue := h.Sum(nil)
	return hex.EncodeToString(hmacValue)
}

func verify(r *http.Request, folder *conf.StorageConf, fileName string) bool {
	if !folder.IsPrivate(fileName) {
		return true
	}
	tokenStr := r.URL.Query().Get("t")
	endTimeStr := r.URL.Query().Get("e")
	if tokenStr == "" || endTimeStr == "" {
		log.Logger().Error().Msgf("filePath:%s is private, but token:%s endTime:%s is empty",
			r.URL.Path, tokenStr, endTimeStr)
		return false
	}
	timestamp, err := strconv.ParseInt(endTimeStr, 10, 64)
	if err != nil {
		log.Logger().Error().Msgf("filePath:%s is private, but endTime:%s is invalid",
			r.URL.Path, endTimeStr)
		return false
	}
	if timestamp < time.Now().Unix() {
		log.Logger().Error().Msgf("filePath:%s is private, but endTime:%s is expired",
			r.URL.Path, endTimeStr)
		return false
	}
	ok := HMAC(fmt.Sprintf("%s.%s", fileName, endTimeStr), folder.Private.Secret) == tokenStr
	if !ok {
		log.Logger().Error().Msgf("filePath:%s is private, but token:%s is invalid",
			r.URL.Path, tokenStr)
	}
	return ok
}

func verifyAuthorization(r *http.Request) bool {
	logger := log.Logger().With().Str("filePath", r.URL.Path).Logger()
	tokenStr := r.Header.Get("Authorization")
	if tokenStr == "" {
		logger.Error().Msgf("token is empty")
		return false
	}
	// bearer
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	tokenStr = strings.TrimSpace(tokenStr)
	logger = logger.With().Str("token", tokenStr).Logger()
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.GetAuthKey()), nil
	})
	if err != nil {
		logger.Error().Err(err).Msgf("token is invalid")
		return false
	}
	exp, err := token.Claims.GetExpirationTime()
	if err != nil {
		logger.Error().Err(err).Msgf("token is invalid")
		return false
	}
	if exp.Before(time.Now()) {
		logger.Error().Msgf("token is expired")
		return false
	}
	return true
}

func get(w http.ResponseWriter, r *http.Request, folder *conf.StorageConf, fileName string) {
	if !verify(r, folder, fileName) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	filePath := folder.FilePath(fileName)
	data, err := fileCache.Get(filePath)
	if err != nil {
		log.Logger().Error().Err(err).Msgf("filePath:%s not found", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", folder.MaxAge))
	w.Header().Set("Expires", time.Now().Add(time.Duration(folder.MaxAge)*time.Second).Format(http.TimeFormat))
	w.Write(data)
}

func put(w http.ResponseWriter, r *http.Request, folder *conf.StorageConf, fileName string) {
	if !verifyAuthorization(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	filePath := folder.FilePath(fileName)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		log.Logger().Error().Err(err).Msgf("filePath:%s read body failed", filePath)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = fileCache.Put(filePath, data)
	if err != nil {
		log.Logger().Error().Err(err).Msgf("filePath:%s put failed", filePath)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func delete(w http.ResponseWriter, r *http.Request, folder *conf.StorageConf, fileName string) {
	if !verifyAuthorization(r) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	filePath := folder.FilePath(fileName)
	err := fileCache.Delete(filePath)
	if err != nil {
		log.Logger().Error().Err(err).Msgf("filePath:%s delete failed", filePath)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func serveFile(w http.ResponseWriter, r *http.Request) {
	folder, fileName := conf.FileInfo(r.URL.Path)
	if folder == nil {
		log.Logger().Error().Msgf("filePath:%s not found", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch r.Method {
	case "GET":
		get(w, r, folder, fileName)
	case "PUT":
		put(w, r, folder, fileName)
	case "DELETE":
		delete(w, r, folder, fileName)
	}
}

func main() {
	configFile := flag.String("env", "env.json", "Set the env file path")
	logLevel := flag.String("log-level", "debug", "Set the log level (debug, info, warn, error, fatal, panic)")
	listenPort := flag.Int("listen", 9001, "Set the listen port")
	flag.Parse()
	log.Init(&lumberjack.Logger{
		Filename:   "./logs/app.log",
		MaxSize:    50, // 兆
		MaxBackups: 0,  // 不删除文件
		MaxAge:     28, // 保留28天
	}, *logLevel)
	conf.Init(*configFile)

	listenAddr := fmt.Sprintf("127.0.0.1:%d", *listenPort)
	log.Logger().Info().Msgf("Server starting on %s...", listenAddr)
	handler := http.NewServeMux()
	handler.HandleFunc("/storage/", serveFile)
	http.ListenAndServe(listenAddr, handler)
}
