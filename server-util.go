package main

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func (s *streamingRespiratorServer) getHost(r *http.Request, defaultPort int) string {
	if _, _, err := net.SplitHostPort(r.URL.Host); err != nil {
		return fmt.Sprintf("%s:%d", r.URL.Host, defaultPort)
	}
	return r.URL.Host
}

func (s *streamingRespiratorServer) parseJsonId(path string) (uint64, bool) {
	n := strings.LastIndexByte(path, '/')
	if n == -1 || n+1 <= len(path) {
		return 0, false
	}
	path = path[n+1:]

	n = strings.IndexByte(path, '.')
	if n == -1 {
		return 0, false
	}
	path = path[:n]

	v, err := strconv.ParseUint(path, 10, 64)
	return v, err == nil
}

func (s *streamingRespiratorServer) getTwitterClient(r *http.Request) (act *Account, ok bool) {
	id, ok := s.parseOwnerId(r)
	if ok {
		for _, account := range Config.Accounts {
			if account.Id == id {
				return account, true
			}
		}
	}

	return
}

var (
	reParseOwnerIdFull = regexp.MustCompile(`oauth_token="?([0-9]+)\-`)
	reParseOwnerId     = regexp.MustCompile(`^([0-9]+)\-`)
)

func (s *streamingRespiratorServer) parseOwnerId(r *http.Request) (id uint64, ok bool) {
	if !r.URL.IsAbs() && r.URL.Path == PathSelf {
		if i, err := strconv.ParseUint(r.URL.Query().Get("id"), 10, 64); err == nil {
			return i, true
		}
	} else {
		parse := func(v string, fullParse bool) (id uint64, ok bool) {
			var m [][]string
			if fullParse {
				m = reParseOwnerIdFull.FindAllStringSubmatch(v, 1)
			} else {
				m = reParseOwnerId.FindAllStringSubmatch(v, 1)
			}
			if len(m) == 0 {
				return
			}

			i, err := strconv.ParseUint(m[0][1], 10, 64)
			if err != nil {
				return
			}
			return i, true
		}

		if id, ok = parse(r.Header.Get("Authorization"), true); ok {
			return
		}
		if id, ok = parse(r.FormValue("oauth_token"), false); ok {
			return
		}
		if id, ok = parse(r.PostFormValue("oauth_token"), false); ok {
			return
		}
	}

	return 0, false
}
