package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"golang.org/x/sync/errgroup"
)

func messagesOverDuration(ctx context.Context, conn io.Writer, msgCount int, fadeDuration time.Duration, readers <-chan io.Reader) error {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	eg, gctx := errgroup.WithContext(ctx)
	tickIntervals := fadeDuration / time.Duration(msgCount)
	ticker := time.NewTicker(tickIntervals)
	fmt.Println("tickIntervals", tickIntervals)

	for i := 0; i < 10; i++ {
		eg.Go(func() error {
			for reader := range readers {

				reader := reader

				select {
				case <-ticker.C:
					eg.Go(func() error {
						if _, err := io.Copy(conn, reader); err != nil {
							cancel()
							return err
						}
						return nil
					})
				case <-gctx.Done():
					return gctx.Err()
				}
			}
			return nil
		})
	}

	return eg.Wait()
}
