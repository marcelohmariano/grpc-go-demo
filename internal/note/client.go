package note

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	notev1 "github.com/marcelohmariano/grpc-go-demo/internal/gen/go/note/v1"
)

type APIClient struct {
	notev1.NoteAPIClient

	conn *grpc.ClientConn
}

func NewAPIClient(ctx context.Context, addr string) (*APIClient, error) {
	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	c := &APIClient{
		NoteAPIClient: notev1.NewNoteAPIClient(conn),
		conn:          conn,
	}
	return c, nil
}

func (c *APIClient) Close() {
	_ = c.conn.Close()
}
