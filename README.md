# ytmigrator

> 한국어 | [English](#english)

YouTube 계정 마이그레이션 데스크톱 앱. 구독 채널, 재생목록, 좋아요 영상, Watch Later를 계정 A에서 계정 B로 옮긴다.

## 기능

- **구독 채널 마이그레이션** — 알림 수준(activity type) 보존, 이미 구독 중이면 skip
- **재생목록 마이그레이션** — 영상 순서 보존, Watch Later 포함
- **좋아요 영상 마이그레이션** — 대상 계정에 like 재적용
- **Export 미리보기 + 선택적 Import** — Export 후 "구독 N개, 재생목록 N개, 좋아요 N개"를 확인하고 원하는 항목만 Import (체크박스 선택)
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

## 사용자 플로우

<img width="600" alt="flow" src="https://user-images.githubusercontent.com/placeholder/flow-ko.png">

```
┌─────────────┐
│  앱 실행    │
└──────┬──────┘
       ↓
┌──────────────────────────┐
│ client_secret.json 선택   │  ← 최초 1회 (자동 저장)
│ 또는 저장된 인증 정보 사용 │  ← 재실행 시
└──────────┬───────────────┘
           ↓
┌──────────────────┐
│ 원본 계정 로그인  │  ← 브라우저 팝업, 읽기 권한
└──────┬───────────┘
       ↓
┌──────────────────┐
│ Export Data 클릭  │  ← /tmp/ytmigrator_export.json
└──────┬───────────┘
       ↓
┌─────────────────────────────────────────────┐
│ Export 미리보기 패널                         │
│  ✅ 구독 247개 / 재생목록 12개 / 좋아요 89개   │  ← 수량 확인
│                                              │
│  ☑️ 구독 [체크]                              │
│  ☑️ 재생목록 [체크]                          │  ← Import 항목 선택
│  ☐ 좋아요 [체크 해제]                        │
└──────┬──────────────────────────────────────┘
       ↓
┌──────────────────┐
│ 대상 계정 로그인  │  ← 브라우저 팝업, 쓰기 권한
└──────┬───────────┘
       ↓
┌──────────────────────────┐
│ Import Selected 클릭      │  ← 선택한 항목만 백그라운드 복사
│ "구독 채널명 (47/247)"    │  ← 실시간 진행 표시
└──────┬───────────────────┘
       ↓
   쿼터 초과? ──예──→ "quota exhausted at 50/247. progress saved — resume tomorrow"
       ↓ 아니오
   ┌─────────────┐
   │ Import 완료  │
   └─────────────┘

다음 날 재실행: 위 플로우 반복 → 이미 완료된 항목은 API 호출 없이 skip
```

## 빠른 시작

### 1. GCP 프로젝트 설정

1. [Google Cloud Console](https://console.cloud.google.com/) 접속 → 프로젝트 생성
2. `API 및 서비스 → API 라이브러리` → **YouTube Data API v3** 사용 설정
3. `사용자 인증 정보 → 사용자 인증 정보 만들기 → OAuth 클라이언트 ID`
4. 애플리케이션 유형: **데스크톱 앱**
5. JSON 다운로드 — 이 파일이 `client_secret.json`
6. `OAuth 동의 화면 → 테스트 사용자`에 자신의 Gmail 추가

### 2. 다운로드 및 실행

**macOS (Apple Silicon)**
```bash
# 릴리즈 페이지에서 ytmigrator-v0.1.0-darwin-arm64.zip 다운로드
unzip ytmigrator-v0.1.0-darwin-arm64.zip

# 실행 (로그도 함께 확인)
./ytmigrator.app/Contents/MacOS/ytmigrator

# GUI로 열기
open ytmigrator.app
```

### 3. 개발 모드

```bash
wails dev
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
Resume 기능으로 중단 시 다음 날 0-unit으로 이어서 진행.

## 알려진 이슈

- **테스트 사용자 오류**: "앱은 현재 테스트 중입니다" → GCP OAuth 동의 화면에서 자신의 Gmail을 테스트 사용자로 추가. Workspace/제한 계정은 추가되지 않을 수 있음
- **쿼터 한도**: 하루 10,000 unit 초과 시 API 호출이 실패. `import.go`가 `quotaExceeded`를 감지하여 즉시 중단하고 진행 상황을 저장함
- **콜백 주소**: macOS에서 IPv6 `::1` 대신 `127.0.0.1` 사용 (`app.go`)

## 참고

- [YouTube Data API v3](https://developers.google.com/youtube/v3)
- [Wails 문서](https://wails.io/docs/gettingstarted/firstproject)
- API 미지원 항목: 시청 기록, 홈피드 알고리즘

---

# English

> [한국어](#ytmigrator) | English

A desktop app for migrating YouTube account data. Move subscriptions, playlists, liked videos, and Watch Later from Account A to Account B.

## Features

- **Subscription Migration** — Preserves notification level (activity type), skips already-subscribed channels
- **Playlist Migration** — Preserves video order, includes Watch Later
- **Liked Videos Migration** — Re-applies "like" ratings to the target account
- **Export Preview + Selective Import** — After export, review counts (subscriptions, playlists, liked videos) and choose which ones to import via checkboxes
- **Resume on Interruption** — Handles the daily quota limit (10,000 units/day). Saves progress to disk and resumes the next day.
- **Async Import** — Runs in a background goroutine; UI never freezes
- **Real-time Progress** — Shows current progress like "Subscriptions: Channel Name (47/247)"
- **Auto-stop on Quota Exhaustion** — Detects `quotaExceeded` and immediately stops while saving progress
- **Desktop OAuth** — Browser-based authentication; tokens live only in memory

## Stack

| Layer | Choice |
|-------|--------|
| Backend | Go 1.23 |
| Desktop | Wails v2 |
| Frontend | React + TypeScript |
| API | YouTube Data API v3 |

## User Flow

<img width="600" alt="flow" src="https://user-images.githubusercontent.com/placeholder/flow-en.png">

```
┌──────────────┐
│  Launch App  │
└──────┬───────┘
       ↓
┌───────────────────────────────┐
│ Select client_secret.json     │  ← First time only (auto-saved)
│ or reuse stored credentials   │  ← On relaunch
└──────────┬────────────────────┘
           ↓
┌──────────────────────┐
│ Login Source Account │  ← Browser popup, read-only scope
└──────┬───────────────┘
       ↓
┌──────────────────────┐
│ Click Export Data    │  ← Creates /tmp/ytmigrator_export.json
└──────┬───────────────┘
       ↓
┌───────────────────────────────────────┐
│ Export Preview Panel                  │
│  ✅ Subscriptions 247 / 12 playlists / 89 likes │ ← Review counts
│                                       │
│  ☑️ Subscriptions [checked]           │
│  ☑️ Playlists [checked]               │  ← Select what to import
│  ☐ Liked Videos [unchecked]           │
└──────┬────────────────────────────────┘
       ↓
┌──────────────────────┐
│ Login Target Account │  ← Browser popup, write scope
└──────┬───────────────┘
       ↓
┌──────────────────────────────────┐
│ Click Import Selected            │  ← Background copy of selected items
│ "Subscriptions: Name (47/247)"   │  ← Real-time progress
└──────┬───────────────────────────┘
       ↓
   Quota hit? ──Yes──→ "quota exhausted at 50/247. progress saved — resume tomorrow"
       ↓ No
   ┌─────────────┐
   │ Import Done  │
   └─────────────┘

Next day relaunch: Repeat the flow → completed items are skipped with zero API cost
```

## Quick Start

### 1. GCP Project Setup

1. Go to [Google Cloud Console](https://console.cloud.google.com/) → Create project
2. `APIs & Services → Library` → Enable **YouTube Data API v3**
3. `Credentials → Create Credentials → OAuth client ID`
4. Application type: **Desktop app**
5. Download JSON — this is `client_secret.json`
6. `OAuth consent screen → Test users` → Add your Gmail

### 2. Download & Run

**macOS (Apple Silicon)**
```bash
# Download ytmigrator-v0.1.0-darwin-arm64.zip from Releases
unzip ytmigrator-v0.1.0-darwin-arm64.zip

# Run (with terminal logs)
./ytmigrator.app/Contents/MacOS/ytmigrator

# Or open via GUI
open ytmigrator.app
```

### 3. Development Mode

```bash
wails dev
```

## Architecture

```
ytmigrator/
├── app.go                      ← Wails entry, async Import + event emission
├── main.go                     ← App config (Bind, OnStartup)
├── internal/
│   ├── youtube/
│   │   ├── model.go            ← Subscription/Playlist/Like structs + ImportProgressCallback
│   │   ├── export.go           ← subscriptions.list, playlists.list, videos.list
│   │   ├── import.go           ← subscriptions.insert, playlists.insert, videos.rate (quotaExceeded detection)
│   │   └── token.go            ← OAuth in-memory token management
│   └── state/
│       └── progress.go         ← Resume support: completed/failed IDs, quota tracking
├── frontend/
│   └── src/
│       ├── App.tsx             ← React UI (EventsOn for real-time progress)
│       └── style.css           ← Dark theme contrast fixes
└── wails.json                  ← Build settings
```

## API Quota

| Operation | units |
|-----------|-------|
| list (subscriptions/playlists/likes) | 1 |
| insert (subscriptions/playlist items) | 50 |
| videos.rate (likes) | 50 |

Default daily quota: 10,000 units. ~200 subscriptions + 10 playlists ≈ 10,000 units.
Resume continues with zero cost for already-completed items.

## Known Issues

- **Test user error**: "App is in testing phase" → Add your Gmail as a test user in GCP OAuth consent screen. Workspace/restricted accounts may fail.
- **Quota limit**: Daily 10,000 units. `import.go` detects `quotaExceeded` and immediately stops while saving progress.
- **Callback address**: Uses `127.0.0.1` instead of IPv6 `::1` on macOS (`app.go`)

## References

- [YouTube Data API v3](https://developers.google.com/youtube/v3)
- [Wails Documentation](https://wails.io/docs/gettingstarted/firstproject)
- Unsupported items: watch history, home feed algorithm
