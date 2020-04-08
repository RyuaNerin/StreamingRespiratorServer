package main

import (
	"context"
	"io"
	"strconv"
	"sync"
	"time"
)

const (
	TLWaitMin   = 1 * time.Second
	TLWaitError = 10 * time.Second
)

type TimeLine struct {
	account *Account

	runningLock      sync.Mutex
	runningCtxCancel context.CancelFunc // 갱신 취소용 함수

	funcGetUrl func(cursor string) (method string, url string) // cursor -> URL
	funcMain   func(r io.Reader, isFirstRefresh bool) (cursor string, streamingStatuses []interface{}, users map[uint64]map[string]interface{})
}

func (tl *TimeLine) Start() {
	tl.runningLock.Lock()
	defer tl.runningLock.Unlock()

	if tl.runningCtxCancel != nil {
		return
	}

	ctx, ctxCacnel := context.WithCancel(context.Background())
	tl.runningCtxCancel = ctxCacnel

	go tl.refreshThread(ctx)
}
func (tl *TimeLine) Stop() {
	tl.runningLock.Lock()
	defer tl.runningLock.Unlock()

	if tl.runningCtxCancel == nil {
		return
	}

	tl.runningCtxCancel()
	tl.runningCtxCancel = nil

}

func (tl *TimeLine) refreshThread(ctx context.Context) {
	var cursor string

	isFirstRefresh := true
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		method, url := tl.funcGetUrl(cursor)
		c, w := tl.update(ctx, method, url, isFirstRefresh)
		if c != "" {
			cursor = c
		}

		isFirstRefresh = false

		select {
		case <-ctx.Done():
			return
		case <-time.After(w):
		}
	}
}

func (tl *TimeLine) update(ctx context.Context, method, url string, isFirstRefresh bool) (cursorNew string, wait time.Duration) {
	req, _ := tl.account.CreateRequest(method, url, nil)
	req.WithContext(ctx)

	res, err := tl.account.httpClient.Do(req)
	if err != nil {
		return "", TLWaitError
	}

	// Todo. user_modified
	cursor, streamingStatuses, users := tl.funcMain(res.Body, isFirstRefresh)

	if !isFirstRefresh {
		tl.account.Send(streamingStatuses...)
	}

	go tl.account.UserCache(users)

	wait = TLWaitError
	if remaining, err := strconv.ParseInt(res.Header.Get("x-rate-limit-remaining"), 10, 64); err == nil {
		if reset, err := strconv.ParseInt(res.Header.Get("x-rate-limit-reset"), 10, 64); err == nil {
			nowUnix := time.Now().Unix()

			if reset < nowUnix {
				// 현재 시간보다 초기화 시간이 작을 때
				wait = TLWaitMin
			} else if remaining == 0 {
				// 리밋이 하나도 남지 않았을 때
				wait = time.Until(time.Unix(reset, 0))
			} else {
				// 분산
				wait = time.Duration((float64(reset)-float64(nowUnix))/float64(remaining)*1000) * time.Millisecond

				if Config.ReduceApiCall {
					wait = wait * 2
				}
			}
		}
	}

	if wait < TLWaitMin {
		wait = TLWaitMin
	}

	return cursor, wait
}
