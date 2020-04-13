# Streaming Respirator (Server)

- [Stremaing Resepirator](https://github.com/RyuaNerin/StreamingRespirator) 의 설정으로, 개인 서버에서 실행하여 사용할 수 있게 하는게 목적입니다.
- 클라이언트에서 [스트리밍 호흡기 CA 인증서](https://raw.githubusercontent.com/RyuaNerin/StreamingRespirator/master/StreamingRespirator/Certificate/ca.crt)를 다운로드 후 설치해주세요
- 이 레포지토리는 기분이 내킬 때만 작업합니다.

- 포트 하나만 열어도 proxy, http, https 모두 지원됩니다.

## Arguments

|인자|필수|설명|
|----|:--:|-|
|`-cfg (VAL)`|O|설정파일 위치 (**계정 정보와 스트리밍 옵션만 불러옵니다.**)|
|`-id (VAL)`|O|프록시 인증용 ID|
|`-pw (VAL)`|O|프록시 인증용 PW|
|`-bind (VAL)`||서버에 사용할 바인딩 주소입니다.|
|`-server-cert (VAL)`|||http 서버에 사용할 SSL/TLS 인증서입니다. (`-bind` 옵션과 함께 사용됩니다.)|
|`-server-key (VAL)`|||http 서버에 사용할 SSL/TLS 인증서입니다. (`-bind` 옵션과 함께 사용됩니다.)|
|`-unix (VAL)`||유닉스 소켓에 바인딩합니다. plain 으로 제공합니다.|
|`-unix-perm (VAL)`||`-unix` 옵션 사용 시 유닉스 소켓의 퍼미션을 설정합니다. 기본값은 `0700` 입니다|
|`-proxy (VAL)`||내부 HTTP Client 에서 사용할 프록시입니다.|
|`-verbose`||자세한 로그를 출력합니다.|
|`-debug`||디버그 모드를 켭니다|

|상황|옵션|
|----|-|
|프록시 / HTTP 직접 연결|`-bind`|
|내부 직접 연결 (nginx 등 이용)|`-unix`|

- 옵션 예시
    - `streaming-respirator -cfg "StreamingRespirator.cfg" -id "auth" -pw "123456789" -bind ":8811"`
    - `streaming-respirator -cfg "StreamingRespirator.cfg" -id "auth" -pw "123456789" -bind ":8812" -server-cert "server.crt" -server-key "server.key"`
    - `streaming-respirator -cfg "StreamingRespirator.cfg" -id "auth" -pw "123456789" -unix "/run/streaming-respirator.sock"` + nginx (proxy_pass)

## 사용 방법

### 네트워크 바인딩 (`-bind`)

- 프록시모드
    - `Proxy-Authorization` 헤더를 설정해주어야 합니다.
        - 인증 방식 : `Basic`
        - `Proxy-Authorization: Basic <credentials>`
            - `<credentials>` : base64(`"<id>:<pw>"`)

    1. HTTP 클라이언트 기본 프록시 설정 사용
        - 아래 두 연결을 사용할 때 스트리밍 호흡기의 포트에 맞게 proxy 설정을 해주세요.
            - `https://streaming.twitter.com`
            - `https://api.twitter.com`

    2. 웹프록시 사용.
        - `https://userstream.twitter.com/A/B/C` →
            - `http://<host>:<port>/userstream.twitter.com/A/B/C`
            - `https://<host>:<port>/userstream.twitter.com/A/B/C`
        - `https://api.twitter.com/A/B/C` →
            - `http://<host>:<port>/api.twitter.com/A/B/C`
            - `https://<host>:<port>/api.twitter.com/A/B/C`

- 호흡기 서버 모드
    - `http://<host>:<port>/userstream?id=<user_id>`
    - `https://<host>:<port>/userstream?id=<user_id>`
    - Min Versino : `TLSv1.2`
    - HTTP 표준에 따라 `Authorization` 헤더를 설정해주어야 합니다.
    - 인증 방식 : `Basic`

### 유닉스 소켓 (`-unix`)

- nginx의 proxy_pass 와 같은 기능으로 사용하기 위한 설정입니다.

- TODO
