# enbu

GitHub/GitLab だけで完結する**キーレス** `.env` 管理 CLI ツール（POC）

## 概要

```bash
enbu auth              # OAuth認証 + age鍵生成 + GHCR登録
enbu add KEY VALUE     # シークレット追加 → CI自動ビルド
enbu pull              # 暗号バンドル取得 → 署名検証 → 復号 → .env生成
```

## セットアップ

### 前提条件

- Go 1.22+
- GitHub OAuth App（Device Flow 有効）
- GitHub org/リポジトリ

### GitHub OAuth App 作成

1. https://github.com/settings/developers → "OAuth Apps" → "New OAuth App"
2. 入力内容:
   - Application name: `enbu`
   - Homepage URL: `https://github.com/<org>/<repo>`
   - Authorization callback URL: `http://localhost`
3. **"Enable Device Flow" にチェック**（これを忘れると Device Flow が動かない）
4. 生成された Client ID を `ENBU_CLIENT_ID` 環境変数に設定

### ビルド

```bash
go build -o enbu .
```

### 使い方

```bash
export ENBU_CLIENT_ID="your-client-id"

# 1. 認証（ブラウザでコード入力が必要）
enbu auth

# 2. シークレット追加
enbu add DATABASE_URL "postgres://localhost/dev"

# 3. バンドル取得
enbu pull
```

## アーキテクチャ

```
enbu auth  → GitHub OAuth Device Flow → age鍵生成 → GHCR push (公開鍵)
enbu add   → GitHub Secrets API (ENBU_BUNDLE) → repository_dispatch
CI         → recipient公開鍵pull → age暗号化 → cosign keyless署名 → GHCR push
enbu pull  → GHCR pull → age復号 → .env生成
```

## 開発時にハマったポイント

### 1. GHCR パッケージの visibility（最重要）

**問題**: GHCR に push した OCI アーティファクトはデフォルトで **private** になる。CI の `GITHUB_TOKEN` はリポジトリスコープなので、同じ org のパッケージでもアクセスできない。

**解決策**: GHCR のパッケージ設定で visibility を **public** に変更する。

- `ghcr.io/<org>/enbu-recipients` → Public
- `ghcr.io/<org>/enbu-bundle` → Public

または OCI manifest に `org.opencontainers.image.source` アノテーションを付けてリポジトリとリンクし、"Inherit access from source repository" を有効にする。

### 2. oras pull でファイルが取得できない

**問題**: `oras pull` はレイヤーに `org.opencontainers.image.title` アノテーション（ファイル名）がないとファイルを書き出さない。Go の oras-go ライブラリで push するとデフォルトでは title アノテーションが付かない。

**解決策**: ワークフロー側で `oras manifest fetch` + `oras blob fetch` を使って digest ベースで直接レイヤーを取得する。

```bash
DIGEST=$(oras manifest fetch "ghcr.io/<org>/enbu-recipients:<tag>" | jq -r '.layers[0].digest')
oras blob fetch --output - "ghcr.io/<org>/enbu-recipients@${DIGEST}"
```

### 3. repository_dispatch の client_payload が null だと 422

**問題**: GitHub API の `POST /repos/{owner}/{repo}/dispatches` で `client_payload` を `null` にすると `422 Invalid request` が返る。

**解決策**: 空でも必ず空オブジェクト `{}` を送る。

```go
if payload == nil {
    payload = make(map[string]string)
}
```

### 4. oras push が絶対パスを拒否する

**問題**: `oras push` に `/tmp/bundle.age` のような絶対パスを渡すと `absolute file path detected` エラーになる。

**解決策**: `cd /tmp` してから相対パスで push する。

```bash
cd /tmp
oras push "ghcr.io/<org>/enbu-bundle:default" \
  bundle.age:application/vnd.enbu.bundle.age.v1
```

### 5. GitHub Secrets API ではシークレットの値を読み取れない

**問題**: GitHub API はシークレットの名前一覧は取得できるが、値は取得できない（セキュリティ上の設計）。そのため `enbu add` で既存バンドルとマージができない。

**現状の回避策（POC）**: 毎回新しいバンドルを作成。本番では CI 側でマージするか、ローカルにバンドルのキー一覧をキャッシュする仕組みが必要。

### 6. GitHub Actions で secrets を列挙できない

**問題**: ワークフロー内で `secrets.*` を動的に列挙する方法がない。シークレットは YAML で明示的に参照する必要がある。

**解決策**: 全 key-value を JSON 化して単一シークレット `ENBU_BUNDLE` に格納。ワークフローは `${{ secrets.ENBU_BUNDLE }}` だけ参照すればよい。

## GHCR パッケージを private のまま運用する方法

POC では public に変更して動作確認したが、本番では private のまま運用可能。

### GITHUB_TOKEN が private パッケージにアクセスする条件

`GITHUB_TOKEN` は以下の条件を満たすと private パッケージにアクセスできる：

1. **パッケージがリポジトリにリンクされている**
2. **ワークフローに `packages: read`（または `write`）権限がある**

リンクさえされていれば、同じリポジトリのワークフローから `GITHUB_TOKEN` で読み書きできる。

### リンク方法（3パターン）

| 方法 | 対象パッケージ | 自動/手動 |
|------|--------------|-----------|
| ワークフローから GITHUB_TOKEN で初回 push | `enbu-bundle` | 自動 |
| OCI アノテーション `org.opencontainers.image.source` 付きで push | `enbu-recipients` | 自動 |
| GitHub UI で "Manage Actions access" → "Add Repository" | 両方 | 手動 |

### enbu-recipients（ユーザーが push）の場合

ユーザーの OAuth トークンで push するため、GITHUB_TOKEN による自動リンクが効かない。
以下のいずれかで対応：

**方法 A: OCI アノテーション（コード側で対応済み）**

`internal/oci/push.go` で `org.opencontainers.image.source` を manifest に付与している。
これにより push 時に GitHub が自動でリポジトリリンクを作成する。

```go
annotations["org.opencontainers.image.source"] = "https://github.com/a-kaibu/enbu-poc"
```

**方法 B: 手動設定（UI）**

1. https://github.com/orgs/a-kaibu/packages/container/enbu-recipients/settings
2. "Manage Actions access" → "Add Repository" → `enbu-poc` を追加（Read 権限）

### enbu-bundle（CI が push）の場合

ワークフローから `GITHUB_TOKEN` で初回 push すると自動でリポジトリにリンクされる。
`permissions: packages: write` が設定済みなのでそのまま動く。

### 手順まとめ（private 運用）

1. `enbu auth` を実行（OCI アノテーション付きで push される）
2. GitHub UI でパッケージ設定を確認:
   - `enbu-recipients`: "Manage Actions access" で `enbu-poc` に Read 付与
   - `enbu-bundle`: 初回 CI push で自動リンク（設定不要）
3. 両パッケージの visibility は private のまま
4. `enbu pull` はユーザーの OAuth トークンで認証するので private でも問題なし

### 注意点

- Fine-grained PAT は Packages API 未対応（classic PAT が必要）
- org 設定で "Allow GitHub Actions to create and approve pull requests" が有効な必要がある場合あり
- パッケージを別リポジトリのワークフローからも読みたい場合は、明示的に "Add Repository" が必要

## POC の制約

- トークンリフレッシュ未実装
- シークレットのローテーション/バージョニングなし
- アクセス取り消し（re-encrypt without revoked user）なし
- 単一 org 前提
- cosign 署名検証は `enbu pull` 側で未実装（CI 側の署名のみ）
- SBOM 生成は最小限

## ライセンス

TBD
