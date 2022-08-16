package log

import (
	"io"
	"os"

	"github.com/tysonmote/gommap"
)

const (
	offWidth uint64 = 4
	posWidth uint64 = 8
	entWidth        = offWidth + posWidth
)

type index struct {
	file *os.File
	mmap gommap.MMap
	size uint64
}

// 指定されたファイルからindexを作成
func newIndex(f *os.File, c Config) (*index, error) {
	idx := &index{
		file: f,
	}
	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}

	// ファイルの現在のサイズを保存
	idx.size = uint64(fi.Size())
	// ファイルを最大のインデックスサイズまで大きくする
	if err = os.Truncate(f.Name(), int64(c.Segment.MaxIndexBytes)); err != nil {
		return nil, err
	}

	// https://twitter.com/__syumai/status/1558354729163706369
	if idx.mmap, err = gommap.Map(idx.file.Fd(), gommap.PROT_READ|gommap.PROT_WRITE, gommap.MAP_SHARED); err != nil {
		return nil, err
	}

	return idx, nil
}

func (i *index) Close() error {
	// メモリにマップされたファイルのデータを永続化ｓれたファイルへ同期
	if err := i.mmap.Sync(gommap.MS_SYNC); err != nil {
		return err
	}

	// 永続化されたファイルの内容を安定したストレージに同期
	if err := i.file.Sync(); err != nil {
		return nil
	}

	// 永続化されたファイルをその中にある実際のデータ量まで切り詰めて
	if err := i.file.Truncate(int64(i.size)); err != nil {
		return err
	}

	// ファイルを閉じる
	return i.file.Close()
}

func (i *index) Read(in int64) (out uint32, pos uint64, err error) {
	// sizeがmaxだからもう書けない、みたいなことじゃないのか
	// -> というよりはiがファイルを持っているかを確認しているだけ？
	if i.size == 0 {
		return 0, 0, io.EOF
	}

	// in == -1のときは末尾の要素を返す
	if in == -1 {
		out = uint32((i.size / entWidth) - 1)
	} else {
		out = uint32(in)
	}
	// index * エントリーの幅 == 位置
	pos = uint64(out) * entWidth

	// インデックスファイルに存在しない位置は読めないのでio.EOFをかえす
	if i.size < pos+entWidth {
		return 0, 0, io.EOF
	}

	// オフセットを読み出してout変数に格納
	out = enc.Uint32(i.mmap[pos : pos+offWidth])
	// 位置を読み出してpos変数に格納
	pos = enc.Uint64(i.mmap[pos+offWidth : pos+entWidth])
	return out, pos, nil
}

func (i *index) Write(off uint32, pos uint64) error {
	//  エントリを書き込み領域があるか確認
	if i.isMaxed() {
		return io.EOF
	}

	// オフセット・位置をそれぞれ書き込み
	enc.PutUint32(i.mmap[i.size:i.size+offWidth], off)
	enc.PutUint64(i.mmap[i.size+offWidth:i.size+entWidth], pos)
	// サイズを更新
	i.size += uint64(entWidth)
	return nil
}

func (i *index) isMaxed() bool {
	return uint64(len(i.mmap)) < i.size+entWidth
}

func (i *index) Name() string {
	return i.file.Name()
}
