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

# 5章 安全なサービスの構築

セキュリティはとても重要でありながら無視されやすい
安全なソリューションを作ることはセキュリティを考慮せずにソリューションを作ることよりも複雑だが、人々が実際に使うものを作るのであればそれは安全でなければならない
また、完成したプロジェクトにセキュリティを後付けするよりも、最初からセキュリティを組み込む方がはるかに簡単であるため、最初からセキュリティを念頭においておく必要がある
私(筆者)はSaaSスタートアップの企業を数社立ち上げた経験から自分のサービスのセキュリティをそのサービスが解決する問題と同じくらい重要だと考えるようになった。その理由は次の通り。
- セキュリティはハッキングからあなたを救います。セキュリティのベストプラクティスに従わないと、侵入や漏洩が驚くほど定期的かつ深刻に発生します
- セキュリティは契約を勝ち取ります。私が手がけたソフトウェアを潜在的な顧客が購入するかの最も重要な要因はそのソフトウェアが何らかのセキュリティ要件を満たしているかどうかです(ドメインにもよりそうだけど弊社でもすごく重要だと思う)
- セキュリティを後から追加するのは大変無ことです。それに比べて最初からセキュリティ昨日を構築するのは比較的容易です。

## 5.1 安全なサービスのための3ステップ
分散サービスにおけるセキュリティは3ステップに分けられる
1. 中間者攻撃から保護するために通信データの暗号化を行う
1. クライアントを識別するために認証を行う
1. 識別されたクライアントの権限を決定するために認可を行う
1つずつ説明していく

### 5.1.1 通信データの暗号化
- 中間者(MITM)攻撃とは
  - 通信データを暗号化することで中間者(MITM: man-in-the-middle)攻撃を防ぐことができる
  - MITM攻撃の例として攻撃者が2人の被害者とそれぞれ独立したコネクションを確立し実際には攻撃者によって会話が制御されているにもかかわらず、被害者同士が直接会話しているように見せかける能動的盗聴がある。これは攻撃者が機密情報を知ることができるだけではなく、被害者間で送信されるメッセージを悪意を持って変更できるため、好ましくない。たとえばBobがPayPalを使ってAliceに送金しようとしたところMalloryが送金先をAliceの口座から自分の口座に変更してしまう、みたいなケース。
  - ここでの問題は本当に信頼できるサーバと通信しているのかが保証できていないこと
- MITM攻撃の防ぎ方
  - MITM攻撃を防ぎ、通信データを暗号化する技術として最も広く使われているのが、SSL(Secure Sockets Layer)の後継であるTLS(Transport Layer Security)である。
  - クライアントとサーバが通信する処理はTLSハンドシェイクによって開始される。
    1. 使われるTLSのバージョンを指定
    1. 使われる暗号スイート(暗号アルゴリズムの集まり)を決める
    1. サーバの秘密鍵と認証局のデジタル署名によりサーバの身元を確認(認証)する必要
    1. ハンドシェイクが完了した後、対称鍵暗号のためのセッションキーを生成する
  - これらの処理はTLSのライブラリが行う。開発者がここですべきことはクライアントとサーバが使う証明書を取得し、その証明書を使ってTLS通信を行うようにgRPCに指示すること。

### 5.1.2 クライアントを特定するための認証
- ほとんどのwebサービスでは、TLSを使って一方向認証を行い、サーバの認証のみを行う。クライアントの認証はアプリケーションに任されており、ユーザ名とパスワードの認証とトークンの組み合わせで行われる。
- TLSの相互認証は一般的に**双方向認証**とも呼ばれ、サーバとクライアントの両方が相手の通信を検証するものであり分散システムのようなマシン間の通信でよく使われる。この設定ではサーバとクライアントの両方が自分の身元を証明するために証明書を使う

### 5.1.3 クライアントの権限を決定するための認可
- 私たちのサービスではアクセスコントロールリスト(ACL)による認可に基づく認可を構築し、クライアントのログの書き込みや読み出し(またはその両方)を許可するかどうかを制御する
次節以降で実際に実装していく

## 5.2 TLSによるサーバの認証
### 5.2.1 CFSSLで独自のCAとして運用
- 第三者機関のCA(Certification Authority: 認証局)を使って証明書を取得することもできるが、費用がかかることもあるし手間もかかるのでここでは自分が運営するCAから発行する。
- ここではCloudFlareが開発してCFSSLというツールを使って証明書を発行する
```shell
# 次のコマンドで cfssl, cfssljsonをインストールする
$ go install github.com/cloudflare/cfssl/cmd/cfssl@v1.6.1
$ go install github.com/cloudflare/cfssl/cmd/cfssljson@v1.6.1
```
- cfsslはTLS証明書の署名・検証・バンドルを行い、結果をJSONとして出力する
- cfssljsonはJSON出力を受け取り、鍵・証明書・CSR・バンドルのファイルに分割する

- `/test/ca-csr.json`に以下の内容を記述する
```json
{
  "CN": "My Awesome CA",
  "key": {
    "algo": "rsa",
    "size": 2048
  },
  "names": [
    {
      "C": "CA",
      "L": "ON",
      "ST": "Toronto",
      "O": "My Awesome Company",
      "OU": "CA Services"
    }
  ]
}
```

- cfsslはこのファイルを使ってCAの証明書を設定する。
- CNはCommon Nameの略で私たちのCAをMy Awesome CAと呼んでいるよ
- keyは証明書に署名するためのアルゴリズムと鍵のサイズを指定する
- namesは証明書に追加されるさまざまな名前情報のリストで、**少なくとも1つのC、L、ST、O、OUの値を含める必要がある**
  - C: 国(Country)
  - L: 地域(locality)や自治体(市など)
  - ST: 州(state)や県
  - O: 組織(organization)
  - OU: 組織単位(organization unit、鍵の所有権を持つ部署など)
- CAのポリシーを定義するために次の内容の`test/ca-config.json`を作成する

```json
```


## 5.3 相互TLS認証によるクライアントの認証
## 5.4 アクセスコントロールリストによる認可
## 5.5 学んだこと
- サービスを安全にする3ステップの方法を学んだ
1. TLSによるコネクションの暗号化
1. クライアントとサーバの身元を確認するための相互TLS認証
1. ACLに基づく認可によるクライアントの操作許可
の3ステップである。
次の章では、メトリクス・ログ・トレースを追加してサービスを監視可能にする。


