package main

import (
	"container/list"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/spf13/cast"
)

const (
	MaxUserCacheCount       = 500
	FriendsRefreshPeriod    = 30 * time.Minute
	VerifyCredentialExpires = time.Minute
)

type Account struct {
	Id               uint64 `json:"id"`
	ScreenName       string `json:"screen_name"`
	Cookie           string `json:"cookie"`
	cookieXCsrfToken string

	httpClient http.Client

	connectionsLock sync.RWMutex
	connections     *list.List // io.Writer. 소켓들

	tlHome    TimeLine
	tlAboutMe TimeLine
	tlDm      TimeLine

	userCacheLock sync.Mutex
	userCache     []userCache

	verifiedLock   sync.Mutex
	verifiedAt     time.Time
	verifiedResult bool
}

type userCache struct {
	id           uint64
	name         string
	screenName   string
	profileImage string

	lastModified time.Time
}

var (
	accountMapLock sync.RWMutex
	accountMap     map[uint64]*Account
)

var (
	regExtractXCsrfToken = regexp.MustCompile(`ct0=([^;]+)`)
)

func (act *Account) Init() {
	act.httpClient.Transport = &http.Transport{
		MaxIdleConnsPerHost:   32,
		Proxy:                 proxy,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 30 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	act.cookieXCsrfToken = regExtractXCsrfToken.FindStringSubmatch(act.Cookie)[1]
	act.connections = list.New()

	act.userCache = make([]userCache, 0, MaxUserCacheCount)

	// 타임라인 초기화
	act.tlHome = TimeLine{
		account:    act,
		funcGetUrl: tlHomeGetUrl,
		funcMain:   tlHomeMain,
	}
	act.tlAboutMe = TimeLine{
		account:    act,
		funcGetUrl: tlAboutMeGetUrl,
		funcMain:   tlAboutMeMain,
	}
	act.tlDm = TimeLine{
		account:    act,
		funcGetUrl: tlDMGetUrl,
		funcMain:   tlDMMain,
	}

	if verbose {
		act.tlHome.name = fmt.Sprintf("Home (@%s)", act.ScreenName)
		act.tlAboutMe.name = fmt.Sprintf("AboutMe (@%s)", act.ScreenName)
		act.tlDm.name = fmt.Sprintf("DM (@%s)", act.ScreenName)
	}
}

func (act *Account) VerifyCredentials(ctx context.Context) bool {
	act.verifiedLock.Lock()
	defer act.verifiedLock.Unlock()

	if time.Now().Before(act.verifiedAt.Add(VerifyCredentialExpires)) {
		return act.verifiedResult
	}

	for i := 0; i < 3; i++ {
		req, _ := act.CreateRequest(
			context.Background(),
			"GET",
			"https://api.twitter.com/1.1/account/verify_credentials.json",
			nil,
		)
		res, err := act.httpClient.Do(req)
		if err != nil {
			logger.Printf("%+v\n", err)
			sentry.CaptureException(err.(error))
		} else {
			defer res.Body.Close()

			if res.StatusCode == http.StatusOK {
				act.verifiedAt = time.Now()
				act.verifiedResult = true
				return true
			}
		}

		time.Sleep(3 * time.Second)
	}

	act.verifiedAt = time.Now()
	act.verifiedResult = false
	return false
}

func (act *Account) AddConnectionAndWait(w http.ResponseWriter, ctx context.Context) {
	conn := newConnection(w, ctx)

	//////////////////////////////////////////////////

	act.connectionsLock.Lock()
	if act.connections.Len() == 0 {
		act.tlAboutMe.Start()
		act.tlDm.Start()
		act.tlHome.Start()
	}
	connNode := act.connections.PushBack(conn)
	act.connectionsLock.Unlock()

	//////////////////////////////////////////////////

	// Friends 날린다
	var tmrWorking int32 = 0
	var tmrFriends *time.Timer

	var sendFriends func()
	sendFriends = func() {
		if atomic.LoadInt32(&tmrWorking) != 0 {
			return
		}

		req, _ := act.CreateRequest(
			context.Background(),
			"GET",
			"https://api.twitter.com/1.1/friends/ids.json?count=5000user_id="+strconv.FormatUint(act.Id, 10),
			nil,
		)
		req.WithContext(ctx)
		res, err := act.httpClient.Do(req)
		if err == nil {
			defer res.Body.Close()

			var friendsCursor struct {
				Ids []uint64 `json:"ids"`
			}
			if err = jsonTwitter.NewDecoder(res.Body).Decode(&friendsCursor); err == nil {
				packetJson := PacketFriends{
					Friends: friendsCursor.Ids,
				}
				if packet, ok := newPacket(&packetJson); ok {
					conn.Send(packet.d)
				}
			}
		}

		tmrFriends = time.AfterFunc(FriendsRefreshPeriod, sendFriends)
	}
	sendFriends()

	//////////////////////////////////////////////////

	select {
	case <-ctx.Done():
	case <-conn.chanClosed:
	}

	atomic.StoreInt32(&tmrWorking, 1)
	tmrFriends.Stop()

	//////////////////////////////////////////////////

	act.connectionsLock.Lock()
	act.connections.Remove(connNode)

	if act.connections.Len() == 0 {
		act.tlAboutMe.Stop()
		act.tlDm.Stop()
		act.tlHome.Stop()

		act.userCacheLock.Lock()
		act.userCache = act.userCache[:0]
		act.userCacheLock.Unlock()
	}
	act.connectionsLock.Unlock()
}

func (act *Account) CreateRequest(ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
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

// 전달된 패킷은 전송 후 Release 됨.
func (act *Account) Send(packetList ...Packet) {
	act.connectionsLock.RLock()
	connList := make([]*Connection, 0, act.connections.Len())
	{
		conn := act.connections.Front()
		for conn != nil {
			connList = append(connList, conn.Value.(*Connection))
			conn = conn.Next()
		}
	}
	act.connectionsLock.RUnlock()

	var wg sync.WaitGroup
	for _, conn := range connList {
		wg.Add(1)

		c := conn
		go func() {
			defer wg.Done()

			for _, d := range packetList {
				go c.Send(d.d)
			}
		}()
	}

	wg.Wait()

	for _, packet := range packetList {
		packet.Release()
	}
}

func (act *Account) SendStatusRemoved(id uint64, userId uint64) {
	var packetJson PacketDelete
	packetJson.Delete.Status = PacketDeleteStatus{
		Id:        id,
		IdStr:     strconv.FormatUint(id, 10),
		UserId:    userId,
		UserIdStr: strconv.FormatUint(userId, 10),
	}

	if packet, ok := newPacket(&packetJson); ok {
		act.Send(packet)
	}
}
func (act *Account) sendStatusRemovedFromStatus(v TwitterStatus) {
	id, err := cast.ToUint64E(v["id"])
	if err == nil {
		return
	}
	user, err := cast.ToStringMapE(v["user"])
	if err != nil {
		return
	}
	userId, err := cast.ToUint64E(user["id"])
	if err != nil {
		return
	}

	act.SendStatusRemoved(id, userId)
}
func (act *Account) SendStatusRemovedWithCheck(id uint64) {
	req, _ := act.CreateRequest(
		context.Background(),
		"GET",
		"https://api.twitter.com/1.1/statuses/show.json?id="+strconv.FormatUint(id, 10),
		nil,
	)

	res, err := act.httpClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	var v struct {
		Id uint64 `json:"id"`
	}
	if err := jsonTwitter.NewDecoder(res.Body).Decode(&v); err != nil && err != io.EOF {
		logger.Printf("%+v\n", err)
		sentry.CaptureException(err.(error))
		return
	}

	act.SendStatusRemoved(v.Id, 0)
}

func (act *Account) GetUserCache(findId uint64, findScreenName string) (id uint64, ScreenName string, ok bool) {
	act.userCacheLock.Lock()
	defer act.userCacheLock.Unlock()

	for _, uc := range act.userCache {
		if findId != 0 && uc.id == findId {
			return uc.id, uc.screenName, true
		}
		if findScreenName != "" && uc.screenName == findScreenName {
			return uc.id, uc.screenName, true
		}
	}

	return
}

func (act *Account) UserCache(users map[uint64]TwitterUser) {
	act.userCacheLock.Lock()
	defer act.userCacheLock.Unlock()

	for _, user := range users {
		id, err := cast.ToUint64E(user["id"])
		if err != nil {
			continue
		}

		name, err := cast.ToStringE(user["name"])
		if err != nil {
			continue
		}
		screenName, err := cast.ToStringE(user["name"])
		if err != nil {
			continue
		}
		profileImage, err := cast.ToStringE(user["profile_image_url"])
		if err != nil {
			continue
		}

		exists := false
		for i, uc := range act.userCache {
			if uc.id == id {
				act.userCache[i].lastModified = time.Now()

				if uc.name == name || uc.screenName != screenName || uc.profileImage != profileImage {
					act.userCache[i].name = name
					act.userCache[i].screenName = screenName
					act.userCache[i].profileImage = profileImage

					go func() {
						packetJson := PacketEvent{
							Event:     "user_update",
							CreatedAt: time.Now(),
							Source:    user,
							Target:    user,
						}
						if packet, ok := newPacket(&packetJson); ok {
							act.Send(packet)
						}
					}()
				}

				exists = true
				break
			}
		}

		if exists {
			continue
		}

		for len(act.userCache) >= MaxUserCacheCount {
			minIndex := 0
			for i := range act.userCache {
				if act.userCache[i].lastModified.Before(act.userCache[minIndex].lastModified) {
					minIndex = i
				}
			}

			act.userCache[minIndex] = act.userCache[len(act.userCache)-1]
			act.userCache = act.userCache[:len(act.userCache)-1]
		}

		act.userCache = append(
			act.userCache,
			userCache{
				id:           id,
				name:         name,
				screenName:   screenName,
				profileImage: profileImage,
				lastModified: time.Now(),
			},
		)
	}
}

func (act *Account) GetUserId(ctx context.Context, screenName string) (userId uint64, ok bool) {
	userId, _, ok = act.GetUserCache(0, screenName)
	if ok {
		return userId, true
	}

	req, _ := act.CreateRequest(
		ctx,
		"GET",
		"https://api.twitter.com/1.1/users/show.json?screen_name="+url.QueryEscape(screenName),
		nil,
	)
	resp, err := act.httpClient.Do(req)
	if err != nil {
		logger.Printf("%+v\n", err)
		sentry.CaptureException(err.(error))
		return
	}
	defer resp.Body.Close()

	var tu struct {
		Id uint64 `json:"id"`
	}
	if err := jsonTwitter.NewDecoder(resp.Body).Decode(&tu); err != nil && err != io.EOF {
		logger.Printf("%+v\n", err)
		sentry.CaptureException(err.(error))
		return
	}

	return tu.Id, true
}
