# Streaming Respirator (Server)

- [Stremaing Resepirator](https://github.com/RyuaNerin/StreamingRespirator) 의 설정으로, 개인 서버에서 실행하여 사용할 수 있게 하는게 목적입니다.

- 이 레포지토리는 기분이 내킬 때만 작업합니다.

- 클라이언트에서 [스트리밍 호흡기 CA 인증서](https://raw.githubusercontent.com/RyuaNerin/StreamingRespirator/master/StreamingRespirator/Certificate/ca.crt)를 다운로드 후 설치해주세요

## args

- 아래의 인자를 모두 넣어 사용해주세요

- `-cfg` : 설정파일 위치
- `-bind` : 서버 바인딩 ip (잘 모르면 공란)
- `-id` : 프록시 인증용 ID
- `-pw` : 프록시 인증용 PW

예) `streaming-respirator -cfg "cfg"  -bind "" -id "auth" -pw "123456789"`

- `-verbose` : 설정 시 자세한 로그를 출력
