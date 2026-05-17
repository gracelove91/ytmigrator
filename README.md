# ytmigrator

YouTube 계정 마이그레이션 데스크톱 앱. 구독 채널, 재생목록, 좋아요 영상, Watch Later를 계정 A 에서 계정 B로 옮긴다.

## 기능

- **구독 채널 마이그레이션** — 알림 수준(activity type) 보존, 이미 구독 중이면 skip
- **재생목록 마이그레이션** — 영상 순서 보존, Watch Later 포함
- **좋아요 영상 마이그레이션** — 대상 계정에 like 재적용
- **작업 중단 시 Resume** — 쿼터 한도(10,000 unit/일) 대응. 진행 상태를 디스크에 저장하여 다음 날 이어서 시작
- **Desktop App OAuth** — 브라우저 연동 인증, 토큰은 메모리에만 보관

## 스택

| 레이어 | 선택 |
|--------|------|
| 백엔드 | Go 1.23 |
| 데스크톱 | Wails v2 |
| 프론트엔드 | React + TypeScript |
| API | YouTube Data API v3 |

## 빠른 시작

### 1. GCP 프로젝트 설정

1. [Google Cloud Console](https://console.cloud.google.com/) 접속 → 프로젝트 생성
2. `API 및 서비스 → API 라이브러리` → **YouTube Data API v3** 사용 설정
3. `사용자 인증 정보 → 사용자 인증 정보 만들기 → OAuth 클라이언트 ID`
4. 애플리케이션 유형: **데스크톱 앱**
5. JSON 다운로드 — 이 파일이 `client_secret.json`

### 2. 빌드 및 실행

```bash
wails build -o ytmigrator
open build/bin/ytmigrator.app
```

또는 개발 모드:
```bash
wails dev
```

### 3. 사용자 플로우

```
[앱 실행]
    ↓
최초: client_secret.json 선택 → 자동 저장
재실행: 저장된 인증 정보 자동 사용
    ↓
[원본 계정 로그인]  ← 브라우저 OAuth, 읽기 권한
    ↓
[Export Data]       ← /tmp/ytmigrator_export.json 생성
    ↓
[대상 계정 로그인]  ← 브라우저 OAuth, 쓰기 권한
    ↓
[Import Data]       ← 구독/재생목록/좋아요 복사
    ↓
Resume              ← 중단 시 다음 실행에서 이어서 진행
```

## 구조

```
ytmigrator/
├── app.go                      ← Wails 진입점, UI 바인딩 메서드
├── main.go                     ← 앱 설정 (Bind, OnStartup)
├── internal/
│   ├── youtube/
│   │   ├── model.go            ← 구독/재생목록/좋아요 구조체
│   │   ├── export.go           ← subscriptions.list, playlists.list, videos.list
│   │   ├── import.go           ← subscriptions.insert, playlists.insert, videos.rate
│   │   └── token.go            ← OAuth 메모리 토큰 관리
│   └── state/
│       └── progress.go         ← Resume 지원: 완료/실패 ID, 할당량 추적
├── frontend/
│   └── src/
│       └── App.tsx             ← React UI (버튼, 상태 표시)
└── wails.json                  ← 빌드 설정
```

## API 할당량

| 작업 | unit |
|------|------|
| list (구독/재생목록/좋아요) | 1 |
| insert (구독/재생목록 항목) | 50 |
| videos.rate (좋아요) | 50 |

기본 하루 할당량 10,000 unit. 200개 구독 + 10개 재생목록 기준 약 10,000 소모. Resume 기능으로 중단 시 다음 날 이어서 진행 가능.

## 참고

- [YouTube Data API v3](https://developers.google.com/youtube/v3)
- [Wails 문서](https://wails.io/docs/gettingstarted/firstproject)
- API 미지원 항목: 시청 기록, 홈피드 알고리즘
