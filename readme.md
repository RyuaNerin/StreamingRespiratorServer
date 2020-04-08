# Streaming Respirator (Server)

- [Stremaing Resepirator](https://github.com/RyuaNerin/StreamingRespirator) 의 설정으로, 개인 서버에서 실행하여 사용할 수 있게 하는게 목적입니다.

- 이 레포지토리는 기분이 내킬 때만 작업합니다.

- 클라이언트에서 [스트리밍 호흡기 CA 인증서](https://raw.githubusercontent.com/RyuaNerin/StreamingRespirator/master/StreamingRespirator/Certificate/ca.crt)를 다운로드 후 설치해주세요

## Arguments

|인자|필수|설명|
|----|:--:|-|
|`-cfg (VAL)`|O|설정파일 위치 (**계정 정보와 스트리밍 옵션만 불러옵니다.**)|
|`-id (VAL)`|O|프록시 인증용 ID|
|`-pw (VAL)`|O|프록시 인증용 PW|
|`-proxy (VAL)`||프록시 바인딩 주소입니다.|
|`-http (VAL)`||http 연결 시 사용할 바인딩 주소입니다. 지정하지 않을 시 프록시 모드만 사용합니다.|
|`-http-plain`||http 연결에 https 를 사용하지 않습니다. (`-http` 옵션과 함께 사용됩니다.)|
|`-unix (VAL)`||유닉스 소켓에 바인딩합니다.|
|`-unix-perm (VAL)`||`-unix` 옵션 사용 시 유닉스 소켓의 퍼미션을 설정합니다. 기본값은 `0700` 입니다|
|`-verbose`||자세한 로그를 출력합니다.|

|상황|옵션|
|----|-|
|프록시|`-proxy`|
|외부 직접 연결|`-http`|
|내부 직접 연결|`-unix`|

- 옵션 예시
    - `streaming-respirator -cfg "StreamingRespirator.cfg" -id "auth" -pw "123456789" -proxy ":8811"`
    - `streaming-respirator -cfg "StreamingRespirator.cfg" -id "auth" -pw "123456789" -proxy ":8811" -http ":8812"`
