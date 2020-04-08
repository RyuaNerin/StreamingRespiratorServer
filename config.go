package main

import (
	"io"
	"os"

	jsoniter "github.com/json-iterator/go"
)

var Config struct {
	Accounts []Account `json:"accounts"`

	ReduceApiCall bool `json:"reduce_api_call"`

	Filter struct {
		Retweeted          bool `json:"retweeted"`
		RetweetWithComment bool `json:"retweet_with_comment"`
		MyRetweet          bool `json:"my_retweet"`
	} `json:"filter"`
}

func LoadConfig(path string) {
	fs, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	err = jsoniter.NewDecoder(fs).Decode(&Config)
	if err != nil && err != io.EOF {
		panic(err)
	}
}
