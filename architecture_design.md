# config-man
# Enterprise Multi-Project Configuration Management System
# 企業級多專案配置管理系統 — Application & System Architecture Design

---

## 1. 專案概述

### 1.1 背景

在擁有約 300 個專案的大型部門中，Config 管理面臨以下痛點：

- 多種格式共存（`.properties`, `.json`, `.yaml`）
- 跨專案共用設定難以同步（功能開關、服務地址、DB 憑證等）
- 部署環境多元（VM、K8s）
- 手動盤點修改耗時且易出錯
- 缺乏版本控制與自動化驗證

### 1.2 目標

建立集中式 Config 管理系統，實現：
- 統一管理介面
- 版本控制與審計追蹤
- 自動化部署與驗證
- 安全的敏感資料管理

### 1.3 第一期定位

`config-man` 第一期以 **最小可行產品（MVP）** 為目標，先解決「集中管理、專案註冊、模板建立、版本追蹤、部署前驗證」這幾個最痛的問題。

第一期不追求完整企業級自動化部署，而是先提供一個可用的 Web 管理平台，讓工程師可以：

- 註冊專案並選擇 Config 模板
- 在網站中維護不同環境的 Config
- 匯入與匯出 `.properties`, `.json`, `.yaml`
- 在部署前做基本驗證與環境差異比對
- 留下修改歷史、修改者與變更原因

> 第一期的核心價值是「先把分散在各專案的設定集中看見、集中修改、集中驗證」，自動同步到 VM/K8s 可放在第二期後逐步強化。

---

## 2. Functional Requirements

| ID | 功能 | 說明 |
|----|------|------|
| FR-01 | 使用者認證與授權 | 支援 LDAP/SSO 登入，RBAC 權限控制 |
| FR-02 | 集中式 Config 管理 | 統一介面管理所有專案的 Config |
| FR-03 | 多格式支援 | 支援 `.properties`, `.json`, `.yaml` 格式的讀寫與轉換 |
| FR-04 | 多專案管理 | 支援 300+ 專案的 Config 管理 |
| FR-05 | 多環境支援 | 支援 dev / testing / staging / prod 環境切換 |
| FR-06 | 版本控制 | 記錄每次 Config 變更的歷程、變更者與原因 |
| FR-07 | 審核流程 | Config 變更需經過審核者 Approve 後才生效 |
| FR-08 | 敏感資料管理 | 加密儲存密碼、API Key 等敏感設定 |
| FR-09 | 搜尋與篩選 | 支援依專案、環境、Key、Value 快速搜尋 |
| FR-10 | Config 繼承與覆蓋 | 支援 Global → Project → Environment 層級繼承 |
| FR-11 | 模板管理 | 可建立可重複使用的 Config 模板 |
| FR-12 | 環境差異比對 | 比對不同環境間的 Config 差異 |
| FR-13 | 跨專案共享設定 | 管理共用的 Config（如共用 DB、MQ 位址） |
| FR-14 | Config 匯入/匯出 | 批量匯入既有 Config、匯出為檔案 |
| FR-15 | 通知機制 | Config 變更時通知相關人員（Email / Webhook） |

### 2.1 第一期 Functional Scope（MVP）

第一期優先完成可以展示完整流程的功能，範圍控制在「網站管理 + 單體 API + PostgreSQL」。

| ID | 功能 | 第一期實作範圍 |
|----|------|----------------|
| P1-FR-01 | 登入與角色權限 | 先採帳號密碼登入，可預留 SSO/LDAP 介面；支援 System Admin、Project Admin、Developer、Reviewer、Viewer |
| P1-FR-02 | 專案註冊 | 建立專案名稱、Repo URL、Owner、環境清單、Config 格式，註冊後可取得一份標準模板 |
| P1-FR-03 | Config 模板 | 提供 Base Template，可定義必要 Key、預設值、型別、是否敏感、說明文字 |
| P1-FR-04 | Config 管理 | 支援 Global / Project / Environment 三層設定，但第一期先以 Project + Environment 為主 |
| P1-FR-05 | 多格式匯入/匯出 | 支援 `.properties`, `.json`, `.yaml` 匯入與匯出；第一期不保證保留原始註解與排版 |
| P1-FR-06 | 版本歷史 | 每次修改記錄變更前後、修改者、時間、原因，支援查詢歷史 |
| P1-FR-07 | Diff 比對 | 支援同一專案 dev / staging / prod 的 Key-Value 差異比對 |
| P1-FR-08 | 自動驗證 | 檢查必填 Key、型別、重複 Key、敏感欄位是否遮蔽、環境必要值是否存在 |
| P1-FR-09 | 審核流程 | Prod 修改需建立 Change Request，由 Reviewer Approve 後才可套用 |
| P1-FR-10 | 匯出部署包 | 產生可下載的 Config Bundle，讓工程師手動放入 VM 或 CI/CD Pipeline 使用 |

### 2.2 第一期 Out of Scope

| 項目 | 延後原因 |
|------|----------|
| 自動推送到 300 個專案 Repo | 需要 Git Provider、Branch Strategy、衝突處理，適合第二期 |
| 直接部署到 VM / K8s | 涉及 Agent、憑證、網路權限與 Rollback，第一期先提供匯出部署包 |
| HashiCorp Vault HA | 第一階段可用 DB 加密欄位 + App Secret，Vault 放第二期 |
| RabbitMQ 事件驅動通知 | 第一期可用同步流程與站內狀態，通知放第二期 |
| Redis Cache | 第一期資料量小，先由 PostgreSQL + Index 支撐 |
| 完整微服務拆分 | 第一期採 Modular Monolith，降低部署與維運成本 |

### 2.3 第一期角色與任務權限

| Role | 主要任務 | 權限範圍 |
|------|----------|----------|
| System Admin | 系統初始化、管理全域模板、管理使用者 | 全系統 |
| Project Admin | 註冊專案、管理專案成員、設定環境 | 指定專案 |
| Developer | 編輯 dev / staging Config、提交 prod 變更申請、執行 Diff | 指定專案 |
| Reviewer | 審核 prod 變更、查看 Diff 與驗證結果 | 指定專案 |
| Viewer | 查詢 Config、查看版本歷史與 Diff | 指定專案，敏感值遮蔽 |

第一期權限原則：

| 操作 | System Admin | Project Admin | Developer | Reviewer | Viewer |
|------|--------------|---------------|-----------|----------|--------|
| 建立模板 | Y | N | N | N | N |
| 註冊專案 | Y | Y | N | N | N |
| 管理成員 | Y | Y | N | N | N |
| 編輯 dev/staging | Y | Y | Y | N | N |
| 編輯 prod | Y | 提交審核 | 提交審核 | N | N |
| Approve prod 變更 | Y | N | N | Y | N |
| 查看敏感值 | Y | Y | 依授權 | N | N |
| 匯出 Config Bundle | Y | Y | Y | Y | N |
| 查看 Audit Log | Y | Y | N | Y | N |

### 2.4 第一期網站資訊架構

| 頁面 | 使用者目標 | 主要功能 |
|------|------------|----------|
| Login | 進入系統 | 帳密登入、角色載入 |
| Dashboard | 快速掌握專案狀態 | 專案數、待審核變更、最近修改、驗證失敗項目 |
| Project Registry | 註冊與管理專案 | 新增專案、設定 Repo URL、Owner、環境、Config 格式、套用模板 |
| Project Config | 集中修改設定 | 依環境切換、Key-Value 編輯、敏感值遮蔽、修改原因 |
| Template Library | 建立標準模板 | 模板 CRUD、必要 Key、預設值、型別、說明 |
| Diff & Validation | 部署前確認 | dev/staging/prod 差異比對、驗證結果、匯出報告 |
| Change Requests | 管理審核 | prod 變更申請、Approve/Reject、審核意見 |
| Version History | 追蹤變更 | 查看歷史版本、變更者、變更時間、變更原因 |
| Users & Roles | 權限管理 | 使用者、角色、專案成員設定 |

### 2.5 第一期前端 UI/UX 設計方向

前端介面希望參考 Apple 官網的視覺語言：簡潔、留白充足、字體階層清楚、互動細膩、有設計感。但 `config-man` 是內部管理工具，因此第一期會採用 **Apple-inspired Admin UI**，把 Apple 的精緻感轉化為適合長時間操作的後台介面。

設計原則：

| 原則 | 說明 |
|------|------|
| Minimal Layout | 畫面保留足夠留白，避免過多邊框與雜訊 |
| Strong Typography | 使用清楚的標題、數字、狀態文字階層，讓資訊一眼可讀 |
| Soft Depth | 使用淡陰影、細邊框、半透明背景表現層次，不做厚重卡片堆疊 |
| Focused Workflow | 每個頁面聚焦一個主要任務，例如註冊專案、編輯 Config、查看 Diff |
| Calm Color Palette | 主色以黑、白、灰為基礎，搭配少量藍色作為互動與重要狀態 |
| Subtle Motion | 按鈕、Tab、Modal、Diff 展開使用輕微轉場，提升精緻感但不干擾操作 |
| Professional Density | 保持 Apple 風格的乾淨感，同時讓 Config 表格、Diff、審核列表具備足夠資訊密度 |

第一期視覺規格建議：

| UI 元素 | 設計方向 |
|---------|----------|
| Top Navigation | 固定頂部導覽，左側為 `config-man`，右側為搜尋、通知、使用者選單 |
| Sidebar | 使用簡潔 icon + label，包含 Dashboard、Projects、Templates、Change Requests、Users |
| Page Header | 大標題 + 一行輔助說明 + 主要操作按鈕 |
| Cards | 只用於 Dashboard 指標、專案列表項目與審核摘要，圓角控制在 8px 以內 |
| Tables | 清楚的列高、狀態標籤、環境切換、快速搜尋，適合大量 Config 掃描 |
| Config Editor | 左側 Key 列表，右側 Value 編輯區，敏感值預設遮蔽 |
| Diff View | 使用左右並排或上下區塊，比對 dev/staging/prod 差異，新增、修改、刪除以不同狀態色標示 |
| Buttons | 主要按鈕使用深色或藍色，次要按鈕保持低對比 |
| Modal | 用於新增專案、提交審核、確認匯出，內容保持短而明確 |

建議色彩：

| 用途 | 色彩 |
|------|------|
| Background | `#f5f5f7` |
| Surface | `#ffffff` |
| Primary Text | `#1d1d1f` |
| Secondary Text | `#6e6e73` |
| Border | `#d2d2d7` |
| Primary Action | `#0071e3` |
| Success | `#248a3d` |
| Warning | `#b86e00` |
| Danger | `#d70015` |

> 介面參考 Apple 的「乾淨、克制、精準」，但不直接複製 Apple 官網版型或素材；`config-man` 的第一畫面應是可操作的 Dashboard，而不是行銷式 Hero Page。

---

## 3. Non-Functional Requirements

| ID | 需求 | 目標 | 說明 |
|----|------|------|------|
| NFR-01 | High Availability | 99.9% uptime | 系統不可用將影響所有 Release 流程 |
| NFR-02 | Scalability | 支援 300+ 專案 | 水平擴展 API 與儲存層 |
| NFR-03 | Low Latency | API < 200ms (P95) | Config 讀取需快速回應 |
| NFR-04 | Security | 加密傳輸與儲存 | TLS + AES-256 加密敏感資料 |
| NFR-05 | Observability | 完整監控 | Metrics、Logging、Tracing |
| NFR-06 | Consistency | 強一致性 | Config 變更後各端資料一致 |
| NFR-07 | Disaster Recovery | RPO < 1hr, RTO < 30min | 定期備份與故障復原 |
| NFR-08 | Auditability | 完整審計 | 所有操作可追蹤 |

### 3.1 第一期 Non-Functional Requirements

第一期以可交付、可展示、可維護為優先，非功能需求目標如下：

| ID | 需求 | 第一期目標 |
|----|------|------------|
| P1-NFR-01 | Availability | 單機部署，工作時間可用；故障時可由備份還原 |
| P1-NFR-02 | Performance | 30 個專案、5,000 筆 Config 下，主要查詢 P95 < 500ms |
| P1-NFR-03 | Security | HTTPS、密碼 Hash、敏感 Config 加密儲存與 UI 遮蔽 |
| P1-NFR-04 | Auditability | 所有新增、修改、審核、匯出操作皆寫入 Audit Log |
| P1-NFR-05 | Maintainability | Backend 採模組化單體，日後可拆分服務 |
| P1-NFR-06 | Backup | PostgreSQL 每日備份，保留至少 7 天 |

---

## 4. Estimate Usage（規模估算）

### 4.1 第一期 — 初期（單一團隊導入）

| 指標 | 估算值 |
|------|--------|
| DAU | ~50 人（1-2 個開發團隊） |
| 管理專案數 | ~30 個 |
| 每人每日操作次數 | ~10 次（查詢 7 次 + 修改 3 次） |
| 每日總請求量 | ~500 requests/day |
| 峰值 QPS | ~1 QPS |
| Config 資料量 | ~5,000 筆 Config entries |
| 儲存量 | ~100 MB（含版本歷史） |

**架構策略：** 單節點部署，PostgreSQL 單機，足以應付。

### 4.2 第二期 — 成長期（部門全面導入）

| 指標 | 估算值 |
|------|--------|
| DAU | ~300 人（全部門工程師） |
| 管理專案數 | ~300 個 |
| 每人每日操作次數 | ~20 次（查詢 12 次 + 修改 5 次 + 比對 3 次） |
| 每日總請求量 | ~6,000 requests/day |
| 峰值 QPS | ~5 QPS |
| Config 資料量 | ~50,000 筆 Config entries |
| 儲存量 | ~2 GB（含版本歷史） |

**架構策略：** 雙節點 HA、Redis Cache、Read Replica。

### 4.3 第三期 — 高流量（企業級 + API 自動化整合）

| 指標 | 估算值 |
|------|--------|
| DAU | ~1,000 人 + CI/CD Pipeline 自動化呼叫 |
| 管理專案數 | ~1,000 個 |
| 每人每日操作次數 | ~30 次 |
| API 自動化呼叫 | ~50,000 requests/day（CI/CD 觸發） |
| 每日總請求量 | ~80,000 requests/day |
| 峰值 QPS | ~50 QPS |
| Config 資料量 | ~500,000 筆 Config entries |
| 儲存量 | ~20 GB（含版本歷史） |

**架構策略：** K8s 多副本、資料庫分區、CDN 快取、Event-Driven 架構。

> **註：** 本系統為企業內部工具，非面向消費者，因此 DAU 規模以企業員工與自動化系統為主。Config 系統的特性是「讀多寫少」，讀寫比約 8:2。

---

## 5. Application Architecture

### 5.1 架構風格選擇

採用 **Modular Monolith → Microservices** 漸進式架構：

- **第一期：** Modular Monolith（模組化單體），降低初期複雜度
- **第二期：** 抽出核心模組為獨立服務（Config Service、Auth Service）
- **第三期：** 完整 Microservices + Event-Driven Architecture

### 5.2 核心模組設計

![Application Architecture Diagram](images/app_architecture.png)

核心模組說明：

| 模組 | 職責 |
|------|------|
| **Config Service** | Config CRUD、版本控制、繼承解析、差異比對、匯入匯出 |
| **Auth Service** | SSO/LDAP 登入、RBAC 權限控制、審核流程管理 |
| **Notification Service** | Email / Webhook / Slack 通知、事件監聽 |
| **API Gateway (Kong)** | 流量控制、身份驗證、路由分發 |

資料儲存層：

| 元件 | 用途 |
|------|------|
| PostgreSQL (Config DB) | 儲存 Config entries、版本歷史、模板 |
| PostgreSQL (Auth DB) | 儲存使用者、角色、審核記錄 |
| Redis | Config 讀取快取（TTL: 5min） |
| RabbitMQ | 事件驅動通知（Config 變更事件） |
| HashiCorp Vault | 敏感資料加密金鑰管理 |

### 5.2.1 第一期模組切分

第一期仍使用單一 Backend Application，但在程式碼內以模組化方式切分，避免日後重構成本過高。

| 模組 | 第一期職責 | 未來演進 |
|------|------------|----------|
| Project Module | 專案註冊、專案成員、環境設定 | 可擴充 Git Provider 整合 |
| Template Module | 模板 CRUD、模板套用、產生初始 Config | 可擴充企業級模板版本管理 |
| Config Module | Config CRUD、匯入/匯出、敏感值遮蔽 | 可獨立為 Config Service |
| Validation Module | 必填、型別、重複 Key、環境差異規則 | 可擴充 Policy Engine |
| Diff Module | 環境間 Key-Value 差異比對 | 可擴充跨專案 Diff |
| Review Module | Prod 變更申請、Approve/Reject | 可擴充多層簽核 |
| Auth/RBAC Module | 帳號、角色、專案權限 | 可串接 LDAP/SSO |
| Audit Module | 記錄操作歷史與版本歷史 | 可串接 SIEM/ELK |

第一期實作邊界：

- Backend 採 Modular Monolith，不拆微服務
- Database 只使用 PostgreSQL
- Sensitive Config 先使用應用程式層加密，並在 UI 預設遮蔽
- Notification 先以系統內待審核列表呈現，Email/Slack 放第二期
- Deploy 先輸出 Config Bundle，不直接控制 VM 或 K8s

### 5.3 API 設計（RESTful）

| Method | Endpoint | 說明 |
|--------|----------|------|
| GET | `/api/v1/projects` | 取得專案列表 |
| GET | `/api/v1/projects/{id}/configs` | 取得專案的 Config |
| GET | `/api/v1/projects/{id}/configs?env=prod` | 依環境篩選 Config |
| POST | `/api/v1/configs` | 新增 Config entry |
| PUT | `/api/v1/configs/{id}` | 修改 Config |
| GET | `/api/v1/configs/{id}/history` | 查詢版本歷史 |
| POST | `/api/v1/configs/{id}/rollback/{version}` | 回滾到指定版本 |
| GET | `/api/v1/configs/diff?env1=dev&env2=prod` | 環境差異比對 |
| POST | `/api/v1/configs/import` | 批量匯入 |
| GET | `/api/v1/configs/export?format=yaml` | 匯出 Config 檔案 |
| POST | `/api/v1/templates` | 建立模板 |
| POST | `/api/v1/reviews` | 提交審核請求 |
| PUT | `/api/v1/reviews/{id}/approve` | 審核通過 |
| GET | `/api/v1/shared-configs` | 取得共享 Config |

### 5.3.1 第一期 API 清單

第一期 API 以網站操作所需為主，先保留簡潔穩定的介面。

| Method | Endpoint | 說明 |
|--------|----------|------|
| POST | `/api/v1/auth/login` | 使用者登入 |
| GET | `/api/v1/me` | 取得目前使用者與角色 |
| GET | `/api/v1/projects` | 取得可存取專案 |
| POST | `/api/v1/projects` | 註冊新專案 |
| GET | `/api/v1/projects/{projectId}` | 取得專案設定 |
| PUT | `/api/v1/projects/{projectId}` | 更新專案基本資料 |
| GET | `/api/v1/templates` | 取得模板列表 |
| POST | `/api/v1/templates` | 建立模板 |
| POST | `/api/v1/projects/{projectId}/apply-template` | 對專案套用模板 |
| GET | `/api/v1/projects/{projectId}/configs?env=dev` | 查詢指定環境 Config |
| PUT | `/api/v1/projects/{projectId}/configs/{configId}` | 修改 Config |
| POST | `/api/v1/projects/{projectId}/configs/import` | 匯入 Config 檔案 |
| GET | `/api/v1/projects/{projectId}/configs/export?env=prod&format=yaml` | 匯出 Config |
| GET | `/api/v1/projects/{projectId}/diff?source=staging&target=prod` | 環境差異比對 |
| POST | `/api/v1/projects/{projectId}/validate` | 執行 Config 驗證 |
| GET | `/api/v1/projects/{projectId}/versions` | 查詢版本歷史 |
| POST | `/api/v1/change-requests` | 建立 prod 變更申請 |
| PUT | `/api/v1/change-requests/{requestId}/approve` | 審核通過 |
| PUT | `/api/v1/change-requests/{requestId}/reject` | 審核退回 |

### 5.4 資料模型設計

```
┌──────────────┐     ┌──────────────────┐     ┌──────────────┐
│   Project    │     │   Config Entry   │     │ Config       │
│              │1───N│                  │1───N│ Version      │
│ id           │     │ id               │     │              │
│ name         │     │ project_id (FK)  │     │ id           │
│ description  │     │ environment      │     │ config_id(FK)│
│ owner_id     │     │ key              │     │ version      │
│ created_at   │     │ value            │     │ value        │
│ updated_at   │     │ value_type       │     │ changed_by   │
└──────────────┘     │ is_sensitive     │     │ change_reason│
                     │ is_inherited     │     │ created_at   │
                     │ source_config_id │     └──────────────┘
                     │ format (json/    │
                     │  yaml/properties)│
                     │ created_at       │
                     │ updated_at       │
                     └──────────────────┘

┌──────────────┐     ┌──────────────────┐     ┌──────────────┐
│   Template   │     │  Review Request  │     │   User       │
│              │     │                  │     │              │
│ id           │     │ id               │     │ id           │
│ name         │     │ config_id (FK)   │     │ username     │
│ description  │     │ requester_id     │     │ email        │
│ content      │     │ reviewer_id      │     │ role         │
│ format       │     │ status (pending/ │     │ team_id      │
│ created_by   │     │  approved/       │     │ created_at   │
│ created_at   │     │  rejected)       │     └──────────────┘
└──────────────┘     │ comment          │
                     │ created_at       │
┌──────────────┐     └──────────────────┘
│ Shared Config│
│              │
│ id           │
│ key          │
│ value        │
│ description  │
│ used_by[]    │ ← 關聯到多個 Project
│ created_at   │
└──────────────┘
```

### 5.4.1 第一期資料表設計

第一期建議先用以下核心資料表即可完成主要流程。

| Table | 主要欄位 | 用途 |
|-------|----------|------|
| `users` | `id`, `username`, `email`, `password_hash`, `system_role`, `created_at` | 使用者與系統角色 |
| `projects` | `id`, `name`, `repo_url`, `description`, `owner_id`, `default_format`, `created_at` | 專案註冊資料 |
| `project_members` | `project_id`, `user_id`, `project_role` | 專案層級 RBAC |
| `project_environments` | `id`, `project_id`, `name`, `sort_order` | dev/staging/prod 等環境 |
| `templates` | `id`, `name`, `description`, `format`, `created_by`, `created_at` | Config 模板主檔 |
| `template_items` | `id`, `template_id`, `key`, `default_value`, `value_type`, `required`, `is_sensitive`, `description` | 模板中的 Key 定義 |
| `config_entries` | `id`, `project_id`, `environment`, `key`, `value_encrypted`, `value_type`, `is_sensitive`, `updated_by` | 實際 Config Key-Value |
| `config_versions` | `id`, `config_id`, `old_value`, `new_value`, `changed_by`, `change_reason`, `created_at` | 版本歷史 |
| `change_requests` | `id`, `project_id`, `environment`, `status`, `requester_id`, `reviewer_id`, `reason`, `created_at` | Prod 變更審核 |
| `audit_logs` | `id`, `actor_id`, `action`, `resource_type`, `resource_id`, `metadata`, `created_at` | 操作審計 |

第一期 Index 建議：

| Table | Index | 目的 |
|-------|-------|------|
| `config_entries` | `(project_id, environment, key)` unique | 避免同環境重複 Key，加速查詢 |
| `config_versions` | `(config_id, created_at DESC)` | 加速版本歷史查詢 |
| `change_requests` | `(project_id, status, created_at DESC)` | 加速待審核列表 |
| `project_members` | `(project_id, user_id)` unique | 加速權限檢查 |

### 5.5 Config 繼承與覆蓋機制

![Config Inheritance & Override Mechanism](images/config_inheritance.png)

解析邏輯（優先級由高到低）：

1. 載入 **Global Default Config**（全域預設值）
2. 合併 **Shared Config**（跨專案共用設定）
3. 合併 **Project Config**（專案層級設定）
4. 最後覆蓋 **Environment Config**（環境層級設定，優先級最高）
5. 若有 `is_inherited = true`，從 `source_config_id` 取值

> 此設計確保環境特定配置（如 prod 的 DB 連線）永遠優先，同時透過繼承機制減少重複設定。

### 5.6 審核流程設計

![Config Change Approval Workflow](images/approval_workflow.png)

流程說明：

1. **Engineer** 在介面上修改 Config
2. 系統自動建立 **Review Request**（status: pending）
3. 第一期先在網站內顯示待審核清單；第二期再透過 Email / Slack Webhook **通知 Reviewer**
4. **Reviewer 審核**：
   - **Approve** → 套用 Config 變更 → 記錄版本歷史 → 觸發通知
   - **Reject** → 附上 comment 通知 Engineer → 回到修改階段
5. 所有操作記錄於 **Audit Log**，可完整追蹤

### 5.7 敏感資料安全設計

| 層級 | 機制 |
|------|------|
| 傳輸層 | TLS 1.3 加密 |
| 應用層 | 敏感 Config 標記 `is_sensitive = true` |
| 儲存層 | AES-256-GCM 加密後存入 DB |
| 密鑰管理 | HashiCorp Vault 管理 encryption key |
| 存取控制 | 僅特定 Role 可查看敏感值 |
| 審計 | 所有敏感資料存取記錄於 Audit Log |

> 第一期可先使用應用程式層加密與環境變數管理 encryption key；Vault 屬於第二期安全強化項目。

---

## 6. System Architecture（依不同時期）

### 6.1 第一期 — MVP 單體部署架構

第一期採用單台 VM 或 Docker Compose 部署，讓系統可以快速交付、容易展示、容易維護。

![Phase 1 System Architecture](images/system_arch_phase1.png)

```
                    ┌─────────────┐
                    │   Users     │
                    │  (Browser)  │
                    └──────┬──────┘
                           │ HTTPS
                           ▼
                    ┌─────────────┐
                    │   Nginx     │
                    │ Reverse     │
                    │ Proxy + SSL │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              ▼                         ▼
     ┌─────────────────┐      ┌─────────────────┐
     │  Frontend       │      │  Backend API    │
     │  React SPA      │      │  Modular        │
     │  Static Files   │      │  Monolith       │
     └─────────────────┘      └───────┬─────────┘
                                      │
                         ┌────────────┼────────────┐
                         ▼                         ▼
                  ┌──────────────┐        ┌────────────────┐
                  │ PostgreSQL   │        │ Local/Object   │
                  │ Config DB    │        │ Storage        │
                  └──────────────┘        │ Export Bundle  │
                                          └────────────────┘
```

第一期部署策略：

| 項目 | 設計 |
|------|------|
| 部署方式 | Docker Compose on VM |
| Frontend | React SPA build 後由 Nginx 提供靜態檔案 |
| Backend | 單一 API Server，內部以模組化方式切分 |
| Database | PostgreSQL single instance |
| File Storage | 匯入檔暫存、匯出 Config Bundle，可先使用本機 volume |
| Backup | `pg_dump` 每日備份，匯出檔定期清理 |
| Secrets | 使用環境變數提供 App Encryption Key，DB 內儲存加密後敏感值 |
| Observability | Application log + Nginx access log + DB backup log |

第一期不部署 Redis、RabbitMQ、Vault、API Gateway，也不需要 Kubernetes。這些元件會在第二期依實際使用量與安全需求導入。

---

### 6.2 第二期 — 高可用架構

```
                         ┌─────────────┐
                         │   Users     │
                         └──────┬──────┘
                                │
                         ┌──────▼──────┐
                         │    CDN      │
                         │(CloudFront) │
                         └──────┬──────┘
                                │
                         ┌──────▼──────┐
                         │    ALB      │
                         │(Load       │
                         │ Balancer)   │
                         └──────┬──────┘
                                │
                    ┌───────────┼───────────┐
                    ▼                       ▼
            ┌──────────────┐       ┌──────────────┐
            │  API Server  │       │  API Server  │
            │  Instance 1  │       │  Instance 2  │
            └──────┬───────┘       └──────┬───────┘
                   │                      │
                   └──────────┬───────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
      ┌──────────────┐┌──────────────┐┌──────────────┐
      │ PostgreSQL   ││    Redis     ││  RabbitMQ    │
      │ Primary +    ││  Sentinel   ││  (Event Bus) │
      │ Read Replica ││  (HA Cache) ││              │
      └──────────────┘└──────────────┘└──────────────┘
              │
              ▼
      ┌──────────────┐
      │ HashiCorp    │
      │ Vault (HA)   │
      └──────────────┘
```

**雲服務對應（AWS 為例）：**

| 元件 | AWS 服務 |
|------|----------|
| Load Balancer | Application Load Balancer (ALB) |
| API Server | ECS Fargate / EKS |
| Database | Amazon RDS PostgreSQL (Multi-AZ) |
| Cache | Amazon ElastiCache (Redis) |
| Message Queue | Amazon MQ (RabbitMQ) |
| Secret Management | AWS Secrets Manager / Vault on EC2 |
| CDN | CloudFront |
| Monitoring | CloudWatch + Grafana |
| Object Storage | S3（備份、Config 匯出檔案） |

> **【需補圖片】** 使用 AWS Architecture Icons 繪製第二期架構圖。

---

### 6.3 第三期 — 企業級高流量架構

```
                              ┌──────────────┐
                              │   Users +    │
                              │   CI/CD      │
                              │   Pipelines  │
                              └──────┬───────┘
                                     │
                              ┌──────▼───────┐
                              │   CloudFront │
                              │   (CDN)      │
                              └──────┬───────┘
                                     │
                              ┌──────▼───────┐
                              │  API Gateway │
                              │  (Kong)      │
                              │  Rate Limit  │
                              │  Auth        │
                              └──────┬───────┘
                                     │
            ┌────────────────────────┼────────────────────────┐
            ▼                        ▼                        ▼
   ┌────────────────┐      ┌────────────────┐      ┌────────────────┐
   │ Config Service │      │ Auth Service   │      │ Notification   │
   │ (K8s: 3 pods)  │      │ (K8s: 2 pods)  │      │ Service        │
   │                │      │                │      │ (K8s: 2 pods)  │
   │ - Config CRUD  │      │ - SSO/LDAP     │      │ - Email        │
   │ - Versioning   │      │ - RBAC         │      │ - Webhook      │
   │ - Inheritance  │      │ - Approval     │      │ - Slack        │
   │ - Diff/Export  │      │                │      │                │
   └───────┬────────┘      └───────┬────────┘      └───────┬────────┘
           │                       │                        │
           │               ┌───────▼────────┐               │
           │               │  Auth DB       │               │
           │               │  (RDS)         │               │
           │               └────────────────┘               │
           ▼                                                ▼
   ┌────────────────┐                              ┌────────────────┐
   │  Config DB     │                              │  RabbitMQ      │
   │  (RDS Multi-AZ)│                              │  Cluster       │
   │  + Read Replica│                              │  (Event Bus)   │
   └────────────────┘                              └────────────────┘
           │
   ┌───────┴───────┐
   ▼               ▼
┌────────┐  ┌──────────┐  ┌──────────────┐
│ Redis  │  │ Vault    │  │ S3           │
│ Cluster│  │ (HA)     │  │ (Backup/     │
│        │  │          │  │  Export)      │
└────────┘  └──────────┘  └──────────────┘

Observability Stack:
┌────────────────────────────────────────────┐
│  Prometheus + Grafana (Metrics)            │
│  ELK Stack (Logging)                       │
│  Jaeger (Distributed Tracing)              │
└────────────────────────────────────────────┘
```

---

## 7. 關鍵架構設計決策

### 7.1 Config 儲存與讀取效率

| 策略 | 說明 |
|------|------|
| DB Indexing | 第一期先對 `project_id`, `environment`, `key` 建立複合索引 |
| Batch API | 批量取得 Config，減少前端 API 呼叫次數 |
| 預計算繼承 | 第二期可將 Config 繼承關係在寫入時預先計算 merged result |
| Redis 快取 | 第二期導入，讀取 Config 先查 Cache（TTL: 5min），Cache Miss 再查 DB |
| Read Replica | 第二期導入讀寫分離，讀取走 Replica |

### 7.2 權限管理設計 (RBAC)

| Role | 權限 |
|------|------|
| Viewer | 檢視非敏感 Config |
| Editor | 新增/修改 Config（需提交 Review） |
| Reviewer | 審核 Config 變更 |
| Project Admin | 管理專案層級設定與成員 |
| System Admin | 全系統管理、共享 Config 管理 |

權限繼承：
```
System Admin → Project Admin → Reviewer → Editor → Viewer
```

### 7.3 高可用與容錯設計

| 機制 | 說明 |
|------|------|
| Multi-AZ 部署 | DB 與服務跨可用區部署 |
| Health Check | K8s Liveness/Readiness Probe |
| Circuit Breaker | 下游服務異常時快速失敗 |
| Graceful Degradation | Vault 不可用時使用本地加密快取 |
| Auto Scaling | K8s HPA 依 CPU/Memory 自動擴縮 |
| Backup | DB 每小時增量備份、每日全量備份 |

---

## 8. 系統測試與驗證策略

| 測試類型 | 範圍 | 工具 |
|----------|------|------|
| Unit Test | Service Layer 業務邏輯 | Jest / JUnit |
| Integration Test | API + DB + Cache 整合 | Supertest / TestContainers |
| E2E Test | 完整使用者流程 | Cypress / Playwright |
| Load Test | 壓力測試（模擬 50 QPS） | k6 / JMeter |
| Security Test | OWASP Top 10 掃描 | OWASP ZAP |

---

## 9. CI/CD 流程

```
  Developer Push Code
        │
        ▼
  GitHub Actions / GitLab CI
        │
        ├── Lint & Code Quality (ESLint, SonarQube)
        ├── Unit Tests
        ├── Integration Tests
        ├── Build Docker Image
        ├── Push to Container Registry (ECR)
        ├── Deploy to Staging (K8s)
        ├── E2E Tests on Staging
        │
        ▼
  Manual Approval Gate
        │
        ▼
  Deploy to Production (K8s Rolling Update)
        │
        ▼
  Post-Deploy Health Check
```

### 9.1 第一期 CI/CD 流程

第一期部署目標為單台 VM + Docker Compose，流程可先簡化如下：

```
  Developer Push Code
        │
        ▼
  GitHub Actions / GitLab CI
        │
        ├── Frontend Lint & Build
        ├── Backend Lint & Unit Tests
        ├── DB Migration Check
        ├── Build Docker Images
        ├── Push to Container Registry
        │
        ▼
  Manual Approval
        │
        ▼
  SSH to VM / Deploy Script
        │
        ├── docker compose pull
        ├── docker compose up -d
        └── Health Check
```

---

## 10. 監控與運維

### 10.1 Observability 三支柱

| 支柱 | 工具 | 監控項目 |
|------|------|----------|
| Metrics | Prometheus + Grafana | API 回應時間、QPS、錯誤率、DB 連線數 |
| Logging | ELK (Elasticsearch + Logstash + Kibana) | 應用日誌、Audit Log、錯誤追蹤 |
| Tracing | Jaeger / OpenTelemetry | 請求鏈路追蹤、跨服務延遲分析 |

### 10.2 告警策略

| 告警等級 | 條件 | 通知方式 |
|----------|------|----------|
| P0 (Critical) | API 可用率 < 99%、DB 不可用 | PagerDuty + 電話 |
| P1 (High) | API P95 > 500ms、錯誤率 > 5% | Slack + Email |
| P2 (Medium) | Cache Hit Rate < 80%、磁碟 > 80% | Slack |
| P3 (Low) | 非關鍵服務異常 | Email |

---

## 11. 技術選型總覽

| 層級 | 技術選擇 | 理由 |
|------|----------|------|
| Frontend | React + TypeScript | 生態系成熟、元件化開發 |
| Backend | Node.js (NestJS) 或 Java (Spring Boot) | 企業級框架、豐富的中介軟體支援 |
| Database | PostgreSQL | JSONB 支援多格式、強一致性、成熟穩定 |
| Cache | Redis | 高效能 Key-Value 快取 |
| Message Queue | RabbitMQ | 輕量級、支援多種 Exchange 模式 |
| Secret Management | HashiCorp Vault | 業界標準密鑰管理 |
| Container | Docker + Kubernetes | Cloud Native 標準 |
| API Gateway | Kong | 開源、Plugin 豐富 |
| CI/CD | GitHub Actions / GitLab CI | 與 Git 整合度高 |
| Monitoring | Prometheus + Grafana + ELK | 開源標準 Observability Stack |

### 11.1 第一期建議技術選型

為了讓第一期快速完成並降低學習與維運成本，建議先採用以下組合：

| 層級 | 第一期選擇 | 說明 |
|------|------------|------|
| Frontend | React + TypeScript | 建立管理後台、表單、Diff 視圖 |
| Backend | Node.js + NestJS | 模組化架構清楚，適合 REST API 與 RBAC |
| Database | PostgreSQL | 支援關聯資料與 JSONB，足以處理第一期資料量 |
| ORM | Prisma 或 TypeORM | 快速建立資料模型與 migration |
| Auth | JWT Session + bcrypt | 第一期帳密登入，預留 SSO 介面 |
| Config Parser | `yaml`, `properties-parser`, native JSON | 支援三種主要 Config 格式匯入/匯出 |
| Diff | 後端產生 structured diff，前端視覺化 | 先比對 key/value/type/sensitive flag |
| Deployment | Docker Compose | 單台 VM 即可部署 |
| CI/CD | GitHub Actions | Lint、Test、Build Docker Image |

第一期可以先不使用：

| 技術 | 延後到 |
|------|--------|
| Redis | 第二期讀取量增加後 |
| RabbitMQ | 第二期導入通知與非同步事件後 |
| Vault | 第二期安全要求提高後 |
| Kubernetes | 第二期或第三期需要 HA / Scale out 後 |
| Kong API Gateway | 第三期多服務化後 |

---

## 12. 總結

本系統採用漸進式架構演進策略，從 Modular Monolith 逐步演化為 Microservices：

- **第一期**：快速交付 MVP，單體架構滿足 30 專案管理需求
- **第二期**：引入 HA、快取與事件驅動，支撐 300 專案全面管理
- **第三期**：完整微服務化，整合 CI/CD Pipeline，支援企業級自動化 Config 管理

核心設計原則：
1. **安全優先** — 敏感資料加密 + RBAC + 審核流程
2. **可靠性** — Multi-AZ + 自動容錯 + 備份復原
3. **可擴展性** — 水平擴展 + 讀寫分離 + 快取策略
4. **可觀測性** — Metrics + Logging + Tracing 全面覆蓋
5. **良好體驗** — Apple-inspired Admin UI，讓複雜 Config 管理保持簡潔、清楚、可操作

---

## References

1. [什麼是架構圖表製作？ - AWS](https://aws.amazon.com/what-is/architecture-diagramming/)
2. [System Design 系統設計學習地圖](https://medium.com/bucketing/system-design-%E7%B3%BB%E7%B5%B1%E8%A8%AD%E8%A8%88%E5%AD%B8%E7%BF%92%E5%9C%B0%E5%9C%96-68f98e8c3df4)
3. [Application Architecture: 6 Common Patterns](https://www.codesee.io/learning-center/application-architecture)
4. [The System Design Primer - GitHub](https://github.com/donnemartin/system-design-primer)
5. [HashiCorp Vault Documentation](https://developer.hashicorp.com/vault/docs)
6. [Kubernetes Documentation](https://kubernetes.io/docs/)
