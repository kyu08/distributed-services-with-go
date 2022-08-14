package log

import (
	"fmt"
	"os"
	"path/filepath"
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
	if off, _, err := s.index.Read(-1); err != nil {
		s.nextOffset = baseOffset
	} else {
		s.nextOffset = baseOffset + uint64(off) + 1
	}

	return s, nil
}
