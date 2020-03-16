package main

import (
	"io"
	"net/http"
	"regexp"
	"sync"
)

type Account struct {
	Id         uint64 `json:"id"`
	ScreenName string `json:"screen_name"`
	Cookie     string `json:"cookie"`

	once sync.Once

	transport http.Client

	lock sync.RWMutex

	cookieXCsrfToken string

	connected       bool
	waitTimeHome    float32
	waitTimeAboutMe float32
	waitTimeDm      float32
}

var (
	accountMapLock sync.RWMutex
	accountMap     map[uint64]*Account
)

var (
	regExtractXCsrfToken = regexp.MustCompile(`ct0=([^;]+)`)
)

func (act *Account) CreateRequest(method string, url string, body io.Reader) (*http.Request, error) {
	act.once.Do(func() {
		act.cookieXCsrfToken = regExtractXCsrfToken.FindStringSubmatch(act.Cookie)[1]
	})

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header = http.Header{
		"Authorization":            []string{"Bearer AAAAAAAAAAAAAAAAAAAAAF7aAAAAAAAASCiRjWvh7R5wxaKkFp7MM%2BhYBqM%3DbQ0JPmjU9F6ZoMhDfI4uTNAaQuTDm2uO9x3WFVr2xBZ2nhjdP0"},
		"Cookie":                   []string{act.Cookie},
		"User-Agent":               []string{"StreamingRespirator"},
		"X-Csrf-Token":             []string{act.cookieXCsrfToken},
		"X-Twitter-Auth-Type":      []string{"OAuth2Session"},
		"X-Twitter-Client-Version": []string{"Twitter-TweetDeck-blackbird-chrome/4.0.190115122859 web/"},
	}

	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	return req, nil
}
