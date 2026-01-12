# StableNet 스트레스 테스트 도구 - 프로젝트 기획서

## 1. 프로젝트 이름 후보 (50개)

### 우주/천체 테마 (Supernova 계열)
| # | 이름 | 의미 |
|---|------|------|
| 1 | **Starburst** | 별의 폭발, 대규모 부하 테스트 |
| 2 | **Pulsar** | 맥동성, 주기적 테스트 |
| 3 | **Quasar** | 준성, 강력한 에너지 |
| 4 | **Nebula** | 성운, 분산 테스트 |
| 5 | **Meteor** | 유성, 빠른 거래 |
| 6 | **Comet** | 혜성, 지속적 테스트 |
| 7 | **Nova** | 신성, 새로운 테스트 도구 |
| 8 | **Solaris** | 태양, 핵심 테스트 |
| 9 | **Cosmos-Hammer** | 우주 해머 |
| 10 | **Starforge** | 별 제조소, 거래 생성 |

### Stable/안정성 테마
| # | 이름 | 의미 |
|---|------|------|
| 11 | **StableStorm** | 안정 폭풍, 역설적 강조 |
| 12 | **StableForce** | 안정 파워 |
| 13 | **StablePulse** | 안정 펄스, 주기적 테스트 |
| 14 | **StableStrike** | 안정 타격 |
| 15 | **StableBlast** | 안정 폭발 |
| 16 | **StableFlood** | 안정 홍수, 대량 거래 |
| 17 | **StableRush** | 안정 러시 |
| 18 | **StableSurge** | 안정 급증 |
| 19 | **StableWave** | 안정 파도 |
| 20 | **StableTornado** | 안정 토네이도 |

### 벤치마크/성능 테마
| # | 이름 | 의미 |
|---|------|------|
| 21 | **Benchmark** | 벤치마크 |
| 22 | **LoadRunner** | 부하 실행기 |
| 23 | **StressForge** | 스트레스 제조소 |
| 24 | **TxHammer** | 거래 해머 |
| 25 | **BlockBuster** | 블록 파괴자 |
| 26 | **ChainBreaker** | 체인 브레이커 |
| 27 | **TxFlood** | 거래 홍수 |
| 28 | **GasGuzzler** | 가스 소비자 |
| 29 | **NetStress** | 네트워크 스트레스 |
| 30 | **LoadForge** | 부하 제조소 |

### 속도/파워 테마
| # | 이름 | 의미 |
|---|------|------|
| 31 | **Thunderbolt** | 번개 |
| 32 | **Lightning** | 라이트닝 |
| 33 | **Blitz** | 전격 |
| 34 | **Velocity** | 속도 |
| 35 | **Turbo** | 터보 |
| 36 | **Hyperdrive** | 하이퍼드라이브 |
| 37 | **Warp** | 워프 |
| 38 | **Sonic** | 소닉 |
| 39 | **Flash** | 플래시 |
| 40 | **Rapid** | 래피드 |

### 창의적/고유 이름
| # | 이름 | 의미 |
|---|------|------|
| 41 | **Anvil** | 모루, 단단한 테스트 |
| 42 | **Siege** | 포위, 지속적 공격 |
| 43 | **Catalyst** | 촉매, 성능 촉진 |
| 44 | **Ignite** | 점화 |
| 45 | **Avalanche** | 눈사태, 대량 거래 |
| 46 | **Tsunami** | 쓰나미 |
| 47 | **Inferno** | 지옥불 |
| 48 | **Vortex** | 소용돌이 |
| 49 | **Typhoon** | 태풍 |
| 50 | **Maelstrom** | 대소용돌이 |

### 추천 TOP 5

| 순위 | 이름 | 이유 |
|------|------|------|
| 1 | **StableStorm** | StableNet과 직접 연관 + 스트레스 테스트 의미 명확 |
| 2 | **Pulsar** | Supernova와 같은 우주 테마, 짧고 기억하기 쉬움 |
| 3 | **Anvil** | 단순하고 강력한 이미지, 테스트 도구로 적합 |
| 4 | **Avalanche** | 대량 거래 홍수 의미, 발음 좋음 |
| 5 | **TxHammer** | 기능을 직접 설명, 개발자 친화적 |

---

## 2. 프로젝트 개요

### 2.1 목적

StableNet L1 블록체인의 성능을 측정하기 위한 스트레스 테스트 CLI 도구 개발

### 2.2 참조 프로젝트

- **Supernova** (`/Users/wm-it-22-00661/Work/github/tools/supernova`)
  - Gno TM2 스트레스 테스트 도구
  - 파이프라인 아키텍처 참조
  - 배치 처리 및 결과 수집 로직 참조

### 2.3 대상 체인

- **go-stablenet** (`/Users/wm-it-22-00661/Work/github/stable-net/test/go-stablenet`)
  - Geth 기반 EVM 호환 L1
  - WBFT (PoA) 합의
  - 스테이블코인 네이티브

---

## 3. 기술 사양

### 3.1 StableNet 핵심 특성

| 항목 | 값 |
|------|-----|
| 기반 | go-ethereum (Geth) 포크 |
| 합의 | WBFT (Anzeon), PoA |
| 블록 시간 | 1초 |
| 에폭 길이 | 10 블록 |
| 최대 블록 가스 | 105,000,000 |
| 기본 거래당 가스 | 21,000 |
| 블록당 예상 TPS | ~5,000 (단순 전송 기준) |

### 3.2 지원 거래 타입

| 타입 | 코드 | 설명 |
|------|------|------|
| Legacy | 0x00 | 기존 이더리움 거래 |
| Access List | 0x01 | EIP-2930 |
| Dynamic Fee | 0x02 | EIP-1559 |
| Blob | 0x03 | EIP-4844 |
| **Fee Delegation** | **0x16** | **StableNet 고유**, 가스비 대납 |

### 3.3 RPC 인터페이스

```
HTTP-RPC: http://localhost:8545
WebSocket: ws://localhost:8546
```

**주요 API:**
- `eth_sendRawTransaction` - 거래 전송
- `eth_getTransactionCount` - Nonce 조회
- `eth_getBalance` - 잔액 조회
- `eth_estimateGas` - 가스 추정
- `eth_gasPrice` - 가스 가격 조회
- `eth_blockNumber` - 블록 높이 조회
- `eth_getBlockByNumber` - 블록 조회
- `eth_getTransactionReceipt` - 거래 영수증 조회

---

## 4. 기능 요구사항

### 4.1 필수 기능 (MVP)

#### Phase 1: 기본 인프라
- [ ] CLI 플래그 파싱 및 설정 관리
- [ ] HTTP/WebSocket RPC 클라이언트
- [ ] 계정 생성 (HD Wallet, BIP39)
- [ ] 거래 서명 (ECDSA secp256k1)

#### Phase 2: 거래 생성
- [ ] Legacy 거래 (Type 0x00)
- [ ] EIP-1559 거래 (Type 0x02)
- [ ] Fee Delegation 거래 (Type 0x16) - **StableNet 특화**
- [ ] 스마트 컨트랙트 배포 거래
- [ ] 스마트 컨트랙트 호출 거래

#### Phase 3: 테스트 실행
- [ ] 자금 배분 (마스터 → 부계정)
- [ ] 배치 거래 전송
- [ ] 논스(Nonce) 관리
- [ ] 가스 가격 동적 조회

#### Phase 4: 결과 수집
- [ ] 거래 확인 및 영수증 수집
- [ ] TPS 계산
- [ ] 블록별 통계 (가스 사용률, 거래 수)
- [ ] JSON 결과 저장

### 4.2 확장 기능

#### Phase 5: 고급 테스트 모드
- [ ] ERC20 토큰 전송 테스트
- [ ] NFT 민팅 테스트
- [ ] DeFi 시뮬레이션 (스왑, 유동성 추가)
- [ ] 혼합 거래 타입 테스트

#### Phase 6: 모니터링 및 분석
- [ ] 실시간 TPS 모니터링
- [ ] 메모리풀(Mempool) 상태 추적
- [ ] 거래 지연 시간 분석
- [ ] 실패 거래 분류 및 분석

---

## 5. 아키텍처 설계

### 5.1 프로젝트 구조

```
{project-name}/
├── cmd/
│   └── root.go                 # CLI 진입점
│
├── internal/
│   ├── config/
│   │   └── config.go           # 설정 관리
│   │
│   ├── client/
│   │   ├── client.go           # RPC 클라이언트 인터페이스
│   │   ├── http.go             # HTTP 클라이언트
│   │   └── ws.go               # WebSocket 클라이언트
│   │
│   ├── wallet/
│   │   ├── wallet.go           # HD Wallet 관리
│   │   └── signer.go           # 거래 서명
│   │
│   ├── txbuilder/
│   │   ├── builder.go          # 거래 빌더 인터페이스
│   │   ├── legacy.go           # Legacy 거래
│   │   ├── dynamic_fee.go      # EIP-1559 거래
│   │   ├── fee_delegation.go   # Fee Delegation 거래 (0x16)
│   │   └── contract.go         # 컨트랙트 거래
│   │
│   ├── distributor/
│   │   └── distributor.go      # 자금 배분
│   │
│   ├── batcher/
│   │   └── batcher.go          # 배치 처리
│   │
│   ├── collector/
│   │   ├── collector.go        # 결과 수집
│   │   └── stats.go            # 통계 계산
│   │
│   ├── pipeline/
│   │   └── pipeline.go         # 메인 오케스트레이션
│   │
│   └── output/
│       ├── terminal.go         # 터미널 출력
│       └── json.go             # JSON 저장
│
├── pkg/
│   └── types/
│       └── types.go            # 공통 타입
│
├── testdata/
│   └── contracts/              # 테스트용 컨트랙트
│
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

### 5.2 실행 흐름

```
┌─────────────────────────────────────────────────────────────────────┐
│                          CLI 실행                                    │
│  {tool} --url http://... --private-key 0x... --txs 1000             │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│  1. 설정 검증 및 RPC 클라이언트 초기화                               │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│  2. 계정 초기화                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ Private Key 또는 Mnemonic → HD Wallet                        │    │
│  │     └─▶ accounts[0] = Master Account (자금 보유)              │    │
│  │     └─▶ accounts[1..N] = Sub Accounts (거래 발송용)           │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│  3. 자금 배분 (Distributor)                                          │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ Master Account ──▶ 각 Sub Account로 필요한 가스비 전송        │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│  4. 거래 생성 (TxBuilder)                                            │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ 모드별 거래 생성:                                             │    │
│  │ • TRANSFER: 단순 전송 (Type 0x02)                             │    │
│  │ • FEE_DELEGATION: 가스비 대납 (Type 0x16)                     │    │
│  │ • CONTRACT_DEPLOY: 컨트랙트 배포                              │    │
│  │ • CONTRACT_CALL: 컨트랙트 호출                                │    │
│  │ • ERC20_TRANSFER: 토큰 전송                                   │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│  5. 배치 전송 (Batcher)                                              │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ 1. 거래 RLP 인코딩                                            │    │
│  │ 2. 배치 분할                                                  │    │
│  │ 3. eth_sendRawTransaction 배치 호출                           │    │
│  │ 4. 거래 해시 수집                                             │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│  6. 결과 수집 (Collector)                                            │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ • eth_getTransactionReceipt로 영수증 수집                     │    │
│  │ • 블록별 가스 사용량, 거래 수 집계                            │    │
│  │ • 성공/실패 거래 분류                                         │    │
│  │ • TPS 계산                                                    │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│  7. 결과 출력                                                        │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ • 터미널 출력 (TPS, 통계, 블록 정보)                          │    │
│  │ • JSON 파일 저장 (선택)                                       │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

### 5.3 테스트 모드

| 모드 | 설명 | 거래 타입 |
|------|------|----------|
| `TRANSFER` | 단순 네이티브 코인 전송 | Type 0x02 (EIP-1559) |
| `FEE_DELEGATION` | 가스비 대납 전송 | Type 0x16 (StableNet 특화) |
| `CONTRACT_DEPLOY` | 스마트 컨트랙트 배포 | Type 0x02 |
| `CONTRACT_CALL` | 컨트랙트 메서드 호출 | Type 0x02 |
| `ERC20_TRANSFER` | ERC20 토큰 전송 | Type 0x02 |
| `MIXED` | 혼합 거래 타입 | 다양 |

---

## 6. CLI 인터페이스

### 6.1 플래그 정의

```bash
USAGE
  {tool} [flags]

FLAGS
  # 필수
  --url string              RPC 엔드포인트 URL (예: http://localhost:8545)
  --private-key string      마스터 계정 개인키 (0x 프리픽스)

  # 또는 HD Wallet 사용
  --mnemonic string         BIP39 니모닉

  # 테스트 설정
  --mode string             테스트 모드 (기본: TRANSFER)
                            [TRANSFER, FEE_DELEGATION, CONTRACT_DEPLOY,
                             CONTRACT_CALL, ERC20_TRANSFER, MIXED]
  --sub-accounts uint       부계정 수 (기본: 10)
  --transactions uint       총 거래 수 (기본: 100)
  --batch uint              배치 크기 (기본: 100)

  # 체인 설정
  --chain-id uint           체인 ID (기본: 자동 감지)
  --gas-limit uint          거래당 가스 한도 (기본: 21000)
  --gas-price string        가스 가격 (기본: 자동)

  # Fee Delegation 모드 전용
  --fee-payer-key string    가스비 대납자 개인키

  # 컨트랙트 모드 전용
  --contract string         대상 컨트랙트 주소
  --method string           호출할 메서드 시그니처
  --args string             메서드 인자 (JSON 형식)

  # 출력
  --output string           결과 JSON 파일 경로
  --verbose                 상세 로그 출력

  # 고급
  --timeout duration        타임아웃 (기본: 5m)
  --rate-limit uint         초당 최대 거래 수 (기본: 무제한)
```

### 6.2 실행 예시

```bash
# 기본 전송 테스트
./stablestorm \
  --url http://localhost:8545 \
  --private-key 0x... \
  --sub-accounts 10 \
  --transactions 1000

# Fee Delegation 테스트
./stablestorm \
  --url http://localhost:8545 \
  --private-key 0x... \
  --mode FEE_DELEGATION \
  --fee-payer-key 0x... \
  --transactions 500

# 컨트랙트 호출 테스트
./stablestorm \
  --url http://localhost:8545 \
  --private-key 0x... \
  --mode CONTRACT_CALL \
  --contract 0x1234... \
  --method "transfer(address,uint256)" \
  --args '["0xabc...", "1000000000000000000"]' \
  --transactions 100
```

---

## 7. 핵심 구현 세부사항

### 7.1 Fee Delegation 거래 (Type 0x16)

StableNet 고유 기능으로, 발신자와 가스비 지불자가 분리됩니다.

```go
type FeeDelegationTx struct {
    ChainID    *big.Int
    Nonce      uint64
    GasTipCap  *big.Int  // Priority fee
    GasFeeCap  *big.Int  // Max fee
    Gas        uint64
    To         *common.Address
    Value      *big.Int
    Data       []byte
    AccessList AccessList

    // 발신자 서명
    V *big.Int
    R *big.Int
    S *big.Int

    // 가스비 지불자 서명
    FeePayer *common.Address
    FV       *big.Int
    FR       *big.Int
    FS       *big.Int
}
```

**서명 순서:**
1. 발신자가 거래 서명 (V, R, S)
2. 가스비 지불자가 추가 서명 (FV, FR, FS)

### 7.2 Nonce 관리

병렬 거래 발송 시 Nonce 충돌 방지:

```go
type NonceManager struct {
    mu      sync.Mutex
    nonces  map[common.Address]uint64
    client  Client
}

func (nm *NonceManager) GetAndIncrement(addr common.Address) (uint64, error) {
    nm.mu.Lock()
    defer nm.mu.Unlock()

    if _, exists := nm.nonces[addr]; !exists {
        nonce, err := nm.client.PendingNonceAt(addr)
        if err != nil {
            return 0, err
        }
        nm.nonces[addr] = nonce
    }

    nonce := nm.nonces[addr]
    nm.nonces[addr]++
    return nonce, nil
}
```

### 7.3 배치 JSON-RPC 요청

```go
type BatchRequest struct {
    requests []rpc.BatchElem
}

func (b *BatchRequest) SendRawTransaction(signedTx []byte) {
    b.requests = append(b.requests, rpc.BatchElem{
        Method: "eth_sendRawTransaction",
        Args:   []interface{}{hexutil.Encode(signedTx)},
        Result: new(common.Hash),
    })
}

func (b *BatchRequest) Execute(client *rpc.Client) error {
    return client.BatchCall(b.requests)
}
```

---

## 8. 개발 일정 (예상)

### Phase 1: 기본 인프라 (1주)
- CLI 프레임워크
- RPC 클라이언트
- 설정 관리

### Phase 2: 지갑 및 거래 (1주)
- HD Wallet 구현
- Legacy/EIP-1559 거래 빌더
- Fee Delegation 거래 빌더

### Phase 3: 테스트 실행 (1주)
- 자금 배분
- 배치 처리
- Nonce 관리

### Phase 4: 결과 수집 (0.5주)
- 거래 추적
- 통계 계산
- 출력 포맷

### Phase 5: 테스트 및 문서화 (0.5주)
- 단위 테스트
- 통합 테스트
- 문서화

**총 예상 기간: 4주**

---

## 9. 의존성

### 9.1 필수 패키지

```go
require (
    github.com/ethereum/go-ethereum v1.13.x    // Geth 라이브러리
    github.com/spf13/cobra v1.8.x              // CLI 프레임워크
    github.com/schollz/progressbar/v3 v3.14.x  // 진행률 표시
    github.com/stretchr/testify v1.9.x         // 테스트
)
```

### 9.2 참조 코드

- **Supernova**: 파이프라인, 배치, 수집 패턴
- **go-stablenet**: Fee Delegation 거래 구조, RLP 인코딩

---

## 10. 성공 기준

### 10.1 기능적 기준
- [ ] 5가지 테스트 모드 모두 동작
- [ ] 1,000+ TPS 처리 가능
- [ ] Fee Delegation 거래 정상 동작
- [ ] 정확한 TPS 및 통계 계산

### 10.2 비기능적 기준
- [ ] 메모리 누수 없음
- [ ] 안정적인 장시간 테스트 (1시간+)
- [ ] 명확한 오류 메시지
- [ ] 90%+ 테스트 커버리지

---

## 11. 참고 자료

- [Supernova 소스코드](/Users/wm-it-22-00661/Work/github/tools/supernova)
- [go-stablenet 소스코드](/Users/wm-it-22-00661/Work/github/stable-net/test/go-stablenet)
- [go-ethereum 문서](https://geth.ethereum.org/docs)
- [EIP-1559](https://eips.ethereum.org/EIPS/eip-1559)
