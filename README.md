# ytmigrator

YouTube 계정 마이그레이션 데스크톱 앱. 구독 채널, 재생목록, 좋아요 영상, Watch Later를 계정 A에서 계정 B로 옮긴다.

## 기능

- **구독 채널 마이그레이션** — 알림 수준(activity type) 보존, 이미 구독 중이면 skip
- **재생목록 마이그레이션** — 영상 순서 보존, Watch Later 포함
- **좋아요 영상 마이그레이션** — 대상 계정에 like 재적용
- **작업 중단 시 Resume** — 쿼터 한도(10,000 unit/일) 대응. 진행 상태를 디스크에 저장하여 다음 날 이어서 시작
- **비동기 Import** — 백그라운드 goroutine으로 실행, UI가 얼어버리지 않음
- **실시간 진행 상황** — "구독 채널명 (47/247)" 형태로 상태 표시
- **쿼터 초과 자동 중단** — quotaExceeded 오류 감지 시 즉시 정지하고 progress 저장
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
6. `OAuth 동의 화면 → 테스트 사용자`에 자신의 Gmail 추가

### 2. 빌드

```bash
wails build -clean
```

### 3. 실행

```bash
# GUI 창 + 터미널 로그 같이 보기
/Users/morris/projects/ytmigrator/build/bin/ytmigrator.app/Contents/MacOS/ytmigrator
```

macOS Finder에서 `open build/bin/ytmigrator.app`을 하면 실행 파일이 아직 생성되지 않은 상태에서 오류가 날 수 있다. 위 경로로 직접 실행하거나, 재빌드 후 다시 시도.

개발 모드:
```bash
wails dev
```

### 4. 사용자 플로우

```
[앱 실행]
    ↓
최초: client_secret.json 선택 → 자동 저장 (~/Library/Application Support/ytmigrator/)
재실행: 저장된 인증 정보 자동 사용
    ↓
[원본 계정 로그인]  ← 브라우저 OAuth, 읽기 권한 (youtube.readonly)
    ↓
[Export Data]       ← /tmp/ytmigrator_export.json 생성
    ↓
[대상 계정 로그인]  ← 브라우저 OAuth, 쓰기 권한 (youtube.force-ssl)
    ↓
[Import Data]       ← 백그라운드에서 구독/재생목록/좋아요 복사 시작
                      UI는 "구독 채널명 (47/247)" 형태로 진행 표시
    ↓
쿼터 한도 도달      ← "quota exhausted at 50/247 subscriptions. progress saved — resume tomorrow"
재실행 후 재개       ← 완료된 항목은 API 호출 없이 skip, 진행 재개
```

## 구조

```
ytmigrator/
├── app.go                      ← Wails 진입점, 비동기 Import + 이벤트 발행
├── main.go                     ← 앱 설정 (Bind, OnStartup)
├── internal/
│   ├── youtube/
│   │   ├── model.go            ← 구독/재생목록/좋아요 구조체 + ImportProgressCallback
│   │   ├── export.go           ← subscriptions.list, playlists.list, videos.list
│   │   ├── import.go           ← subscriptions.insert, playlists.insert, videos.rate (quotaExceeded 감지)
│   │   └── token.go            ← OAuth 메모리 토큰 관리
│   └── state/
│       └── progress.go         ← Resume 지원: 완료/실패 ID, 할당량 추적
├── frontend/
│   └── src/
│       ├── App.tsx             ← React UI (EventsOn으로 실시간 진행 수신)
│       └── style.css           ← 다크 테마 배경/글자색 대비 조정
└── wails.json                  ← 빌드 설정
```

## API 할당량

| 작업 | unit |
|------|------|
| list (구독/재생목록/좋아요) | 1 |
| insert (구독/재생목록 항목) | 50 |
| videos.rate (좋아요) | 50 |

기본 하루 할당량 10,000 unit. 200개 구독 + 10개 재생목록 기준 약 10,000 소모.
Import 시각 달라도 전체 리스트를 읽으므로 소스 export 자체는 쿼터를 소모하지 않음 (내보내기는 API에서 읽어오지만 소스 계정의 한도만 사용).

## 알려진 이슈

- **테스트 사용자 오류**: "앱은 현재 테스트 중입니다" → GCP OAuth 동의 화면에서 자신의 Gmail을 테스트 사용자로 추가. Workspace/제한 계정은 추가되지 않을 수 있음
- **쿼터 한도**: 하루 10,000 unit 초과 시 API 호출이 실패. `import.go`가 `quotaExceeded`를 감지하여 즉시 중단하고 진행 상황을 저장함
- **콜백 주소**: macOS에서 IPv6 `::1` 대신 `127.0.0.1` 사용 (`app.go`)

## 참고

- [YouTube Data API v3](https://developers.google.com/youtube/v3)
- [Wails 문서](https://wails.io/docs/gettingstarted/firstproject)
- API 미지원 항목: 시청 기록, 홈피드 알고리즘
