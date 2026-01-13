# 블록체인 스트레스 테스트 도구 비교 분석

이 문서는 블록체인 네트워크 성능 테스트를 위한 4개의 내부 프로젝트를 비교 분석합니다.

## 목차

1. [프로젝트 개요](#1-프로젝트-개요)
2. [기능 비교](#2-기능-비교)
3. [아키텍처 비교](#3-아키텍처-비교)
4. [핵심 차이점](#4-핵심-차이점)
5. [공통 설계 패턴](#5-공통-설계-패턴)
6. [txhammer 포지셔닝](#6-txhammer-포지셔닝)
7. [병목 지점 발견 전략](#7-병목-지점-발견-전략)
8. [향후 개선 방향](#8-향후-개선-방향)

---

## 1. 프로젝트 개요

### 1.1 요약 테이블

| 항목 | supernova | tpser | txhammer | pandoras-box |
|------|-----------|-------|----------|--------------|
| **언어** | Go | Go | Go | TypeScript |
| **대상 체인** | Gno (Tendermint2) | Ethereum/EVM | StableNet (EVM) | Ethereum/EVM |
| **주 목적** | TPS 벤치마크 | TPS 측정/장시간 송금 | 스트레스/성능 테스트 | 스트레스 테스트 |
| **코드량** | ~5,000 LOC | ~2,000 LOC | ~9,000 LOC | ~2,000 LOC |
| **라이선스** | - | - | MIT | Apache 2.0 |

### 1.2 개별 프로젝트 설명

#### supernova
Gno Tendermint 2 블록체인 네트워크를 위한 CLI 스트레스 테스트 도구입니다.
- Realm/Package 배포 및 호출 테스트
- gnolang/benchmarks 리포지토리와 연동
- Amino 인코딩 기반 트랜잭션 처리

#### tpser
Ethereum 호환 네트워크용 성능 테스트 및 분석 도구입니다.
- 세 가지 동작 모드: BlocksFetcher, LongSender, TxInfo
- Prometheus 메트릭 노출
- 장시간(수시간~수일) 지속 테스트 지원

#### txhammer
StableNet L1 블록체인을 위한 종합 스트레스 테스트 도구입니다.
- 5가지 트랜잭션 타입 지원
- Fee Delegation (Type 0x16) 지원
- 상세 레이턴시 분포 분석 (P50, P95, P99)

#### pandoras-box
TypeScript 기반 Ethereum 스트레스 테스트 도구입니다.
- EOA, ERC20, ERC721 테스트 지원
- npm 패키지로 배포
- 토큰 분배 자동화

---

## 2. 기능 비교

### 2.1 트랜잭션 타입 지원

| 기능 | supernova | tpser | txhammer | pandoras-box |
|------|:---------:|:-----:|:--------:|:------------:|
| EOA Transfer | - | O | O | O |
| Realm/Package Deploy | O | - | - | - |
| Contract Deploy | - | - | O | - |
| Contract Call | O | - | O | - |
| ERC20 Transfer | - | 계획 | O | O |
| ERC721 Mint | - | - | - | O |
| **Fee Delegation** | - | - | O | - |
| EIP-1559 | - | O | O | - |

### 2.2 실행 모드

| 기능 | supernova | tpser | txhammer | pandoras-box |
|------|:---------:|:-----:|:--------:|:------------:|
| Batch 송신 | O | - | O | O |
| Streaming 모드 | - | O | O | - |
| Fire-and-forget | - | O | O | - |
| Dry Run | - | - | O | - |
| 블록 분석 전용 | - | O | - | - |
| 장시간 테스트 | - | O | - | - |

### 2.3 출력 및 리포팅

| 기능 | supernova | tpser | txhammer | pandoras-box |
|------|:---------:|:-----:|:--------:|:------------:|
| JSON 리포트 | O | - | O | O |
| CSV 리포트 | - | - | O | - |
| 레이턴시 분포 | - | - | O | - |
| Prometheus 메트릭 | - | O | - | - |
| 실시간 진행률 | O | O | O | O |
| 블록별 통계 | O | O | O | O |

### 2.4 계정 관리

| 기능 | supernova | tpser | txhammer | pandoras-box |
|------|:---------:|:-----:|:--------:|:------------:|
| HD Wallet (BIP39) | O | O | O | O |
| 다중 서브 계정 | O | O | O | O |
| 자동 자금 분배 | O | - | O | O |
| 토큰 분배 | - | - | - | O |
| Private Key 직접 입력 | - | O | O | - |

---

## 3. 아키텍처 비교

### 3.1 supernova 아키텍처

```
supernova/
├── cmd/                    # CLI 진입점, 플래그 파싱
└── internal/
    ├── client/             # JSON-RPC HTTP/WS 클라이언트 (Amino 인코딩)
    ├── runtime/            # 트랜잭션 생성 전략
    │   ├── realm_deployment.go
    │   ├── package_deployment.go
    │   └── realm_call.go
    ├── batcher/            # 순차적 배치 송신
    ├── collector/          # 블록/트랜잭션 결과 수집
    ├── distributor/        # 서브 계정 자금 분배
    ├── signer/             # HD 키 파생 및 서명
    └── common/             # 공유 타입, 수수료 계산
```

**특징:**
- Gno 체인 전용 메시지 타입 (`vm.MsgAddPackage`, `vm.MsgCall`)
- Amino 바이너리 인코딩으로 트랜잭션 마샬링
- 순차적 배치 전송 (논스 순서 보장)

### 3.2 tpser 아키텍처

```
tpser/
├── main.go                 # 진입점
├── app/
│   └── app.go              # uber/fx 기반 DI 부트스트랩
└── pkg/
    ├── conf/               # CLI 플래그 및 설정 검증
    ├── eth/
    │   ├── eth.go          # 모드 팩토리
    │   ├── types/          # BlockInfo 등 타입 정의
    │   ├── tools/
    │   │   ├── txsigner/   # 트랜잭션 서명
    │   │   ├── txsender/   # 트랜잭션 전송
    │   │   └── txreceipts/ # 영수증 확인
    │   └── modes/
    │       ├── getblocks/  # BlocksFetcher 구현
    │       ├── longsender/ # LongSender 구현
    │       └── txinfo/     # TxInfo 구현
    ├── logger/             # logrus/zap 로깅
    └── prom/               # Prometheus 메트릭
```

**특징:**
- `go.uber.org/fx` 의존성 주입 프레임워크 사용
- 팩토리 패턴으로 모드 선택
- Prometheus 메트릭 HTTP 엔드포인트 노출

### 3.3 txhammer 아키텍처

```
txhammer/
├── cmd/
│   └── main.go             # Cobra CLI 진입점
├── internal/
│   ├── config/             # 설정 구조체 및 검증
│   ├── client/             # go-ethereum 클라이언트 래퍼
│   ├── wallet/             # HD 지갑 관리
│   ├── txbuilder/          # 트랜잭션 빌더 (Factory 패턴)
│   │   ├── builder.go      # 기본 빌더 인터페이스
│   │   ├── factory.go      # 빌더 팩토리
│   │   ├── transfer.go     # EIP-1559 전송
│   │   ├── fee_delegation.go # Type 0x16 빌더
│   │   ├── contract.go     # 컨트랙트 배포/호출
│   │   └── erc20.go        # ERC20 전송
│   ├── distributor/        # 자금 분배 로직
│   ├── batcher/            # 배치 + 스트리밍 전송
│   ├── collector/          # 영수증 수집 및 메트릭 계산
│   └── pipeline/           # 6단계 파이프라인 오케스트레이션
└── pkg/
    └── types/              # 공개 타입 정의
```

**특징:**
- 6단계 파이프라인: INIT → DISTRIBUTE → BUILD → SEND → COLLECT → REPORT
- Factory 패턴으로 5가지 트랜잭션 빌더 관리
- 최대 100개 동시 배치 요청 지원
- 상세 레이턴시 백분위 수 계산

### 3.4 pandoras-box 아키텍처

```
pandoras-box/
└── src/
    ├── index.ts            # CLI 진입점 및 오케스트레이터
    ├── runtime/
    │   ├── runtimes.ts     # Runtime 인터페이스
    │   ├── eoa.ts          # EOA 전송 런타임
    │   ├── erc20.ts        # ERC20 토큰 런타임
    │   ├── erc721.ts       # ERC721 NFT 런타임
    │   ├── engine.ts       # 트랜잭션 구성 파이프라인
    │   ├── batcher.ts      # JSON-RPC 배치 전송
    │   └── signer.ts       # 지갑 및 서명
    ├── distributor/
    │   ├── distributor.ts      # 네이티브 통화 분배
    │   └── tokenDistributor.ts # 토큰 분배
    ├── stats/
    │   └── collector.ts    # 성능 메트릭 수집
    ├── logger/             # 컬러 콘솔 로깅
    ├── outputter/          # JSON 파일 내보내기
    └── contracts/          # ERC20/ERC721 ABI
```

**특징:**
- TypeScript로 작성, npm 패키지로 배포
- ethers.js v5 기반 블록체인 상호작용
- 힙(Heap) 자료구조로 효율적인 자금 분배 우선순위 관리

---

## 4. 핵심 차이점

### 4.1 대상 블록체인

| 프로젝트 | 대상 체인 | 특이사항 |
|----------|-----------|----------|
| supernova | Gno (Tendermint2) | Cosmos SDK 기반, 비 EVM |
| tpser | Ethereum/EVM | 범용 EVM 체인 |
| txhammer | StableNet (EVM) | Fee Delegation 지원 |
| pandoras-box | Ethereum/EVM | 범용 EVM 체인 |

### 4.2 txhammer 고유 기능

| 기능 | 설명 |
|------|------|
| **Fee Delegation (Type 0x16)** | StableNet 전용 트랜잭션 타입. 수수료 대납자가 가스비 부담 |
| **6단계 파이프라인** | 체계적인 실행 흐름으로 각 단계별 에러 처리 및 모니터링 |
| **레이턴시 백분위 수** | P50, P95, P99 계산으로 정확한 응답 시간 분포 파악 |
| **다중 리포트 포맷** | JSON + 3종 CSV (summary, transactions, blocks) |
| **동시성 제어** | 최대 100개 동시 배치 요청, 세마포어 기반 제어 |
| **Graceful Shutdown** | SIGINT/SIGTERM 시그널 처리로 안전한 종료 |
| **Dry Run 모드** | 실제 전송 없이 트랜잭션 빌드만 수행 |

### 4.3 tpser 고유 기능

| 기능 | 설명 |
|------|------|
| **LongSender 모드** | 수시간~수일 장시간 테스트 지원 |
| **BlocksFetcher 모드** | 블록 분석 전용 모드 (트랜잭션 미전송) |
| **TxInfo 모드** | 특정 트랜잭션 상세 정보 조회 |
| **Prometheus 메트릭** | `tpser` 네임스페이스로 메트릭 노출 |
| **uber/fx DI** | 의존성 주입으로 테스트 용이성 향상 |

### 4.4 supernova 고유 기능

| 기능 | 설명 |
|------|------|
| **Gno 전용 메시지** | `vm.MsgAddPackage`, `vm.MsgCall` 지원 |
| **Amino 인코딩** | Tendermint 표준 인코딩 형식 |
| **Realm/Package 배포** | Gno 스마트 컨트랙트 배포 테스트 |
| **벤치마크 연동** | gnolang/benchmarks 리포지토리 결과 게시 |

### 4.5 pandoras-box 고유 기능

| 기능 | 설명 |
|------|------|
| **ERC721 민팅** | NFT 민팅 스트레스 테스트 |
| **토큰 분배** | ERC20 토큰 자동 분배 |
| **TypeScript** | Node.js 생태계 활용 |
| **npm 패키지** | `pandoras-box`로 npm에서 설치 가능 |

---

## 5. 공통 설계 패턴

### 5.1 HD Wallet (BIP39/BIP44)

모든 프로젝트가 동일한 키 파생 방식을 사용합니다:

```
Master Seed (Mnemonic)
    └── m/44'/60'/0'/0/0  → 마스터 계정 (자금 분배용)
    └── m/44'/60'/0'/0/1  → 서브 계정 1
    └── m/44'/60'/0'/0/2  → 서브 계정 2
    └── ...
    └── m/44'/60'/0'/0/N  → 서브 계정 N
```

### 5.2 자금 분배 (Distributor)

```
1. 마스터 계정 잔액 확인
2. 각 서브 계정의 필요 자금 계산
   └── 필요 자금 = (가스 가격 × 가스 한도 + 전송값) × 트랜잭션 수 × 버퍼(1.2)
3. 잔액 부족 계정 식별
4. 마스터 → 서브 계정 자금 전송
5. 전송 확인 대기
```

### 5.3 배치 처리 (Batcher)

```
1. 트랜잭션 목록을 배치 크기로 분할
2. JSON-RPC 배치 요청 구성
   └── [{"jsonrpc":"2.0","method":"eth_sendRawTransaction","params":[tx1],"id":1}, ...]
3. HTTP POST로 배치 전송
4. 응답에서 트랜잭션 해시 추출
5. 에러 처리 및 재시도
```

### 5.4 결과 수집 (Collector)

```
1. 전송된 트랜잭션 해시 목록 유지
2. 주기적으로 영수증 폴링 (500ms~5s 간격)
3. 블록 데이터 수집
   └── 블록 번호, 타임스탬프, 가스 사용량, 트랜잭션 수
4. TPS 계산
   └── TPS = 총 트랜잭션 / (마지막 블록 시간 - 첫 블록 시간)
5. 메트릭 집계 및 리포트 생성
```

### 5.5 논스 관리

```
각 서브 계정별 로컬 논스 카운터 유지
    └── 트랜잭션 생성 시 논스 증가
    └── 네트워크 재조회 최소화
    └── 동시성 제어 (Mutex/Atomic)
```

---

## 6. txhammer 포지셔닝

### 6.1 go-stablenet 테스트 적합성

txhammer는 go-stablenet 프로젝트를 위한 스트레스 테스트 도구로서 다음과 같은 장점을 가집니다:

| 요구사항 | txhammer 지원 |
|----------|---------------|
| StableNet Fee Delegation | O (Type 0x16) |
| EIP-1559 트랜잭션 | O |
| 고성능 배치 전송 | O (100 동시 배치) |
| 상세 레이턴시 분석 | O (P50, P95, P99) |
| 컨트랙트 테스트 | O (Deploy, Call) |
| ERC20 토큰 테스트 | O |
| 다양한 리포트 포맷 | O (JSON, CSV) |

### 6.2 타 프로젝트 대비 강점

```
                    기능 풍부도
                         ^
                         |
              txhammer   |  * (가장 포괄적)
                         |
                         |
         supernova  ●    |    ● tpser (모니터링 특화)
                         |
                         |
      pandoras-box  ●    |
                         |
                         +-------------------------> 범용성
                    Gno 전용              EVM 범용
```

### 6.3 현재 한계점

| 한계점 | 설명 | 참고 프로젝트 |
|--------|------|---------------|
| Prometheus 미지원 | 실시간 메트릭 노출 없음 | tpser |
| 장시간 테스트 미지원 | 지속 시간 기반 테스트 없음 | tpser (LongSender) |
| 블록 분석 전용 모드 없음 | 기존 블록 분석만 필요할 때 | tpser (BlocksFetcher) |

---

## 7. 병목 지점 발견 전략

### 7.1 테스트 시나리오

#### 시나리오 1: 기본 처리량 측정

```bash
./txhammer \
  --url http://localhost:8545 \
  --private-key $MASTER_KEY \
  --mode TRANSFER \
  --sub-accounts 50 \
  --transactions 10000 \
  --batch 100
```

**측정 항목:**
- TPS (전송/확인)
- 블록별 가스 사용률
- 평균 레이턴시

**병목 지점 판단:**
- TPS < 예상치: 네트워크/합의 병목
- 가스 사용률 < 80%: 블록 생성 빈도 또는 가스 한도 문제
- 레이턴시 P99 >> P50: 특정 조건에서 지연 발생

#### 시나리오 2: 컨트랙트 부하 테스트

```bash
./txhammer \
  --url http://localhost:8545 \
  --private-key $MASTER_KEY \
  --mode CONTRACT_CALL \
  --contract $CONTRACT_ADDRESS \
  --sub-accounts 20 \
  --transactions 5000
```

**측정 항목:**
- 컨트랙트별 가스 소비량
- EVM 실행 시간 (레이턴시)
- 상태 변경 처리량

**병목 지점 판단:**
- 가스 소비량 급증: 컨트랙트 최적화 필요
- 레이턴시 증가: 상태 읽기/쓰기 병목

#### 시나리오 3: Fee Delegation 성능

```bash
./txhammer \
  --url http://localhost:8545 \
  --private-key $MASTER_KEY \
  --fee-payer-key $FEE_PAYER_KEY \
  --mode FEE_DELEGATION \
  --sub-accounts 30 \
  --transactions 3000
```

**측정 항목:**
- Fee Delegation 처리 오버헤드
- 성공률
- P95/P99 레이턴시

**병목 지점 판단:**
- 성공률 < 100%: Fee Payer 잔액 또는 논스 충돌
- 오버헤드 > 10%: Fee Delegation 구현 최적화 필요

#### 시나리오 4: 피크 부하 테스트

```bash
./txhammer \
  --url http://localhost:8545 \
  --private-key $MASTER_KEY \
  --mode TRANSFER \
  --streaming \
  --streaming-rate 5000 \
  --sub-accounts 100 \
  --transactions 100000 \
  --skip-collection
```

**측정 항목:**
- 최대 전송 TPS
- 노드 응답 시간
- 에러율

**병목 지점 판단:**
- 에러율 증가: mempool 포화 또는 RPC 병목
- 응답 시간 급증: 노드 처리 능력 한계

### 7.2 분석 워크플로우

```
1. 기준선 측정 (Baseline)
   └── 소규모 테스트로 정상 동작 확인

2. 점진적 부하 증가
   └── 서브 계정 수 / 트랜잭션 수 / 배치 크기 조절

3. 병목 지점 식별
   └── TPS 감소 시점
   └── 레이턴시 급증 시점
   └── 에러율 증가 시점

4. 원인 분석
   └── 리포트 CSV 분석
   └── 노드 로그 확인
   └── 시스템 리소스 모니터링

5. 최적화 및 재테스트
   └── 설정 조정 후 재테스트
   └── 개선 효과 측정
```

---

## 8. 향후 개선 방향

### 8.1 우선순위별 개선 사항

#### 높은 우선순위

| 기능 | 설명 | 참고 |
|------|------|------|
| Prometheus 메트릭 | 실시간 메트릭 노출 (`/metrics` 엔드포인트) | tpser |
| 장시간 테스트 모드 | 지속 시간 기반 테스트 (분/시간/일 단위) | tpser LongSender |
| 실시간 TPS 모니터링 | 테스트 중 현재 TPS 표시 | tpser |

#### 중간 우선순위

| 기능 | 설명 | 참고 |
|------|------|------|
| 블록 분석 전용 모드 | 기존 블록 데이터만 분석 | tpser BlocksFetcher |
| 트랜잭션 정보 조회 | 특정 트랜잭션 상세 정보 | tpser TxInfo |
| 동적 부하 조절 | 테스트 중 TPS 조절 | - |

#### 낮은 우선순위

| 기능 | 설명 | 참고 |
|------|------|------|
| ERC721 민팅 테스트 | NFT 민팅 스트레스 테스트 | pandoras-box |
| 토큰 자동 분배 | ERC20 토큰 분배 자동화 | pandoras-box |
| 웹 대시보드 | 테스트 결과 시각화 | - |

### 8.2 아키텍처 개선

```
현재 구조:
  pipeline/ → 6단계 순차 실행

개선 방향:
  1. 메트릭 서버 추가 (Prometheus)
  2. 장시간 모드 추가 (duration 기반)
  3. 분석 전용 모드 추가 (블록 조회만)
  4. 실시간 모니터링 goroutine 추가
```

### 8.3 예상 구현 범위

```go
// 새로운 모드 추가
const (
    ModeLongSender    Mode = "LONG_SENDER"    // 장시간 테스트
    ModeBlockAnalyzer Mode = "BLOCK_ANALYZER" // 블록 분석 전용
)

// Prometheus 메트릭
type Metrics struct {
    TxSent        prometheus.Counter
    TxConfirmed   prometheus.Counter
    TxLatency     prometheus.Histogram
    CurrentTPS    prometheus.Gauge
    ErrorCount    prometheus.Counter
}
```

---

## 참고 자료

- [txhammer ARCHITECTURE.md](./ARCHITECTURE.md)
- [txhammer FEE_DELEGATION.md](./FEE_DELEGATION.md)
- [txhammer SPEC.md](./SPEC.md)
- [go-ethereum Documentation](https://geth.ethereum.org/docs)
- [StableNet Documentation](https://docs.stablenet.io)
