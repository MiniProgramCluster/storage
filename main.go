package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"storage/conf"
	"storage/filecache"
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

func verify(r *http.Request, conf *conf.StorageConf, fileName string) bool {
	if !conf.IsPrivate(fileName) {
		return true
	}
	tokenStr := r.URL.Query().Get("t")
	endTimeStr := r.URL.Query().Get("e")
	if tokenStr == "" || endTimeStr == "" {
		fmt.Printf("filePath:%s is private, but token:%s endTime:%s is empty\n",
			r.URL.Path, tokenStr, endTimeStr)
		return false
	}
	timestamp, err := strconv.ParseInt(endTimeStr, 10, 64)
	if err != nil {
		fmt.Printf("filePath:%s is private, but endTime:%s is invalid\n",
			r.URL.Path, endTimeStr)
		return false
	}
	if timestamp < time.Now().Unix() {
		fmt.Printf("filePath:%s is private, but endTime:%s is expired\n",
			r.URL.Path, endTimeStr)
		return false
	}
	ok := HMAC(fmt.Sprintf("%s.%s", fileName, endTimeStr), conf.Private.Secret) == tokenStr
	if !ok {
		fmt.Printf("filePath:%s is private, but token:%s is invalid\n",
			r.URL.Path, tokenStr)
	}
	return ok
}

func serveFile(w http.ResponseWriter, r *http.Request) {
	conf, fileName := conf.FileInfo(r.URL.Path)
	if conf == nil {
		fmt.Printf("filePath:%s not found\n", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if !verify(r, conf, fileName) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	filePath := conf.FilePath(fileName)
	data, err := fileCache.Get(filePath)
	if err != nil {
		fmt.Printf("filePath:%s not found\n", r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", conf.MaxAge))
	w.Header().Set("Expires", time.Now().Add(time.Duration(conf.MaxAge)*time.Second).Format(http.TimeFormat))
	w.Write(data)
}

func main() {
	configFile := flag.String("env", ".env.json", "Set the env file path")
	listenPort := flag.Int("listen", 9001, "Set the listen port")
	flag.Parse()

	conf.Init(*configFile)

	listenAddr := fmt.Sprintf("127.0.0.1:%d", *listenPort)
	fmt.Printf("Server starting on %s...", listenAddr)
	http.HandleFunc("/storage/", serveFile)
	http.ListenAndServe(listenAddr, nil)
}
