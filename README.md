### go versionについて
1.18.5だとうまく動かなかったので1.18をローカルにインストールした

<details>
<summary>以下経緯とやったこと</summary>

### 起きていたこと
`go test ./...`とか`go build ./cmd/server/main.go`とかすると以下のようなエラーが出てしまっていた。
<details>
<summary>エラー内容</summary>

```
# crypto/cipher
/opt/homebrew/Cellar/go/1.18.5/libexec/src/crypto/cipher/gcm.go:139:20: binary.BigEndian.Uint64 undefined (type binary.bigEndian has no field or method Uint64)
/opt/homebrew/Cellar/go/1.18.5/libexec/src/crypto/cipher/gcm.go:140:20: binary.BigEndian.Uint64 undefined (type binary.bigEndian has no field or method Uint64)
/opt/homebrew/Cellar/go/1.18.5/libexec/src/crypto/cipher/gcm.go:325:29: binary.BigEndian.Uint64 undefined (type binary.bigEndian has no field or method Uint64)
/opt/homebrew/Cellar/go/1.18.5/libexec/src/crypto/cipher/gcm.go:326:30: binary.BigEndian.Uint64 undefined (type binary.bigEndian has no field or method Uint64)
# crypto/md5
/opt/homebrew/Cellar/go/1.18.5/libexec/src/crypto/md5/md5.go:103:33: binary.BigEndian.Uint64 undefined (type binary.bigEndian has no field or method Uint64)
# math/big
/opt/homebrew/Cellar/go/1.18.5/libexec/src/math/big/nat.go:1185:32: binary.BigEndian.Uint64 undefined (type binary.bigEndian has no field or method Uint64)
```
</details>
go1.18.0環境のDockerコンテナを立てて同じコマンドを実行したところ問題なく実行できたのでgo1.18.5だとうまく動かないっぽいことがわかった。

### やったこと
```shell
which go
// /opt/homebrew/bin/go
brew uninstall go
which go
// /usr/local/go/bin/go
go version
// go version go1.18 darwin/amd64
// よさそう
```
</details>

# 2章 プロトコルバッファによる構造化データ
- 非公開のAPIを構築する場合、JSONに比べて生産性が高く、速く、多くの機能をもち、バグのすくないサービスを作ることができるデータの構造化と送信の機構を利用するべき
- その機構はプロトコルバッファ(Protocol Buffers: protobufとも)である。

## protobufの利点
- 型の安全性を保証する
- スキーマ違反を防ぐ
- 高速なシリアライズを可能にする
- 後方互換性を提供する

## protobufを使う理由
- 一貫性のあるスキーマ
- バージョン管理
- ボイラープレートコードの削減
  - protobufライブラリがエンコードとでコードを行うので自動生成されるので人間が書く必要がない
- 拡張性
- 言語寛容性
- パフォーマンス

※protoファイルからgoコードを生成する前に別途`$ brew install protoc-gen-go`する必要があった

# 3章 ログパッケージの作成
## ログは強力なツール
- ログは分散サービスを構築する上で最も重要なツールキット
- ログはDBのシステムの耐久性を高めるために仕組みだったり、reduxだったりとさまざまな場面で利用されている
- 完全なログは最新の状態だけでなく過去の全ての状態を保持している
- そのおかげで他の方法では構築するのが複雑になってしまう機能を構築できる

## ログの仕組み
### ログとは
- ログは追加専用のレコード列である。ログの最後にレコードを追加し、通常上から下へ、古いレコードから新しいレコードへ読んでいく
- ログにレコードを追加するとログはそのレコードに一意の連続したオフセット番号を割り当て、その番号がレコードのIDの役割を果たす。ログはレコードを常に時間順に並べ、**オフセット**と**作成時間**で各レコードにインデックスつけるテーブルのようなもの

## セグメント
- ディスク容量は無限ではないため、同じファイルに永遠にログを追加していくことはできない。そのため、ログをセグメントに分割する。ログが大きくなりすぎるとすでに処理してアーカイブした古いセグメントを削除してディスク容量を確保する
- セグメントの集まりの中には常に一つの特別なセグメントがあり、それが**アクティブ(active)**セグメントである。アクティブセグメントは活発(アクティブ)に’書き込む唯一のセグメントである。アクティブセグメントがいっぱいになると新たなセグメントを作成し、その新たなセグメントをアクティブセグメントにする

## ストアファイルとインデックスファイル
- 各セグメントはストア(**store**)ファイルとインデックス(**index**)ファイルで構成される。
- **ストアファイル**は**レコードデータを保存する場所**。このファイルに継続的にレコードを追加する
- **インデックスファイル**は**ストアファイル内の各レコードへのインデックスを保存する場所**。レコードのオフセットをストアファイル内の位置に対応付けることでストアファイルからのレコードの読み取りが速くなる

## オフセットが指定されたレコードの読み取りの手順
- オフセットが指定されたレコードの読み取りは2つの手順からなる
1. インデックスファイルからレコードのエントリを取得し、ストアファイル内でのレコードの位置を知る
1. ストアファイル内のその位置のレコードを読み取る

- インデックスファイルに必要なのは「レコードのオフセット」と「格納位置」という二つの小さなフィールドだけなので、インデックスファイルはストアファイルよりもはるかに小さい

## gommapとは
- goでmmapするためのライブラリ
> mmap() は、UNIXのシステムコールのひとつで、ファイルやデバイスなどのオペレーティングシステム (OS) 上のリソースの一部または全部を連続した仮想アドレス空間にマッピングする関数である。(wikipedia)
- 今回はパフォーマンスのためにファイルをメモリ上にマッピングして扱っているっぽい

## インデックスファイル内のイメージ

```
       offset部   position部        offset部   position部        offset部   position部        ...
...****===========******************===========******************===========******************...
      ↑ pos      ↑ pos+offWidth    ↑ pos+entWidth
      |          |                 |
      <----------><---------------->
        offWidth       posWidth
      <---------------------------->
               entWidth

```

# 4章 gRPCによるリクエスト処理
- サーバーを立ち上げる時に0番ポートを指定すると自動的に空いてるポートに割り振ってくれるらしい


