package util //nolint:revive

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/grafana/loki/v4/pkg/storage/chunk"
)

func BenchmarkGetParallelChunks(b *testing.B) {
	ctx := context.Background()
	in := make([]chunk.Chunk, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := GetParallelChunks(ctx, 150, in,
			func(_ context.Context, _ *chunk.DecodeContext, c chunk.Chunk) (chunk.Chunk, error) {
				return c, nil
			})
		if err != nil {
			b.Fatal(err)
		}
		if len(res) != len(in) {
			b.Fatal("unexpected number of chunk returned")
		}
	}
}

func TestGetParallelChunksPrefersStorageErrorOverMissing(t *testing.T) {
	in := make([]chunk.Chunk, 64)
	for i := range in {
		in[i].Fingerprint = uint64(i)
	}
	storageErr := errors.New("quorum not reached")
	for run := 0; run < 50; run++ {
		res, err := GetParallelChunks(context.Background(), 8, in,
			func(_ context.Context, _ *chunk.DecodeContext, c chunk.Chunk) (chunk.Chunk, error) {
				switch {
				case c.Fingerprint == 7:
					return c, storageErr
				case c.Fingerprint%5 == 0:
					return c, fmt.Errorf("%w: key", ErrChunkMissing)
				}
				return c, nil
			})
		if !errors.Is(err, storageErr) {
			t.Fatalf("run %d: expected storage error, got %v", run, err)
		}
		if len(res) != 64-13-1 {
			t.Fatalf("run %d: unexpected result size %d", run, len(res))
		}
	}
}

func TestGetParallelChunksAllMissing(t *testing.T) {
	in := make([]chunk.Chunk, 10)
	for i := range in {
		in[i].Fingerprint = uint64(i)
	}
	res, err := GetParallelChunks(context.Background(), 4, in,
		func(_ context.Context, _ *chunk.DecodeContext, c chunk.Chunk) (chunk.Chunk, error) {
			if c.Fingerprint < 3 {
				return c, fmt.Errorf("%w: key", ErrChunkMissing)
			}
			return c, nil
		})
	if !errors.Is(err, ErrChunkMissing) {
		t.Fatalf("expected missing error, got %v", err)
	}
	if len(res) != 7 {
		t.Fatalf("unexpected result size %d", len(res))
	}
}
