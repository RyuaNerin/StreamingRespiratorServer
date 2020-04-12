package main

import (
	"io"
	"os"
	"syscall"

	jsoniter "github.com/json-iterator/go"
)

var Config struct {
	Accounts []*Account `json:"accounts"`

	ReduceApiCall bool `json:"reduce_api_call"`

	Filter struct {
		Retweeted          bool `json:"retweeted"`
		RetweetWithComment bool `json:"retweet_with_comment"`
		MyRetweet          bool `json:"my_retweet"`
	} `json:"filter"`
}

func loadConfig(path string) {
	fs, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	// ignore DOM
	b := make([]byte, 3)
	_, err = fs.Read(b)
	if err != nil {
		panic(err)
	}

	if b[0] != 0xEF || b[1] != 0xBB || b[2] != 0xBF {
		_, err := fs.Seek(0, syscall.FILE_BEGIN)
		if err != nil {
			panic(err)
		}
	}

	err = jsoniter.NewDecoder(fs).Decode(&Config)
	if err != nil && err != io.EOF {
		panic(err)
	}
}
