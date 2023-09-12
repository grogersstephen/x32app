package main

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func Test_messagesOverDuration(t *testing.T) {
	type args struct {
		ctx          context.Context
		msgCount     int
		fadeDuration time.Duration
		readers      <-chan io.Reader
	}
	chs := make(chan io.Reader, 10)

	tests := []struct {
		name     string
		args     args
		wantConn string
		wantErr  bool
	}{
		// {
		// 	name: "should pass; 100 * 1",
		// 	args: args{
		// 		ctx:          context.Background(),
		// 		msgCount:     100,
		// 		fadeDuration: time.Second * 1,
		// 		readers:      chs,
		// 	},
		// },
		// {
		// 	name: "should pass; 1000 * 10",
		// 	args: args{
		// 		ctx:          context.Background(),
		// 		msgCount:     100,
		// 		fadeDuration: time.Second * 10,
		// 		readers:      chs,
		// 	},
		// },
		{
			name: "should pass; 256 * 2",
			args: args{
				ctx:          context.Background(),
				msgCount:     256,
				fadeDuration: time.Second * 2,
				readers:      chs,
			},
		},
		// {
		// 	name: "should pass; 1024 * 3",
		// 	args: args{
		// 		ctx:          context.Background(),
		// 		msgCount:     1024,
		// 		fadeDuration: time.Second * 3,
		// 		readers:      chs,
		// 	},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := time.Now()
			ctx, cancel := context.WithTimeout(tt.args.ctx, tt.args.fadeDuration+time.Millisecond*50500)
			defer cancel()

			go func() {
				defer close(chs)
				for i := 0; i < tt.args.msgCount; i++ {
					chs <- bytes.NewReader([]byte("test"))
				}
			}()

			conn := &bytes.Buffer{}
			if err := messagesOverDuration(ctx, conn, tt.args.msgCount, tt.args.fadeDuration, tt.args.readers); (err != nil) != tt.wantErr {
				t.Errorf("messagesOverDuration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			t.Logf("\nfinished in: %s\n", time.Since(ts))

			// if gotConn := conn.String(); gotConn != tt.wantConn {
			// 	t.Errorf("messagesOverDuration() = %v, want %v", gotConn, tt.wantConn)
			// }
		})
	}
}
