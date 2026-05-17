# ytmigrator

YouTube 계정 간 구독 채널·재생목록·좋아요·Watch Later를 옮기는 데스크톱 앱.

## 기능

- 구독 채널 마이그레이션 (알림 수준 보존)
- 재생목록 마이그레이션 (영상 순서 보존)
- 좋아요 표시 영상 마이그레이션
- Watch Later 마이그레이션
- 작업 중단 시 Resume 지원 (할당량 한도 대응)
- OAuth2 설치형 앱 흐름 (Desktop App)

## 스택

| 레이어 | 선택 | 이유 |
|--------|------|------|
| 백엔드 | Go | 싱글 바이너리, 뛰어난 동시성 |
| 데스크톱 | Wails v2 | Go + React → 네이티브 앱 |
| 프론트엔드 | React + TypeScript | 선언형 UI, 풍부한 생태계 |

## 빠른 시작

### 1. GCP 프로젝트 설정

- [Google Cloud Console](https://console.cloud.google.com/)에서 프로젝트 생성
- YouTube Data API v3 활성화
- OAuth 2.0 클라이언트 ID 생성 (애플리케이션 유형: **데스크톱 앱**)
- `client_secret.json` 다운로드

### 2. 앱 실행

```bash
wails dev
```

### 3. 인증 및 마이그레이션

1. 앱에서 `client_secret.json` 선택
2. 원본 계정으로 Google 인증 (읽기 권한)
3. 데이터 추출 (Export)
4. 대상 계정으로 Google 인증 (쓰기 권한)
5. 데이터 가져오기 (Import)

## 학습 포인트

이 프로젝트는 Go 언어 학습을 겸하고 있다:

- **OAuth2 installed-app flow**: 랜덤 localhost 포트, 브라우저 연동
- **Go ↔ JS 바인딩**: Wails의 `Bind` 메커니즘
- **구조체와 메서드**: `type App struct`, receiver
- **채널(channel)**: `make(chan string)`, `select`
- **고루틴(goroutine)**: `go func()`
- **에러 처리**: `(string, error)` 다중 반환, `fmt.Errorf(..., %w)`
- **sync.Mutex**: 할당량 카운터 보호
- **Context**: `context.Background()`, `runtime.BrowserOpenURL`

## 빌드

```bash
wails build
```

## 참고

- [YouTube Data API v3](https://developers.google.com/youtube/v3)
- [Wails 문서](https://wails.io/docs/gettingstarted/firstproject)
