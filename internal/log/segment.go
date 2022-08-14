package log

import (
	"fmt"
	"os"
	"path/filepath"

	api "github.com/kyu08/distributed-services-with-go/api/v1"
	"google.golang.org/protobuf/proto"
)

type segment struct {
	store                  *store
	index                  *index
	baseOffset, nextOffset uint64
	config                 Config
}

func newSegment(dir string, baseOffset uint64, c Config) (*segment, error) {
	s := &segment{
		baseOffset: baseOffset,
		config:     c,
	}

	// ストアファイルをオープン
	storeFile, err := os.OpenFile(
		filepath.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".store")),

		os.O_RDWR|os.O_CREATE|os.O_APPEND,
		0600,
	)

	if err != nil {
		return nil, err
	}

	// ストアを作成
	if s.store, err = newStore(storeFile); err != nil {
		return nil, err
	}

	// インデックスファイルをオープン
	indexFile, err := os.OpenFile(
		filepath.Join(dir, fmt.Sprintf("%d%s", baseOffset, ".index")),
		os.O_RDWR|os.O_CREATE,
		0600,
	)

	if err != nil {
		return nil, err
	}
	// インデックスを作成
	if s.index, err = newIndex(indexFile, c); err != nil {
		return nil, err
	}

	// TODO: ここちゃんとわかってない(P.40)
	// 最後に、セグメントの次のオフセットを設定して、次に追加されるレコードのための準備 をします。
	// インデックスが空の場合、セグメントに追加される次のレコードが最初のレコードとなり、
	//  そのオフセットはセグメントのベースオフセットになります。インデックスに少なくとも一つ のエントリがある場合、
	// 次に書き込まれるレコードのオフセットはセグメントの最後のオフセット を使う必要があり、
	// ベースオフセットと相対オフセットの和に 1 を加算して得られます。
	// この後で 説明するメソッドをすべて書き終えた時点で、セグメントはログへの書き込みとログからの読み出 しの準備が整います。
	if off, _, err := s.index.Read(-1); err != nil {
		s.nextOffset = baseOffset
	} else {
		s.nextOffset = baseOffset + uint64(off) + 1
	}

	return s, nil
}

func (s *segment) Append(record *api.Record) (offset uint64, err error) {
	cur := s.nextOffset
	record.Offset = cur
	p, err := proto.Marshal(record)
	if err != nil {
		return 0, err
	}

	// ストアファイルにレコードを追加
	_, pos, err := s.store.Append(p)
	if err != nil {
		return 0, err
	}

	// インデックスファイルにレコードのオフセットと位置を書き込み
	if err = s.index.Write(uint32(s.nextOffset-uint64(s.baseOffset)), pos); err != nil {
		return 0, err
	}
	s.nextOffset++
	return cur, nil
}

func (s *segment) Read(off uint64) (*api.Record, error) {
	// 絶対オフセットを相対オフセットに変換してインデックスからレコードのオフセットと位置を読み出し
	_, pos, err := s.index.Read(int64(off - s.baseOffset))
	if err != nil {
		return nil, err
	}
	// 位置を使ってストアからレコードを取得
	p, err := s.store.Read(pos)
	if err != nil {
		return nil, err
	}

	// アンマーシャルして返却
	record := &api.Record{}
	err = proto.Unmarshal(p, record)
	return record, err
}

func (s *segment) IsMaxed() bool {
	fmt.Printf("s.store.size: %v\n", s.store.size)
	return s.store.size >= s.config.Segment.MaxStoreBytes ||
		s.index.size >= s.config.Segment.MaxIndexBytes ||
		s.index.isMaxed()
}

func (s *segment) Remove() error {
	// インデックスとストアをそれぞれクローズしてから削除
	if err := s.Close(); err != nil {
		return err
	}
	if err := os.Remove(s.index.Name()); err != nil {
		return err
	}
	if err := os.Remove(s.store.Name()); err != nil {
		return err
	}

	return nil
}

func (s *segment) Close() error {
	// インデックスとストアをそれぞれクローズ
	if err := s.index.Close(); err != nil {
		return err
	}

	if err := s.store.Close(); err != nil {
		return err
	}

	return nil
}
