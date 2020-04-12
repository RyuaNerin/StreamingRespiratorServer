package main

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func parseJsonId(path string) (uint64, bool) {
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

func getTwitterClient(req *http.Request, isHttpServer bool) (act *Account, ok bool) {
	id, ok := parseOwnerId(req, isHttpServer)
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

func parseOwnerId(req *http.Request, isHttpServer bool) (id uint64, ok bool) {
	if isHttpServer {
		if i, err := strconv.ParseUint(req.URL.Query().Get("id"), 10, 64); err == nil {
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

		if id, ok = parse(req.Header.Get("Authorization"), true); ok {
			return
		}
		if id, ok = parse(req.FormValue("oauth_token"), false); ok {
			return
		}
		if id, ok = parse(req.PostFormValue("oauth_token"), false); ok {
			return
		}
	}

	return 0, false
}
