package main

import (
	"container/list"
	"io"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/spf13/cast"
)

const (
	MaxUserCacheCount = 500
)

type Account struct {
	Id         uint64 `json:"id"`
	ScreenName string `json:"screen_name"`
	Cookie     string `json:"cookie"`

	onceInit         sync.Once
	cookieXCsrfToken string

	httpClient http.Client

	connectionsLock sync.RWMutex
	connections     *list.List // io.Writer. 소켓들

	tlHome    TimeLine
	tlAboutMe TimeLine
	tlDm      TimeLine

	userCacheLock sync.Mutex
	userCache     []userCache
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
		MaxIdleConnsPerHost: 32,
	}

	act.cookieXCsrfToken = regExtractXCsrfToken.FindStringSubmatch(act.Cookie)[1]
	act.connections = list.New()

	act.userCache = make([]userCache, 0, MaxUserCacheCount)

	// 타임라인 초기화
}

func (act *Account) AddConnectionAndWait(w io.Writer) {
	conn := Connection{
		w:    w,
		wait: make(chan struct{}),
		data: make(chan []byte),
	}

	act.connectionsLock.Lock()
	if act.connections.Len() == 0 {
		act.tlAboutMe.Start()
		act.tlDm.Start()
		act.tlHome.Start()
	}
	connNode := act.connections.PushBack(
		&conn,
	)
	act.connectionsLock.Unlock()

	go conn.Broadcaster()
	<-conn.wait

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

func (act *Account) CreateRequest(method string, url string, body io.Reader) (*http.Request, error) {
	act.onceInit.Do(act.Init)

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

func (act *Account) Send(packetList ...Packet) {
	act.onceInit.Do(act.Init)

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
						if packet, ok := NewPacket(&packetJson); ok {
							act.Send(packet)
							packet.Release()
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
